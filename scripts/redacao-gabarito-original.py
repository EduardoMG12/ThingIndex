#!/usr/bin/env python3
"""Teste de redacao e ganho da expansao com o gabarito ORIGINAL das baterias.

O artigo (Secao 8.7) e o Anexo A8 citam o resultado do teste de redacao
(literal contra parafrase) e do ganho da expansao com o gabarito que valia
na primeira medicao (81,7% sem expansao), antes das revisoes v4 a v7
(docs/article/results/alteracoes-pos-medicao.md, linhas 4, 6, 7 e 8).

Este script nao confia na coluna `correct` de CSV antigo: ele recorrige as
respostas gravadas HOJE em docs/article/results/expansao/{off,on}-*.csv com
o gabarito lido do git no commit anterior a revisao v4 (54a4d70^), pela
mesma regra de api/cmd/generation-eval/metrics.go (texto dobrado em caixa e
acento; todas as answer_patterns casam, nenhuma reject_pattern casa; linha
com erro de API conta como errada) -- a funcao `correta` de
scripts/auditoria-padroes.py, que reproduz o corretor.

Criterio dos pares (o mesmo de scripts/numeros-geracao.py): uma versao conta
como certa quando acerta nas tres repeticoes. Teste: McNemar exato
(binomial bilateral sobre os pares discordantes).

Uma nota sobre docs/article/results/resumo-bateria.md: ele lista 9 pares
so-literal e 1 so-parafrase porque foi gerado pela versao antiga de
scripts/resumo-bateria.py, que guardava so a ULTIMA repeticao de cada
pergunta (bug corrigido em d718081). `--ultima-repeticao` reproduz aquela
saida.

Uso: python3 scripts/redacao-gabarito-original.py [--ultima-repeticao]
"""
from __future__ import annotations

import argparse
import csv
import importlib.util
import json
import subprocess
from math import comb
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
RES = RAIZ / "docs" / "article" / "results" / "expansao"
TEMAS = ("seguranca-trabalho", "produto", "licitacoes")
# Ultimo commit antes da revisao v4 dos gabaritos das baterias (54a4d70).
REV_ORIGINAL = "54a4d70^"


def _correta():
    spec = importlib.util.spec_from_file_location("auditoria_padroes", RAIZ / "scripts" / "auditoria-padroes.py")
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod.correta


def gabarito_original(tema: str) -> dict[str, dict]:
    bruto = subprocess.run(
        ["git", "show", f"{REV_ORIGINAL}:api/internal/evalcorpus/testdata/gabarito-{tema}.json"],
        cwd=RAIZ, capture_output=True, text=True, check=True).stdout
    return {q["id"]: q for q in json.loads(bruto)["questions"]}


def recorrige(braco: str) -> list[dict]:
    """Linhas de {braco}-*.csv com `ok` recalculado pelo gabarito original."""
    correta = _correta()
    out = []
    for tema in TEMAS:
        g = gabarito_original(tema)
        with open(RES / f"{braco}-{tema}.csv", encoding="utf-8") as f:
            for r in csv.DictReader(f):
                q = g[r["question_id"]]
                r["ok"] = r["error"] == "" and correta(r["answer"], q)
                out.append(r)
    return out


def mcnemar(a: int, b: int) -> float:
    n = a + b
    if n == 0:
        return 1.0
    return min(1.0, 2 * sum(comb(n, i) for i in range(min(a, b) + 1)) / 2 ** n)


def por_pergunta(linhas: list[dict], ultima: bool = False) -> dict[str, bool]:
    d: dict[str, list[bool]] = {}
    for r in linhas:
        d.setdefault(r["question_id"], []).append(r["ok"])
    return {q: (v[-1] if ultima else all(v)) for q, v in d.items()}


def redacao(off: list[dict], ultima: bool = False) -> dict:
    pq = por_pergunta(off, ultima)
    par = {r["question_id"]: r["pair_id"] for r in off if r["pair_id"]}
    var = {r["question_id"]: r["variant"] for r in off}
    pares: dict[str, dict[str, bool]] = {}
    for q, pid in par.items():
        pares.setdefault(pid, {})[var[q]] = pq[q]
    lit = sorted(p for p, v in pares.items() if v.get("literal") and not v.get("parafrase"))
    pa = sorted(p for p, v in pares.items() if v.get("parafrase") and not v.get("literal"))
    return {"pares": len(pares), "so_literal": lit, "so_parafrase": pa, "p": mcnemar(len(lit), len(pa))}


def expansao(off: list[dict], on: list[dict]) -> dict:
    c, e = por_pergunta(off), por_pergunta(on)
    ganhou = sorted(q for q in c if e[q] and not c[q])
    perdeu = sorted(q for q in c if c[q] and not e[q])
    return {"ganhou": ganhou, "perdeu": perdeu, "p": mcnemar(len(ganhou), len(perdeu))}


def calcula() -> dict:
    off, on = recorrige("off"), recorrige("on")
    return {
        "off": (sum(r["ok"] for r in off), len(off)),
        "on": (sum(r["ok"] for r in on), len(on)),
        "redacao": redacao(off),
        "redacao_ultima_repeticao": redacao(off, ultima=True),
        "expansao": expansao(off, on),
    }


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--ultima-repeticao", action="store_true",
                    help="reproduz o criterio com bug de resumo-bateria.md (so a ultima repeticao)")
    args = ap.parse_args()
    r = calcula()
    print(f"gabarito: {REV_ORIGINAL}")
    for b in ("off", "on"):
        k, n = r[b]
        print(f"{b}: {k}/{n} = {k / n:.3f}")
    red = r["redacao_ultima_repeticao" if args.ultima_repeticao else "redacao"]
    print(f"redacao ({red['pares']} pares): so-literal {len(red['so_literal'])} {red['so_literal']}; "
          f"so-parafrase {len(red['so_parafrase'])} {red['so_parafrase']}; McNemar exato p = {red['p']:.4f}")
    ex = r["expansao"]
    print(f"expansao: ganhou {len(ex['ganhou'])} {ex['ganhou']}; perdeu {len(ex['perdeu'])} {ex['perdeu']}; "
          f"McNemar exato p = {ex['p']:.4f}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
