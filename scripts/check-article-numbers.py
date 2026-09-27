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
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
CSV = RAIZ / "docs" / "article" / "results" / "results.csv"
ARTIGO = RAIZ / "docs" / "article" / "article-overleaf-v2.md"

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

    print(f"afirmacoes conferidas: {len(afirmacoes)}")

    if falhas:
        print(f"\n{falhas} divergencia(s) entre o artigo e a medicao.", file=sys.stderr)
        return 1

    print("\nOK: todo numero afirmado no artigo bate com o CSV de resultados.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
