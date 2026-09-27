#!/usr/bin/env python3
"""Proveniencia da geracao (acervo de ensino) com os dois criterios da banca.

Para T e F, e para as mesmas respostas:

  exibida  -- a pagina-alvo esta entre as fontes que o sistema mostra (so T)
  prosa    -- alguma pagina escrita no texto da resposta e pagina-alvo,
              aceitando tambem o folio impresso da pagina-alvo
              (docs/article/results/folios-si.json).

Tambem conta por pergunta (acerto de proveniencia nas tres repeticoes) e
sem as 12 perguntas de origem 'rodada-cc5', ja usadas para ajustar o sistema.
"""
import csv
import json
import re
import unicodedata
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
R = RAIZ / "docs/article/results"
G = json.loads((RAIZ / "api/internal/evalcorpus/testdata/gabarito-si.json").read_text())
FOLIOS = json.loads((R / "folios-si.json").read_text())
Q = {q["id"]: q for q in G["questions"]}

PAG = re.compile(r"\b(?:p\.|pp\.|pag\.|pags?\.|pagina|paginas)\s*((?:\d+\s*(?:,|e|a|-|ate)?\s*)+)")


def fold(s: str) -> str:
    return "".join(c for c in unicodedata.normalize("NFD", s.lower()) if unicodedata.category(c) != "Mn")


def paginas_prosa(texto: str) -> set[int]:
    out = set()
    for m in PAG.finditer(fold(texto)):
        out.update(int(n) for n in re.findall(r"\d+", m.group(1)) if int(n) > 0)
    return out


def alvos(q, com_folio: bool) -> set[int]:
    s = set()
    for t in q["targets"]:
        for p in t["pages"]:
            s.add(p)
            if com_folio and str(p) in FOLIOS.get(t["document_id"], {}):
                s.add(FOLIOS[t["document_id"]][str(p)])
    return s


def resumo(arq: str, braco: str) -> dict:
    rows = [r for r in csv.DictReader(open(R / arq, encoding="utf-8")) if Q[r["question_id"]]["targets"]]
    out = {}
    for nome, filtro in (("todas", lambda r: True), ("sem-cc5", lambda r: Q[r["question_id"]]["origin"] != "rodada-cc5")):
        rs = [r for r in rows if filtro(r)]
        prosa = [bool(paginas_prosa(r["answer"]) & alvos(Q[r["question_id"]], False)) for r in rs]
        prosa_f = [bool(paginas_prosa(r["answer"]) & alvos(Q[r["question_id"]], True)) for r in rs]
        cita = [bool(paginas_prosa(r["answer"])) for r in rs]
        exib = [r["provenance_ok"] == "true" for r in rs] if braco == "T" else None
        acerto = [r["correct"] == "true" for r in rs]
        porq = {}
        for r, ok in zip(rs, prosa_f):
            porq.setdefault(r["question_id"], []).append(ok)
        out[nome] = {
            "respostas": len(rs),
            "acerto": sum(acerto),
            "citam_pagina": sum(cita),
            "prosa": sum(prosa),
            "prosa_com_folio": sum(prosa_f),
            "exibida": sum(exib) if exib else None,
            "perguntas": len(porq),
            "prosa_folio_3_de_3": sum(all(v) for v in porq.values()),
        }
    return out


if __name__ == "__main__":
    for arq, b in (("generation-results-T-ordem.csv", "T"), ("generation-results-F-document.csv", "F")):
        print(b, json.dumps(resumo(arq, b), ensure_ascii=False))
