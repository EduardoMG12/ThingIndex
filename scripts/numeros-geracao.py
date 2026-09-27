#!/usr/bin/env python3
"""Recalcula, a partir dos CSV versionados, os numeros das secoes de geracao,
bateria multidominio, expansao e enriquecimento do artigo.

Imprime um relatorio legivel. As mesmas funcoes sao usadas por
scripts/check-article-numbers.py para travar os numeros citados no texto.
"""

from __future__ import annotations

import csv
import json
import math
import random
from collections import defaultdict
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
RES = RAIZ / "docs" / "article" / "results"
GAB = RAIZ / "api" / "internal" / "evalcorpus" / "testdata"
TEMAS = ("seguranca-trabalho", "produto", "licitacoes")


def ler(caminho: Path) -> list[dict]:
    with caminho.open(encoding="utf-8") as f:
        return list(csv.DictReader(f))


def wilson(k: int, n: int, z: float = 1.959964) -> tuple[float, float]:
    """Intervalo de Wilson para proporcao binomial."""
    if n == 0:
        return (math.nan, math.nan)
    p = k / n
    den = 1 + z * z / n
    centro = (p + z * z / (2 * n)) / den
    meia = z * math.sqrt(p * (1 - p) / n + z * z / (4 * n * n)) / den
    return (max(0.0, centro - meia), min(1.0, centro + meia))


def sinais(a: int, b: int) -> float:
    """Teste de sinais exato bilateral (equivale ao McNemar exato)."""
    n = a + b
    if n == 0:
        return 1.0
    k = min(a, b)
    cauda = sum(math.comb(n, i) for i in range(k + 1)) / 2**n
    return min(1.0, 2 * cauda)


def estrato_si() -> dict[str, str]:
    g = json.loads((GAB / "gabarito-si.json").read_text(encoding="utf-8"))
    return {q["id"]: q["stratum"] for q in g["questions"]}


def geracao() -> dict:
    """Tabela de geracao: T (contexto recuperado) contra F (documento inteiro)."""
    estrato = estrato_si()
    out = {}
    for braco, arq in (("T", "generation-results-T-ordem.csv"), ("F", "generation-results-F-document.csv")):
        linhas = ler(RES / arq)
        por = defaultdict(lambda: {"n": 0, "ok": 0, "pn": 0, "pok": 0, "tok": 0})
        for r in linhas:
            for chave in (estrato[r["question_id"]], "todas"):
                d = por[chave]
                d["n"] += 1
                d["ok"] += r["correct"] == "true"
                d["tok"] += int(r["prompt_tokens"] or 0)
                if r["provenance_ok"] != "" and estrato[r["question_id"]] != "negativa":
                    d["pn"] += 1
                    d["pok"] += r["provenance_ok"] == "true"
        out[braco] = {
            k: {
                "acerto": v["ok"] / v["n"],
                "prov": (v["pok"] / v["pn"]) if v["pn"] else None,
                "tokens": v["tok"] / v["n"],
                "n": v["n"],
                "ok": v["ok"],
                "pn": v["pn"],
                "pok": v["pok"],
            }
            for k, v in por.items()
        }
    return out


def bateria(braco: str = "off") -> dict:
    """Bateria multidominio. braco off = sem expansao (C), on = com expansao (E)."""
    linhas = []
    for t in TEMAS:
        for r in ler(RES / "expansao" / f"{braco}-{t}.csv"):
            r["_tema"] = t
            linhas.append(r)
    def taxa(rs):
        return sum(r["correct"] == "true" for r in rs), len(rs)
    res: dict = {"linhas": linhas}
    res["tema"] = {t: taxa([r for r in linhas if r["_tema"] == t]) for t in TEMAS}
    res["forma"] = {f: taxa([r for r in linhas if r["stratum"] == f]) for f in sorted({r["stratum"] for r in linhas})}
    res["variante"] = {v: taxa([r for r in linhas if r["variant"] == v]) for v in ("literal", "parafrase")}
    res["total"] = taxa(linhas)
    pv = [r for r in linhas if r["provenance_ok"] != ""]
    res["prov"] = (sum(r["provenance_ok"] == "true" for r in pv), len(pv))
    res["tokens"] = sum(int(r["prompt_tokens"] or 0) for r in linhas) / len(linhas)
    # Estabilidade por pergunta: certa nas 3, errada nas 3, oscila.
    por_q = defaultdict(list)
    for r in linhas:
        por_q[r["question_id"]].append(r["correct"] == "true")
    res["sempre_certa"] = sum(all(v) for v in por_q.values())
    res["sempre_errada"] = sum(not any(v) for v in por_q.values())
    res["oscila"] = len(por_q) - res["sempre_certa"] - res["sempre_errada"]
    res["por_q"] = {q: all(v) for q, v in por_q.items()}
    res["pares"] = {}
    for r in linhas:
        if r["pair_id"]:
            res["pares"].setdefault(r["pair_id"], {})[r["variant"]] = res["por_q"][r["question_id"]]
    return res


def efeito_formulacao(b: dict) -> dict:
    lit = par = conc = 0
    por_tema = defaultdict(lambda: [0, 0])
    for pid, v in b["pares"].items():
        if "literal" not in v or "parafrase" not in v:
            continue
        tema = next(r["_tema"] for r in b["linhas"] if r["pair_id"] == pid)
        if v["literal"] and not v["parafrase"]:
            lit += 1
            por_tema[tema][0] += 1
        elif v["parafrase"] and not v["literal"]:
            par += 1
            por_tema[tema][1] += 1
        else:
            conc += 1
    return {"a_favor_literal": lit, "a_favor_parafrase": par, "concordantes": conc, "p": sinais(lit, par), "por_tema": dict(por_tema)}


def por_pergunta(arm: str, temas: tuple[str, ...]) -> dict[str, list[bool]]:
    """Acerto de cada repeticao, agrupado por pergunta."""
    d: dict[str, list[bool]] = {}
    for t in temas:
        for r in ler(RES / "expansao" / f"{arm}-{t}.csv"):
            d.setdefault(r["question_id"], []).append(r["correct"] == "true")
    return d


def ic_conglomerado(arm: str, temas: tuple[str, ...], seed: int = 7, reps: int = 10000) -> tuple[float, float]:
    """IC de 95% do acerto por resposta, reamostrando PERGUNTAS (bootstrap de
    conglomerado). As tres repeticoes de uma pergunta nao sao independentes;
    tratar as 213 respostas como tais estreitaria o intervalo."""
    d = por_pergunta(arm, temas)
    ks = list(d)
    rng = random.Random(seed)
    b = []
    for _ in range(reps):
        a = [d[rng.choice(ks)] for _ in ks]
        b.append(sum(sum(v) for v in a) / sum(len(v) for v in a))
    b.sort()
    return b[int(0.025 * reps) - 1], b[int(0.975 * reps) - 1]


def efeito_expansao(c: dict, e: dict) -> dict:
    ganhou = [q for q in c["por_q"] if e["por_q"][q] and not c["por_q"][q]]
    perdeu = [q for q in c["por_q"] if c["por_q"][q] and not e["por_q"][q]]
    return {"ganhou": ganhou, "perdeu": perdeu, "concordantes": len(c["por_q"]) - len(ganhou) - len(perdeu), "p": sinais(len(ganhou), len(perdeu))}


def enriquecimento() -> dict:
    d = RES / "enriquecimento-reexecucao"
    out = {}
    for braco in ("content", "content-summary", "content-questions", "max"):
        rs = ler(d / f"braco-seguranca-trabalho-{braco}.csv")
        ok = lambda s: sum(r["correct"] == "true" for r in s)
        pv = [r for r in rs if r["provenance_ok"] != ""]
        out[braco] = {
            "geral": (ok(rs), len(rs)),
            "literal": (ok([r for r in rs if r["variant"] == "literal"]), sum(r["variant"] == "literal" for r in rs)),
            "parafrase": (ok([r for r in rs if r["variant"] == "parafrase"]), sum(r["variant"] == "parafrase" for r in rs)),
            "prov": (sum(r["provenance_ok"] == "true" for r in pv), len(pv)),
            "fontes": {r["question_id"]: r["cited_pages"] for r in rs},
        }
    qs = out["content"]["fontes"].keys()
    out["fontes_mudam"] = sum(len({out[b]["fontes"][q] for b in ("content", "content-summary", "content-questions", "max")}) > 1 for q in qs)
    return out


def main() -> None:
    g = geracao()
    print("== Geracao (T = contexto recuperado, F = documento inteiro)")
    for est in ("prosa", "tabela", "calendario", "negativa", "todas"):
        t, f = g["T"][est], g["F"][est]
        fmt = lambda x: "  -  " if x is None else f"{x:.3f}"
        print(f"  {est:10s} acerto T {t['acerto']:.3f} F {f['acerto']:.3f} | prov T {fmt(t['prov'])} F {fmt(f['prov'])} | tokens T {t['tokens']:.0f} F {f['tokens']:.0f}")
    t, f = g["T"]["todas"], g["F"]["todas"]
    print(f"  prov T {t['pok']}/{t['pn']} IC {wilson(t['pok'], t['pn'])} ; F {f['pok']}/{f['pn']} IC {wilson(f['pok'], f['pn'])}")
    print(f"  razao de tokens F/T = {f['tokens'] / t['tokens']:.1f}")

    c, e = bateria("off"), bateria("on")
    print("\n== Bateria (C, sem expansao)")
    for tema, (k, n) in c["tema"].items():
        print(f"  {tema:20s} {k}/{n} = {k / n:.3f} IC {tuple(round(x, 3) for x in wilson(k, n))}")
    k, n = c["total"]
    print(f"  total {k}/{n} = {k / n:.3f} IC {tuple(round(x, 3) for x in wilson(k, n))}")
    for f_, (k, n) in c["forma"].items():
        print(f"  forma {f_:15s} {k}/{n} = {k / n:.3f}")
    for v, (k, n) in c["variante"].items():
        print(f"  variante {v:10s} {k}/{n} = {k / n:.3f}")
    print(f"  proveniencia {c['prov'][0]}/{c['prov'][1]} = {c['prov'][0] / c['prov'][1]:.3f}  tokens {c['tokens']:.0f}")
    print(f"  estabilidade: sempre certa {c['sempre_certa']}, sempre errada {c['sempre_errada']}, oscila {c['oscila']}")
    ef = efeito_formulacao(c)
    print(f"  formulacao: {ef}")

    print("\n== Expansao (E)")
    for v, (k, n) in e["variante"].items():
        print(f"  variante {v:10s} {k}/{n} = {k / n:.3f}")
    k, n = e["total"]
    print(f"  total {k}/{n} = {k / n:.3f}; proveniencia {e['prov'][0]}/{e['prov'][1]} = {e['prov'][0] / e['prov'][1]:.3f}; tokens {e['tokens']:.0f}")
    ee = efeito_formulacao(e)
    print(f"  formulacao em E: {ee}")
    ex = efeito_expansao(c, e)
    print(f"  C->E por pergunta: ganhou {len(ex['ganhou'])} {ex['ganhou']}, perdeu {ex['perdeu']}, concordantes {ex['concordantes']}, p = {ex['p']:.4f}")

    en = enriquecimento()
    print("\n== Enriquecimento (reexecucao)")
    for b in ("content", "content-summary", "content-questions", "max"):
        print(f"  {b:18s} {en[b]['geral']} literal {en[b]['literal']} parafrase {en[b]['parafrase']} prov {en[b]['prov']}")
    print(f"  perguntas cujas fontes mudam entre bracos: {en['fontes_mudam']}")


if __name__ == "__main__":
    main()
