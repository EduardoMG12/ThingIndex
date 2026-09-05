#!/usr/bin/env python3
"""Anota o corpus C1 com secoes e gabarito em nivel de no (issue #16).

Cada julgamento abaixo foi feito LENDO o conteudo da secao e decidindo qual
delas responde a necessidade de informacao da consulta -- nao por casamento de
palavra. O caso mais ilustrativo e q06 ("quanto tempo a empresa tem para pagar
a nota"): a resposta esta em "4 DOS PRAZOS", nao em "3 DA PROPOSTA E DOS
PRECOS", que e a secao que fala de dinheiro. E exatamente a distincao que
ranquear documento inteiro nao consegue capturar.

Os ids sao autorais e estaveis (doc#sN). NUNCA derivados do titulo, porque o
titulo e saida do extrator -- foi assim que o benchmark anterior virou medida de
autoconsistencia.
"""
import json
import pathlib

CORPUS = pathlib.Path("internal/evalcorpus/testdata/corpus-c1.json")

# ancora = texto literal que abre a secao no documento-fonte.
SECOES = {
    "acordao-trabalhista": [
        ("s1", "1 RELATÓRIO", "Relatório"),
        ("s2", "2 FUNDAMENTAÇÃO", "Fundamentação"),
        ("s3", "3 DISPOSITIVO", "Dispositivo"),
    ],
    "edital-pregao": [
        ("s1", "1 DO OBJETO", "Do objeto"),
        ("s2", "2 DA HABILITAÇÃO", "Da habilitação"),
        ("s3", "3 DA PROPOSTA E DOS PREÇOS", "Da proposta e dos preços"),
        ("s4", "4 DOS PRAZOS", "Dos prazos"),
    ],
    "contrato-sla": [
        ("s1", "1 DISPONIBILIDADE", "Disponibilidade"),
        ("s2", "2 TEMPO DE ATENDIMENTO", "Tempo de atendimento"),
        ("s3", "3 PENALIDADES", "Penalidades"),
        ("s4", "4 REAJUSTE", "Reajuste"),
    ],
    "politica-privacidade": [
        ("s1", "1 FINALIDADE DO TRATAMENTO", "Finalidade do tratamento"),
        ("s2", "2 COMPARTILHAMENTO", "Compartilhamento"),
        ("s3", "3 DIREITOS DO TITULAR", "Direitos do titular"),
        ("s4", "4 RETENÇÃO", "Retenção"),
    ],
    "manual-onboarding": [
        ("s1", "1 PRIMEIRO DIA", "Primeiro dia"),
        ("s2", "2 ACESSOS E FERRAMENTAS", "Acessos e ferramentas"),
        ("s3", "3 PERÍODO DE EXPERIÊNCIA", "Período de experiência"),
        ("s4", "4 BENEFÍCIOS", "Benefícios"),
    ],
}

# Gabarito em nivel de no. O grau segue a mesma escala 0-3 do nivel de
# documento; o comentario registra POR QUE aquela secao responde.
RELEVANCIA_SECOES = {
    # "horas extraordinarias adicional noturno vinculo" -> o pedido esta no
    # relatorio; a condenacao esta no dispositivo. Ambos respondem, o
    # dispositivo com menos contexto sobre o pedido original.
    "q01-literal": {"acordao-trabalhista#s1": 3, "acordao-trabalhista#s3": 2},
    "q02-literal": {"edital-pregao#s2": 3},
    # "disponibilidade mensal severidade contorno" cruza duas secoes: o
    # percentual esta em DISPONIBILIDADE, os tempos em TEMPO DE ATENDIMENTO.
    "q03-literal": {"contrato-sla#s1": 3, "contrato-sla#s2": 3},
    "q04-literal": {"politica-privacidade#s3": 3},
    "q05-literal": {"manual-onboarding#s3": 3},
    # Caso que justifica a granularidade fina: prazo de PAGAMENTO esta em
    # "4 DOS PRAZOS", nao na secao de precos.
    "q06-para": {"edital-pregao#s4": 3, "contrato-sla#s2": 1},
    # "se o fornecedor ficar fora do ar" -> penalidade; disponibilidade define
    # o limite que foi violado.
    "q07-para": {"contrato-sla#s3": 3, "contrato-sla#s1": 2},
    "q08-para": {"politica-privacidade#s4": 3},
    # "o juiz aceitou o registro de jornada" -> a analise da prova esta na
    # fundamentacao.
    "q09-para": {"acordao-trabalhista#s2": 3},
    "q10-para": {"manual-onboarding#s4": 3},
    "q11-para": {"edital-pregao#s3": 3},
    "q12-para": {"contrato-sla#s4": 3, "edital-pregao#s4": 1},
    "q13-noisy": {"edital-pregao#s4": 3},
    "q14-noisy": {"contrato-sla#s3": 3},
    "q15-noisy": {"politica-privacidade#s3": 3},
    # "onboarding acesso rede treinamento" -> credenciais e treinamento de
    # seguranca; o primeiro dia e contexto adjacente.
    "q16-noisy": {"manual-onboarding#s2": 3, "manual-onboarding#s1": 1},
}


def main() -> None:
    corpus = json.loads(CORPUS.read_text(encoding="utf-8"))

    for doc in corpus["documents"]:
        secoes = SECOES.get(doc["id"])
        if not secoes:
            continue
        doc["sections"] = [
            {"id": f"{doc['id']}#{sid}", "anchor": anchor, "title": title}
            for sid, anchor, title in secoes
        ]
        # Falha cedo se uma ancora nao existir: o Validate tambem barra, mas
        # errar aqui produz mensagem melhor.
        for section in doc["sections"]:
            if section["anchor"] not in doc["content"]:
                raise SystemExit(
                    f"ancora {section['anchor']!r} nao existe em {doc['id']}"
                )

    for query in corpus["queries"]:
        julgamentos = RELEVANCIA_SECOES.get(query["id"])
        if julgamentos:
            query["relevance_sections"] = julgamentos

    CORPUS.write_text(
        json.dumps(corpus, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )

    total = sum(len(v) for v in SECOES.values())
    pares = sum(len(v) for v in RELEVANCIA_SECOES.values())
    print(f"secoes declaradas: {total}")
    print(f"pares (consulta, secao) julgados: {pares}")


if __name__ == "__main__":
    main()
