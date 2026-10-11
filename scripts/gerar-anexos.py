#!/usr/bin/env python3
"""Gera os anexos do artigo em PDF (pasta anexos/ na raiz do repositorio).

Cada anexo e montado como HTML numa pasta de trabalho e impresso em PDF A4 pelo
Chromium headless. As fontes de cada anexo sao arquivos versionados:

- diagramas: docs/article/anexos/diagramas/*.puml, renderizados pelo PlantUML
  (imagem Docker plantuml/plantuml) em SVG; o script tambem regrava os .png de
  alta resolucao ao lado dos .puml;
- esquema do banco (A9): os .puml esquema-banco-*.puml sao gerados aqui a
  partir de docs/schema/thingindex.dbml, nunca editados a mao;
- tabelas de recuperacao (A7): convertidas de docs/article/tables/*.tex, com
  verificacao de que todos os numeros do .tex aparecem no PDF, na mesma ordem;
- telas (A6): docs/article/anexos/telas/*.png.

Requer chromium, docker, rsvg-convert e pdftotext no PATH.

Uso: python3 scripts/gerar-anexos.py [pasta-de-trabalho]
"""

from __future__ import annotations

import html
import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
ART = RAIZ / "docs" / "article"
DIAG = ART / "anexos" / "diagramas"
TELAS = ART / "anexos" / "telas"
TAB = ART / "tables"
DBML = RAIZ / "docs" / "schema" / "thingindex.dbml"
SAIDA = RAIZ / "anexos"

LINHA_INSTITUCIONAL = ("ThingIndex – Trabalho de Conclusão de Curso – Sistemas de Informação – "
                       "IFPR Campus Palmas")
FONTES_LIBERATION = "/usr/share/fonts/liberation"

# Altura util da pagina (A4 menos margens), descontada uma folga de 2 mm para o
# Chromium nao empurrar a ultima linha para uma pagina em branco.
MARGEM = {"portrait": 20, "landscape": 14}
ALTURA_UTIL = {"portrait": 297 - 2 * 20 - 2, "landscape": 210 - 2 * 14 - 2}

CSS = """
@page { size: A4 %(orient)s; margin: %(margem)smm; }
* { box-sizing: border-box; }
html, body { background: #fff; }
body { font-family: "Liberation Serif", "Times New Roman", "DejaVu Serif", serif;
       font-size: 11pt; color: #000; line-height: 1.35; margin: 0; }
header { margin: 0 0 10pt 0; }
h1 { font-size: 14pt; font-weight: bold; margin: 0; }
.inst { font-size: 9pt; margin: 2pt 0 0 0; }
.desc { margin: 0 0 10pt 0; }
h2 { font-size: 12pt; font-weight: bold; margin: 16pt 0 6pt 0; break-after: avoid; }
p { margin: 0 0 6pt 0; text-align: justify; }
table { border-collapse: collapse; width: 100%%; margin: 0 0 4pt 0; font-size: 10pt; }
thead { display: table-header-group; }
tr { break-inside: avoid; }
th, td { padding: 2.5pt 6pt; vertical-align: top; text-align: left; }
thead th { border-top: 1.2pt solid #000; border-bottom: 0.6pt solid #000; font-weight: bold; }
tbody tr:last-child td { border-bottom: 1.2pt solid #000; }
tr.regra td { border-top: 0.6pt solid #000; }
td.id { white-space: nowrap; width: 1%%; }
td small { font-size: 8.5pt; white-space: nowrap; }
table.longa { font-size: 9.5pt; }
table.longa td { padding-top: 1.5pt; padding-bottom: 1.5pt; }
.titulo-tab { font-size: 10pt; margin: 14pt 0 4pt 0; text-align: justify; break-after: avoid; }
.bloco { break-inside: avoid; }
code { font-family: "Liberation Mono", "DejaVu Sans Mono", monospace; font-size: 0.9em; }
.folha { display: flex; flex-direction: column; overflow: hidden; }
.folha + .folha { break-before: page; }
figure { margin: 0; flex: 1 1 auto; min-height: 0; display: flex; flex-direction: column; }
figure .img { flex: 1 1 auto; min-height: 0; display: flex; align-items: center;
              justify-content: center; }
figure img { max-width: 100%%; max-height: 100%%; width: 100%%; height: 100%%;
             object-fit: contain; }
figcaption { font-size: 10pt; text-align: center; margin-top: 6pt; }
figure .fonte { font-size: 9pt; text-align: center; margin: 2pt 0 0 0; }
.pagina { break-before: page; }
.leitura { margin: 6pt 0 4pt 0; }
table.quadro { font-size: 10pt; }
table.quadro td { text-align: left; }
ul { margin: 0 0 6pt 0; padding-left: 16pt; }
li { margin: 0 0 3pt 0; text-align: justify; }
h3 { font-size: 11pt; font-weight: bold; margin: 0 0 1pt 0; break-after: avoid; }
.alteracao { margin: 12pt 0 0 0; padding-top: 6pt; border-top: 0.6pt solid #000; }
.alteracao p, .alteracao li { margin-bottom: 3pt; }
.quando { font-style: italic; font-size: 10pt; }
"""


def esc(s: str) -> str:
    return html.escape(s, quote=False)


def documento(cod: str, titulo: str, desc: str, corpo: str, orient: str = "portrait",
              cabecalho_na_folha: bool = False) -> str:
    """Monta o HTML de um anexo. Com cabecalho_na_folha, o corpo e uma lista de
    folhas de figura e o cabecalho entra na primeira delas."""
    css = CSS % {"orient": orient, "margem": MARGEM[orient]}
    cab = (f"<header><h1>Anexo {cod} – {esc(titulo)}</h1>"
           f"<p class='inst'>{LINHA_INSTITUCIONAL}</p></header>"
           + (f"<p class='desc'>{desc}</p>" if desc else ""))
    if cabecalho_na_folha:
        corpo = corpo.replace("<!--CABECALHO-->", cab, 1)
        cab = ""
    return (f'<!doctype html><html lang="pt-BR"><head><meta charset="utf-8">'
            f"<title>Anexo {cod} – {esc(titulo)}</title><style>{css}</style></head>"
            f"<body>{cab}{corpo}</body></html>")


def folha(img: Path, legenda: str, orient: str, primeira: bool, extra: str = "") -> str:
    """Uma pagina com uma figura que ocupa toda a area util restante."""
    altura = ALTURA_UTIL[orient]
    return (f"<section class='folha' style='height:{altura}mm'>"
            + ("<!--CABECALHO-->" if primeira else "")
            + f"<figure><div class='img'><img src='{img.as_uri()}' alt=''></div>"
            f"<figcaption>{legenda}</figcaption>"
            + (f"<p class='fonte'>{extra}</p>" if extra else "")
            + "</figure></section>")


def imprimir(html_txt: str, nome: str, build: Path) -> Path:
    fonte = build / (nome + ".html")
    fonte.write_text(html_txt, encoding="utf-8")
    pdf = SAIDA / (nome + ".pdf")
    subprocess.run(["chromium", "--headless", "--disable-gpu", "--no-pdf-header-footer",
                    "--allow-file-access-from-files", f"--print-to-pdf={pdf}",
                    fonte.as_uri()], check=True, capture_output=True)
    texto = subprocess.run(["pdftotext", str(pdf), "-"], check=True,
                           capture_output=True, text=True).stdout
    proibido = re.findall(r"(?i)html|xlsx|planilha", texto)
    if proibido:
        raise SystemExit(f"{pdf.name}: menciona {sorted(set(proibido))}")
    print("gerado", pdf.relative_to(RAIZ))
    return pdf


# ------------------------------------------------------------ PlantUML

def plantuml(arquivos: list[Path], build: Path) -> dict[str, Path]:
    """Renderiza os .puml em SVG (pasta build/svg) e regrava o .png ao lado de
    cada .puml. Monta as fontes Liberation no conteiner para que o PlantUML
    meça o texto com a mesma fonte usada no PDF."""
    svg = build / "svg"
    svg.mkdir(parents=True, exist_ok=True)
    cmd = ["docker", "run", "--rm", "-u", f"{os.getuid()}:{os.getgid()}",
           "-v", f"{DIAG}:/data", "-v", f"{svg}:/out"]
    if Path(FONTES_LIBERATION).is_dir():
        cmd += ["-v", f"{FONTES_LIBERATION}:/usr/share/fonts/truetype/liberation:ro"]
    cmd += ["plantuml/plantuml", "-tsvg", "-o", "/out"] + [f"/data/{a.name}" for a in arquivos]
    subprocess.run(cmd, check=True)
    saida = {}
    for a in arquivos:
        s = svg / (a.stem + ".svg")
        if not s.exists():
            raise SystemExit(f"PlantUML nao gerou {s.name}")
        if "Syntax Error" in s.read_text(encoding="utf-8"):
            raise SystemExit(f"erro de sintaxe em {a.name}")
        if not a.stem.startswith("esquema-banco-"):
            subprocess.run(["rsvg-convert", "-z", "3", "-b", "white", str(s),
                            "-o", str(DIAG / (a.stem + ".png"))], check=True)
        saida[a.stem] = s
    return saida


# --------------------------------------------------- DBML -> diagrama ER

NUCLEO = {"users", "document_categories", "collections", "documents"}

GRUPOS_BANCO = [
    ("nucleo", "núcleo do acervo e estrutura extraída dos documentos",
     ["users", "document_categories", "collections", "documents", "document_pages",
      "paragraph_nodes", "table_sections", "document_outline_nodes", "document_chunks",
      "metadata_pairs"], 0),
    ("processamento", "operação: processamento, auditoria e provedores",
     ["document_processing_attempts", "outbox_events", "document_audit_events",
      "user_provider_configs", "provider_model_catalogs", "provider_audit_events",
      "user_indexing_preferences", "enrichment_cache_entries"], 4),
    ("produto", "operação: preços de modelos, conversas e estado do workspace",
     ["pricing_sources", "model_prices", "model_price_histories", "chat_conversations",
      "chat_conversation_entries", "paragraph_highlights", "workspace_view_states"], 4),
]


def ler_dbml() -> tuple[dict[str, list[tuple[str, str, str]]], list[tuple[str, str, str, str]]]:
    dbml = DBML.read_text(encoding="utf-8")
    tabelas = {}
    for nome, corpo in re.findall(r"^Table (\w+) \{\n(.*?)^\}", dbml, re.M | re.S):
        cols = []
        for l in corpo.split("indexes {")[0].strip().splitlines():
            m = re.match(r"\s*(\w+)\s+(\S+)(?:\s+\[(.*)\])?", l)
            if m:
                cols.append((m.group(1), m.group(2), m.group(3) or ""))
        tabelas[nome] = cols
    refs = re.findall(r"^Ref:\s*(\w+)\.(\w+)\s*>\s*(\w+)\.(\w+)", dbml, re.M)
    declarado = re.search(r"Total: (\d+) tabelas", dbml)
    if declarado and int(declarado.group(1)) != len(tabelas):
        raise SystemExit("DBML: total declarado difere das tabelas lidas")
    return tabelas, refs


def gera_er() -> list[tuple[str, str, Path, list[str]]]:
    """Escreve um .puml de diagrama ER por grupo de tabelas e devolve
    (chave, titulo, caminho, tabelas)."""
    tabelas, refs = ler_dbml()
    agrupadas = [t for _, _, ts, _ in GRUPOS_BANCO for t in ts]
    if sorted(agrupadas) != sorted(tabelas):
        raise SystemExit("grupos do A9 desatualizados frente ao DBML: "
                         f"{sorted(set(tabelas) ^ set(agrupadas))}")
    fks = {(o, c) for o, c, _, _ in refs}
    saida = []
    for chave, titulo, nomes, colunas_grade in GRUPOS_BANCO:
        l = ["@startuml",
             "' GERADO por scripts/gerar-anexos.py a partir de docs/schema/thingindex.dbml.",
             "' Nao edite a mao.",
             "!include estilo.iuml",
             "skinparam defaultFontSize 13",
             "skinparam linetype ortho",
             "skinparam nodesep 40",
             "skinparam ranksep 50",
             "hide circle"]

        def entidade(nome: str, so_pk: bool = False) -> None:
            rotulo = nome + (" (Figura 1)" if so_pk else "")
            l.append(f'entity "{rotulo}" as {nome} {{')
            for c, tipo, attr in tabelas[nome]:
                pk = "pk" in [x.strip() for x in attr.split(",")]
                if so_pk and not pk:
                    continue
                marca = "<b>PK</b> " if pk else ("<i>FK</i> " if (nome, c) in fks else "")
                l.append(f"  {{field}} {marca}{c} : {tipo}")
                if pk and not so_pk:
                    l.append("  --")
            l.append("}")

        for n in nomes:
            entidade(n)
        externas = sorted({d for o, _, d, _ in refs if o in nomes and d not in nomes})
        for n in externas:
            entidade(n, so_pk=True)
        ligadas = set()
        # As tabelas do nucleo ficam numa linha (relacoes para a direita) e as
        # demais abaixo delas, para a figura caber numa pagina em paisagem.
        for o, c, d, _ in refs:
            if o in nomes:
                seta = "-right-" if o in NUCLEO and d in NUCLEO else "--"
                l.append(f"{d} ||{seta}o{{ {o}")
                ligadas |= {o, d}
        # Posse: user_id tem chave estrangeira para users e ja entra pelas
        # referencias acima, inclusive nos grupos sem a tabela users (que
        # aparece so com a PK).
        donas = [o for o, _, d, _ in refs if o in nomes and d == "users"]
        # Muitas tabelas penduradas no mesmo users viram uma faixa larga e
        # ilegivel; quebra em fileiras de 4 com ligacoes invisiveis.
        if "users" not in nomes and len(donas) > 4:
            for a, b in zip(donas, donas[4:]):
                l.append(f"{a} -[hidden]down- {b}")
        # O outbox aponta o documento por aggregate_id, campo generico do padrao.
        if "outbox_events" in nomes:
            if "documents" not in nomes and "documents" not in externas:
                entidade("documents", so_pk=True)
            l.append("documents ||..o{ outbox_events : aggregate_id")
            ligadas |= {"outbox_events", "documents"}
        if colunas_grade:  # tabelas soltas em grade, para a figura nao virar uma faixa
            soltas = [n for n in nomes if n not in ligadas]
            colunas = [soltas[i::colunas_grade] for i in range(colunas_grade)]
            for col in colunas:
                for a, b in zip(col, col[1:]):
                    l.append(f"{a} -[hidden]down- {b}")
        l.append("@enduml")
        caminho = DIAG / f"esquema-banco-{chave}.puml"
        caminho.write_text("\n".join(l) + "\n", encoding="utf-8")
        saida.append((chave, titulo, caminho, nomes))
    return saida


# ---------------------------------------------------------------- LaTeX -> HTML

def chave(s: str, i: int) -> tuple[str, int]:
    """Conteudo do grupo {...} que comeca em s[i] e indice apos ele."""
    assert s[i] == "{", s[i:i + 20]
    prof, j = 0, i
    while True:
        if s[j] == "{":
            prof += 1
        elif s[j] == "}":
            prof -= 1
            if prof == 0:
                return s[i + 1:j], j + 1
        j += 1


def separa(s: str, sep: str) -> list[str]:
    """Divide s em sep apenas fora de chaves."""
    partes, prof, ini, i = [], 0, 0, 0
    while i < len(s):
        c = s[i]
        if c == "{":
            prof += 1
        elif c == "}":
            prof -= 1
        elif prof == 0 and s.startswith(sep, i) and not (sep == "&" and s[i - 1] == "\\"):
            partes.append(s[ini:i])
            i += len(sep)
            ini = i
            continue
        i += 1
    partes.append(s[ini:])
    return partes


def matematica(m: str) -> str:
    if m == "'":
        return "′"
    m = re.sub(r"\\sqrt\{(.*?)\}", r"√\1", m)
    m = m.replace(r"\Delta", "Δ").replace(r"\rightarrow", "→").replace("-", "−")
    return re.sub(r"\b([A-Za-z])\b", r"<i>\1</i>", m)


def comando(s: str, nome: str, ini: str, fim: str) -> str:
    while (k := s.find("\\" + nome + "{")) >= 0:
        dentro, j = chave(s, k + len(nome) + 1)
        s = s[:k] + ini + dentro + fim + s[j:]
    return s


def inline(s: str) -> str:
    """Converte o LaTeX de uma celula ou legenda em HTML."""
    s = s.strip().replace("{,}", ",").replace(r"\%", "%")
    s = re.sub(r"\\label\{.*?\}", "", s)
    while (k := s.find(r"\shortstack{")) >= 0:
        dentro, j = chave(s, k + len(r"\shortstack"))
        a, b = (x.strip() for x in separa(dentro, r"\\"))
        b = b.replace(r"\scriptsize", "").strip()
        s = s[:k] + f"{a}<br><small>{b}</small>" + s[j:]
    s = s.replace("``", "“").replace("''", "”").replace("---", "—").replace("--", "–")
    s = re.sub(r"\$(.+?)\$", lambda m: matematica(m.group(1)), s)
    s = s.replace("~", " ")
    for nome, a, b in (("emph", "<em>", "</em>"), ("textit", "<em>", "</em>"),
                       ("textbf", "<strong>", "</strong>"), ("texttt", "<code>", "</code>")):
        s = comando(s, nome, a, b)
    return s


def alinhamento(spec: str) -> list[str]:
    spec = re.sub(r"p\{.*?\}", "l", spec)
    return [{"l": "left", "c": "center", "r": "right"}[c] for c in spec if c in "lcr"]


def corpo_tabular(caminho: Path) -> tuple[str, str]:
    """Legenda e corpo (entre \\begin e \\end) do tabular/longtable de um .tex."""
    s = "\n".join(l for l in caminho.read_text(encoding="utf-8").splitlines()
                  if not l.lstrip().startswith("%"))
    k = s.find(r"\caption{")
    legenda, _ = chave(s, k + len(r"\caption"))
    m = re.search(r"\\begin\{(tabular|longtable)\}\{([^}]*)\}", s)
    amb = m.group(1)
    corpo = s[m.end():s.find(r"\end{" + amb + "}")]
    if amb == "longtable":  # cabecalho do primeiro trecho + corpo apos \endhead
        cab = corpo[corpo.find(r"\toprule"):corpo.find(r"\endfirsthead")]
        corpo = cab + corpo[corpo.find(r"\endhead") + len(r"\endhead"):]
    corpo = re.sub(r"\\(caption|label)\{.*?\}\s*(\\\\)?", "", corpo)
    return legenda, m.group(2), corpo


def tabela_tex(caminho: Path, cabecalho: list[str] | None = None,
               rotulo=None) -> tuple[str, str]:
    """Converte table/tabular ou longtable de um .tex gerado em (legenda, <table>).

    cabecalho troca o texto das celulas do cabecalho (mesma quantidade); rotulo,
    se dado, e aplicado ao texto da primeira celula de cada linha. Os numeros
    das celulas nao mudam, e confere_numeros verifica isso no PDF."""
    legenda, spec, corpo = corpo_tabular(caminho)
    alin = alinhamento(spec)
    linhas, cab, regra, depois_cab = [], None, False, False
    for bruto in separa(corpo, r"\\"):
        t = bruto.replace(r"\toprule", "").replace(r"\bottomrule", "")
        if r"\midrule" in t:
            regra = True
            t = t.replace(r"\midrule", "")
        t = t.strip()
        if not t:
            continue
        html_cels, col = [], 0
        for c in separa(t, "&"):
            c = c.strip()
            span = 1
            mc = re.match(r"\\multicolumn\{(\d+)\}\{([^}]*)\}", c)
            if mc:
                span = int(mc.group(1))
                c, _ = chave(c, mc.end())
            a = alin[col] if col < len(alin) else "left"
            html_cels.append((inline(c), span, a))
            col += span
        if cab is None:
            if cabecalho is not None:
                if len(cabecalho) != len(html_cels):
                    raise SystemExit(f"{caminho.name}: cabecalho com {len(html_cels)} colunas")
                html_cels = [(n, sp, al) for n, (_, sp, al) in zip(cabecalho, html_cels)]
            cab, regra = html_cels, False
            continue
        if rotulo is not None:
            c0, sp0, al0 = html_cels[0]
            html_cels[0] = (rotulo(c0), sp0, al0)
        linhas.append((html_cels, regra and depois_cab))
        regra, depois_cab = False, True
    th = "".join(f'<th colspan="{sp}" style="text-align:{a}">{c}</th>' for c, sp, a in cab)
    tr = "".join(
        f'<tr{" class=regra" if r else ""}>' +
        "".join(f'<td colspan="{sp}" style="text-align:{a}">{c}</td>' for c, sp, a in cels) +
        "</tr>" for cels, r in linhas)
    return inline(legenda), f"<table><thead><tr>{th}</tr></thead><tbody>{tr}</tbody></table>"


def numeros_tex(caminho: Path) -> list[str]:
    """Numeros das celulas de um .tex, na ordem, para conferir o PDF."""
    _, _, corpo = corpo_tabular(caminho)
    corpo = re.sub(r"\\multicolumn\{\d+\}\{[^}]*\}", "", corpo)
    corpo = re.sub(r"\\[a-zA-Z]+", " ", corpo).replace("{,}", ",")
    # "BM25" e nome de metodo, nao numero; o A7 abrevia os bracos pela letra
    corpo = corpo.replace("BM25", "BM")
    return re.findall(r"\d+(?:,\d+)?", corpo)


def confere_numeros(pdf: Path, arquivos: list[Path]) -> None:
    """Todo numero dos .tex precisa aparecer no texto do PDF, na mesma ordem."""
    texto = subprocess.run(["pdftotext", "-raw", str(pdf), "-"], check=True,
                           capture_output=True, text=True).stdout
    no_pdf = re.findall(r"\d+(?:,\d+)?", texto)
    i, total = 0, 0
    for a in arquivos:
        for n in numeros_tex(a):
            total += 1
            while i < len(no_pdf) and no_pdf[i] != n:
                i += 1
            if i == len(no_pdf):
                raise SystemExit(f"{pdf.name}: numero {n} de {a.name} ausente ou fora de ordem")
            i += 1
    print(f"  conferidos {total} numeros de {len(arquivos)} tabelas .tex")


# ------------------------------------------------------------------- A1

REQUISITOS = [
    ("RF01", "Cadastrar usuário, fazer login, redefinir a senha e editar o perfil."),
    ("RF02", "Organizar os documentos em categorias e coleções."),
    ("RF03", "Enviar documentos em PDF para uma coleção."),
    ("RF04", "Indexar o documento em segundo plano, extraindo páginas, parágrafos, tabelas e sumário."),
    ("RF05", "Gerar, de forma opcional, resumos das páginas com um LLM."),
    ("RF06", "Disponibilizar o conteúdo indexado de uma coleção por API."),
    ("RF07", "Responder perguntas sobre uma coleção, citando as fontes."),
    ("RF08", "Abrir cada fonte no PDF, na página e no parágrafo de origem."),
    ("RF09", "Cadastrar provedores de LLM e de armazenamento, com a credencial guardada cifrada."),
    ("RF10", "Acompanhar o andamento da indexação e o consumo de tokens."),
    ("RF11", "Registrar o histórico de ações sobre documentos e provedores."),
    ("RF12", "Reprocessar documentos cuja indexação falhou."),
]

# Cada item conferido no codigo de api/ (bcrypt em usecase/auth_usecase.go, JWT em
# api/middleware/auth.go, AES-GCM em pkg/crypto/aes.go, limites em api/routes.go,
# teste de determinismo em retrieval/determinism_test.go, provedor OLLAMA em
# domain/provider.go).
NAO_FUNCIONAIS = [
    "Armazenar as senhas com bcrypt.",
    "Autenticar a sessão por token JWT.",
    "Cifrar as credenciais dos provedores com AES-GCM.",
    "Limitar o número de requisições nas rotas de autenticação, de envio de documentos e de chat.",
    "Garantir integridade referencial e transações no banco de dados (PostgreSQL).",
    "Organizar o código em camadas: handlers, casos de uso e repositórios.",
    "Permitir a execução reproduzível do sistema com Docker Compose.",
    "Recuperar de forma determinística: a mesma pergunta sobre o mesmo índice traz as mesmas "
    "fontes, na mesma ordem.",
    "Permitir o uso de um modelo local (Ollama), sem enviar o conteúdo a terceiros.",
]


def tabela_simples(cab: tuple[str, str], linhas: list[tuple[str, str]]) -> str:
    tr = "".join(f"<tr><td class='id'>{a}</td><td>{b}</td></tr>" for a, b in linhas)
    return (f"<table><thead><tr><th>{cab[0]}</th><th>{cab[1]}</th></tr></thead>"
            f"<tbody>{tr}</tbody></table>")


def a1() -> str:
    rnf = [(f"RNF{i:02d}", r) for i, r in enumerate(NAO_FUNCIONAIS, 1)]
    corpo = ("<p class='titulo-tab'>Quadro 1 – Requisitos funcionais.</p>"
             + tabela_simples(("Id", "Requisito"), REQUISITOS)
             + "<p class='titulo-tab'>Quadro 2 – Requisitos não funcionais.</p>"
             + tabela_simples(("Id", "Requisito"), rnf))
    return documento("A1", "Requisitos do sistema",
                     "Requisitos funcionais e não funcionais do ThingIndex.", corpo)


# ------------------------------------------------------------------- A7

# Nomes simples dos bracos, usados no lugar dos rotulos tecnicos dos .tex.
NOMES_BRACOS = {
    "A, contagem": "A – contagem de palavras",
    "B, BM25": "B – BM25",
    "C, denso exato": "C – significado, exata",
    "C′, denso IVF": "C′ – significado, aproximada",
    "D, RRF(A,B)": "D – combinação de A e B",
    "E, RRF(B,C)": "E – combinação de B e C",
    "<em>Teto alcançável</em>": "<em>Máximo possível</em>",
}


def rotulo_braco(c: str) -> str:
    return NOMES_BRACOS.get(c, c)


def rotulo_comparacao(c: str) -> str:
    """'A, contagem vs B, BM25' -> 'A × B' (as letras estao no Quadro 1)."""
    lados = c.split(" vs ")
    if len(lados) != 2 or any(l.strip() not in NOMES_BRACOS for l in lados):
        raise SystemExit(f"comparacao desconhecida: {c}")
    return " × ".join(l.strip().split(",")[0] for l in lados)


ROTULOS_DETERMINISMO = {
    "C (exato), 30 execuções": "C (exata), repetida 30 vezes",
    "C′ (IVF), 30 sementes": "C′ (aproximada), 30 sorteios diferentes das gavetas",
    "C′ (IVF), 30 ordens de inserção": "C′ (aproximada), textos inseridos em 30 ordens",
    "C′ (IVF), <code>probes</code> de 1 a <code>lists</code>":
        "C′ (aproximada), abrindo de 1 gaveta até todas",
}

ROTULOS_ANN = {
    "IVFFlat, √<i>N</i> listas, 1 sonda": "IVFFlat, √<i>N</i> gavetas, 1 aberta por busca",
    "IVFFlat, recomendação do pgvector": "IVFFlat, configuração recomendada",
    "HNSW, padrões do pgvector": "HNSW, configuração padrão",
}


def de_dicionario(d: dict[str, str]):
    def f(c: str) -> str:
        if c not in d:
            raise SystemExit(f"rotulo sem traducao: {c}")
        return d[c]
    return f


CAB_DESEMPENHO = ["Braço", "P@1<br><small>1º é o certo</small>",
                  "P@3<br><small>certos nos 3 primeiros</small>",
                  "R@5<br><small>certo nos 5 primeiros</small>",
                  "MRR<br><small>posição do certo</small>",
                  "nDCG@10<br><small>ordem dos 10 primeiros</small>"]
CAB_VARIANTES = ["Braço", "Literal", "Paráfrase", "Ruidosa",
                 "Queda da literal para a paráfrase"]
CAB_NEGATIVAS = ["Braço", "Taxa de falso positivo", "Resultados devolvidos (média)"]
CAB_COMPARACAO = ["Comparação", "Medida", "Δ (diferença)", "<i>n</i>", "valor-p",
                  "valor-p corrigido (Holm)", "Vitórias (1º/2º)"]

BRACOS = [
    ("A", "contagem de palavras",
     "Conta quantas palavras da pergunta aparecem no texto; quanto mais palavras em comum, "
     "mais alto o texto fica na lista. Era a busca da versão antiga do ThingIndex."),
    ("B", "BM25",
     "Busca por palavras usada hoje pelo sistema. Também compara palavras, mas com uma "
     "fórmula clássica que dá mais peso às palavras raras, que identificam melhor o assunto, "
     "e impede que uma palavra repetida muitas vezes domine a nota."),
    ("C", "busca por significado (vetores), exata",
     "Um modelo de linguagem transforma a pergunta e cada texto numa lista de números, "
     "chamada vetor, que representa o sentido; textos de sentido parecido ficam com vetores "
     "próximos, mesmo sem palavras em comum. “Exata” quer dizer que a pergunta é comparada "
     "com todos os textos."),
    ("C′", "busca por significado com índice aproximado",
     "Os mesmos vetores de C, guardados num índice aproximado: os vetores são separados em "
     "grupos (gavetas) e cada busca abre só algumas delas. É mais rápido em acervos grandes, "
     "mas pode deixar o texto certo de fora."),
    ("D", "combinação de A e B",
     "Junta as listas de A e de B numa só: cada texto ganha pontos conforme a posição que "
     "ocupou em cada lista (o método se chama RRF)."),
    ("E", "combinação de B e C",
     "Igual a D, mas juntando a lista da busca por palavras (B) com a da busca por "
     "significado (C)."),
]

METRICAS = [
    ("P@1", "Em que fração das perguntas o 1º resultado era o texto certo?",
     "0,750 = em 3 de cada 4 perguntas, o 1º resultado era o certo."),
    ("P@3", "Dos 3 primeiros resultados, que fração era certa?",
     "Como quase toda pergunta tem um só texto certo, mesmo uma busca perfeita teria 1 certo "
     "em 3 (0,333). Por isso as tabelas trazem a linha “máximo possível”."),
    ("R@5", "O texto certo apareceu entre os 5 primeiros resultados?",
     "1,000 = em todas as perguntas, o certo estava entre os 5 primeiros."),
    ("MRR", "Quão perto do topo veio o texto certo?",
     "1º lugar vale 1; 2º vale 0,5; 3º vale 0,33; e assim por diante. O MRR é a média dessas "
     "notas: se numa pergunta o certo veio em 1º e noutra em 2º, o MRR é (1 + 0,5) ÷ 2 = 0,75."),
    ("nDCG@10", "Parecido com o MRR, mas olhando os 10 primeiros resultados.",
     "Dá mais crédito a quem põe os textos certos no topo; 1 = ordem ideal."),
    ("Intervalo de confiança de 95%",
     "Os números entre colchetes, abaixo de cada nota: a faixa onde o valor real provavelmente "
     "está.",
     "0,750 com [0,500, 0,938]: com tão poucas perguntas, o valor real pode estar em qualquer "
     "ponto dessa faixa. Se as faixas de dois braços se sobrepõem, não dá para dizer que um é "
     "melhor. A faixa foi calculada sorteando as perguntas 10 mil vezes (técnica chamada "
     "<em>bootstrap</em>)."),
    ("Valor-p", "Qual é a chance de a diferença entre dois braços ser só acaso?",
     "Abaixo de 0,05, costuma-se dizer que a diferença não é acaso."),
    ("Falso positivo",
     "Devolver resultados para uma pergunta que não tem resposta em nenhum texto.",
     "Taxa 1,000 = devolveu resultados em todas as perguntas sem resposta; 0,000 = reconheceu "
     "todas."),
    ("Limiar", "Nota mínima: resultados com nota abaixo dela são descartados.",
     "Serve para tentar fazer a busca devolver “nada” quando não há resposta."),
    ("Índice de Youden (<i>J</i>)", "Quão bem um limiar separa os textos certos dos errados.",
     "1 = separa perfeitamente; 0 = não é melhor que o acaso."),
]


def quadro(titulo: str, cab: tuple[str, ...], linhas: list[tuple[str, ...]],
           larguras: tuple[int, ...]) -> str:
    cols = "".join(f"<col style='width:{w}%'>" for w in larguras)
    th = "".join(f"<th>{c}</th>" for c in cab)
    tr = "".join("<tr>" + "".join(f"<td{' class=id' if i == 0 else ''}>{c}</td>"
                                  for i, c in enumerate(l)) + "</tr>" for l in linhas)
    return (f"<p class='titulo-tab'>{titulo}</p><table class='quadro'><colgroup>{cols}"
            f"</colgroup><thead><tr>{th}</tr></thead><tbody>{tr}</tbody></table>")


# (titulo da secao, texto antes das tabelas, tabelas, "o que mostra")
# tabelas: (arquivo .tex, legenda, cabecalho, rotulo)
SECOES_A7 = [
    ("1. Desempenho de cada braço", "",
     [("retrieval-summary.tex",
       "Desempenho de cada braço escolhendo entre os 5 textos inteiros (unidade: documento). "
       "Abaixo de cada nota, entre colchetes, o intervalo de confiança de 95%. A última linha "
       "é o máximo possível de cada medida.", CAB_DESEMPENHO, rotulo_braco),
      ("retrieval-summary-section.tex",
       "Desempenho de cada braço escolhendo entre as 19 seções (unidade: seção). Mesmo formato "
       "da tabela anterior. Só os braços de busca por palavras foram medidos.",
       CAB_DESEMPENHO, rotulo_braco)],
     "Escolhendo entre os textos inteiros, o BM25 (B) e as duas combinações (D e E) tiveram "
     "as notas mais altas, com nDCG@10 de 0,890 a 0,896, seguidos da contagem de palavras "
     "(0,869), da busca por significado exata (0,844) e da aproximada (0,797); a combinação E "
     "teve o maior MRR (0,880). Mas as faixas entre colchetes se sobrepõem bastante, e com só "
     "5 textos a tarefa é fácil demais. Escolhendo entre seções, a diferença aparece: o 1º "
     "resultado foi o certo em 0,562 das perguntas com a contagem de palavras e em 0,750 com o "
     "BM25, e o certo apareceu entre os 5 primeiros em 0,719 e 0,812. Esse resultado levou o "
     "sistema a trocar a contagem de palavras pelo BM25."),
    ("2. Perguntas com as palavras do texto ou com outras palavras",
     "As perguntas são de três tipos: <em>literal</em> (usa as mesmas palavras do texto), "
     "<em>paráfrase</em> (pede a mesma coisa com outras palavras) e <em>ruidosa</em> (com "
     "erros de digitação e sem acentos). As tabelas mostram a P@1, isto é, em que fração das "
     "perguntas de cada tipo o 1º resultado era o certo.",
     [("retrieval-variants.tex",
       "P@1 por tipo de pergunta (unidade: documento). A última coluna é quanto a nota cai da "
       "pergunta literal para a paráfrase: quanto maior a queda, mais o braço depende de a "
       "pergunta usar as mesmas palavras do texto.", CAB_VARIANTES, rotulo_braco),
      ("retrieval-variants-section.tex",
       "P@1 por tipo de pergunta (unidade: seção). Mesmo formato da tabela anterior.",
       CAB_VARIANTES, rotulo_braco)],
     "As buscas por palavras (A, B e D) acertaram todas as perguntas literais e caíram nas "
     "paráfrases; a queda foi maior na contagem de palavras (0,429) que no BM25 (0,286). A "
     "busca por significado quase não depende das palavras da pergunta (queda de 0,029 em C), "
     "mas acertou menos as literais (0,600). Uma queda negativa, como a de C′, quer dizer que a "
     "paráfrase foi melhor que a literal. Por seção, a dependência das palavras cresce: a "
     "contagem cai 0,857 e acerta só 14% das paráfrases (0,143)."),
    ("3. Perguntas sem resposta",
     "Duas das 18 perguntas não têm resposta em nenhum dos 5 textos. O certo, nelas, seria a "
     "busca não devolver nada.",
     [("retrieval-negatives.tex",
       "Perguntas sem resposta (unidade: documento): com que frequência cada braço devolveu "
       "resultados mesmo assim (taxa de falso positivo) e quantos resultados devolveu, em "
       "média. 0,000 seria reconhecer que não há resposta; 1,000 é devolver resultados sempre.",
       CAB_NEGATIVAS, rotulo_braco),
      ("retrieval-negatives-section.tex",
       "Perguntas sem resposta (unidade: seção). Mesmo formato da tabela anterior.",
       CAB_NEGATIVAS, rotulo_braco),
      ("threshold.tex",
       "Tentativa de usar uma nota mínima (limiar) para descartar resultados. Sobreposição: "
       "fração dos textos errados que tiraram nota maior que o texto certo de nota mais baixa "
       "(quanto maior, mais misturadas estão as notas de certos e errados). Limiar: a melhor "
       "nota mínima encontrada. <i>J</i>: índice de Youden nesse limiar. As duas últimas "
       "colunas são a taxa de falso positivo sem e com o limiar.",
       ["Braço", "Sobreposição", "Limiar", "<i>J</i>", "Falso positivo sem limiar",
        "Falso positivo com limiar"], rotulo_braco)],
     "Todos os braços devolveram resultados nas duas perguntas sem resposta (taxa 1,000). Na "
     "busca por palavras, qualquer palavra em comum já dá nota positiva, e as notas de textos "
     "certos e errados se misturam (sobreposição de 0,782 e 0,980). C′ devolveu menos "
     "resultados (2,5 em média) porque só procura numa parte dos textos, mas também não deixou "
     "de responder. Mesmo com o melhor limiar, a taxa só cai para 0,500: uma das duas "
     "perguntas continua recebendo resultados. A nota da busca, portanto, não serve para "
     "decidir que não há resposta; no sistema completo, quem reconhece a falta de resposta é "
     "o modelo de linguagem, ao redigir a resposta (Seção 8.2 do artigo)."),
    ("4. Estabilidade: a mesma pergunta traz sempre a mesma lista?",
     "Um requisito do sistema é que a mesma pergunta traga sempre as mesmas fontes, na mesma "
     "ordem. No índice aproximado, os vetores são distribuídos em gavetas por um sorteio, e "
     "cada busca abre algumas delas; o sorteio, a ordem em que os textos são inseridos e o "
     "número de gavetas abertas podem mudar o resultado. O pgvector, extensão do banco "
     "PostgreSQL usada pelo sistema para guardar vetores, oferece dois tipos de índice "
     "aproximado: o IVFFlat, que usa gavetas, e o HNSW, que liga cada vetor aos seus vizinhos "
     "mais próximos e procura caminhando por essas ligações.",
     [("dense-determinism.tex",
       "Estabilidade da busca por significado em C1 (5 textos, 18 perguntas). Uma pergunta é "
       "estável quando a lista completa de resultados, com as notas, sai idêntica em todas as "
       "repetições (conferido por uma impressão digital da lista, o <em>hash</em> SHA-256).",
       ["O que foi variado", "Perguntas estáveis", "Listas diferentes obtidas",
        "Resultados perdidos frente à busca exata"], de_dicionario(ROTULOS_DETERMINISMO)),
      ("ann-pgvector.tex",
       "Índices aproximados do pgvector (versão 0.8.6) com 1.000, 5.000 e 20.000 vetores "
       "sintéticos (768 números cada) e 20 perguntas. √<i>N</i> gavetas: tantas gavetas quanto "
       "a raiz quadrada do número de vetores. Revocação nos 10 primeiros: fração dos 10 "
       "primeiros resultados da busca exata que o índice também encontrou. As duas últimas colunas contam em quantas "
       "das 20 perguntas os 10 primeiros resultados mudaram ao reconstruir o índice 5 vezes "
       "sobre os mesmos dados e ao inserir os mesmos vetores em 5 ordens diferentes.",
       ["Índice", "Vetores (<i>N</i>)", "Revocação nos 10 primeiros",
        "Mudaram ao reconstruir", "Mudaram ao trocar a ordem"], de_dicionario(ROTULOS_ANN))],
     "A busca exata por significado deu a mesma lista nas 30 repetições (a contagem de "
     "palavras, o BM25 e a combinação dos dois também, como relata a Seção 8.3 do artigo). O "
     "índice aproximado C′, configurado com muitas gavetas e só uma aberta por busca, mudou de "
     "resultado com o sorteio e com a ordem de inserção, e perdeu em média 48,4% dos "
     "resultados da busca exata. A segunda tabela mostra que isso depende da configuração: na "
     "configuração recomendada, o IVFFlat deu sempre o mesmo resultado da busca exata; o HNSW "
     "foi exato com 1.000 vetores e, com 20.000, encontrou 91,0% dos resultados, e 6 das 20 "
     "perguntas mudaram ao reconstruir o índice e 17 ao trocar a ordem. A estabilidade, "
     "portanto, não é uma vantagem exclusiva da busca por palavras."),
    ("5. Origem de cada unidade guardada no banco",
     "Ao indexar um documento, o sistema guarda no banco unidades de vários tipos: páginas, "
     "parágrafos, trechos (os pedaços de texto citados como fonte), itens do sumário e "
     "metadados (estatísticas do documento inteiro, como o número de parágrafos). A taxa de "
     "proveniência completa (TPC) é a fração dessas unidades cuja origem está inteira no "
     "banco: documento, página, posição e conteúdo.",
     [("provenance.tex",
       "TPC por tipo de unidade, em C1. Os metadados não estão em página nenhuma e por isso "
       "são avaliados sem exigência de página; os demais tipos precisam ter página.",
       ["Tipo de unidade", "Unidades", "Com origem completa", "TPC"], None)],
     "As 155 unidades guardadas têm origem completa (TPC 1,000). Dois cuidados: C1 é texto "
     "sem paginação, então a tabela mostra que o sistema grava a origem, e não que a página "
     "esteja certa; e a primeira medição deu 0,329, valor que subiu depois de acrescentar a "
     "página aos trechos (defeito corrigido) e de declarar que os metadados pertencem ao "
     "documento inteiro (ajuste da medida), como detalha a Seção 8.4 do artigo."),
    ("6. Comparações de dois em dois: as diferenças são acaso?",
     "Cada linha compara dois braços numa medida. Como as tabelas são longas, os braços "
     "aparecem só pela letra (Quadro 1). <strong>Δ (diferença)</strong>: nota do primeiro "
     "braço menos a do segundo; positiva, o primeiro foi melhor. <strong><i>n</i></strong>: em "
     "quantas perguntas os dois braços tiveram resultado diferente. <strong>Valor-p</strong>: "
     "chance de a diferença ser acaso, calculada por um teste que troca ao acaso, pergunta a "
     "pergunta, os resultados dos dois braços e vê com que frequência surge uma diferença tão "
     "grande. <strong>Valor-p corrigido (Holm)</strong>: o mesmo valor ajustado porque foram "
     "feitas muitas comparações ao mesmo tempo (75 por documento e 15 por seção); quanto mais "
     "comparações, maior a chance de alguma parecer diferente por sorte, e a correção compensa "
     "isso. É este o valor a ler. <strong>Vitórias</strong>: em quantas das <i>n</i> perguntas "
     "o primeiro e o segundo braço tiveram a nota maior.",
     [("retrieval-significance.tex",
       "Comparações entre todos os pares de braços (unidade: documento).",
       CAB_COMPARACAO, rotulo_comparacao),
      ("retrieval-significance-section.tex",
       "Comparações entre os pares de braços medidos por seção (unidade: seção).",
       CAB_COMPARACAO, rotulo_comparacao)],
     "Nenhuma comparação ficou abaixo de 0,05: o menor valor-p sem correção foi 0,125 por "
     "documento e 0,062 por seção, e todos os valores corrigidos ficaram entre 0,938 e 1,000. "
     "Com 16 perguntas com resposta, no máximo 8 tiveram resultado diferente em qualquer "
     "comparação, o que é pouco para detectar diferenças desse tamanho. C1, portanto, não "
     "mostra que a busca por palavras seja igual ou melhor que a busca por significado; mostra "
     "só que, neste conjunto, nenhuma diferença foi detectável."),
]

CONCLUSOES_A7 = [
    "Escolhendo entre textos inteiros, todos os braços ficaram próximos, e nenhuma diferença "
    "pode ser separada do acaso com só 16 perguntas com resposta.",
    "Escolhendo entre seções, tarefa mais difícil e mais parecida com o uso real, o BM25 pôs o "
    "texto certo em 1º lugar mais vezes que a contagem de palavras (0,750 contra 0,562); por "
    "isso o sistema passou a usar o BM25.",
    "A busca por palavras depende de a pergunta usar as palavras do texto: acerta todas as "
    "literais e cai nas paráfrases. A busca por significado depende menos disso, mas acerta "
    "menos as literais.",
    "Nenhuma forma de busca reconhece sozinha uma pergunta sem resposta; no sistema, isso fica "
    "a cargo do modelo de linguagem que redige a resposta.",
    "A busca por palavras e a busca exata por significado dão sempre a mesma lista; o índice "
    "aproximado só foi instável numa configuração fora da recomendada. As 155 unidades "
    "guardadas têm a origem completa no banco.",
]


def a7() -> tuple[str, list[Path]]:
    corpo = (
        "<h2>Como ler este anexo</h2>"
        "<p>Este anexo traz todos os números do primeiro experimento do artigo (Seções 8.1 a "
        "8.4). O experimento comparou formas diferentes de encontrar, entre vários textos, o "
        "trecho que responde a uma pergunta. Para isso, usou um conjunto pequeno, chamado C1 "
        "no artigo: 5 textos curtos escritos pelo autor (jurídico, contratual e normativo) e "
        "18 perguntas, 16 com resposta num dos textos e 2 sem resposta em nenhum. Como o autor "
        "escreveu os textos, sabe-se de antemão qual é o texto certo de cada pergunta, e dá "
        "para contar quantas vezes cada forma de busca o encontrou. Cada forma de busca testada "
        "é chamada de <em>braço</em> do experimento. As notas vão de 0 a 1, e maior é melhor: "
        "0,750 quer dizer 75%. As medições foram feitas de dois jeitos: escolhendo entre os 5 "
        "textos inteiros (unidade documento), tarefa fácil, em que um sorteio já acertaria o "
        "1º lugar cerca de 20% das vezes; e escolhendo entre as 19 seções em que os textos "
        "foram divididos (unidade seção), tarefa mais difícil e mais parecida com o que o "
        "sistema faz. Por seção, só os braços de busca por palavras foram medidos, porque a "
        "busca por significado (braços C, C′ e E do Quadro 1) foi preparada por texto "
        "inteiro.</p>"
        "<div class='bloco'>"
        + quadro("Quadro 1 – Braços comparados.", ("Braço", "Nome", "Como funciona"),
                 BRACOS, (8, 27, 65)) + "</div>"
        "<div class='bloco'>"
        + quadro("Quadro 2 – Medidas usadas nas tabelas.",
                 ("Medida", "O que responde", "Como ler"), METRICAS, (17, 38, 45)) + "</div>")
    n, usados = 0, []
    for titulo, antes, tabelas, mostra in SECOES_A7:
        corpo += f"<h2>{titulo}</h2>" + (f"<p>{antes}</p>" if antes else "")
        for arq, legenda, cab, rot in tabelas:
            n += 1
            usados.append(TAB / arq)
            _, tabela = tabela_tex(TAB / arq, cab, rot)
            # tabelas de ate 15 linhas nao se partem entre paginas
            longa = tabela.count("<tr") > 16
            if longa:
                tabela = tabela.replace("<table>", "<table class='longa'>", 1)
            corpo += (f"<div{'' if longa else ' class=bloco'}><p class='titulo-tab'>Tabela {n} "
                      f"– {legenda}</p>{tabela}</div>")
        rotulo = "esta tabela mostra" if len(tabelas) == 1 else "estas tabelas mostram"
        corpo += f"<p class='leitura'><strong>O que {rotulo}:</strong> {mostra}</p>"
    corpo += ("<div class='bloco'><h2>O que concluir</h2><ul>"
              + "".join(f"<li>{c}</li>" for c in CONCLUSOES_A7) + "</ul></div>")
    return documento("A7", "Tabelas completas de recuperação", "", corpo), usados


# ------------------------------------------------------------------- A8

# Fonte: docs/article/results/alteracoes-pos-medicao.md, na ordem do registro
# (a linha 5, sobre o desenvolvimento, vai por ultimo, como la). Os numeros
# sao os do registro; o texto foi reescrito para quem nao conhece os termos.
TERMOS_A8 = [
    ("Gabarito", "Para cada pergunta, o padrão de texto que a resposta certa precisa conter "
     "(por exemplo, o número de um artigo de lei) e, às vezes, padrões que invalidam a "
     "resposta (por exemplo, uma recusa). A correção das respostas é automática, por esses "
     "padrões. Cada revisão do gabarito ganhou um número de versão (v1, v2...)."),
    ("Recusa", "Resposta que diz que a informação não está nos documentos (“não consta”, "
     "“não há informação sobre isso”)."),
    ("Acervo de ensino", "Os 10 PDFs institucionais do IFPR, com 72 perguntas, usados durante "
     "o desenvolvimento do sistema. Cada pergunta foi feita três vezes, num total de 216 "
     "respostas."),
    ("Bateria", "O teste feito depois do desenvolvimento: 71 perguntas sobre três acervos "
     "(segurança do trabalho, produto e consumidor, licitações), cada uma feita três vezes, "
     "num total de 213 respostas."),
    ("T e F", "As duas formas comparadas no acervo de ensino. T (trechos): o sistema como o "
     "usuário o usa, que busca os trechos relevantes e envia só eles ao modelo de linguagem. "
     "F (documento inteiro): o modelo recebe o documento inteiro, sem busca."),
    ("C e E", "As duas formas comparadas na bateria. C: a busca usa a pergunta como o usuário "
     "a escreveu. E: antes da busca, o modelo de linguagem acrescenta à pergunta termos "
     "técnicos relacionados (a <em>expansão da pergunta</em>)."),
    ("Formas de pergunta", "<em>Negativa</em>: a resposta não está no acervo, e o certo é "
     "recusar. <em>Existência</em>: pergunta se o acervo trata de um assunto. <em>Premissa "
     "falsa</em>: a pergunta traz um dado errado (por exemplo, uma modalidade de licitação que "
     "já não existe), e o certo é corrigir esse dado."),
    ("Literal e paráfrase", "Duas versões da mesma pergunta: uma com as palavras do documento "
     "e outra com as palavras de quem não leu o documento."),
    ("Códigos das perguntas", "Na bateria, lc = licitações e pr = produto e consumidor; a letra "
     "final indica a versão literal (L) ou a paráfrase (P): lc4P é a paráfrase da pergunta 4 "
     "de licitações. Sem a letra final (lc3), vale para as duas versões. No acervo de ensino, "
     "p = prosa, t = tabela e c = calendário."),
    ("Banca simulada", "Revisão crítica do trabalho feita antes da defesa, simulando as "
     "perguntas de uma banca."),
]

# (quando, titulo curto, o que mudou, por que, efeito nos numeros)
ALTERACOES = [
    ("10 a 18/09/2026", "Gabarito do acervo de ensino",
     "O gabarito do acervo de ensino passou por quatro versões (v1 a v4). Em quatro perguntas "
     "(t04, t06, c08 e c13), os padrões de texto foram afrouxados para aceitar outras redações "
     "da mesma resposta. Em outras quatro (p01, p20, p22 e t21), foram acrescentadas páginas "
     "certas.",
     "Respostas certas eram recusadas só porque estavam redigidas de outro modo, e a "
     "informação de algumas perguntas estava em mais de uma página.",
     "<ul><li>F, que não mudou no período: 204 → 213 respostas certas de 216 "
     "(<strong>subiu</strong>), só pela revisão do gabarito.</li>"
     "<li>T: de 67 perguntas certas de 72 (numa só rodada, em 09/09/2026) para 216 respostas "
     "certas de 216 (<strong>subiu</strong>), somando ajustes no sistema e no gabarito.</li>"
     "<li>Consequência: na prática, o acervo de ensino serviu de conjunto de desenvolvimento, "
     "e não de teste (ver a última alteração).</li></ul>"),
    ("19/09/2026", "Recusas nas perguntas negativas da bateria",
     "O padrão que reconhece uma recusa nas perguntas negativas da bateria foi ampliado para "
     "aceitar outras formas, como “não consta” e “não há”.",
     "Recusas corretas, redigidas de outro modo (por exemplo, “os documentos não mencionam o "
     "preço”), eram marcadas como erradas.",
     "<ul><li>Acerto no acervo de produto e consumidor, com a pergunta original (C): 0,783 → "
     "0,841 (<strong>subiu</strong>).</li></ul>"),
    ("27/09/2026", "Nome dos documentos na medida de origem",
     "O gabarito da bateria passou a aceitar também as variações do título de cada documento. "
     "Essas variações ficaram fora da impressão digital (<em>hash</em>) que garante que as "
     "perguntas não mudaram.",
     "A medida de origem (se a página certa estava entre as fontes mostradas) comparava o "
     "título do documento citado com o identificador do gabarito, e contava erro quando os "
     "dois eram escritos de modo diferente, mesmo com o documento certo.",
     "<ul><li>Origem com a pergunta original (C): 0,371 → 0,581 (<strong>subiu</strong>).</li>"
     "<li>Origem com a pergunta expandida (E): 0,532 → 0,823 (<strong>subiu</strong>).</li>"
     "<li>Nenhuma resposta mudou de certa para errada ou o contrário.</li></ul>"),
    ("27/09/2026", "Recusas não contam mais como acerto (gabarito v4 da bateria, v2 do edital)",
     "Nas perguntas de existência e na lc3, uma recusa passou a invalidar a resposta. Nas de "
     "premissa falsa, a resposta passou a ter de trazer o dado que corrige a premissa. A lc3 "
     "deixou de exigir a palavra “não”. A pr2 passou a aceitar o prazo de 2 anos, que o termo "
     "de garantia também traz.",
     "A banca simulada mostrou que uma recusa fixa, dada a todas as perguntas, acertaria 24 "
     "perguntas de existência e de premissa falsa, e que a resposta da lc3 com a pergunta "
     "expandida (E), que estava correta, era marcada como errada. Foi criado um teste "
     "automático que confere que recusas não contam como acerto.",
     "<ul><li>Pergunta original (C): 174 → 149 respostas certas de 213, ou 0,817 → 0,700 "
     "(<strong>caiu</strong>).</li>"
     "<li>Pergunta expandida (E): 198 → 192 de 213, ou 0,930 → 0,901 "
     "(<strong>caiu</strong>).</li>"
     "<li>Premissa falsa com a pergunta original (C): 35 → 13 de 36 (<strong>caiu</strong>)."
     "</li>"
     "<li>Ganho da expansão da pergunta (perguntas que passaram a acertar e que deixaram de "
     "acertar com E): de 10 ganhos e 1 perda para 15 ganhos e 0 perdas; o valor-p, chance de "
     "a diferença ser acaso, foi de 0,012 para 0,0001 (a diferença ficou "
     "<strong>mais clara</strong>).</li>"
     "<li>Enriquecimento do índice (Seção 8.8 do artigo): 9 de 12 nos quatro braços → 7, 8, "
     "7 e 8 de 12 (<strong>caiu</strong>).</li>"
     "<li>Edital municipal (acervo exploratório com um edital de concurso): sem mudança."
     "</li></ul>"),
    ("27/09/2026", "Respostas que negam o assunto (gabarito v5 da bateria, v3 do edital)",
     "Nas perguntas de existência, respostas com o verbo negado (“não existe”, “não consta”, "
     "“não trata”, “não prevê”) passaram a ser invalidadas. Na lc3, a resposta oposta à certa "
     "(“não foi revogada”, “continua em vigor”) passou a ser invalidada. A lc4 passou a exigir "
     "o art. 190 da lei ou a regra que ele estabelece, e a invalidar recusa e resposta oposta.",
     "A segunda leitura da banca simulada mostrou três brechas: a frase “Não existe previsão "
     "sobre isso nos documentos.” acertava as 12 perguntas de existência; “Não, a lei não foi "
     "revogada.” acertava a lc3; e as 6 respostas da lc4P, todas recusas, contavam como "
     "certas. O teste automático passou a cobrir nove recusas e as respostas opostas.",
     "<ul><li>Pergunta original (C): 149 → 146 de 213, ou 0,700 → 0,685 "
     "(<strong>caiu</strong>).</li>"
     "<li>Pergunta expandida (E): 192 → 189 de 213, ou 0,901 → 0,887 "
     "(<strong>caiu</strong>).</li>"
     "<li>Só mudaram as 6 respostas da lc4P, conferidas uma a uma.</li>"
     "<li>Efeito da redação (pares literal e paráfrase em que só uma das versões acertou, a "
     "favor da literal × a favor da paráfrase): 12×0 → 13×0. Com o gabarito original (v3) "
     "eram 8×2, com valor-p 0,109, que não permitia descartar o acaso: a vantagem da pergunta "
     "literal só fica clara depois desta revisão e da anterior.</li>"
     "<li>Responder “Sim.” a tudo ainda acerta as 12 perguntas de existência, cuja resposta "
     "certa é sempre sim; o artigo declara isso.</li></ul>"),
    ("27/09/2026", "Últimas brechas e um falso negativo (gabarito v6 da bateria, v4 do edital)",
     "Nas perguntas de sim ou não, recusas amplas (“não está previsto”, “nenhum dos documentos "
     "trata”, “nada consta”, “não pode ser encontrada”) passaram a ser invalidadas, assim como "
     "a resposta que começa negando o que o gabarito afirma, ou o contrário. A lc12 e a lc13 "
     "passaram a invalidar a resposta oposta. A pr8 passou a aceitar o plural “píxeis”.",
     "A terceira leitura da banca simulada encontrou recusas e respostas opostas que a versão "
     "anterior ainda aceitava (nenhuma delas aparecia nas respostas reais) e um falso "
     "negativo, isto é, uma resposta certa marcada como errada: “píxeis”, sem acento, não "
     "contém a palavra “pixel”.",
     "<ul><li>Só a pr8L mudou: 2 respostas da pergunta original (C), de erradas para certas, "
     "conferidas.</li>"
     "<li>Pergunta original (C): 146 → 148 de 213, ou 0,685 → 0,695 "
     "(<strong>subiu</strong>).</li>"
     "<li>Pergunta expandida (E): sem mudança (0,887).</li>"
     "<li>Ganho da expansão (ganhos × perdas): 15×0 → 14×0 (<strong>caiu</strong>). Efeito "
     "da redação: 13×0 → 14×0 (<strong>subiu</strong>).</li>"
     "<li>Foram lidas as 30 respostas erradas restantes, uma por pergunta e braço, à procura "
     "de outras respostas certas marcadas como erradas: só a pr8 era um caso claro. A lc1P, "
     "caso de fronteira, foi mantida como estava: a resposta lista as cinco modalidades de "
     "licitação da Lei 14.133, e o gabarito, escrito antes da medição, pede também as da Lei "
     "8.666 (“depende da lei”).</li></ul>"),
    ("Durante o desenvolvimento", "Ajustes da busca medidos no acervo de ensino",
     "Regras da busca do sistema foram mantidas ou descartadas conforme o efeito medido nas 72 "
     "perguntas do acervo de ensino: exigir que a fonte contenha pelo menos 2 palavras da "
     "pergunta; procurar também a forma no plural de palavras com 6 ou mais letras; deixar de "
     "dar pontos extras a palavras do título; e levar junto o começo da página seguinte quando "
     "o texto da fonte continua nela.",
     "O trabalho seguiu a pesquisa em ciência do design, em que o sistema é melhorado em ciclos "
     "de construir, medir e ajustar.",
     "<ul><li>O acerto e a origem de T no acervo de ensino foram medidos no mesmo conjunto "
     "em que o sistema foi ajustado, e por isso não valem como teste independente.</li>"
     "<li>As perguntas da bateria, escritas depois, são o conjunto de teste.</li></ul>"),
]


def a8() -> str:
    termos = "".join(f"<tr><td class='id'>{t}</td><td>{d}</td></tr>" for t, d in TERMOS_A8)
    corpo = (
        "<h2>Por que existe este registro</h2>"
        "<p>Para corrigir automaticamente centenas de respostas, cada pergunta dos experimentos "
        "tem um gabarito: o padrão de texto que a resposta certa precisa conter e, às vezes, "
        "padrões que a invalidam. Quando um gabarito ou um critério de correção é alterado "
        "depois de ver os resultados, existe o risco de ajustá-lo, mesmo sem querer, para que o "
        "sistema pareça melhor do que é. Por isso, cada alteração feita depois de uma medição "
        "foi anotada com a data, o motivo e o efeito nos números, inclusive as que pioraram o "
        "resultado do próprio sistema. Os números publicados no artigo são sempre os de depois "
        "da última alteração.</p>"
        "<div class='bloco'><p class='titulo-tab'>Quadro 1 – Termos usados neste anexo.</p>"
        "<table class='quadro'><colgroup><col style='width:20%'><col style='width:80%'>"
        f"</colgroup><thead><tr><th>Termo</th><th>Significado</th></tr></thead>"
        f"<tbody>{termos}</tbody></table></div>"
        "<h2>As alterações</h2>")
    for i, (quando, titulo, oque, porque, efeito) in enumerate(ALTERACOES, 1):
        corpo += (f"<div class='alteracao'><h3>Alteração {i} – {titulo}</h3>"
                  f"<p class='quando'>{quando}</p>"
                  f"<p><strong>O que mudou.</strong> {oque}</p>"
                  f"<p><strong>Por quê.</strong> {porque}</p>"
                  "<p><strong>Efeito nos números</strong>"
                  + (" (antes → depois)" if "→" in efeito else "") + f":</p>{efeito}</div>")
    return documento("A8", "Alterações feitas depois de medir",
                     "Registro de cada alteração de gabarito ou de critério de correção feita "
                     "depois de conhecidos os resultados, com o motivo e o efeito nos números.",
                     corpo)


# ------------------------------------------------------------------- main

def figuras(cod: str, titulo: str, desc: str, itens: list[tuple], orient: str,
            extra: str = "") -> str:
    """itens: (imagem, legenda) ou (imagem, legenda, nota propria da figura)."""
    corpo = "".join(folha(it[0], f"Figura {i} – {it[1]}", orient, i == 1,
                          it[2] if len(it) > 2 else extra)
                    for i, it in enumerate(itens, 1))
    return documento(cod, titulo, desc, corpo, orient, cabecalho_na_folha=True)


def main() -> None:
    build = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(tempfile.mkdtemp(prefix="anexos-"))
    build.mkdir(parents=True, exist_ok=True)
    SAIDA.mkdir(exist_ok=True)

    er = gera_er()
    svg = plantuml([DIAG / "casos-de-uso.puml", DIAG / "sequencia.puml",
                    DIAG / "esquema-proveniencia.puml", DIAG / "recuperacao.puml"]
                   + [p for _, _, p, _ in er], build)

    imprimir(a1(), "A1-requisitos-funcionais", build)
    imprimir(figuras("A2", "Diagrama de casos de uso",
                     "Casos de uso do ThingIndex, agrupados por área do sistema.",
                     [(svg["casos-de-uso"], "Diagrama de casos de uso do ThingIndex.")],
                     "landscape"), "A2-casos-de-uso", build)
    imprimir(figuras("A3", "Diagrama de sequência",
                     "Envio e indexação de um documento e pergunta a uma coleção.",
                     [(svg["sequencia"], "Diagrama de sequência do envio de um documento e "
                                         "de uma pergunta à coleção.")],
                     "portrait"), "A3-sequencia-envio-e-pergunta", build)
    imprimir(figuras("A4", "Recorte do esquema com o caminho da proveniência",
                     "Tabelas que ligam cada fonte citada ao documento, à página e ao parágrafo "
                     "de origem. As demais tabelas do banco (conversas, auditoria, fila de "
                     "processamento, provedores e preços) não participam desse caminho e estão "
                     "no Anexo A9.",
                     [(svg["esquema-proveniencia"], "Recorte do esquema do banco de dados com "
                       "o caminho da proveniência.")],
                     "landscape",
                     "PK: chave primária; FK: chave estrangeira. Cada linha é uma chave "
                     "estrangeira."), "A4-esquema-proveniencia", build)
    imprimir(figuras("A5", "Fluxo de recuperação",
                     "Etapas da recuperação de contexto para uma pergunta.",
                     [(svg["recuperacao"], "Fluxo de recuperação de contexto do ThingIndex.")],
                     "portrait"), "A5-fluxo-de-recuperacao", build)
    # Telas numeradas na ordem de uso, com as legendas de telas/legendas.txt;
    # a do chat com resposta (workspace.png) entra logo depois do chat vazio.
    telas = []
    for linha in (TELAS / "legendas.txt").read_text(encoding="utf-8").splitlines():
        if "|" not in linha:
            continue
        arquivo, legenda = (x.strip() for x in linha.split("|", 1))
        telas.append((TELAS / arquivo, legenda.rstrip(".") + "."))
        if arquivo.startswith("05-"):
            telas.append((TELAS / "workspace.png", "Área de trabalho com uma resposta do chat: "
                          "fontes numeradas (esquerda), PDF na página citada com o parágrafo "
                          "destacado (centro) e resumo do documento (direita)."))
    imprimir(figuras("A6", "Telas do protótipo",
                     "Telas do ThingIndex em funcionamento, na ordem em que o usuário as percorre.",
                     telas, "landscape"), "A6-telas-do-prototipo", build)
    html_a7, usados = a7()
    pdf = imprimir(html_a7, "A7-tabelas-completas-de-recuperacao", build)
    confere_numeros(pdf, usados)
    imprimir(a8(), "A8-alteracoes-pos-medicao", build)

    tabelas, _ = ler_dbml()
    def nota(ns: list[str]) -> str:
        texto = "PK: chave primária; FK: chave estrangeira. "
        if "outbox_events" in ns:
            texto += ("Linha contínua: chave estrangeira; linha tracejada: no outbox, o "
                      "documento a indexar (aggregate_id), campo genérico do padrão, sem "
                      "chave estrangeira.")
        else:
            texto += "Cada linha é uma chave estrangeira."
        if "users" not in ns:
            texto += " Tabelas marcadas com (Figura 1) aparecem completas na Figura 1."
        return texto

    itens = [(svg[p.stem], f"Esquema do banco de dados: {t} ({len(ns)} tabelas).", nota(ns))
             for _, t, p, ns in er]
    imprimir(figuras("A9", "Esquema do banco de dados",
                     f"Diagrama entidade-relacionamento das {len(tabelas)} tabelas do "
                     "banco de dados PostgreSQL.", itens, "landscape"),
             "A9-esquema-do-banco", build)


if __name__ == "__main__":
    main()
