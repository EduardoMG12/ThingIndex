#!/usr/bin/env python3
"""Lista uma resposta errada por pergunta e braco da bateria, para a varredura
de falsos negativos do gabarito (acertos que o corretor recusa)."""
import sys
from pathlib import Path as _P
ROOT = str(_P(__file__).resolve().parent.parent) + "/"
import csv, json
from pathlib import Path
R = Path(ROOT + "docs/article/results/expansao")
G = Path(ROOT + "api/internal/evalcorpus/testdata")
out = []
for t in ("seguranca-trabalho", "produto", "licitacoes"):
    g = {q["id"]: q for q in json.load(open(G / f"gabarito-{t}.json"))["questions"]}
    for arm in ("off", "on"):
        vistos = set()
        for r in csv.DictReader(open(R / f"{arm}-{t}.csv")):
            if r["correct"] == "true" or (r["question_id"], arm) in vistos:
                continue
            vistos.add((r["question_id"], arm))
            q = g[r["question_id"]]
            out.append(f"### {arm} {q['id']} [{q['stratum']}] rep{r['repeat']}\nQ: {q['text']}\nESPERADO: {q['expected']}\nRESP: {r['answer'][:700]}\n")
open(sys.argv[1] if len(sys.argv) > 1 else "/dev/stdout", "w").write("\n".join(out))
print(len(out))
