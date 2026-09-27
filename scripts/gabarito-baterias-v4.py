#!/usr/bin/env python3
"""Revisao v4/v5/v6 dos gabaritos das baterias (existencia, premissa falsa, lc3, lc4).

v5 (segunda leitura da banca): os padroes da existencia aceitavam
"nao existe", "nao consta", "nao trata" e "nao preve", porque casavam o verbo
sem exigir que ele fosse afirmado; lc3 aceitava "nao foi revogada"; lc4 aceitava
qualquer resposta que mencionasse "leis anteriores", inclusive a recusa.

v6 (terceira leitura): as recusas passaram a ser vetadas em toda pergunta nao
negativa, com uma lista mais ampla ("nao esta previsto", "nenhum dos
documentos trata", "nada consta", "nao pode ser encontrad"); nas perguntas de
sim ou nao (existencia e confirmacao) a resposta que ABRE negando o que o
gabarito afirma (ou afirmando o que ele nega) e vetada; lc12 e lc13 vetam a
resposta oposta; pr8 aceita o plural "pixeis".

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
    r"(informac|mencao|referenc|dados|detalhe|especific|indica|previs|determinar)"
    r"|nenhuma informac|sem informac"
    r"|(os|nos|o|no) (documentos?|trechos?|textos?)[^.]{0,40}nao "
    r"(trazem|traz|mencionam?|contem|tratam?|abordam?|especificam?|informam?|apresentam?|indicam?|preveem|preve)"
)

# Recusas que o v5 ainda deixava passar (terceira leitura da banca).
RECUSA_AMPLA = (
    r"nao (esta|estao|foi|foram) (previst|mencionad|encontrad|tratad|abordad)"
    r"|nao ha (dispositivo|artigo|norma|trecho|regra)[^.]{0,30}(prev|trat|mencion|regul)"
    r"|nenhum (dos|documento|trecho|texto)[^.]{0,30}(trata|menciona|preve|aborda|traz|contem)"
    r"|nao se trata disso|nada consta|nao pode ser (encontrad|determinad|respondid)"
    r"|nao (e|foi) possivel (encontrar|determinar|responder|identificar)"
)

# Abertura da resposta: as perguntas de sim ou nao sao julgadas tambem pela
# polaridade da primeira frase. Uma resposta que abre com "Nao," quando o
# gabarito diz sim (ou com "Sim," quando diz nao) esta errada, ainda que o
# resto do texto case com o padrao.
ABRE_NEGANDO = r"^\W*nao\b"
ABRE_AFIRMANDO = r"^\W*sim\b"

# O verbo que a existencia pede, negado: "nao existe", "nao consta", "nao
# trata", "nao preve(m)". Vale para a resposta inteira: quem nega o verbo em
# qualquer ponto nao afirmou que o tema existe.
VERBO_NEGADO = r"\bnao (existem?|constam?|tratam?|preveem|preve|ha previsao)\b"

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
    # pr8: "pixeis" (plural sem acento) e resposta certa
    "pr8": [r"pixe(l|is)|subpixe(l|is)"],
    # contrato anterior a lei nova: o art. 190 ou a regra que ele estabelece
    "lc4": [r"art\.? ?190|legislacao revogada|regras? anteriores?[^.]{0,40}(continu|mant|reg)|(continu|permanec|mant)\w*[^.]{0,40}(regid|regulad)[^.]{0,40}(lei|legislacao|regras?) (anterior|antiga|revogada)|(regid|regulad)\w*[^.]{0,20}(pela|pela lei|pela legislacao) (8\.?666|anterior|antiga|revogada)"],
}

# Vetos alem da recusa: a resposta oposta a certa.
OPOSTA = {
    "lc4": r"(aplica|passa a valer|vale)\w*[^.]{0,30}imediatamente|muda de regra|passa a ser regid\w* pela (lei )?(14\.?133|nova)",
    "lc3": r"nao (foi|esta|estava) (integralmente |totalmente )?revogad|continua (em vigor|valendo|vigente)|ainda (esta em vigor|vale|e valida|vigora)",
    "lc12": r"nao (foi|esta|estava) (integralmente |totalmente )?revogad|continua (em vigor|valendo|vigente)",
    "lc13": r"(ainda|continua a) exist|nao foi (revogad|extint)|continua (valid|previst|vigente)",
}
# Na premissa falsa o padrao ja exige o dado que corrige a premissa; uma
# resposta que abre com "nao ha mencao a esse prazo" e em seguida da o dado
# esta certa, entao a recusa nao veta ali. Na existencia e em lc3 a resposta
# certa e afirmativa, e a recusa veta.
REJEITA_POR_ESTRATO = {"existencia", "confirmacao", "premissa-falsa", "aberta"}
REJEITA_POR_PAR = {"lc3", "lc4", "lc12", "lc13"}

for nome, versao in (("seguranca-trabalho", "v6"), ("produto", "v6"), ("licitacoes", "v6"), ("prefeitura", "v4")):
    path = DIR / f"gabarito-{nome}.json"
    g = json.loads(path.read_text(encoding="utf-8"))
    for q in g["questions"]:
        par = q.get("pair_id") or q["id"]
        if par in NOVOS:
            q["answer_patterns"] = NOVOS[par]
        if q["stratum"] in REJEITA_POR_ESTRATO or par in REJEITA_POR_PAR:
            # Na premissa falsa e na aberta a recusa so veta quando o padrao
            # nao exige o dado: o padrao delas ja exige o conteudo, e uma
            # resposta que abre com "nao ha mencao a esse prazo" e depois da
            # o dado esta certa. Por isso a recusa ampla vale so nas perguntas
            # de sim ou nao, e nas demais so a lista original e so onde ja era
            # usada.
            # Na confirmacao a resposta pode ser parcial ("no manual, sim; o
            # termo de garantia nao menciona"): a recusa original ("o
            # documento nao menciona") vetaria uma resposta certa (pr9L), entao
            # ali so a recusa ampla, que exige recusa da resposta inteira, e a
            # polaridade da abertura.
            vetos = []
            if q["stratum"] == "existencia" or par in REJEITA_POR_PAR:
                vetos += [RECUSA, RECUSA_AMPLA]
            elif q["stratum"] == "confirmacao":
                vetos.append(RECUSA_AMPLA)
            if q["stratum"] == "existencia":
                vetos.append(VERBO_NEGADO)
            if q["stratum"] in ("existencia", "confirmacao"):
                esperado = q.get("expected", "").strip().lower()
                vetos.append(ABRE_AFIRMANDO if esperado.startswith("nao") else ABRE_NEGANDO)
            if par in OPOSTA:
                vetos.append(OPOSTA[par])
            if vetos:
                q["reject_patterns"] = vetos
    g["version"] = g["version"].rsplit("-v", 1)[0] + "-" + versao
    path.write_text(json.dumps(g, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(path.name, g["version"])
