#!/usr/bin/env python3
"""Confere os numeros afirmados no artigo contra o CSV de resultados.

Prosa se descola do artefato que ela descreve. O caso mais perigoso nao e o
numero errado obvio -- e o numero que ficou desatualizado depois de uma
remedicao: continua plausivel, ninguem relê a frase, e sobrevive ate a defesa.

Este script nao interpreta o texto. Ele mantem uma lista EXPLICITA de
afirmacoes (numero, onde aparece, como e calculado) e recalcula cada uma a
partir de docs/article/results/results.csv. Alterar um numero no artigo sem
atualizar aqui quebra o CI, que e exatamente a intencao.

Uso:
    python3 scripts/check-article-numbers.py
"""

from __future__ import annotations

import collections
import csv
import importlib.util
import sys
import re
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
CSV = RAIZ / "docs" / "article" / "results" / "results.csv"
ARTIGO = RAIZ / "docs" / "article" / "artigo.tex"

# Tolerancia: o artigo publica 3 casas decimais.
TOL = 0.0005


def carregar() -> list[dict]:
    if not CSV.exists():
        sys.exit(f"erro: {CSV} nao existe -- rode 'go run ./cmd/retrieval-eval' primeiro")
    with CSV.open(encoding="utf-8") as f:
        linhas = list(csv.DictReader(f))
    if not linhas:
        sys.exit(f"erro: {CSV} nao tem dados")
    return linhas


def media(rows: list[dict], gran: str, arm: str, metrica: str) -> float | None:
    vals = [
        float(r[metrica])
        for r in rows
        if r["negative"] == "false" and r["granularity"] == gran and r["arm"] == arm
    ]
    return sum(vals) / len(vals) if vals else None


def _numeros_geracao():
    spec = importlib.util.spec_from_file_location("numeros_geracao", RAIZ / "scripts" / "numeros-geracao.py")
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def _proveniencia_geracao():
    spec = importlib.util.spec_from_file_location("proveniencia_geracao", RAIZ / "scripts" / "proveniencia-geracao.py")
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def _br(valor: float, casas: int = 3) -> str:
    return f"{valor:.{casas}f}".replace(".", ",")


def _pct(valor: float) -> str:
    """Proporcao como o artigo a publica: em porcentagem, com uma casa.
    Parte do valor ja arredondado em 3 casas (_br), para que 0,8125 vire 81,2
    exatamente como no texto, e nao 81,3 por arredondamento binario."""
    milesimos = round(float(_br(valor).replace(",", ".")) * 1000)
    return f"{milesimos / 10:.1f}".replace(".", ",")


def _milhar(valor: float) -> str:
    return f"{round(valor):,}".replace(",", ".")


def conferir_geracao(texto: str) -> int:
    """Recalcula dos CSV os numeros das secoes de geracao, bateria,
    enriquecimento e expansao e exige que cada um apareca no artigo como
    esta escrito. Um numero que mude no CSV, ou que suma do texto, falha."""
    ng = _numeros_geracao()
    g = ng.geracao()
    c, e = ng.bateria("off"), ng.bateria("on")
    en = ng.enriquecimento()
    ef = ng.efeito_formulacao(c)
    ex = ng.efeito_expansao(c, e)

    def taxa(par):
        return par[0] / par[1]

    def p_texto(p: float) -> str:
        return r"p < 0{,}001" if p < 0.001 else "p = " + _br(p).replace(",", "{,}")

    pg = _proveniencia_geracao()
    prov_t = pg.resumo("generation-results-T-ordem.csv", "T")["todas"]
    prov_f = pg.resumo("generation-results-F-document.csv", "F")["todas"]
    ic_c = ng.ic_conglomerado("off", ("seguranca-trabalho", "produto", "licitacoes"))
    ic_e = ng.ic_conglomerado("on", ("seguranca-trabalho", "produto", "licitacoes"))

    esperados = {
        "tokens T": _milhar(g["T"]["todas"]["tokens"]),
        "tokens F": _milhar(g["F"]["todas"]["tokens"]),
        "pagina exibida T em respostas": f"{prov_t['exibida']} de {prov_t['respostas']}",
        "pagina citada T": f"correta em {prov_t['prosa_com_folio']} de {prov_t['respostas']} respostas em T",
        "pagina citada F": f"{prov_f['prosa_com_folio']} de {prov_f['respostas']} em F",
        "pagina citada F sem folio": f"F cairia para {prov_f['prosa']}",
        # Cada numero ancorado na frase em que aparece: um valor solto casaria
        # com qualquer outra ocorrencia (a banca mostrou que 0,685 -> 0,686
        # passava).
        "acerto da bateria (resumo)": f"a taxa de acerto foi de {_pct(taxa(c['total']))}\\% com a pergunta original e de {_pct(taxa(e['total']))}\\% com expans",
        "acerto da bateria (secao)": f"a taxa de acerto foi de {_pct(taxa(c['total']))}\\%, com os tr",
        "acerto da bateria (conclusao)": f"a taxa de acerto foi de {_pct(taxa(c['total']))}\\% com a pergunta original. As par",
        "acerto E (conclusao)": f"elevou o acerto a {_pct(taxa(e['total']))}\\%, em configura",
        "faixa dos acervos": f"entre {_pct(min(taxa(c['tema'][t]) for t in c['tema']))}\\% e {_pct(max(taxa(c['tema'][t]) for t in c['tema']))}\\% e intervalos",
        "literal x parafrase C": f"o acerto foi {_pct(taxa(c['variante']['literal']))}\\% nas literais e {_pct(taxa(c['variante']['parafrase']))}\\% nas par",
        "literal x parafrase C (discussao)": f"as paráfrases acertaram {_pct(taxa(c['variante']['parafrase']))}\\% contra {_pct(taxa(c['variante']['literal']))}\\% das literais",
        "lacuna literal-parafrase": f"cai de {_pct(taxa(c['variante']['literal']) - taxa(c['variante']['parafrase']))} para {_pct(taxa(e['variante']['literal']) - taxa(e['variante']['parafrase']))} pontos percentuais",
        "lacuna E (discussao)": f"distância a {_pct(taxa(e['variante']['literal']) - taxa(e['variante']['parafrase']))} pontos percentuais",
        "proveniencia C": _pct(taxa(c["prov"])),
        "literal E": _pct(taxa(e["variante"]["literal"])),
        "parafrase E": _pct(taxa(e["variante"]["parafrase"])),
        "acerto E": _pct(taxa(e["total"])),
        "proveniencia E": _pct(taxa(e["prov"])),
        "tokens C": _milhar(c["tokens"]),
        "tokens E": _milhar(e["tokens"]),
        "sempre certa": f"{c['sempre_certa']} das 71 acertaram nas tr",
        "sempre errada": f"{c['sempre_errada']} erraram nas tr",
        "pares a favor da literal": f"{ef['a_favor_literal']} discordantes, todos a favor da literal",
        "p da formulacao": p_texto(ef["p"]),
        "expansao ganhou": f"{len(ex['ganhou'])} passaram a acertar",
        "expansao perdeu": "nenhuma deixou de acertar" if not ex["perdeu"] else f"{len(ex['perdeu'])} deixou de acertar",
        "p da expansao": p_texto(ex["p"]),
        "enriquecimento, fontes mudam": f"em {en['fontes_mudam']} das 12 perguntas",
    }
    for braco in ("content", "content-summary", "content-questions", "max"):
        k, n = en[braco]["geral"]
        esperados[f"enriquecimento {braco}"] = f"{k}/{n}"

    # Linhas inteiras das tabelas: um numero isolado pode reaparecer em outro
    # paragrafo e mascarar a troca; a linha da tabela, nao.
    def linha(*celulas):
        return " & ".join(celulas) + r" \\"

    t, f = g["T"], g["F"]
    for est, rot in (("prosa", "Prosa (30)"), ("tabela", "Tabela (21)"), ("calendario", "Calendário (16)")):
        esperados[f"tabela geracao {est}"] = linha(
            f"{rot:<15}", _pct(t[est]["acerto"]), _pct(f[est]["acerto"]), _pct(t[est]["prov"]),
            _pct(pg.taxa_estrato("generation-results-T-ordem.csv", est)),
            _pct(pg.taxa_estrato("generation-results-F-document.csv", est)),
            _milhar(t[est]["tokens"]), _milhar(f[est]["tokens"]))
    esperados["tabela geracao todas"] = linha(
        f"{'Todas (72)':<15}", _pct(t["todas"]["acerto"]), _pct(f["todas"]["acerto"]), _pct(t["todas"]["prov"]),
        _pct(prov_t["prosa_com_folio"] / prov_t["respostas"]), _pct(prov_f["prosa_com_folio"] / prov_f["respostas"]),
        _milhar(t["todas"]["tokens"]), _milhar(f["todas"]["tokens"]))
    for nome, b in (("Sem expansão", c), ("Com expansão", e)):
        esperados[f"tabela expansao {nome}"] = linha(
            nome, _pct(taxa(b["variante"]["literal"])), _pct(taxa(b["variante"]["parafrase"])), _pct(taxa(b["total"])),
            _pct(taxa(b["prov"])), _milhar(b["tokens"]), f"{b['sempre_certa']}/71")
    temas = (("seguranca-trabalho", "Segurança do trabalho"), ("produto", "Produto e consumidor "), ("licitacoes", "Licitações           "))
    for tema, rot in temas:
        k, n = c["tema"][tema]
        lo, hi = ng.ic_conglomerado("off", (tema,))
        certas = sum(all(v) for q, v in ng.por_pergunta("off", (tema,)).items())
        total_q = len(ng.por_pergunta("off", (tema,)))
        esperados[f"tabela bateria {tema}"] = linha(rot, str(n), _pct(k / n), f"{_pct(lo)}--{_pct(hi)}", f"{certas}/{total_q}")
    esperados["tabela bateria total"] = linha(
        f"{'Total':<21}", str(c["total"][1]), _pct(taxa(c["total"])), f"{_pct(ic_c[0])}--{_pct(ic_c[1])}", f"{c['sempre_certa']}/71")
    esperados["ic E (texto)"] = f"intervalo de {_pct(ic_e[0])}\\% a {_pct(ic_e[1])}\\%, contra {_pct(ic_c[0])}\\% a {_pct(ic_c[1])}\\% sem ela"
    for forma, rot in (("negativa", "Negativa      "), ("existencia", "Existência    "), ("confirmacao", "Confirmação   "),
                       ("aberta", "Aberta        "), ("premissa-falsa", "Premissa falsa")):
        k, n = c["forma"][forma]
        esperados[f"tabela forma {forma}"] = linha(f"{rot:<14}", f"{k}/{n}", _pct(k / n))
    # O teste de redacao com o gabarito original, citado no texto e na
    # Tabela de alteracoes.
    antigo = redacao_com_gabarito_original()
    p_antigo = _br(antigo[2]).replace(",", "{,}")
    esperados["redacao, gabarito original"] = f"eram {antigo[0]} pares a favor da literal e {antigo[1]} a favor da paráfrase ($p = {p_antigo}$)"
    esperados["gpt-oss sem falhas de API"] = gpt_oss_sem_falhas()
    esperados.update(expansao_por_variante())

    falhas = 0
    for descricao, trecho in esperados.items():
        if trecho not in texto:
            print(f"FALHA  {descricao}: o CSV da '{trecho}', que nao aparece no artigo", file=sys.stderr)
            falhas += 1
    return falhas


def queda_literal_parafrase(rows: list[dict], gran: str, arm: str) -> float | None:
    """P@1(parafrase) - P@1(literal): a dependencia de casamento lexical."""
    por_variante = collections.defaultdict(list)
    for r in rows:
        if r["negative"] == "false" and r["granularity"] == gran and r["arm"] == arm:
            por_variante[r["variant"]].append(float(r["p_at_1"]))

    if not por_variante.get("literal") or not por_variante.get("paraphrase"):
        return None

    lit = sum(por_variante["literal"]) / len(por_variante["literal"])
    par = sum(por_variante["paraphrase"]) / len(por_variante["paraphrase"])
    return par - lit


def redacao_com_gabarito_original() -> tuple[int, int, float]:
    """Pares a favor da literal, da parafrase e p exato, com os CSVs anteriores
    a revisao do gabarito (commit 10f4846^)."""
    import io
    import math
    import subprocess
    pares: dict[str, dict[str, list[bool]]] = {}
    for tema in ("seguranca-trabalho", "produto", "licitacoes"):
        saida = subprocess.run(["git", "show", f"10f4846^:docs/article/results/expansao/off-{tema}.csv"],
                               cwd=RAIZ, capture_output=True, text=True, check=True).stdout
        for r in csv.DictReader(io.StringIO(saida)):
            q = r["question_id"]
            if r.get("stratum") == "negativa" or q[-1] not in "LP":
                continue
            pares.setdefault(q[:-1], {}).setdefault(q[-1], []).append(r["correct"] == "true")
    a = sum(1 for v in pares.values() if "L" in v and "P" in v and all(v["L"]) and not all(v["P"]))
    b = sum(1 for v in pares.values() if "L" in v and "P" in v and all(v["P"]) and not all(v["L"]))
    n = a + b
    p = min(1.0, 2 * sum(math.comb(n, i) for i in range(min(a, b) + 1)) / 2 ** n) if n else 1.0
    return a, b, p


def gpt_oss_sem_falhas() -> str:
    """Acerto do F com gpt-oss:120b, descontadas as falhas da API do provedor,
    contra o modelo principal nas MESMAS perguntas (banca, terceira leitura)."""
    res = RAIZ / "docs" / "article" / "results"
    rows = list(csv.DictReader(open(res / "generation-results-F-document-gpt-oss-120b.csv", encoding="utf-8")))
    falha = {r["question_id"] for r in rows
             if re.search(r"prompt is too long|timeout|deadline", r.get("answer", "") + r.get("error", ""), re.I)}
    validas = [r for r in rows if r["question_id"] not in falha]
    k = sum(r["correct"] == "true" for r in validas)
    ids = {r["question_id"] for r in validas}
    base = [r for r in csv.DictReader(open(res / "generation-results-F-document.csv", encoding="utf-8"))
            if r["question_id"] in ids]
    kb = sum(r["correct"] == "true" for r in base)
    return (f"{_pct(k / len(validas))}\\% contra {_pct(kb / len(base))}\\% do modelo principal nas mesmas "
            f"{len(ids)} perguntas, descontadas {len(falha)} falhas")


def expansao_por_variante() -> dict[str, str]:
    """Paraframes e literais que erravam sem expansao e quantas ela corrigiu."""
    res = RAIZ / "docs" / "article" / "results" / "expansao"
    def por_pergunta(arm):
        d: dict[str, list[bool]] = {}
        for t in ("seguranca-trabalho", "produto", "licitacoes"):
            for r in csv.DictReader(open(res / f"{arm}-{t}.csv", encoding="utf-8")):
                d.setdefault(r["question_id"], []).append(r["correct"] == "true")
        return d
    c, e = por_pergunta("off"), por_pergunta("on")
    err = [q for q, v in c.items() if not all(v)]
    par = [q for q in err if q.endswith("P")]
    lit = [q for q in err if q.endswith("L")]
    ok = lambda qs: sum(all(e[q]) for q in qs)
    return {"expansao por variante": f"corrigiu {ok(par)} de {len(par)} paráfrases e {ok(lit)} de {len(lit)} literais"}


def main() -> int:
    rows = carregar()

    # (descricao, valor afirmado no artigo, funcao que recalcula)
    afirmacoes = [
        ("nDCG@10 relacional, documento", 0.869, lambda: media(rows, "document", "A-relacional", "ndcg_at_10")),
        ("nDCG@10 denso exato, documento", 0.844, lambda: media(rows, "document", "C-denso", "ndcg_at_10")),
        ("nDCG@10 denso IVF, documento", 0.797, lambda: media(rows, "document", "C-denso-ivf", "ndcg_at_10")),
        ("P@1 relacional, documento", 0.750, lambda: media(rows, "document", "A-relacional", "p_at_1")),
        ("P@1 BM25, documento", 0.812, lambda: media(rows, "document", "B-bm25", "p_at_1")),
        ("P@1 relacional, secao", 0.562, lambda: media(rows, "section", "A-relacional", "p_at_1")),
        ("P@1 BM25, secao", 0.750, lambda: media(rows, "section", "B-bm25", "p_at_1")),
        ("R@5 relacional, secao", 0.719, lambda: media(rows, "section", "A-relacional", "r_at_5")),
        ("R@5 BM25, secao", 0.812, lambda: media(rows, "section", "B-bm25", "r_at_5")),
        ("queda literal->parafrase relacional, documento", -0.429, lambda: queda_literal_parafrase(rows, "document", "A-relacional")),
        ("queda literal->parafrase relacional, secao", -0.857, lambda: queda_literal_parafrase(rows, "section", "A-relacional")),
        ("queda literal->parafrase BM25, documento", -0.286, lambda: queda_literal_parafrase(rows, "document", "B-bm25")),
    ]

    falhas = 0
    for descricao, no_artigo, calcular in afirmacoes:
        medido = calcular()
        if medido is None:
            print(f"FALHA  {descricao}: nao ha dados para recalcular", file=sys.stderr)
            falhas += 1
            continue
        if abs(no_artigo - medido) > TOL:
            print(
                f"FALHA  {descricao}: artigo diz {no_artigo:.3f}, medicao da {medido:.3f}",
                file=sys.stderr,
            )
            falhas += 1

    # R@5 saturado em documento e a premissa da Secao de Granularidade: se
    # deixar de ser 1,000, o argumento do artigo muda.
    for arm in ("A-relacional", "B-bm25", "D-rrf"):
        valor = media(rows, "document", arm, "r_at_5")
        if valor is not None and abs(valor - 1.0) > TOL:
            print(
                f"FALHA  R@5 de {arm} em documento deveria ser 1,000 (saturado); veio {valor:.3f}"
                " -- a Secao de Granularidade argumenta a partir dessa saturacao",
                file=sys.stderr,
            )
            falhas += 1

    # A queda precisa ser MAIOR em secao: e a conclusao central da Secao de
    # Granularidade.
    d = queda_literal_parafrase(rows, "document", "A-relacional")
    s = queda_literal_parafrase(rows, "section", "A-relacional")
    if d is not None and s is not None and s >= d:
        print(
            f"FALHA  a queda em secao ({s:+.3f}) deveria ser MAIOR que em documento ({d:+.3f})"
            " -- se a diferenca sumiu, a conclusao sobre dependencia lexical precisa ser revista",
            file=sys.stderr,
        )
        falhas += 1

    # O texto do artigo precisa mesmo conter os numeros conferidos, senao este
    # script estaria validando afirmacoes que ninguem faz.
    texto = ARTIGO.read_text(encoding="utf-8")
    for trecho in ("86,9", "84,4", "56,2", "75,0", "85,7"):
        if trecho not in texto:
            print(
                f"FALHA  o artigo nao contem mais '{trecho}': a lista deste script esta desatualizada",
                file=sys.stderr,
            )
            falhas += 1

    falhas += conferir_geracao(texto)

    print(f"afirmacoes conferidas: {len(afirmacoes)} de recuperacao, mais as de geracao")

    if falhas:
        print(f"\n{falhas} divergencia(s) entre o artigo e a medicao.", file=sys.stderr)
        return 1

    print("\nOK: todo numero afirmado no artigo bate com o CSV de resultados.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
