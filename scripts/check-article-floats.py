#!/usr/bin/env python3
"""Toda tabela e figura do artigo e chamada no texto, e a chamada vem antes.

O script de shell anterior (check-article-tables.sh) so olhava as tabelas
incluidas por \\input e deixou passar seis tabelas escritas no proprio
artigo sem nenhum \\ref. Este olha todo \\label{tab:...} e \\label{fig:...}
do artigo e dos .tex incluidos e exige:

  1. pelo menos um \\ref{...} no artigo;
  2. o primeiro \\ref aparece ANTES do ambiente (table/figure/longtable) no
     fonte, que e o que a norma pede: o leitor encontra a chamada e depois o
     objeto.

A posicao e medida no fonte expandido (cada \\input substituido pelo
conteudo), por caractere. Um \\ref dentro da propria legenda nao conta.

Uso: python3 scripts/check-article-floats.py
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
ARTIGO = RAIZ / "docs" / "article" / "artigo.tex"
DIR = ARTIGO.parent

AMBIENTE = re.compile(r"\\begin\{(table|figure|longtable)\*?\}(.*?)\\end\{\1\*?\}", re.S)
LABEL = re.compile(r"\\label\{((?:tab|fig):[^}]+)\}")


def sem_comentarios(texto: str) -> str:
    return re.sub(r"(?<!\\)%.*", "", texto)


def expandir(texto: str) -> str:
    def trocar(m: re.Match) -> str:
        caminho = DIR / (m.group(1) + ("" if m.group(1).endswith(".tex") else ".tex"))
        return sem_comentarios(caminho.read_text(encoding="utf-8")) if caminho.exists() else m.group(0)

    return re.sub(r"\\input\{([^}]+)\}", trocar, texto)


def main() -> int:
    fonte = expandir(sem_comentarios(ARTIGO.read_text(encoding="utf-8")))
    # Apendices ficam fora da regra de ordem: a tabela do apendice e chamada
    # no corpo, paginas antes.
    falhas = 0
    vistos = 0
    for amb in AMBIENTE.finditer(fonte):
        corpo = amb.group(2)
        for label in LABEL.findall(corpo):
            vistos += 1
            refs = [m.start() for m in re.finditer(r"\\ref\{" + re.escape(label) + r"\}", fonte)]
            # Ref dentro do proprio ambiente (legenda) nao conta como chamada.
            refs = [p for p in refs if not (amb.start() <= p < amb.end())]
            if not refs:
                print(f"FALHA  {label}: nunca e chamado no texto", file=sys.stderr)
                falhas += 1
            elif refs[0] > amb.start():
                linha = fonte.count("\n", 0, amb.start()) + 1
                print(f"FALHA  {label}: aparece antes da primeira chamada (fonte expandido, linha {linha})", file=sys.stderr)
                falhas += 1
    print(f"tabelas e figuras conferidas: {vistos}")
    if falhas:
        print(f"\n{falhas} objeto(s) sem chamada ou chamados depois de aparecer.", file=sys.stderr)
        return 1
    print("OK: toda tabela e figura e chamada no texto antes de aparecer.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
