#!/usr/bin/env python3
"""Compara os braços C (sem expansão) e E (com expansão) pergunta a pergunta.

A análise é pareada por pergunta entre os braços, e não por diferença de
médias: são as mesmas 71 perguntas medidas duas vezes. Perguntas que acertam
nos dois braços, ou erram nos dois, não carregam informação sobre a diferença e
ficam fora do teste, que é o que o pré-registro fixou
(docs/prds/pre-registro-expansao-consulta.md).
"""

import argparse
import csv
import json
import pathlib
from math import comb

TEMAS = ("seguranca-trabalho", "produto", "licitacoes")


def carrega(caminho, gabarito):
    """Resultado por pergunta: acertou em TODAS as repetições?

    O critério é exigente de propósito. Uma pergunta que acerta em duas de três
    não virou acerto por causa da expansão; virou moeda, e a bateria anterior
    mostrou que isso é raro (2 de 71).
    """
    por_id = {}
    for linha in csv.DictReader(caminho.open(encoding="utf-8")):
        por_id.setdefault(linha["question_id"], []).append(linha["correct"] == "true")
    meta = {q["id"]: q for q in json.loads(gabarito.read_text(encoding="utf-8"))["questions"]}
    return {k: (all(v), meta.get(k, {})) for k, v in por_id.items()}


def sinais(mais, menos):
    """Teste de sinais exato, bilateral, sobre os discordantes."""
    n = mais + menos
    if n == 0:
        return None
    k = max(mais, menos)
    cauda = sum(comb(n, i) for i in range(k, n + 1)) / 2 ** n
    return min(1.0, 2 * cauda)


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--saida", type=pathlib.Path,
                   default=pathlib.Path("docs/article/results/expansao"))
    args = p.parse_args()
    testdata = pathlib.Path("api/internal/evalcorpus/testdata")

    linhas, ganhou, perdeu = [], [], []
    for tema in TEMAS:
        off = args.saida / f"off-{tema}.csv"
        on = args.saida / f"on-{tema}.csv"
        if not (off.exists() and on.exists()):
            print(f"aviso: faltando CSV de {tema}, pulando")
            continue
        gabarito = testdata / f"gabarito-{tema}.json"
        c, e = carrega(off, gabarito), carrega(on, gabarito)
        for qid in sorted(set(c) & set(e)):
            certo_c, meta = c[qid]
            certo_e, _ = e[qid]
            linhas.append((qid, meta.get("variant", ""), certo_c, certo_e))
            if certo_e and not certo_c:
                ganhou.append((qid, meta))
            elif certo_c and not certo_e:
                perdeu.append((qid, meta))

    if not linhas:
        return 1

    def taxa(sel, idx):
        alvo = [x for x in linhas if sel(x)]
        return (sum(1 for x in alvo if x[idx]), len(alvo)) if alvo else (0, 0)

    print("## Braço C (sem expansão) contra braço E (com expansão)\n")
    print("| Recorte | C | E |")
    print("|---|---|---|")
    for rotulo, sel in (
        ("todas", lambda x: True),
        ("literal", lambda x: x[1] == "literal"),
        ("paráfrase", lambda x: x[1] == "parafrase"),
    ):
        (ac, n), (ae, _) = taxa(sel, 2), taxa(sel, 3)
        if n:
            print(f"| {rotulo} | {ac}/{n} ({ac/n:.3f}) | {ae}/{n} ({ae/n:.3f}) |")

    print(f"\n## Pareado por pergunta\n")
    print(f"- só E acertou: **{len(ganhou)}**")
    print(f"- só C acertou: **{len(perdeu)}**")
    print(f"- concordantes: {len(linhas) - len(ganhou) - len(perdeu)} de {len(linhas)}")
    p_valor = sinais(len(ganhou), len(perdeu))
    if p_valor is None:
        print("\nNenhuma pergunta mudou de resultado: o teste de sinais não se aplica.")
    else:
        veredito = "significativo" if p_valor < 0.05 else "direcional e inconclusivo"
        print(f"\nTeste de sinais bilateral: **p = {p_valor:.4f}** ({veredito}).")

    for titulo, grupo in (("Ganhou com a expansão", ganhou), ("Perdeu com a expansão", perdeu)):
        if grupo:
            print(f"\n### {titulo}\n")
            for qid, meta in grupo:
                print(f"- `{qid}` ({meta.get('variant','?')}, {meta.get('stratum','?')}): {meta.get('text','')}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
