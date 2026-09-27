#!/usr/bin/env python3
"""Revisao v4 dos gabaritos das baterias (existencia, premissa falsa, lc3).

Regras escritas a partir do que cada forma de pergunta exige, antes de
reavaliar as respostas:

  existencia     -- a resposta afirma que o tema existe no acervo. "ha " saiu
                    da alternancia (casava com "nao ha"); uma recusa invalida.
  premissa-falsa -- a resposta traz o dado que corrige a premissa. "nao"
                    isolado deixou de bastar.
  lc3 (vigencia) -- a resposta diz que a lei foi revogada; "nao" deixou de
                    ser exigido (a resposta "foi revogada em 30/12/2023" e
                    correta e nao tem "nao"). Recusa invalida.

As perguntas negativas nao mudam.
"""
import json
from pathlib import Path

DIR = Path(__file__).resolve().parent.parent / "api" / "internal" / "evalcorpus" / "testdata"

RECUSA = (
    r"nao (ha|foram encontrad\w*|encontrei|foi possivel|e possivel)[^.]{0,40}"
    r"(informac|mencao|referenc|dados|detalhe|especific|indica)"
    r"|nenhuma informac|sem informac"
    r"|(os|nos) (documentos|trechos|textos)[^.]{0,40}nao "
    r"(trazem|traz|mencionam|contem|tratam|abordam|especificam|informam|apresentam|indicam)"
)

NOVOS = {
    # existencia: afirmacao positiva, sem "ha " solto
    "sg5": [r"\bsim\b|existe|trata|prev"],
    "sg10": [r"\bsim\b|existe|trata|prev"],
    "pr5": [r"\bsim\b|existe|consta|exclu"],
    "pr10": [r"\bsim\b|existe|consta|orient"],
    "lc8": [r"\bsim\b|existe|prev|trata"],
    "lc10": [r"\bsim\b|existe|prev|seguro-garantia"],
    # premissa falsa: o dado que corrige a premissa
    "sg1": [r"gratuit|sem (custo|onus)|nao (pode|e permitido|ha previsao de) (descont|cobr)"],
    "sg12": [r"gratuit|sem (custo|onus)|(organizacao|empregador|empresa)[^.]{0,60}fornec|fornec[^.]{0,60}(organizacao|empregador|empresa)"],
    # O termo de garantia traz dois prazos, 1 ano (p. 1) e 2 anos (p. 20);
    # qualquer um corrige a premissa de 3 anos.
    "pr2": [r"1 \(um\) ano|\bum ano\b|12 meses|\b1 ano\b|2 \(dois\) anos|\bdois anos\b|\b2 anos\b"],
    "pr12": [r"nao (cobre|abrange|inclui)[^.]{0,80}instala|instala[^.]{0,80}(nao (e|esta) cobert|exclu|nao (cobre|abrange))|exclu[^.]{0,80}instala"],
    "lc5": [r"revogad|extint|deixou de existir|nao (existe|consta|e mais|esta mais|ha mais)[^.]{0,60}(modalidade|tomada|rol)"],
    "lc13": [r"revogad|extint|deixou de existir|nao (existe|consta|e mais|esta mais|ha mais)[^.]{0,60}(modalidade|convite|rol)"],
    # vigencia da Lei 8.666
    "lc3": [r"revogad"],
}
# Na premissa falsa o padrao ja exige o dado que corrige a premissa; uma
# resposta que abre com "nao ha mencao a esse prazo" e em seguida da o dado
# esta certa, entao a recusa nao veta ali. Na existencia e em lc3 a resposta
# certa e afirmativa, e a recusa veta.
REJEITA_POR_ESTRATO = {"existencia"}
REJEITA_POR_PAR = {"lc3"}

for nome, versao in (("seguranca-trabalho", "v4"), ("produto", "v4"), ("licitacoes", "v4"), ("prefeitura", "v2")):
    path = DIR / f"gabarito-{nome}.json"
    g = json.loads(path.read_text(encoding="utf-8"))
    for q in g["questions"]:
        par = q.get("pair_id") or q["id"]
        if par in NOVOS:
            q["answer_patterns"] = NOVOS[par]
        if q["stratum"] in REJEITA_POR_ESTRATO or par in REJEITA_POR_PAR:
            q["reject_patterns"] = [RECUSA]
    g["version"] = g["version"].rsplit("-v", 1)[0] + "-" + versao
    path.write_text(json.dumps(g, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(path.name, g["version"])
