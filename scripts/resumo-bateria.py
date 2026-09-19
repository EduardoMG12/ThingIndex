#!/usr/bin/env python3
"""Consolida os CSV da bateria de temas em três recortes.

Os recortes são por tema, por forma de pergunta e por formulação (literal
contra paráfrase). O terceiro é o que responde a pergunta central do trabalho
sobre reuso de vocabulário, e ele só existe porque cada pergunta foi escrita em
duas versões com a mesma resposta esperada.

Saída em markdown, para colar direto no artigo ou no slide.
"""

import argparse
import csv
import json
import pathlib
import sys

FORMAS = ["aberta", "confirmacao", "existencia", "premissa-falsa", "negativa"]


def carrega(csv_path, gabarito_path):
    """Junta o CSV de resultado com o gabarito, pelo id da pergunta.

    A variante vive no CSV desde 19/09/2026, mas CSV antigo não a tem, então a
    do gabarito vale como reserva. Sem isso, um arquivo gravado antes daquela
    data sairia da conta de formulação sem avisar.
    """
    gabarito = json.loads(gabarito_path.read_text(encoding="utf-8"))
    por_id = {q["id"]: q for q in gabarito["questions"]}
    linhas = []
    with csv_path.open(encoding="utf-8") as f:
        for linha in csv.DictReader(f):
            q = por_id.get(linha["question_id"], {})
            linha["variant"] = linha.get("variant") or q.get("variant", "")
            linha["pair_id"] = linha.get("pair_id") or q.get("pair_id", "")
            linhas.append(linha)
    return linhas


def taxa(linhas, filtro=lambda _: True):
    alvo = [x for x in linhas if filtro(x)]
    if not alvo:
        return None, 0, 0
    certas = sum(1 for x in alvo if x["correct"] == "true")
    return certas / len(alvo), certas, len(alvo)


def linha_md(rotulo, r):
    frac, certas, total = r
    if frac is None:
        return f"| {rotulo} | — | — |"
    return f"| {rotulo} | {certas}/{total} | {frac:.3f} |"



def estabilidade(linhas):
    """Cada pergunta deu o mesmo resultado em todas as repetições?

    É o número que responde "como você sabe que essa taxa não foi sorte":
    uma pergunta que oscila entre repetições não tem taxa, tem moeda. A
    contagem separa acerto estável, erro estável e oscilação.
    """
    por = {}
    for x in linhas:
        por.setdefault(x["question_id"], []).append(x["correct"] == "true")
    reps = {len(v) for v in por.values()}
    sempre = sum(1 for v in por.values() if all(v))
    nunca = sum(1 for v in por.values() if not any(v))
    oscilou = [k for k, v in por.items() if len(set(v)) > 1]
    return por, reps, sempre, nunca, oscilou


def main():
    p = argparse.ArgumentParser()
    p.add_argument("saida", type=pathlib.Path)
    p.add_argument("--tema")
    args = p.parse_args()

    raiz = pathlib.Path(__file__).resolve().parent.parent
    testdata = raiz / "api" / "internal" / "evalcorpus" / "testdata"

    todos = []
    por_tema = {}
    for csv_path in sorted(args.saida.glob("bateria-*.csv")):
        tema = csv_path.stem.replace("bateria-", "")
        if args.tema and tema != args.tema:
            continue
        gabarito = testdata / f"gabarito-{tema}.json"
        if not gabarito.exists():
            print(f"aviso: sem gabarito para {tema}, pulando", file=sys.stderr)
            continue
        linhas = carrega(csv_path, gabarito)
        por_tema[tema] = linhas
        todos.extend(linhas)

    if not todos:
        print("nenhum CSV de bateria encontrado em " + str(args.saida), file=sys.stderr)
        return 1

    print("\n## Por tema\n")
    print("| Tema | Acertos | Taxa |")
    print("|---|---|---|")
    for tema, linhas in sorted(por_tema.items()):
        print(linha_md(tema, taxa(linhas)))
    print(linha_md("**todos**", taxa(todos)))

    print("\n## Por forma da pergunta\n")
    print("| Forma | Acertos | Taxa |")
    print("|---|---|---|")
    for forma in FORMAS:
        print(linha_md(forma, taxa(todos, lambda x, f=forma: x["stratum"] == f)))

    print("\n## Por formulação\n")
    print("| Formulação | Acertos | Taxa |")
    print("|---|---|---|")
    for variante in ("literal", "parafrase"):
        print(linha_md(variante, taxa(todos, lambda x, v=variante: x["variant"] == v)))

    lit = taxa(todos, lambda x: x["variant"] == "literal")[0]
    par = taxa(todos, lambda x: x["variant"] == "parafrase")[0]
    if lit is not None and par is not None:
        print(f"\nQueda literal para paráfrase: **{lit:.3f} para {par:.3f}** "
              f"({(lit - par) * 100:.1f} pontos).")

    # Pares em que uma variante acertou e a outra não: é a evidência item a item
    # da queda, e serve para escolher exemplos para o slide.
    quebrados = []
    for linhas in por_tema.values():
        pares = {}
        for x in linhas:
            if x["pair_id"]:
                pares.setdefault(x["pair_id"], {})[x["variant"]] = x
        for pid, v in pares.items():
            if len(v) == 2:
                cl = v.get("literal", {}).get("correct") == "true"
                cp = v.get("parafrase", {}).get("correct") == "true"
                if cl != cp:
                    quebrados.append((pid, "literal" if cl else "parafrase"))
    if quebrados:
        print("\n## Pares em que a formulação decidiu o resultado\n")
        for pid, ganhou in sorted(quebrados):
            print(f"- `{pid}`: só a versão **{ganhou}** acertou")
    por, reps, sempre, nunca, oscilou = estabilidade(todos)
    if reps and max(reps) > 1:
        print("\n## Estabilidade entre repetições\n")
        print(f"Repetições por pergunta: {sorted(reps)}")
        print("\n| Comportamento | Perguntas |")
        print("|---|---|")
        print(f"| Acertou em todas | {sempre} |")
        print(f"| Errou em todas | {nunca} |")
        print(f"| Oscilou entre repetições | {len(oscilou)} |")
        print(f"| **Total** | **{len(por)}** |")
        if oscilou:
            print("\nOscilaram: " + ", ".join(f"`{k}`" for k in sorted(oscilou)))
        else:
            print("\nNenhuma pergunta mudou de resultado entre repetições: "
                  "a taxa é do sistema, não da amostragem do modelo.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
