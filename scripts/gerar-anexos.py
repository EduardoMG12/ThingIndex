#!/usr/bin/env python3
"""Gera os anexos do artigo em PDF (pasta anexos/ na raiz do repositorio).

Cada anexo e montado como HTML numa pasta de trabalho e impresso em PDF A4 pelo
Chromium headless. As fontes de cada anexo sao arquivos versionados:

- diagramas: docs/article/anexos/diagramas/*.puml, renderizados pelo PlantUML
  (imagem Docker plantuml/plantuml) em SVG; o script tambem regrava os .png de
  alta resolucao ao lado dos .puml;
- esquema do banco (A9): os .puml esquema-banco-*.puml sao gerados aqui a
  partir de docs/schema/thingindex.dbml, nunca editados a mao;
- resultados do teste de busca (A7): texto em linguagem simples com tabelas
  montadas de docs/article/tables/*.tex e, na secao "Detalhes completos", as
  tabelas inteiras com as proporcoes convertidas para porcentagem ou nota de 0 a
  100; confere_numeros verifica que todo numero do .tex, convertido, aparece no
  PDF (regra de conversao em para_cem);
- correcoes depois de medir (A8): de docs/article/results/alteracoes-pos-medicao.md,
  com verificacao de que todo numero citado existe no registro (confere_registro);
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
from collections import Counter
from decimal import ROUND_HALF_EVEN, Decimal
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
/* Quadros de texto: linha fina entre os itens, para ver onde cada um termina. */
table.quadro tbody tr + tr td, table.legenda tbody tr + tr td { border-top: 0.4pt solid #999 !important; }
ul { margin: 0 0 6pt 0; padding-left: 16pt; }
li { margin: 0 0 3pt 0; text-align: justify; }
h3 { font-size: 11pt; font-weight: bold; margin: 0 0 1pt 0; break-after: avoid; }
.alteracao { margin: 12pt 0 0 0; padding-top: 6pt; border-top: 0.6pt solid #000; }
.alteracao p, .alteracao li { margin-bottom: 3pt; }
.quando { font-style: italic; font-size: 10pt; }
th.n, td.n { text-align: center; }
.resumo { margin: 4pt 0 6pt 0; }
.caixa { border: 0.6pt solid #000; padding: 5pt 7pt; margin: 6pt 0 8pt 0; }
.titulo-caixa { font-weight: bold; margin: 0 0 3pt 0; }
table.legenda { font-size: 9pt !important; margin: 0; }
table.legenda td { padding: 1.5pt 4pt; border: none !important; }
table.detalhe { font-size: 8.5pt !important; margin-bottom: 4pt; }
.detalhes .titulo-tab { margin-top: 8pt; font-size: 9.5pt; }
table.detalhe td { padding-top: 1.5pt; padding-bottom: 1.5pt; }
.colunas { columns: 2; column-gap: 7mm; }
table.par { font-size: 8pt !important; break-inside: avoid; margin: 0 0 5pt 0; }
table.par th, table.par td { padding: 1pt 3pt; }
table.par th.nome-par { text-align: left; font-size: 8pt; padding-top: 2pt; }
table.par thead tr + tr th { border-top: none; font-weight: normal; font-style: italic; }
table.a8 { font-size: 9.5pt; }
table.a8 td:nth-child(n+3) { font-size: 9pt; }
.nota-a8 { font-size: 10pt; margin-top: 6pt; }
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


def celulas_tex(caminho: Path) -> tuple[list[str], list[tuple[list[str], bool]]]:
    """Cabecalho e linhas de um table/tabular ou longtable de um .tex gerado.

    Cada celula vem convertida para HTML (sem trocar numeros); cada linha traz
    junto se havia um \\midrule antes dela."""
    _, _, corpo = corpo_tabular(caminho)
    linhas, cab, regra, depois_cab = [], None, False, False
    for bruto in separa(corpo, r"\\"):
        t = bruto.replace(r"\toprule", "").replace(r"\bottomrule", "")
        if r"\midrule" in t:
            regra = True
            t = t.replace(r"\midrule", "")
        t = t.strip()
        if not t:
            continue
        cels = []
        for c in separa(t, "&"):
            c = c.strip()
            mc = re.match(r"\\multicolumn\{(\d+)\}\{([^}]*)\}", c)
            if mc:
                c, _ = chave(c, mc.end())
            cels.append(inline(c))
        if cab is None:
            cab, regra = cels, False
            continue
        linhas.append((cels, regra and depois_cab))
        regra, depois_cab = False, True
    return cab, linhas


# Conversao dos numeros das tabelas .tex para o A7 (e para a conferencia):
# - proporcao com virgula decimal entre 0 e 1 -> multiplicada por 100, com uma
#   casa decimal e arredondamento meio-para-o-par ('0,812' e '0,8125' -> '81,2';
#   '1,000' -> '100,0'). Vale para taxas (P@1, P@3, R@5, falso positivo,
#   sobreposicao, origem completa), que viram porcentagem; para MRR, nDCG e o
#   J de Youden, que viram "nota de 0 a 100"; para os valores-p, que viram a
#   chance de sorte em %; e para as diferencas (Delta, queda), que viram pontos
#   percentuais (o sinal fica fora do numero e nao muda);
# - numero sem virgula (contagens, '20/20', '1.000') nao muda;
# - numero com virgula maior que 1 nao e proporcao e nao muda: o limiar (nota
#   da propria busca, '4,000' e '2,127'), a media de resultados devolvidos
#   ('5,0') e os valores que o .tex ja traz em porcentagem ('48,4\%').
# Nao ha similaridade de cosseno nas tabelas; se houvesse, entre 0 e 1, viraria
# nota de 0 a 100 pela mesma regra.

def para_cem(num: str) -> str:
    if "," not in num:
        return num
    d = Decimal(num.replace(",", "."))
    if d > 1:
        return num
    return str((d * 100).quantize(Decimal("0.1"), rounding=ROUND_HALF_EVEN)).replace(".", ",")


def converte(celula: str) -> str:
    """Aplica para_cem a cada numero da celula; o intervalo '[a, b]' vira '(de a a b)'."""
    def f(m: re.Match) -> str:
        if m[1]:
            return f"(de {para_cem(m[1])} a {para_cem(m[2])})"
        return para_cem(m[3])
    return re.sub(r"\[(\d+,\d+), (\d+,\d+)\]|(\d+,\d+)", f, celula)


def numeros_tex(caminho: Path) -> list[str]:
    """Numeros das celulas de um .tex, ja convertidos por para_cem."""
    _, _, corpo = corpo_tabular(caminho)
    corpo = re.sub(r"\\multicolumn\{\d+\}\{[^}]*\}", "", corpo)
    corpo = re.sub(r"\\[a-zA-Z]+", " ", corpo).replace("{,}", ",")
    # "BM25" e nome de metodo, e "P@1", "nDCG@10" sao nomes de medida: nao sao numeros
    corpo = corpo.replace("BM25", "BM")
    corpo = re.sub(r"@\d+", "", corpo)
    return [para_cem(n) for n in re.findall(r"\d+(?:,\d+)?", corpo)]


MARCA_DETALHES = "Tabelas completas"


def confere_numeros(pdf: Path, arquivos: list[Path]) -> None:
    """Todo numero de cada .tex, convertido, precisa aparecer no PDF, no trecho da
    sua tabela na secao de detalhes ("Tabela D<i> –" ate a proxima). A conferencia
    e por contagem (cada numero tantas vezes quanto no .tex), e nao por ordem,
    porque algumas tabelas foram divididas em duas para caber em poucas colunas."""
    texto = subprocess.run(["pdftotext", "-raw", str(pdf), "-"], check=True,
                           capture_output=True, text=True).stdout
    k = texto.rfind(MARCA_DETALHES)
    if k < 0:
        raise SystemExit(f"{pdf.name}: secao '{MARCA_DETALHES}' ausente")
    partes = re.split(r"Tabela D(\d+)\s*–", texto[k:])
    trechos = {int(partes[i]): partes[i + 1] for i in range(1, len(partes), 2)}
    total = 0
    for i, a in enumerate(arquivos, 1):
        if i not in trechos:
            raise SystemExit(f"{pdf.name}: Tabela D{i} ({a.name}) ausente")
        esperados = numeros_tex(a)
        faltam = Counter(esperados) - Counter(re.findall(r"\d+(?:,\d+)?", trechos[i]))
        if faltam:
            raise SystemExit(f"{pdf.name}: Tabela D{i} sem os numeros {dict(faltam)} "
                             f"de {a.name}")
        total += len(esperados)
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

# Rotulo do metodo no .tex (depois de inline) -> nome por extenso usado no A7.
METODOS = {
    "A, contagem": "Contagem de palavras (versão antiga)",
    "B, BM25": "BM25 (busca por palavras, usada no sistema)",
    "C, denso exato": "Busca por significado, exata",
    "C′, denso IVF": "Busca por significado, aproximada",
    "D, RRF(A,B)": "Contagem + BM25 combinados",
    "E, RRF(B,C)": "BM25 + significado combinados",
}
ORDEM_METODOS = list(METODOS)

COMO_FUNCIONA = {
    "A, contagem": "Conta quantas palavras da pergunta aparecem em cada texto. Foi a busca "
                   "da primeira versão do ThingIndex.",
    "B, BM25": "Também compara palavras, mas pesa mais as raras, que dizem mais sobre o "
               "assunto, e menos as que se repetem. É a busca usada hoje pelo sistema.",
    "C, denso exato": "Um modelo de linguagem converte a pergunta e cada texto em vetores "
                      "numéricos, de modo que textos de sentido parecido fiquem próximos, "
                      "mesmo sem palavras em comum. A pergunta é comparada com todos os textos.",
    "C′, denso IVF": "Os mesmos vetores, mas guardados num índice que os divide em grupos e "
                     "consulta só parte deles. É mais rápida em acervos grandes, ao custo de "
                     "às vezes deixar o texto certo de fora.",
    "D, RRF(A,B)": "Combina as listas da contagem e do BM25, somando pontos pela posição "
                   "de cada texto em cada lista.",
    "E, RRF(B,C)": "Mesma combinação, entre o BM25 e a busca por significado exata.",
}

ROTULOS_DETALHE = dict(METODOS, **{
    "<em>Teto alcançável</em>": "<em>Máximo possível</em>",
    # dense-determinism.tex
    "C (exato), 30 execuções": "Busca por significado, exata: repetida 30 vezes",
    "C′ (IVF), 30 sementes": "Busca por significado, aproximada: 30 sorteios diferentes "
                            "dos grupos",
    "C′ (IVF), 30 ordens de inserção": "Busca por significado, aproximada: textos inseridos "
                                      "em 30 ordens",
    "C′ (IVF), <code>probes</code> de 1 a <code>lists</code>":
        "Busca por significado, aproximada: olhando de 1 grupo até todos",
    # ann-pgvector.tex
    "IVFFlat, √<i>N</i> listas, 1 sonda": "IVFFlat como no teste: muitos grupos, só 1 "
                                          "consultado",
    "IVFFlat, recomendação do pgvector": "IVFFlat, configuração recomendada",
    "HNSW, padrões do pgvector": "HNSW, configuração padrão",
    # provenance.tex
    "Trecho": "Trecho (o pedaço citado como fonte)",
    "Metadado": "Dado geral do documento",
    "Nó de sumário": "Item do sumário",
    "Página": "Página",
    "Parágrafo": "Parágrafo",
    "<strong>Total</strong>": "<strong>Total</strong>",
})


def rotulo_detalhe(c: str) -> str:
    if c not in ROTULOS_DETALHE:
        raise SystemExit(f"rotulo sem traducao: {c}")
    return ROTULOS_DETALHE[c]


def linhas_de(arq: str) -> dict[str, list[str]]:
    """{rotulo do .tex: demais celulas} de uma tabela."""
    _, linhas = celulas_tex(TAB / arq)
    return {c[0]: c[1:] for c, _ in linhas}


def valor(celula: str) -> str:
    """Numero principal da celula (sem o intervalo), convertido."""
    return converte(celula.split("<br>")[0]).strip()


def tabela(cab: list[str], linhas: list[list[str]], larguras: list[int] | None = None,
           classe: str = "", regras: list[bool] | None = None, centro: bool = True) -> str:
    cols = ("<colgroup>" + "".join(f"<col style='width:{w}%'>" for w in larguras)
            + "</colgroup>") if larguras else ""
    n = (lambda i: " class=n" if i and centro else "")
    th = "".join(f"<th{n(i)}>{c}</th>" for i, c in enumerate(cab))
    tr = "".join(
        f"<tr{' class=regra' if regras and regras[k] else ''}>"
        + "".join(f"<td{n(i)}>{c}</td>" for i, c in enumerate(l))
        + "</tr>" for k, l in enumerate(linhas))
    return (f"<table class='{classe}'>{cols}<thead><tr>{th}</tr></thead>"
            f"<tbody>{tr}</tbody></table>")


def resumo(txt: str) -> str:
    return f"<p>{txt[0].upper() + txt[1:]}</p>"


def pct(celula: str) -> str:
    return valor(celula) + "%"


def a7_corpo() -> str:
    doc = linhas_de("retrieval-summary.tex")
    sec = linhas_de("retrieval-summary-section.tex")
    var_doc = linhas_de("retrieval-variants.tex")
    var_sec = linhas_de("retrieval-variants-section.tex")
    det = linhas_de("dense-determinism.tex")
    _, ann = celulas_tex(TAB / "ann-pgvector.tex")
    _, sig_doc = celulas_tex(TAB / "retrieval-significance.tex")
    _, sig_sec = celulas_tex(TAB / "retrieval-significance-section.tex")

    def nome(k: str) -> str:
        return METODOS[k]

    # numeros do teste estatistico, tirados das tabelas
    def minimo(linhas, col):
        return min(linhas, key=lambda l: Decimal(l[0][col].replace(",", ".")))[0][col]
    p_doc, p_sec = minimo(sig_doc, 4), minimo(sig_sec, 4)
    holm = minimo(sig_doc + sig_sec, 5)
    max_n = max(int(l[0][3]) for l in sig_doc + sig_sec)
    faixa_b = re.search(r"\[(.*?), (.*?)\]", doc["B, BM25"][0])
    faixa_a = re.search(r"\[(.*?), (.*?)\]", doc["A, contagem"][0])

    c = []
    B, A, C = "B, BM25", "A, contagem", "C, denso exato"
    c.append("<h2>Por que este teste foi feito</h2>"
             "<p>A pergunta do TCC é se dá para montar o sistema sem banco de vetores, só com o "
             "banco relacional e uma busca por palavras. Antes de construir o resto do sistema em "
             "cima dessa escolha, era preciso saber duas coisas: se a busca por palavras encontra o "
             "trecho certo pelo menos tão bem quanto a busca por significado, que é a opção usual, "
             "e se ela devolve sempre o mesmo resultado, já que o sistema promete fontes que possam "
             "ser conferidas de novo.</p>"
             "<p>O teste é pequeno e controlado, separado dos acervos de PDFs públicos usados no "
             "restante do artigo. São cinco textos curtos e fictícios (um acórdão, um edital de "
             "pregão, um contrato de TI, uma política de privacidade e um manual de integração) e "
             "18 perguntas sobre eles; 16 têm resposta e 2 não. Como os textos foram criados para o "
             "teste, sabe-se qual é o texto certo de cada pergunta. Ao longo deste anexo, a medida é "
             "uma só: em quantas perguntas o primeiro resultado da busca era o texto certo.</p>")

    linhas = [[nome(k), pct(doc[k][0]), pct(sec[k][0]) if k in sec else "não medido"]
              for k in ORDEM_METODOS]
    c.append("<h2>1. A busca por palavras acha o texto certo?</h2>"
             "<p>Se a busca por palavras errasse muito mais que a busca por significado, a proposta "
             "do TCC não se sustentaria. Foram comparados seis métodos: a contagem de palavras da "
             "primeira versão do sistema, o BM25 (busca por palavras que pesa mais as palavras "
             "raras, usada hoje), a busca por significado com vetores, em versão exata e "
             "aproximada, e duas combinações. Cada um escolheu o texto certo entre os cinco textos "
             "inteiros e, num teste mais difícil e mais parecido com o sistema real, a seção certa "
             "entre 19 seções.</p>"
             "<div class='bloco'><p class='titulo-tab'>Tabela 1 – Perguntas em que o primeiro "
             "resultado era o texto certo.</p>"
             + tabela(["Método", "Entre 5 textos", "Entre 19 seções"], linhas, [52, 24, 24])
             + "</div>"
             f"<p>O BM25 acertou {pct(doc[B][0])} entre os textos inteiros, e a busca por "
             f"significado, {pct(doc[C][0])}. São duas perguntas de diferença em 16, pouco para "
             "afirmar que um é melhor (ver a seção 3), mas o suficiente para mostrar que a busca "
             "por palavras não ficou atrás. Entre as seções, o BM25 acertou "
             f"{pct(sec[B][0])} e a contagem de palavras, {pct(sec[A][0])}. Como o sistema devolve "
             "trechos, e não documentos inteiros, foi esse resultado que levou à troca da contagem "
             "pelo BM25, usado em todos os experimentos seguintes.</p>")

    tipos = ["Com as palavras do texto", "Com outras palavras"]
    l_doc = [[nome(k)] + [pct(x) for x in var_doc[k][:2]] for k in (A, B, C)]
    c.append("<h2>2. Onde cada busca falha?</h2>"
             "<p>Quem consulta uma norma nem sempre conhece os termos dela. Por isso as perguntas "
             "foram escritas de três jeitos: repetindo as palavras do texto, com outras palavras "
             "(como faria quem não leu o documento) e com erros de digitação.</p>"
             "<div class='bloco'><p class='titulo-tab'>Tabela 2 – Primeiro resultado certo, por "
             "jeito de escrever a pergunta (entre 5 textos).</p>"
             + tabela(["Método"] + tipos, l_doc, [52, 24, 24]) + "</div>"
             "<p>As buscas por palavras acertaram todas as perguntas que repetem o texto e caíram "
             "bastante quando a pergunta usou outras palavras. Entre as seções a queda foi maior: a "
             f"contagem acertou só {pct(var_sec[A][1])} dessas perguntas. A busca por significado "
             "quase não muda com as palavras, mas erra mais justamente quando a pergunta repete o "
             "texto. Esse ponto fraco da busca por palavras levou ao teste de expansão da pergunta, "
             "em que o modelo de linguagem sugere os termos técnicos antes da busca (Seção 8.9 do "
             "artigo).</p>")

    c.append("<h2>3. As diferenças são confiáveis?</h2>"
             "<p>Com 16 perguntas, cada pergunta vale pouco mais de seis pontos percentuais, e uma "
             "diferença pequena pode ser sorte. Cada par de métodos passou por um teste estatístico "
             "que calcula a chance de a diferença ter aparecido por acaso; em geral, só se aceita "
             "uma diferença como real quando essa chance fica abaixo de 5%. Entre os textos "
             f"inteiros, a menor chance foi de {para_cem(p_doc)}% (75 comparações) e, entre as "
             f"seções, de {para_cem(p_sec)}% (15 comparações). Nenhuma diferença pôde ser "
             "confirmada.</p>"
             "<p>O teste, então, não diz que a busca por palavras é melhor nem pior que a busca por "
             "significado. Diz que, nessa escala, as duas ficaram próximas. Foi por isso que o "
             "artigo tratou a escolha da busca por palavras como uma questão de simplicidade de "
             "operação.</p>")

    c.append("<h2>4. A mesma pergunta traz sempre o mesmo resultado?</h2>"
             "<p>Uma fonte conferida hoje precisa continuar aparecendo amanhã. Cada método "
             "respondeu às perguntas 30 vezes, e as listas foram comparadas. A busca por palavras, "
             "as combinações e a busca por significado exata deram sempre a mesma lista. A versão "
             "aproximada, que usa um índice para acelerar a busca em acervos grandes, mudou o "
             "resultado de todas as perguntas e deixou de fora, em média, "
             f"{det['C′ (IVF), 30 sementes'][2]} do que a busca exata encontrava.</p>"
             "<p>O teste foi repetido no pgvector, a extensão do PostgreSQL para vetores, com até "
             "20.000 textos artificiais. Com a configuração recomendada pela documentação, o "
             "índice aproximado foi tão estável quanto a busca exata (Tabela D3, no fim do anexo). A "
             "instabilidade vinha da configuração usada no primeiro teste. Dar sempre o mesmo "
             "resultado, portanto, não é uma vantagem só da busca por palavras.</p>")

    c.append("<h2>5. E quando a pergunta não tem resposta?</h2>"
             "<p>Nas duas perguntas sem resposta, todos os métodos devolveram algum texto. Na "
             "busca por palavras, basta uma palavra em comum para um texto receber nota, e nem uma "
             "nota mínima de corte resolveu. No sistema completo, quem reconhece a falta de "
             "resposta é o modelo de linguagem, que recebe só as fontes encontradas e, nos outros "
             "acervos, declarou que a informação não consta em todas as perguntas sem resposta "
             "(Seção 8.2 do artigo).</p>")

    c.append("<h2>6. A origem de cada trecho está gravada?</h2>"
             "<p>É a promessa central do sistema: toda fonte aponta documento e página. Das 155 "
             "partes que o sistema guardou no banco para os cinco textos, só 32,9% tinham a origem "
             "completa na primeira medição. O motivo era um defeito real: a tabela de trechos não "
             "tinha coluna de página. A coluna foi criada (45,2%), e os dados gerais do documento, "
             "como o número de parágrafos, que não pertencem a nenhuma página, passaram a declarar "
             "que vêm do documento inteiro (100%).</p>")

    c.append("<h2>O que o teste decidiu</h2>"
             "<p>O teste mostrou que a busca por palavras não fica atrás da busca por significado "
             "e levou à troca da contagem de palavras pelo BM25. Também apontou o ponto fraco da "
             "busca por palavras, a pergunta escrita com outras palavras, que motivou a expansão da "
             "pergunta. E encontrou a falta da página na tabela de trechos, corrigida antes dos "
             "experimentos com PDFs reais.</p>")
    return "".join(c)


LEGENDA_DETALHES = [
    ("1º lugar", "Porcentagem de perguntas em que o primeiro resultado era o texto certo.",
     "P@1"),
    ("3 primeiros", "Porcentagem dos três primeiros resultados que eram certos. Como quase toda "
                    "pergunta tem um só texto certo, nem uma busca perfeita chega perto de 100%; "
                    "o máximo possível aparece na última linha.", "P@3"),
    ("5 primeiros", "Porcentagem de perguntas em que o texto certo apareceu entre os cinco "
                    "primeiros resultados.", "R@5"),
    ("Posição", "Nota de 0 a 100 pela posição do texto certo: 100 em primeiro lugar, 50 em "
                "segundo, 33,3 em terceiro; a nota final é a média.", "MRR"),
    ("Ordem", "Nota de 0 a 100 parecida com a anterior, considerando os dez primeiros "
              "resultados.", "nDCG@10"),
    ("(de X a Y)", "Intervalo de confiança de 95%, calculado sorteando as perguntas 10 mil "
                   "vezes.", "bootstrap"),
]


def detalhe(n: int, arq: str, legenda: str, grupos: list[tuple[list[str], list[int],
            list[int]]]) -> str:
    """Tabela D<n>: uma ou mais tabelas com colunas do .tex (indices), numeros
    convertidos por converte e primeira coluna traduzida."""
    _, linhas = celulas_tex(TAB / arq)
    html_t = ""
    for cab, idx, larg in grupos:
        corpo = [[rotulo_detalhe(l[0])] + [converte(l[i]) for i in idx] for l, _ in linhas]
        html_t += tabela(cab, corpo, larg, "detalhe", [r for _, r in linhas])
    return (f"<div class='bloco'><p class='titulo-tab'>Tabela D{n} – {legenda}</p>"
            f"{html_t}</div>")


MEDIDAS = {"P@1": "1º lugar", "P@3": "3 primeiros", "R@5": "5 primeiros", "MRR": "Posição",
           "nDCG@10": "Ordem"}


def detalhe_comparacoes(n: int, arq: str, legenda: str) -> str:
    _, linhas = celulas_tex(TAB / arq)
    pares: dict[str, list[list[str]]] = {}
    for l, _ in linhas:
        pares.setdefault(l[0], []).append(l[1:])
    blocos = ""
    for par, ls in pares.items():
        a, b = (x.strip() for x in par.split(" vs "))
        tr = "".join(
            f"<tr><td>{MEDIDAS.get(m.strip(), m)}</td><td class=n>{converte(d)}</td>"
            f"<td class=n>{q} ({v.replace('/', ' e ')})</td><td class=n>{converte(p)}</td>"
            f"<td class=n>{converte(h)}</td></tr>" for m, d, q, p, h, v in ls)
        blocos += (f"<table class='par'><colgroup><col style='width:20%'><col style='width:18%'>"
                   f"<col style='width:24%'><col style='width:19%'><col style='width:19%'>"
                   f"</colgroup><thead><tr><th colspan=5 class='nome-par'>{METODOS[a]} × "
                   f"{METODOS[b]}</th></tr></thead><tbody>{tr}</tbody></table>")
    return (f"<p class='titulo-tab'>Tabela D{n} – {legenda}</p>"
            f"<div class='colunas'>{blocos}</div>")


def a7_detalhes() -> tuple[str, list[Path]]:
    pct_ = "(%)"
    nota = "(nota 0–100)"
    resumo_cols = [(["Método", f"1º lugar {pct_}", f"3 primeiros {pct_}",
                     f"5 primeiros {pct_}", f"Posição {nota}", f"Ordem {nota}"],
                    [1, 2, 3, 4, 5], [25, 15, 15, 15, 15, 15])]
    var_cols = [(["Método", "Palavras do texto (%)", "Outras palavras (%)",
                  "Erros de digitação (%)", "Queda com outras palavras (pontos)"],
                 [1, 2, 3, 4], [32, 17, 17, 17, 17])]
    neg_cols = [(["Método", "Devolveu algo (%)", "Resultados devolvidos (média)"], [1, 2],
                 [50, 25, 25])]
    itens = [
        ("retrieval-summary.tex", "Todas as medidas, escolhendo entre 5 textos. Abaixo de "
         "cada número, o intervalo de confiança de 95%.", resumo_cols),
        ("retrieval-summary-section.tex", "Todas as medidas, escolhendo entre 19 seções "
         "(só os métodos medidos nesse nível).", resumo_cols),
        ("ann-pgvector.tex", "Busca aproximada no pgvector 0.8.6, com textos artificiais "
         "(768 números cada) e 20 perguntas. As duas últimas colunas contam as perguntas cujos "
         "10 primeiros resultados mudaram (5 reconstruções do índice; 5 ordens de inserção).",
         [(["Índice", "Textos", "Igual à busca exata nos 10 primeiros", "Mudou ao "
            "reconstruir", "Mudou com outra ordem"], [1, 2, 3, 4], [36, 14, 20, 15, 15])]),
    ]
    legenda = "".join(f"<tr><td class='id'><strong>{s_}</strong></td><td>{d}</td>"
                      f"<td class='lit'>{lit}</td></tr>" for s_, d, lit in LEGENDA_DETALHES)
    c = (f"<h2>{MARCA_DETALHES} (para conferência)</h2>"
         "<p>As seções anteriores bastam para entender o teste. Esta parte é para "
         "conferência: traz todas as medidas calculadas, em porcentagem ou em nota de 0 a 100. "
         "O Quadro 1 explica cada uma, com o nome usado na literatura de recuperação de "
         "informação.</p>"
         "<div class='bloco'><p class='titulo-tab'>Quadro 1 – Medidas usadas nas tabelas "
         "completas.</p>"
         "<table class='legenda'><colgroup><col style='width:17%'><col style='width:63%'>"
         "<col style='width:20%'></colgroup><thead><tr><th>Medida</th><th>O que mede</th>"
         f"<th>Nome técnico</th></tr></thead><tbody>{legenda}</tbody></table></div>")
    usados = []
    for i, (arq, leg, grupos) in enumerate(itens, 1):
        usados.append(TAB / arq)
        c += detalhe(i, arq, leg, grupos)
    c += ("<p>As demais tabelas do teste (por jeito de escrever a pergunta, perguntas sem "
          "resposta, nota mínima de corte, estabilidade, origem completa e as 90 comparações "
          "estatísticas, 75 entre textos e 15 entre seções) estão no repositório do projeto, em "
          "<code>docs/article/tables</code>, e os dados brutos em <code>docs/article/results"
          "</code>.</p>")
    return f"<div class='detalhes'>{c}</div>", usados


def a7() -> tuple[str, list[Path]]:
    detalhes, usados = a7_detalhes()
    estilo = "<style>body { font-size: 10.5pt; } table { font-size: 9.5pt; }</style>"
    return (documento("A7", "Resultados do teste de busca", "", estilo + a7_corpo() + detalhes),
            usados)


# ------------------------------------------------------------------- A8

REGISTRO_A8 = ART / "results" / "alteracoes-pos-medicao.md"

# Uma linha por correcao do registro que mudou numero publicado, na ordem do
# registro (linhas 1, 2, 3, 4, 6 e 7; a linha 5 vai no texto abaixo da tabela).
# Antes e Depois: lista de (rotulo, valor); todo numero aqui e conferido contra
# o registro por confere_registro.
CORRECOES_A8 = [
    ("Na correção das perguntas usadas durante o desenvolvimento (PDFs do IFPR), respostas "
     "certas escritas de outro jeito passaram a ser aceitas, e algumas perguntas ganharam "
     "mais de uma página certa.",
     "Respostas certas eram marcadas como erradas só por usarem outras palavras, e a "
     "informação de algumas perguntas estava em mais de uma página.",
     [("Sistema sem mudanças", "204 de 216 respostas")],
     [("Sistema sem mudanças", "213 de 216 respostas")]),
    ("Nas perguntas sem resposta do teste final, mais formas de recusar passaram a contar "
     "como certas (como “não consta” e “não há”).",
     "Recusas corretas escritas de outro jeito, como “os documentos não mencionam o preço”, "
     "eram marcadas como erradas.",
     [("Produto e consumidor, original", "78,3%")],
     [("Produto e consumidor, original", "84,1%")]),
    ("Na medida que confere se a página certa aparece entre as fontes, o nome de cada "
     "documento passou a ser aceito em todas as suas formas.",
     "O sistema grava o título do documento, e a correção esperava um código. A fonte certa "
     "contava como erro quando os dois eram escritos de jeitos diferentes. Nenhuma resposta "
     "mudou de certa para errada.",
     [("Original", "37,1%"), ("Ampliada", "53,2%")],
     [("Original", "58,1%"), ("Ampliada", "82,3%")]),
    ("Recusas (“não há informação sobre isso”) deixaram de contar como certas nas perguntas "
     "que têm resposta. Nas perguntas que trazem um dado errado, a resposta passou a ter de "
     "corrigir esse dado.",
     "Uma resposta fixa de recusa, igual para todas as perguntas, acertaria 24 perguntas.",
     [("Original", "81,7%"), ("Ampliada", "93,0%"), ("Com dado errado", "35 de 36"),
      ("Chance de a vantagem da ampliada ser sorte", "1,2%"),
      ("Teste da Seção 8.8 do artigo", "9 de 12")],
     [("Original", "70,0%"), ("Ampliada", "90,1%"), ("Com dado errado", "13 de 36"),
      ("Chance de a vantagem da ampliada ser sorte", "0,01%"),
      ("Teste da Seção 8.8 do artigo", "7 a 8 de 12")]),
    ("Respostas que negam o assunto (“não existe”, “não consta”) ou que dizem o contrário da "
     "resposta certa também deixaram de contar como certas.",
     "A frase “Não existe previsão sobre isso nos documentos.” acertava as 12 perguntas que "
     "perguntam se um assunto existe, e 6 recusas a uma mesma pergunta contavam como certas.",
     [("Original", "70,0%"), ("Ampliada", "90,1%")],
     [("Original", "68,5%"), ("Ampliada", "88,7%")]),
    ("Mais formas de recusa passaram a ser barradas, e uma resposta certa escrita “píxeis” "
     "(plural de pixel) passou a ser aceita.",
     "“Píxeis” sem acento não contém a palavra “pixel”, e a resposta certa era marcada como "
     "errada. As outras recusas barradas não apareciam nas respostas reais.",
     [("Original", "68,5%"), ("Ampliada", "88,7%")],
     [("Original", "69,5%"), ("Ampliada", "88,7%")]),
]
TOTAL_A8 = ("<strong>Efeito somado das três últimas correções</strong>",
            "Respostas que não respondem nada deixaram de contar como certas.",
            [("Original", "81,7%"), ("Ampliada", "93,0%")],
            [("Original", "69,5%"), ("Ampliada", "88,7%")])


def celula_valores(vals: list[tuple[str, str]]) -> str:
    return "<br>".join((f"{r}: " if r else "") + f"<strong>{v}</strong>" for r, v in vals)


def confere_registro(texto_html: str) -> None:
    """Todo numero do A8 precisa existir no registro: contagens e numeros sem %
    como estao; porcentagens como a proporcao decimal do registro (81,7% <-
    0,817; 0,01% <- 0,0001), com o mesmo numero de casas do texto."""
    texto = html.unescape(re.sub(r"<[^>]+>", " ", texto_html))
    reg = REGISTRO_A8.read_text(encoding="utf-8")
    reg_nums = set(re.findall(r"\d+(?:,\d+)?", reg))
    reg_dec = [Decimal(x.replace(",", ".")) for x in reg_nums if "," in x]
    faltam = []
    for m in re.finditer(r"(\d+(?:,\d+)?)(\s*%)?", texto):
        num, por = m[1], m[2]
        if por:
            casas = len(num.split(",")[1]) if "," in num else 0
            v = Decimal(num.replace(",", "."))
            ok = any((d * 100).quantize(Decimal(1).scaleb(-casas)) == v for d in reg_dec)
        else:
            ok = num in reg_nums
        if not ok:
            faltam.append(m.group().strip())
    if faltam:
        raise SystemExit(f"A8: numeros fora do registro: {faltam}")
    print(f"  A8: {len(re.findall(r"\d+(?:,\d+)?", texto))} numeros conferidos no registro")


def a8() -> str:
    linhas = CORRECOES_A8 + [TOTAL_A8]
    tr = "".join(
        f"<tr{' class=regra' if i == len(linhas) - 1 else ''}><td>{o}</td><td>{p}</td>"
        f"<td>{celula_valores(a)}</td><td>{celula_valores(d)}</td></tr>"
        for i, (o, p, a, d) in enumerate(linhas))
    corpo = (
        "<p>Depois de ver os resultados, algumas regras de correção das respostas foram "
        "ajustadas. Como isso poderia deixar os números artificialmente melhores, cada ajuste "
        "foi anotado com o motivo e o efeito. O ajuste mais importante deixou o resultado do "
        "sistema pior, e não melhor.</p>"
        "<p>A correção é automática: cada pergunta tem um padrão de texto que a resposta certa "
        "precisa conter e, às vezes, padrões que a tornam errada, como uma recusa. A maioria "
        "das linhas abaixo trata do teste final, com 213 respostas sobre leis, normas e "
        "manuais públicos, escritas depois de o sistema ficar pronto. Nele, cada número "
        "aparece de dois modos: com a pergunta como o usuário escreveu (<em>original</em>) e "
        "com a pergunta <em>ampliada</em> por um LLM, que acrescenta termos técnicos antes "
        "da busca.</p>"
        "<p class='titulo-tab'>Quadro 1 – Correções que mudaram números publicados, na ordem "
        "em que foram feitas.</p>"
        "<table class='quadro a8'><colgroup><col style='width:29%'><col style='width:29%'>"
        "<col style='width:21%'><col style='width:21%'></colgroup><thead><tr>"
        "<th>O que mudou</th><th>Por quê</th><th>Antes</th><th>Depois</th></tr></thead>"
        f"<tbody>{tr}</tbody></table>"
        "<p class='nota-a8'>As regras da busca também foram ajustadas olhando as 72 perguntas "
        "usadas durante o desenvolvimento. Por isso, os números dessas perguntas não valem "
        "como teste independente; o teste de verdade é o teste final, com perguntas escritas "
        "depois.</p>"
        + resumo("as três correções mais importantes fizeram o acerto do sistema cair de 81,7% "
                 "para 69,5%, porque respostas que não respondiam nada deixaram de contar como "
                 "certas. As outras corrigiram erros da própria correção, que marcava respostas "
                 "certas como erradas; o artigo publica sempre os números de depois da última "
                 "correção."))
    confere_registro(corpo)
    return documento("A8", "Correções feitas depois de medir", "", corpo)


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
