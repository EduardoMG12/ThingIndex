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


def _br(valor: float, casas: int = 3) -> str:
    return f"{valor:.{casas}f}".replace(".", ",")


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

    esperados = {
        "acerto T na geracao": _br(g["T"]["todas"]["acerto"]),
        "acerto F na geracao": _br(g["F"]["todas"]["acerto"]),
        "proveniencia T": _br(g["T"]["todas"]["prov"]),
        "proveniencia F": _br(g["F"]["todas"]["prov"]),
        "tokens T": _milhar(g["T"]["todas"]["tokens"]),
        "tokens F": _milhar(g["F"]["todas"]["tokens"]),
        "prov T em respostas": f"{g['T']['todas']['pok']} de {g['T']['todas']['pn']}",
        "prov F em respostas": f"{g['F']['todas']['pok']} de {g['F']['todas']['pn']}",
        "acerto da bateria": _br(taxa(c["total"])),
        "acerto seguranca": _br(taxa(c["tema"]["seguranca-trabalho"])),
        "acerto produto": _br(taxa(c["tema"]["produto"])),
        "acerto licitacoes": _br(taxa(c["tema"]["licitacoes"])),
        "literal C": _br(taxa(c["variante"]["literal"])),
        "parafrase C": _br(taxa(c["variante"]["parafrase"])),
        "proveniencia C": _br(taxa(c["prov"])),
        "literal E": _br(taxa(e["variante"]["literal"])),
        "parafrase E": _br(taxa(e["variante"]["parafrase"])),
        "acerto E": _br(taxa(e["total"])),
        "proveniencia E": _br(taxa(e["prov"])),
        "tokens C": _milhar(c["tokens"]),
        "tokens E": _milhar(e["tokens"]),
        "sempre certa": f"{c['sempre_certa']} acertaram nas tr",
        "sempre errada": f"{c['sempre_errada']} erraram nas tr",
        "pares a favor da literal": f"{ef['a_favor_literal']} deles a favor da literal",
        "p da formulacao": f"p = {_br(ef['p']).replace(',', '{,}')}",
        "expansao ganhou": f"{len(ex['ganhou'])} passaram a acertar",
        "expansao perdeu": f"{len(ex['perdeu'])} deixou de acertar",
        "p da expansao": f"p = {_br(ex['p']).replace(',', '{,}')}",
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
            f"{rot:<15}", _br(t[est]["acerto"]), _br(f[est]["acerto"]), _br(t[est]["prov"]), _br(f[est]["prov"]),
            _milhar(t[est]["tokens"]), _milhar(f[est]["tokens"]))
    esperados["tabela geracao todas"] = linha(
        f"{'Todas (72)':<15}", _br(t["todas"]["acerto"]), _br(f["todas"]["acerto"]), _br(t["todas"]["prov"]),
        _br(f["todas"]["prov"]), _milhar(t["todas"]["tokens"]), _milhar(f["todas"]["tokens"]))
    for nome, b in (("C, sem expansão", c), ("E, com expansão", e)):
        esperados[f"tabela expansao {nome}"] = linha(
            nome, _br(taxa(b["variante"]["literal"])), _br(taxa(b["variante"]["parafrase"])), _br(taxa(b["total"])),
            _br(taxa(b["prov"])), _milhar(b["tokens"]))
    for tema, rot in (("seguranca-trabalho", "Segurança do trabalho"), ("produto", "Produto e consumidor "), ("licitacoes", "Licitações           ")):
        k, n = c["tema"][tema]
        lo, hi = ng.wilson(k, n)
        esperados[f"tabela bateria {tema}"] = linha(rot, _br(k / n), f"{_br(lo)}--{_br(hi)}", str(n))

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
    for trecho in ("0,869", "0,844", "0,562", "0,750", "-0{,}857"):
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
