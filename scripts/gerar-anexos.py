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
FONTE = "Fonte: elaborado pelo autor."
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
td small { font-size: 8.5pt; }
table.longa { font-size: 9.5pt; }
table.longa td { padding-top: 1.5pt; padding-bottom: 1.5pt; }
.titulo-tab { font-size: 10pt; margin: 14pt 0 4pt 0; text-align: justify; break-after: avoid; }
.fonte-tab { font-size: 9pt; margin: 2pt 0 10pt 0; }
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
            + f"<p class='fonte'>{FONTE}</p></figure></section>")


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
        if "users" in nomes:  # posse por user_id, sem chave estrangeira declarada
            for n in nomes:
                if n != "users" and any(c == "user_id" for c, _, _ in tabelas[n]):
                    seta = ".right." if n in NUCLEO else ".."
                    l.append(f"users ||{seta}o{{ {n}")
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


def tabela_tex(caminho: Path) -> tuple[str, str]:
    """Converte table/tabular ou longtable de um .tex gerado em (legenda, <table>)."""
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
            cab, regra = html_cels, False
            continue
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
             + f"<p class='fonte-tab'>{FONTE}</p>"
             + "<p class='titulo-tab'>Quadro 2 – Requisitos não funcionais.</p>"
             + tabela_simples(("Id", "Requisito"), rnf)
             + f"<p class='fonte-tab'>{FONTE}</p>")
    return documento("A1", "Requisitos do sistema",
                     "Requisitos funcionais e não funcionais do ThingIndex.", corpo)


# ------------------------------------------------------------------- A7

BRACOS = [("A", "Contagem de termos da pergunta presentes no texto."),
          ("B", "BM25."),
          ("C", "Busca densa exata."),
          ("C′", "Busca densa com índice aproximado."),
          ("D", "Fusão de A e B."),
          ("E", "Fusão de B e C.")]
METRICAS = [("P@1", "Fração das consultas cujo primeiro resultado é relevante."),
            ("P@3", "Fração relevante dos três primeiros resultados."),
            ("R@5", "Fração dos itens relevantes encontrada entre os cinco primeiros resultados."),
            ("MRR", "Média do inverso da posição do primeiro resultado relevante."),
            ("nDCG@10", "Qualidade da ordenação dos dez primeiros resultados, com mais peso "
                        "para os relevantes no topo (1 = ordenação ideal).")]

TABELAS_A7 = [
    ("Desempenho por braço", ["retrieval-summary.tex", "retrieval-summary-section.tex"]),
    ("Variantes de consulta", ["retrieval-variants.tex", "retrieval-variants-section.tex"]),
    ("Consultas negativas e limiar de corte",
     ["retrieval-negatives.tex", "retrieval-negatives-section.tex", "threshold.tex"]),
    ("Comparações pareadas", ["retrieval-significance.tex", "retrieval-significance-section.tex"]),
    ("Determinismo, índices aproximados e proveniência",
     ["dense-determinism.tex", "ann-pgvector.tex", "provenance.tex"]),
]


def a7() -> tuple[str, list[Path]]:
    corpo = ("<p>As tabelas reproduzem integralmente os resultados de recuperação das Seções 8.1 "
             "a 8.3 do artigo. Todas as métricas variam de 0 a 1, e maior é melhor; valores entre "
             "colchetes são intervalos de confiança de 95%. Nas comparações pareadas, a coluna "
             "Vitórias A/B conta as consultas em que o primeiro (A) ou o segundo (B) braço da "
             "comparação teve o valor maior.</p>"
             "<div class='bloco'><p class='titulo-tab'>Quadro 1 – Braços de recuperação comparados.</p>"
             + tabela_simples(("Braço", "Descrição"), BRACOS) + "</div>"
             "<div class='bloco'><p class='titulo-tab'>Quadro 2 – Métricas de recuperação.</p>"
             + tabela_simples(("Métrica", "Significado"), METRICAS) + "</div>")
    n, usados = 0, []
    for titulo, arqs in TABELAS_A7:
        corpo += f"<h2>{titulo}</h2>"
        for a in arqs:
            n += 1
            usados.append(TAB / a)
            legenda, tabela = tabela_tex(TAB / a)
            if a.startswith("retrieval-") and "unidade de recuperação" not in legenda:
                unidade = "seção" if "-section" in a else "documento"
                legenda += f" Unidade de recuperação: {unidade}."
            # tabelas de ate 8 linhas nao se partem entre paginas
            longa = tabela.count("<tr") > 9
            classe = "" if longa else " class='bloco'"
            if longa:
                tabela = tabela.replace("<table>", "<table class='longa'>", 1)
            corpo += (f"<div{classe}><p class='titulo-tab'>Tabela {n} – {legenda}</p>{tabela}"
                      f"<p class='fonte-tab'>{FONTE}</p></div>")
    return documento("A7", "Tabelas completas de recuperação",
                     "", corpo), usados


# ------------------------------------------------------------------- A8

# Redacao fiel de docs/article/results/alteracoes-pos-medicao.md, na mesma
# ordem; os numeros sao os do registro.
ALTERACOES = [
    ("10 a 18/09/2026",
     "Quatro versões do gabarito do acervo de ensino (v1 a v4): expressões regulares "
     "afrouxadas em t04, t06, c08 e c13; novas páginas-alvo em p01, p20, p22 e t21.",
     "Respostas certas eram recusadas por redação diferente; a informação estava presente "
     "em mais de uma página.",
     "F (sistema inalterado): 204 → 213/216 respostas. T passou de 67/72 perguntas "
     "(1 repetição, 09/09/2026) a 216/216 respostas, somando ajustes do sistema e do "
     "gabarito. Na prática, o acervo de ensino foi conjunto de desenvolvimento (Seção 5)."),
    ("19/09/2026",
     "Padrão de recusa das perguntas negativas das baterias ampliado (“não consta”, "
     "“não há”, ...).",
     "Recusas corretas redigidas de outro modo eram marcadas como erradas.",
     "Produto: 0,783 → 0,841 (C)."),
    ("27/09/2026",
     "Variantes do título de cada documento incluídas no gabarito das baterias, fora do "
     "hash das perguntas.",
     "O registro das páginas citadas gravava o título da fonte, e não o identificador do "
     "gabarito; a proveniência contava erro quando o título diferia.",
     "Proveniência: C 0,371 → 0,581; E 0,532 → 0,823. Nenhum veredito de acerto mudou."),
    ("27/09/2026",
     "Gabaritos das baterias v4 e do edital v2: as perguntas de existência e lc3 passam a "
     "vetar recusa; premissa falsa passa a exigir o dado que corrige a premissa; lc3 deixa "
     "de exigir “não”; pr2 aceita o prazo de 2 anos que o termo de garantia também traz.",
     "A banca simulada mostrou que uma recusa constante acertava 24 perguntas de existência "
     "e de premissa falsa, e que lc3 no braço E, correta, era marcada como errada. Foi "
     "criado um teste de regressão para recusas.",
     "C 174 → 149/213 (0,817 → 0,700); E 198 → 192/213 (0,930 → 0,901). Premissa falsa: "
     "C 35 → 13/36. Expansão: 10 ganhos e 1 perda → 15 ganhos e 0 perdas "
     "(p 0,012 → 0,0001). Enriquecimento: 9/12 nos quatro braços → 7, 8, 7 e 8/12. "
     "Edital: sem mudança."),
    ("27/09/2026",
     "Gabaritos das baterias v5 e do edital v3: existência veta o verbo negado (“não "
     "existe”, “não consta”, “não trata”, “não prevê”); lc3 veta a resposta oposta (“não "
     "foi revogada”, “continua em vigor”); lc4 passa a exigir o art. 190 ou a regra que "
     "ele estabelece e veta recusa e resposta oposta.",
     "A segunda leitura da banca simulada mostrou que “Não existe previsão sobre isso nos "
     "documentos.” acertava as 12 perguntas de existência, que “Não, a lei não foi "
     "revogada.” acertava lc3 e que as 6 respostas de lc4P, todas recusas, contavam como "
     "certas. O teste de regressão passou a cobrir nove recusas e as respostas opostas.",
     "C 149 → 146/213 (0,700 → 0,685); E 192 → 189/213 (0,901 → 0,887). Mudaram só as 6 "
     "respostas de lc4P, conferidas uma a uma. Teste de redação: 12×0 → 13×0 pares; com o "
     "gabarito original (v3) era 8×2, p = 0,109: o efeito da redação só é significativo "
     "depois desta revisão e da anterior. A resposta constante “Sim.” ainda acerta as 12 "
     "perguntas de existência, cuja resposta certa é sempre sim; o artigo declara isso."),
    ("27/09/2026",
     "Gabaritos das baterias v6 e do edital v4: recusa ampla (“não está previsto”, "
     "“nenhum dos documentos trata”, “nada consta”, “não pode ser encontrada”) vetada nas "
     "perguntas de sim ou não; nelas, a resposta que abre negando o que o gabarito afirma "
     "(ou o contrário) é vetada; lc12 e lc13 vetam a resposta oposta; pr8 aceita o plural "
     "“píxeis”.",
     "A terceira leitura da banca simulada mostrou recusas e respostas opostas que o v5 "
     "ainda aceitava (nenhuma nas respostas reais) e um falso negativo: “píxeis” não "
     "contém “pixel” depois de retirados os acentos.",
     "Só pr8L mudou (2 respostas de C, de errado para certo, conferidas). C 146 → 148/213 "
     "(0,685 → 0,695); E inalterado (0,887). Expansão: 15×0 → 14×0; redação: 13×0 → 14×0. "
     "A leitura das 30 respostas erradas restantes, uma por pergunta e braço, encontrou só "
     "pr8 como falso negativo claro. lc1P, caso de fronteira, foi mantido: a resposta lista "
     "as cinco modalidades da Lei 14.133, e o gabarito, escrito antes da medição, pede "
     "também as da Lei 8.666 (“depende da lei”)."),
    ("Durante o desenvolvimento",
     "Heurísticas do ranqueador (cobertura mínima de 2 termos, variante de plural com 6 ou "
     "mais letras, descarte do reforço por título, regra de continuação de página) mantidas "
     "ou descartadas pelo efeito medido nas 72 perguntas do acervo de ensino.",
     "Ciclos da pesquisa em ciência do design (DSR).",
     "O acerto e a proveniência de T no acervo de ensino são resultados dentro da amostra "
     "de desenvolvimento. As baterias, escritas depois, são o conjunto de teste."),
]


def a8() -> str:
    tr = "".join(f"<tr><td class='id'>{d}</td><td>{o}</td><td>{p}</td><td>{e}</td></tr>"
                 for d, o, p, e in ALTERACOES)
    corpo = ("<p class='titulo-tab'>Quadro 1 – Alterações de gabarito ou de critério feitas "
             "depois de medir.</p>"
             "<table style='font-size:9.5pt'><colgroup><col style='width:11%'><col style='width:32%'>"
             "<col style='width:25%'><col style='width:32%'></colgroup><thead><tr><th>Data</th>"
             f"<th>Alteração</th><th>Motivo</th><th>Efeito nos números</th></tr></thead>"
             f"<tbody>{tr}</tbody></table>"
             "<p class='fonte-tab'>Braços: T, trechos recuperados; F, documento inteiro; C, "
             "pergunta original; E, pergunta expandida por LLM. Os números publicados no "
             "artigo são os da coluna Efeito nos números.</p>"
             f"<p class='fonte-tab'>{FONTE}</p>")
    return documento("A8", "Alterações feitas depois de medir",
                     "Registro de cada alteração de gabarito ou de critério feita depois de "
                     "conhecidos os resultados, com o efeito nos números.", corpo, "landscape")


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
                     "de origem.",
                     [(svg["esquema-proveniencia"], "Recorte do esquema do banco de dados com "
                       "o caminho da proveniência.")],
                     "landscape",
                     "Linha contínua: chave estrangeira; linha tracejada: posse pelo usuário "
                     "(user_id)."), "A4-esquema-proveniencia", build)
    imprimir(figuras("A5", "Fluxo de recuperação",
                     "Etapas da recuperação de contexto para uma pergunta.",
                     [(svg["recuperacao"], "Fluxo de recuperação de contexto do ThingIndex.")],
                     "portrait"), "A5-fluxo-de-recuperacao", build)
    imprimir(figuras("A6", "Telas do protótipo", "",
                     [(TELAS / "workspace.png", "Workspace de uma coleção: chat com as fontes "
                       "(esquerda), PDF na página citada com o parágrafo destacado (centro) e "
                       "resumo do documento (direita)."),
                      (TELAS / "operacoes.png", "Tela de operações: andamento da indexação, "
                       "tentativas, consumo de tokens e falhas.")],
                     "landscape"), "A6-telas-do-prototipo", build)
    html_a7, usados = a7()
    pdf = imprimir(html_a7, "A7-tabelas-completas-de-recuperacao", build)
    confere_numeros(pdf, usados)
    imprimir(a8(), "A8-alteracoes-pos-medicao", build)

    tabelas, _ = ler_dbml()
    nota_nucleo = ("PK: chave primária; FK: chave estrangeira. Linha contínua: chave "
                   "estrangeira; linha tracejada: posse pelo usuário (user_id).")
    nota_operacao = ("PK: chave primária; FK: chave estrangeira; linha contínua: chave "
                     "estrangeira. As colunas user_id indicam o usuário dono do registro. "
                     "Tabelas marcadas com (Figura 1) aparecem completas na Figura 1.")
    itens = [(svg[p.stem], f"Esquema do banco de dados: {t} ({len(ns)} tabelas).",
              nota_nucleo if "users" in ns else nota_operacao)
             for _, t, p, ns in er]
    imprimir(figuras("A9", "Esquema do banco de dados",
                     f"Diagrama entidade-relacionamento das {len(tabelas)} tabelas do "
                     "banco de dados PostgreSQL.", itens, "landscape"),
             "A9-esquema-do-banco", build)


if __name__ == "__main__":
    main()
