#!/usr/bin/env python3
"""Mapa pagina fisica -> folio impresso dos PDFs do acervo de ensino.

O braco F recebe o documento com marcas "[pagina i]" pelo indice fisico,
mas os PDFs trazem o numero impresso no rodape, que e outro. Quando o modelo
cita o folio impresso da pagina certa, a proveniencia por indice fisico conta
erro. A banca apontou isso; este script extrai, para cada pagina-alvo do
gabarito, o numero isolado na ultima ou na primeira linha nao vazia da
pagina, que e onde os documentos imprimem o folio.

Saida: docs/article/results/folios-si.json, {documento: {pagina_fisica: folio}}.
"""
import json
import re
import subprocess
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
G = json.loads((RAIZ / "api/internal/evalcorpus/testdata/gabarito-si.json").read_text())
PDF = {d["id"]: RAIZ / "examples" / "faculdade" / d["file"] for d in G["documents"]}

alvos: dict[str, set[int]] = {}
for q in G["questions"]:
    for t in q["targets"]:
        alvos.setdefault(t["document_id"], set()).update(t["pages"])

saida: dict[str, dict[str, int]] = {}
for doc, paginas in sorted(alvos.items()):
    for p in sorted(paginas):
        txt = subprocess.run(["pdftotext", "-f", str(p), "-l", str(p), "-layout", str(PDF[doc]), "-"],
                             capture_output=True, text=True, check=True).stdout
        linhas = [l.strip() for l in txt.splitlines() if l.strip()]
        folio = None
        for l in (linhas[-1:] + linhas[:1]) if linhas else []:
            if re.fullmatch(r"\d{1,4}", l):
                folio = int(l)
                break
        if folio is not None and folio != p:
            saida.setdefault(doc, {})[str(p)] = folio

(RAIZ / "docs/article/results/folios-si.json").write_text(json.dumps(saida, indent=2, sort_keys=True) + "\n")
print(json.dumps(saida, indent=2, sort_keys=True))
