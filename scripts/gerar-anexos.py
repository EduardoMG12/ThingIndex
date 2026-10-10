#!/usr/bin/env python3
"""Gera os anexos do artigo em PDF (pasta anexos/ na raiz do repositorio).

Orientacao do colegiado: diagramas, prototipos e tabelas completas ficam no
GitHub, com o link citado no artigo. Cada anexo e montado como HTML numa pasta
temporaria e impresso em PDF A4 pelo Chromium headless. Todo conteudo vem de
arquivos versionados: imagens de docs/article/anexos/, tabelas LaTeX de
docs/article/tables/, o registro docs/article/results/alteracoes-pos-medicao.md
e o esquema docs/schema/thingindex.dbml. Nada e digitado aqui alem dos textos
de apresentacao, dos requisitos (copiados do artigo) e das legendas das telas.

Uso: python3 scripts/gerar-anexos.py [pasta-temporaria]
"""

from __future__ import annotations

import html
import re
import subprocess
import sys
import tempfile
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
ART = RAIZ / "docs" / "article"
FIG = ART / "anexos"
TAB = ART / "tables"
SAIDA = RAIZ / "anexos"
SUBTITULO = "TCC ThingIndex — Charles Eduardo Mello Guimarães — IFPR Palmas"

CSS = """
@page { size: A4 %(orient)s; margin: 2cm; }
body { font-family: "Noto Sans", "Liberation Sans", sans-serif; font-size: 10pt;
       color: #111; line-height: 1.4; margin: 0; }
h1 { font-size: 16pt; margin: 0 0 2pt 0; }
h2 { font-size: 12pt; margin: 14pt 0 4pt 0; break-after: avoid; }
h3 { font-size: 10.5pt; margin: 10pt 0 3pt 0; break-after: avoid; }
.sub { color: #555; font-size: 9pt; margin: 0 0 8pt 0;
       border-bottom: 1px solid #999; padding-bottom: 4pt; }
.leia { background: #f3f3f3; padding: 6pt 8pt; margin: 0 0 10pt 0; font-size: 9.5pt; }
table { border-collapse: collapse; width: 100%%; margin: 4pt 0 10pt 0; font-size: 8.5pt; }
thead { display: table-header-group; }
tr { break-inside: avoid; }
th, td { padding: 2.5pt 5pt; vertical-align: top; }
thead th { border-top: 1.5px solid #000; border-bottom: 1px solid #000; text-align: left; }
tbody tr:last-child td { border-bottom: 1.5px solid #000; }
tr.regra td { border-top: 1px solid #000; }
td small { color: #555; font-size: 7.5pt; }
caption { caption-side: top; text-align: left; font-size: 9pt; padding-bottom: 4pt; }
.grade td, .grade th { border: 1px solid #bbb; }
code { font-family: "Liberation Mono", monospace; font-size: 0.92em; }
figure { margin: 0; text-align: center; break-inside: avoid; }
figure img { max-width: 100%%; max-height: %(imgmax)s; object-fit: contain; }
figcaption { text-align: left; font-size: 9pt; margin-top: 6pt; }
.pagina { break-before: page; }
.rotulo { font-weight: bold; font-size: 9pt; margin: 12pt 0 2pt 0; break-after: avoid; }
caption { break-after: avoid; }
.md td:nth-child(2) { white-space: nowrap; }
dl dt { font-weight: bold; } dl dd { margin: 0 0 3pt 14pt; }
"""


def esc(s: str) -> str:
    return html.escape(s, quote=False)


def pagina(n: int, titulo: str, leia: str, corpo: str, paisagem=False) -> str:
    css = CSS % {"orient": "landscape" if paisagem else "portrait",
                 "imgmax": "128mm" if paisagem else "215mm"}
    return (f'<!doctype html><html lang="pt-BR"><head><meta charset="utf-8">'
            f"<title>Anexo {n}</title><style>{css}</style></head><body>"
            f"<h1>Anexo {n} — {esc(titulo)}</h1><p class='sub'>{SUBTITULO}</p>"
            f"<p class='leia'>{leia}</p>{corpo}</body></html>")


def imprimir(html_txt: str, nome: str, build: Path) -> None:
    fonte = build / (nome + ".html")
    fonte.write_text(html_txt, encoding="utf-8")
    pdf = SAIDA / (nome + ".pdf")
    subprocess.run(["chromium", "--headless", "--disable-gpu", "--no-pdf-header-footer",
                    "--allow-file-access-from-files", f"--print-to-pdf={pdf}",
                    fonte.as_uri()], check=True, capture_output=True)
    print("gerado", pdf.relative_to(RAIZ))


def figura(caminho: Path, legenda: str = "", altura: str = "") -> str:
    leg = f"<figcaption>{legenda}</figcaption>" if legenda else ""
    est = f' style="max-height:{altura}"' if altura else ""
    return f'<figure><img src="{caminho.as_uri()}" alt=""{est}>{leg}</figure>'


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
    """Converte o LaTeX de uma celula, legenda ou item em HTML."""
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


def tabela_tex(caminho: Path) -> str:
    """Converte table/tabular ou longtable de um .tex gerado em <table> HTML."""
    s = "\n".join(l for l in caminho.read_text(encoding="utf-8").splitlines()
                  if not l.lstrip().startswith("%"))
    k = s.find(r"\caption{")
    legenda, _ = chave(s, k + len(r"\caption"))
    m = re.search(r"\\begin\{(tabular|longtable)\}\{([^}]*)\}", s)
    amb, alin = m.group(1), alinhamento(m.group(2))
    corpo = s[m.end():s.find(r"\end{" + amb + "}")]
    if amb == "longtable":  # cabecalho do primeiro trecho + corpo apos \endhead
        cab = corpo[corpo.find(r"\toprule"):corpo.find(r"\endfirsthead")]
        corpo = cab + corpo[corpo.find(r"\endhead") + len(r"\endhead"):]
    corpo = re.sub(r"\\(caption|label)\{.*?\}\s*(\\\\)?", "", corpo)
    linhas, cab, regra, depois_cab = [], None, False, False
    for bruto in separa(corpo, r"\\"):
        t = bruto.replace(r"\toprule", "").replace(r"\bottomrule", "")
        if r"\midrule" in t:
            regra = True
            t = t.replace(r"\midrule", "")
        t = t.strip()
        if not t:
            continue
        cels, html_cels, col = separa(t, "&"), [], 0
        for c in cels:
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
    return (f"<table><caption>{inline(legenda)}</caption><thead><tr>{th}</tr></thead>"
            f"<tbody>{tr}</tbody></table>")


# ------------------------------------------------------------- Markdown -> HTML

def md_inline(s: str) -> str:
    s = esc(s)
    s = re.sub(r"`([^`]+)`", r"<code>\1</code>", s)
    s = re.sub(r"\*\*(.+?)\*\*", r"<strong>\1</strong>", s)
    s = re.sub(r"(?<![\w*])\*(?!\s)(.+?)(?<!\s)\*(?![\w*])", r"<em>\1</em>", s)
    return re.sub(r"\[([^\]]+)\]\(([^)]+)\)", r'<a href="\2">\1</a>', s)


def markdown(txt: str, rebaixa: int = 1) -> str:
    """Conversor minimo: titulos, paragrafos, listas com '-' e tabelas."""
    out, par, linhas = [], [], txt.splitlines()

    def fecha():
        if par:
            out.append("<p>" + md_inline(" ".join(par)) + "</p>")
            par.clear()

    i = 0
    while i < len(linhas):
        l = linhas[i].rstrip()
        if m := re.match(r"(#+)\s+(.*)", l):
            fecha()
            n = min(len(m.group(1)) + rebaixa, 6)
            out.append(f"<h{n}>{md_inline(m.group(2))}</h{n}>")
        elif l.startswith("|"):
            fecha()
            bloco = []
            while i < len(linhas) and linhas[i].startswith("|"):
                bloco.append([c.strip() for c in linhas[i].strip().strip("|").split("|")])
                i += 1
            th = "".join(f"<th>{md_inline(c)}</th>" for c in bloco[0])
            tr = "".join("<tr>" + "".join(f"<td>{md_inline(c)}</td>" for c in r) + "</tr>"
                         for r in bloco[2:])
            out.append(f"<table class='grade md'><thead><tr>{th}</tr></thead><tbody>{tr}</tbody></table>")
            continue
        elif l.lstrip().startswith("- "):
            fecha()
            itens = []
            while i < len(linhas) and linhas[i].strip():
                x = linhas[i].strip()
                if x.startswith("- "):
                    itens.append(x[2:])
                else:
                    itens[-1] += " " + x
                i += 1
            out.append("<ul>" + "".join(f"<li>{md_inline(x)}</li>" for x in itens) + "</ul>")
            continue
        elif not l.strip():
            fecha()
        else:
            par.append(l.strip())
        i += 1
    fecha()
    return "\n".join(out)


# ------------------------------------------------------------------- Anexos

# Copiados de docs/article/anexos/README.md (secao "Requisitos funcionais").
REQUISITOS = [
    ("RF01", "Cadastro, autenticação, redefinição de senha e edição de perfil", r"rotas \texttt{/auth/*} e \texttt{/user/me}"),
    ("RF02", "Criar, listar e renomear categorias; criar coleções e mover entre categorias", "rotas de categoria e coleção"),
    ("RF03", "Enviar documento para uma coleção (PDF, HTML e XLSX), com validação de tipo e tamanho", r"\texttt{POST /documents/upload}"),
    ("RF04", "Indexar de forma assíncrona páginas, parágrafos, tabelas, sumário, trechos e metadados", r"\emph{worker} \texttt{indexer}"),
    ("RF05", "Enriquecer o índice com LLM, opcional e configurável por usuário", "preferências de indexação"),
    ("RF06", "Recuperar o contexto de uma coleção por API", "rota de contexto da coleção"),
    ("RF07", r"Responder perguntas sobre a coleção com as fontes citadas, em \emph{streaming}", r"\texttt{POST /chat/stream} (SSE)"),
    ("RF08", "Abrir cada fonte no PDF, na página e na região de origem", r"\emph{workspace} do \emph{frontend}"),
    ("RF09", "Configurar provedores de LLM e de armazenamento, com credencial cifrada", r"rotas \texttt{/user/providers}"),
    ("RF10", r"Acompanhar status, tentativas, \emph{tokens} e custo da indexação", "painel de operações"),
    ("RF11", "Registrar trilha de auditoria de documentos e provedores", "tabelas de eventos de auditoria"),
    ("RF12", "Reprocessar documento com falha e marcar documento cujo arquivo sumiu", r"\texttt{POST /documents/:id/retry}"),
]

# Copiados da secao Requisitos de docs/article/artigo.tex (branch main).
NAO_FUNCIONAIS = [
    "integridade referencial e transações ACID no armazenamento das estruturas extraídas;",
    "separação em camadas, com dependência em um só sentido;",
    r"segurança: senha com bcrypt, sessão por \emph{token} JWT (\emph{JSON Web Token}), credencial de provedor cifrada com AES-GCM (cifra simétrica autenticada), limite de requisições por conexão e por e-mail nas rotas de autenticação e por usuário nas de envio e \emph{chat}; sem um proxy reverso configurado para repassar o IP, o limite por conexão é uma cota única da instalação, que terceiros podem esgotar, e o limite por e-mail permite bloquear temporariamente o acesso a uma conta conhecida; as duas escolhas são aceitáveis no uso individual e impedem que o IP seja forjado;",
    r"privacidade: com um modelo servido na própria instalação (por exemplo, pelo Ollama local), nenhum conteúdo sai dela; com provedor externo, incluído o Ollama em nuvem, a pergunta, o histórico da conversa e os trechos recuperados, e, se o enriquecimento estiver ligado, o texto das páginas, são enviados a ele, e cabe à organização a base legal para esse tratamento, nos termos da Lei Geral de Proteção de Dados (LGPD);",
    "execução reproduzível por Docker Compose;",
    "recuperação determinística: a mesma pergunta sobre o mesmo índice produz as mesmas fontes na mesma ordem.",
]

# Legendas das figuras fig:workspace e fig:operacoes de docs/article/artigo.tex (main).
LEGENDA_WORKSPACE = r"\emph{Workspace} de uma coleção de normas de segurança do trabalho: \emph{chat} com a primeira fonte expandida (à esquerda), PDF na página citada com o parágrafo destacado (ao centro) e o resumo do documento gerado pelo LLM na indexação (à direita)."
LEGENDA_OPERACOES = r"Tela de operações, voltada à manutenção técnica: modelo ativo, cobertura da indexação, \emph{tokens} estimados da indexação (caracteres divididos por 4; custo não informado, porque não há preço cadastrado para o modelo), um escore heurístico da estrutura extraída, não validado contra o acerto, e contadores de nova tentativa, fila de mensagens mortas, falhas permanentes e órfãos. A tela não indica se o provedor configurado é local ou externo."


def a1() -> str:
    tr = "".join(f"<tr><td>{i}</td><td>{inline(r)}</td><td>{inline(im)}</td></tr>"
                 for i, r, im in REQUISITOS)
    nf = "".join(f"<li>{inline(x)}</li>" for x in NAO_FUNCIONAIS)
    corpo = (f"<h2>Requisitos funcionais</h2><table class=grade><thead><tr><th>Id</th>"
             f"<th>Requisito</th><th>Implementação</th></tr></thead><tbody>{tr}</tbody></table>"
             f"<h2>Requisitos não funcionais</h2><ul>{nf}</ul>")
    return pagina(1, "Requisitos funcionais e não funcionais",
                  "Lista dos requisitos do sistema (Seção 6.1 do artigo). Na tabela, cada linha "
                  "é um requisito funcional e a coluna Implementação diz onde ele está no código "
                  "(rota da API, <em>worker</em> ou tela). Abaixo, os requisitos não funcionais, "
                  "copiados do texto do artigo.", corpo)


def diagrama(n, titulo, leia, arquivo, paisagem, legenda=""):
    leia += " A imagem está em alta resolução: use o zoom do leitor de PDF para ler os detalhes."
    return pagina(n, titulo, leia, figura(FIG / "diagramas" / arquivo, legenda), paisagem)


BRACOS = [("A", "contagem de termos da pergunta que aparecem no texto"),
          ("B", "BM25, ranqueamento lexical clássico que pondera termos raros"),
          ("C", "denso exato: similaridade entre vetores de <em>embedding</em>, comparando com todos os itens"),
          ("C′", "denso com índice aproximado IVF (pgvector), que compara só com parte dos itens"),
          ("D", "fusão de A e B por RRF (soma de 1/(k + posição) em cada lista)"),
          ("E", "fusão de B e C por RRF")]
METRICAS = [("P@1", "o primeiro resultado é relevante? (média sobre as consultas)"),
            ("P@3", "fração dos 3 primeiros resultados que é relevante"),
            ("R@5", "fração dos itens relevantes que aparece entre os 5 primeiros"),
            ("MRR", "média de 1/posição do primeiro resultado relevante (1 = sempre em 1º)"),
            ("nDCG@10", "qualidade da ordem dos 10 primeiros, premiando relevantes no topo (1 = ordem ideal)")]

TABELAS_A7 = [
    ("Desempenho por braço", ["retrieval-summary.tex", "retrieval-summary-section.tex"]),
    ("Variantes de consulta", ["retrieval-variants.tex", "retrieval-variants-section.tex"]),
    ("Consultas negativas e limiar de corte", ["retrieval-negatives.tex", "retrieval-negatives-section.tex", "threshold.tex"]),
    ("Comparações pareadas", ["retrieval-significance.tex", "retrieval-significance-section.tex"]),
    ("Determinismo, índices aproximados e proveniência", ["dense-determinism.tex", "ann-pgvector.tex", "provenance.tex"]),
]


def a7() -> str:
    bra = "".join(f"<dt>{b}</dt><dd>{d}</dd>" for b, d in BRACOS)
    met = "".join(f"<dt>{b}</dt><dd>{d}</dd>" for b, d in METRICAS)
    corpo = (f"<h2>Legenda</h2><h3>Braços (estratégias de recuperação comparadas)</h3><dl>{bra}</dl>"
             f"<h3>Métricas (todas de 0 a 1; maior é melhor)</h3><dl>{met}</dl>"
             "<p>Números entre colchetes são intervalos de confiança de 95%. Vírgula é o "
             "separador decimal. Na coluna Vitórias A/B das comparações pareadas, A e B são o "
             "primeiro e o segundo braço da comparação, não os braços A e B. A unidade de "
             "recuperação diz o que conta como acerto: o documento certo ou a seção certa.</p>")
    n = 0
    for titulo, arqs in TABELAS_A7:
        corpo += f"<h2 class=pagina>{titulo}</h2>"
        for a in arqs:  # o sufixo -section e o mesmo de scripts/analyze.py (UNIDADE)
            n += 1
            unidade = ("unidade de recuperação: seção" if "-section" in a else
                       "unidade de recuperação: documento" if a.startswith("retrieval-") else "")
            corpo += (f"<p class=rotulo>Tabela 7.{n} — <code>{a}</code>"
                      f"{' — ' + unidade if unidade else ''}</p>" + tabela_tex(TAB / a))
    return pagina(7, "Tabelas completas de recuperação",
                  "Versão integral das tabelas de recuperação das Seções 8.1 a 8.3 do artigo, "
                  "convertidas sem alteração dos arquivos <code>docs/article/tables/*.tex</code>, "
                  "que são gerados por script a partir dos CSV de resultados. Cada tabela traz "
                  "acima a legenda original; leia primeiro a legenda de braços e métricas abaixo.",
                  corpo)


def a8() -> str:
    txt = (ART / "results" / "alteracoes-pos-medicao.md").read_text(encoding="utf-8")
    return pagina(8, "Alterações feitas depois de medir",
                  "Registro de toda alteração de gabarito ou de critério feita depois de ver os "
                  "resultados (Seção 5 do artigo), reproduzido de "
                  "<code>docs/article/results/alteracoes-pos-medicao.md</code>. Cada linha da "
                  "tabela diz o que mudou, por quê, e o efeito nos números; os números publicados "
                  "no artigo são os da coluna Efeito.", markdown(txt), paisagem=True)


GRUPOS = [
    ("Núcleo: usuário e acervo", ["users", "document_categories", "collections", "documents"]),
    ("Estrutura extraída dos documentos", ["document_pages", "paragraph_nodes", "table_sections",
     "document_outline_nodes", "document_chunks", "metadata_pairs"]),
    ("Operação: fila, auditoria, provedores, preços, conversas e preferências", ["document_processing_attempts",
     "outbox_events", "document_audit_events", "user_provider_configs", "provider_model_catalogs",
     "provider_audit_events", "pricing_sources", "model_prices", "model_price_histories",
     "chat_conversations", "chat_conversation_entries", "paragraph_highlights", "workspace_view_states",
     "user_indexing_preferences", "enrichment_cache_entries"]),
]


def a9() -> str:
    dbml = (RAIZ / "docs" / "schema" / "thingindex.dbml").read_text(encoding="utf-8")
    refs = {}
    for o, d in re.findall(r"^Ref:\s*(\S+)\s*>\s*(\S+)", dbml, re.M):
        refs[o] = d
    tabelas = {}
    for nome, corpo in re.findall(r"^Table (\w+) \{\n(.*?)^\}", dbml, re.M | re.S):
        cols, idx = [], []
        partes = corpo.split("indexes {")
        for l in partes[0].strip().splitlines():
            m = re.match(r"\s*(\w+)\s+(\S+)(?:\s+\[(.*)\])?", l)
            if m:
                cols.append(m.groups())
        if len(partes) > 1:
            for l in partes[1].splitlines():
                m = re.match(r"\s*(\(.*?\)|\w+)\s+\[(.*)\]", l)
                if m:
                    idx.append(m.group(1) + (" único" if "unique" in m.group(2) else ""))
        tabelas[nome] = (cols, idx)
    agrupadas = [t for _, ts in GRUPOS for t in ts]
    assert sorted(agrupadas) == sorted(tabelas), "grupos do A9 desatualizados frente ao DBML"

    def restr(a):
        if not a:
            return ""
        out = []
        for x in (y.strip() for y in a.split(",")):
            if x == "pk":
                out.append("chave primária")
            elif x == "not null":
                out.append("obrigatório")
            elif x == "unique":
                out.append("único")
            elif x == "increment":
                out.append("autoincremento")
            elif x.startswith("default:"):
                out.append("padrão " + x.split(":", 1)[1].strip().strip("`"))
        return ", ".join(out)

    corpo = ""
    for titulo, nomes in GRUPOS:
        corpo += f"<h2>{titulo} ({len(nomes)} tabelas)</h2>"
        for t in nomes:
            cols, idx = tabelas[t]
            tr = "".join(
                f"<tr><td><code>{c}</code></td><td><code>{esc(tp)}</code></td><td>{restr(a)}</td>"
                f"<td>{('→ <code>' + refs[t + '.' + c] + '</code>') if t + '.' + c in refs else ''}</td></tr>"
                for c, tp, a in cols)
            ix = "; ".join(f"<code>{esc(i)}</code>" for i in idx)
            corpo += (f"<div style='break-inside:avoid'><h3><code>{t}</code></h3><table class=grade><thead><tr><th>Coluna</th>"
                      f"<th>Tipo</th><th>Restrições</th><th>Referência</th></tr></thead>"
                      f"<tbody>{tr}</tbody></table>"
                      + (f"<p style='font-size:8pt;margin-top:-6pt'>Índices: {ix}</p>" if ix else "") + "</div>")
    return pagina(9, f"Esquema do banco de dados ({len(tabelas)} tabelas)",
                  "Listagem das tabelas do PostgreSQL, gerada de <code>docs/schema/thingindex.dbml</code>, "
                  "arquivo que por sua vez é gerado dos modelos da aplicação (<code>api/db/models.go</code>). "
                  "Cada tabela mostra suas colunas, tipo, restrições e, na coluna Referência, a chave "
                  "estrangeira declarada. As colunas <code>user_id</code> indicam o dono do registro "
                  "(posse verificada em toda consulta, sem chave estrangeira declarada; ver Anexo 4). "
                  "O arquivo pode ser aberto em dbdiagram.io para ver o diagrama.", corpo)


def main() -> None:
    build = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(tempfile.mkdtemp(prefix="anexos-"))
    build.mkdir(parents=True, exist_ok=True)
    SAIDA.mkdir(exist_ok=True)
    imprimir(a1(), "A1-requisitos-funcionais", build)
    imprimir(diagrama(2, "Diagrama de casos de uso",
                      "Casos de uso do ThingIndex, derivados das rotas da API (Seção 6.1). Cada "
                      "elipse é uma ação que o usuário faz no sistema; as linhas ligam o ator às ações.",
                      "casos-de-uso.png", True), "A2-casos-de-uso", build)
    imprimir(diagrama(3, "Sequência do envio e da pergunta",
                      "Diagrama de sequência (Seção 6.2) dos dois fluxos principais: o envio e a "
                      "indexação de um documento e uma pergunta à coleção. Leia de cima para baixo; "
                      "cada coluna é um componente e cada seta, uma mensagem entre eles.",
                      "sequencia.png", True), "A3-sequencia-envio-e-pergunta", build)
    imprimir(diagrama(4, "Recorte do esquema com o caminho da proveniência",
                      "Recorte do modelo de dados (Seção 6.3): do usuário até as seis tabelas de "
                      "estrutura extraída, por onde cada fonte citada chega ao documento e à página. "
                      "<strong>Linha contínua</strong> = chave estrangeira declarada; "
                      "<strong>linha tracejada</strong> = posse por <code>user_id</code>, verificada "
                      "em toda consulta. O esquema completo está no Anexo 9.",
                      "esquema-proveniencia.png", True), "A4-esquema-proveniencia", build)
    imprimir(diagrama(5, "Fluxo de recuperação",
                      "Fluxo da recuperação de contexto (Seção 6.5): carga do índice em número fixo "
                      "de consultas, ranqueamento BM25 por tipo de unidade e seleção das fontes. "
                      "Leia de cima para baixo.", "recuperacao.png", False),
             "A5-fluxo-de-recuperacao", build)
    telas = (figura(FIG / "telas" / "workspace.png", "<strong>Tela 1.</strong> " + inline(LEGENDA_WORKSPACE), "112mm")
             + "<div class=pagina></div>"
             + figura(FIG / "telas" / "operacoes.png", "<strong>Tela 2.</strong> " + inline(LEGENDA_OPERACOES),
                    "140mm"))
    imprimir(pagina(6, "Telas do protótipo",
                    "Capturas do protótipo em funcionamento (Seção 6.6), uma por página, com a "
                    "legenda usada no artigo.", telas, True), "A6-telas-do-prototipo", build)
    imprimir(a7(), "A7-tabelas-completas-de-recuperacao", build)
    imprimir(a8(), "A8-alteracoes-pos-medicao", build)
    imprimir(a9(), "A9-esquema-do-banco", build)


if __name__ == "__main__":
    main()
