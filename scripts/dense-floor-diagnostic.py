#!/usr/bin/env python3
"""Diagnostico do piso de similaridade do braco denso.

Nao faz parte do experimento: serve para ESCOLHER e JUSTIFICAR o valor de
DenseSimilarityFloor com base na separacao observada, em vez de chutar um
numero redondo. O valor escolhido e depois declarado no artigo.
"""
import json
import pathlib
import sys

root = pathlib.Path(__file__).resolve().parents[1]
art = json.loads((root / "api/internal/evalcorpus/testdata/embeddings-c1.json").read_text())
corpus = json.loads((root / "api/internal/evalcorpus/testdata/corpus-c1.json").read_text())

dv = {d["id"]: d["vector"] for d in art["documents"]}
qv = {q["id"]: q["vector"] for q in art["queries"]}
meta = {q["id"]: q for q in corpus["queries"]}


def dot(a, b):
    return sum(x * y for x, y in zip(a, b))


rel_scores, irrel_scores, neg_scores = [], [], []

print(f"{'query':12} {'variante':11} {'top1':22} {'sim':>7} {'acertou':>8}")
for q in corpus["queries"]:
    qid = q["id"]
    rel = q["relevance"]
    sims = sorted(((dot(qv[qid], dv[d]), d) for d in dv), reverse=True)
    top_sim, top_doc = sims[0]
    hit = "" if q.get("negative") else ("sim" if rel.get(top_doc, 0) >= 1 else "NAO")
    print(f"{qid:12} {q['variant']:11} {top_doc:22} {top_sim:7.4f} {hit:>8}")

    for s, d in sims:
        if q.get("negative"):
            neg_scores.append(s)
        elif rel.get(d, 0) >= 1:
            rel_scores.append(s)
        else:
            irrel_scores.append(s)

print()
for name, scores in [
    ("relevantes", rel_scores),
    ("irrelevantes", irrel_scores),
    ("negativas (todos)", neg_scores),
]:
    print(f"{name:20} n={len(scores):3}  min={min(scores):.4f}  mediana={sorted(scores)[len(scores)//2]:.4f}  max={max(scores):.4f}")

print()
print("Escolha do piso: manter TODOS os relevantes e cortar o maximo de ruido.")
print(f"{'piso':>6} {'relev.mantidos':>15} {'irrelev.mantidos':>17} {'negativos mantidos':>19}")
for floor in [0.70, 0.75, 0.78, 0.80, 0.82, 0.84, 0.86, 0.88, 0.90]:
    kr = sum(1 for s in rel_scores if s >= floor)
    ki = sum(1 for s in irrel_scores if s >= floor)
    kn = sum(1 for s in neg_scores if s >= floor)
    flag = "  <- perde relevante" if kr < len(rel_scores) else ""
    print(f"{floor:6.2f} {kr:>10}/{len(rel_scores):<4} {ki:>12}/{len(irrel_scores):<4} {kn:>14}/{len(neg_scores):<4}{flag}")

sys.exit(0)
