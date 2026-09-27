#!/usr/bin/env python3
"""Conta os arquivos de teste de integracao que o CI de fato executa.

Um arquivo conta quando: tem build tag 'integration', tem ao menos uma
func Test, seu pacote aparece num 'go test -tags=integration' do ci.yml e, se
aquele passo tem -run, ao menos uma das funcoes casa o filtro. Chamado por
scripts/contagem-testes.sh.
"""
import re
import subprocess
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
ci = (RAIZ / ".github" / "workflows" / "ci.yml").read_text(encoding="utf-8")

passos = []
for linha in ci.splitlines():
    m = re.search(r"go test\s.*-tags=integration\s(.*)$", linha)
    if not m:
        continue
    resto = m.group(1)
    pacotes = {p.strip("./").rstrip("/").removesuffix("/...") for p in re.findall(r"\./[\w/.-]+", resto)}
    filtro = re.search(r"-run\s+'([^']+)'", resto)
    passos.append((pacotes, re.compile(filtro.group(1)) if filtro else None))

arquivos = subprocess.run(["git", "ls-files", "api/*_test.go"], cwd=RAIZ, capture_output=True, text=True, check=True).stdout.split()
total = no_ci = 0
for f in arquivos:
    texto = (RAIZ / f).read_text(encoding="utf-8")
    if not re.search(r"^//go:build.*integration", texto, re.M):
        continue
    funcs = re.findall(r"^func (Test\w+)", texto, re.M)
    if not funcs:
        continue
    total += 1
    pacote = str(Path(f).parent.relative_to("api"))
    if any(pacote in pacotes and (filtro is None or any(filtro.search(fn) for fn in funcs)) for pacotes, filtro in passos):
        no_ci += 1

print(total, no_ci)
sys.exit(0)
