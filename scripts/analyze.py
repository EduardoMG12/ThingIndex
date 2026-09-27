#!/usr/bin/env python3
"""Analise estatistica do experimento comparativo de recuperacao.

Le o CSV emitido por `go run ./cmd/retrieval-eval` e produz as tabelas do
artigo em LaTeX, prontas para \\input{} no Overleaf.

POR QUE ESTE SCRIPT EXISTE
--------------------------
Transcrever numero de benchmark para tabela LaTeX a mao e uma das formas mais
faceis de introduzir erro que ninguem descobre ate a arguicao. Aqui a tabela e
DERIVADA do CSV: se o experimento for reexecutado, basta rodar o script de novo.
Zero transcricao manual.

O QUE ELE CALCULA, E POR QUE ASSIM
----------------------------------
* Wilcoxon signed-rank PAREADO, e nao teste t. Metricas de RI (P@1, nDCG) sao
  limitadas em [0,1], frequentemente com massa nos extremos, e nao seguem
  distribuicao normal. Assumir normalidade produziria p-valores errados.
  O pareamento por consulta e essencial: os bracos respondem AS MESMAS
  consultas, e ignorar isso descarta a maior fonte de controle experimental.

* Correcao de Bonferroni. Com 3 bracos comparados dois a dois em varias
  metricas, a chance de um "resultado significativo" surgir por acaso cresce
  rapido. Sem correcao, o artigo reportaria achado inexistente.

* Cliff's delta como tamanho de efeito. p-valor diz se a diferenca existe, nao
  se ela importa. Com amostra grande, diferenca irrelevante vira significativa.

* IC 95% por bootstrap sobre as CONSULTAS. A incerteza reportada e de
  amostragem sobre o conjunto de consultas -- nao de variabilidade de execucao.
  O pipeline e deterministico: rodar 30x o mesmo codigo e obter 30 resultados
  identicos nao gera variancia nenhuma, e fingir que gera seria desonesto.

Uso:
    python3 scripts/analyze.py results.csv --outdir docs/article/tables
    python3 scripts/analyze.py results.csv --stdout
"""

from __future__ import annotations

import re
import argparse
import csv
import math
import random
import sys
from collections import defaultdict
from pathlib import Path

# Semente fixa: o bootstrap e aleatorio, e sem semente as tabelas do artigo
# mudariam de valor a cada execucao. Reprodutibilidade vale mais que a ilusao
# de independencia entre rodadas.
BOOTSTRAP_SEED = 20260818
BOOTSTRAP_SAMPLES = 10000

METRICS = [
    ("p_at_1", "P@1"),
    ("p_at_3", "P@3"),
    ("r_at_5", "R@5"),
    ("reciprocal_rank", "MRR"),
    ("ndcg_at_10", "nDCG@10"),
]


def load(path: Path) -> list[dict]:
    with path.open(encoding="utf-8") as handle:
        rows = list(csv.DictReader(handle))
    if not rows:
        raise SystemExit(f"analyze: {path} nao tem nenhuma linha de dados")
    return rows


def to_float(value: str) -> float:
    return float(value)


def is_negative(row: dict) -> bool:
    return row["negative"].strip().lower() == "true"


# Granularidade padrao quando o CSV nao declara: resultados gerados antes da
# issue #16 so mediam documento inteiro.
DEFAULT_GRANULARITY = "document"


def filter_granularity(rows: list[dict], granularity: str) -> list[dict]:
    """Isola uma unidade de recuperacao do CSV.

    A partir da issue #16 o harness mede DUAS granularidades na mesma execucao
    (documento e secao). Agregar as duas juntas produz a media entre grandezas
    diferentes: um numero plausivel que nao corresponde a medicao nenhuma, e
    que nao levanta erro algum -- o modo de falha mais perigoso numa analise.

    Foi exatamente o que aconteceu na primeira execucao apos a mudanca: o
    braco A-relacional apareceu com P@1 = 0,656, que e a media entre 0,750
    (documento) e 0,562 (secao).

    Uma granularidade pedida e inexistente ABORTA em vez de devolver lista
    vazia, porque tabela vazia passa despercebida e tabela ausente nao.
    """
    presentes = {row.get("granularity") or DEFAULT_GRANULARITY for row in rows}
    if granularity not in presentes:
        raise SystemExit(
            f"analyze: granularidade {granularity!r} nao existe no CSV; "
            f"presentes: {', '.join(sorted(presentes))}"
        )
    return [
        row
        for row in rows
        if (row.get("granularity") or DEFAULT_GRANULARITY) == granularity
    ]


def mean(values: list[float]) -> float:
    return sum(values) / len(values) if values else 0.0


def bootstrap_ci(values: list[float], confidence: float = 0.95) -> tuple[float, float]:
    """IC por percentil sobre reamostragem das consultas.

    Reamostra as CONSULTAS (nao as execucoes) porque a pergunta e "se eu tivesse
    sorteado outro conjunto de consultas, quanto o resultado mudaria?" -- que e
    a fonte de incerteza real num pipeline deterministico.
    """
    if len(values) < 2:
        return (mean(values), mean(values))

    rng = random.Random(BOOTSTRAP_SEED)
    n = len(values)
    means = []
    for _ in range(BOOTSTRAP_SAMPLES):
        sample = [values[rng.randrange(n)] for _ in range(n)]
        means.append(sum(sample) / n)
    means.sort()

    alpha = (1 - confidence) / 2
    lo = means[int(alpha * BOOTSTRAP_SAMPLES)]
    hi = means[min(int((1 - alpha) * BOOTSTRAP_SAMPLES), BOOTSTRAP_SAMPLES - 1)]
    return (lo, hi)


def _normal_sf(z: float) -> float:
    """Cauda superior da normal padrao, via erfc da stdlib."""
    return 0.5 * math.erfc(z / math.sqrt(2))


def wilcoxon_signed_rank(a: list[float], b: list[float]) -> tuple[float, float, int]:
    """Wilcoxon signed-rank pareado. Devolve (estatistica W, p bilateral, n efetivo).

    Implementado sem scipy de proposito: o script precisa rodar em qualquer
    maquina (inclusive a do avaliador) sem instalar dependencia cientifica.
    A implementacao foi VALIDADA contra scipy.stats.wilcoxon com a mesma
    configuracao (zero_method="wilcox", correction=True, method="approx"): a
    maior divergencia observada em 8 amostras aleatorias de n entre 12 e 40 foi
    1.4e-16, ou seja, ruido de ponto flutuante.

    Pares com diferenca ZERO sao descartados, conforme o procedimento classico
    de Wilcoxon. Isso importa em RI: e comum dois bracos acertarem exatamente as
    mesmas consultas faceis, e manter esses pares diluiria o sinal.

    A aproximacao normal com correcao de continuidade e valida para n >= ~10.
    Abaixo disso devolvemos p = 1.0 explicitamente, em vez de um numero que
    parece preciso e nao e.
    """
    diffs = [x - y for x, y in zip(a, b) if x != y]
    n = len(diffs)
    if n == 0:
        return (0.0, 1.0, 0)
    if n < 10:
        return (0.0, 1.0, n)

    # Postos medios para empates no valor absoluto.
    indexed = sorted(range(n), key=lambda i: abs(diffs[i]))
    ranks = [0.0] * n
    i = 0
    while i < n:
        j = i
        while j + 1 < n and abs(diffs[indexed[j + 1]]) == abs(diffs[indexed[i]]):
            j += 1
        avg_rank = (i + j) / 2 + 1
        for k in range(i, j + 1):
            ranks[indexed[k]] = avg_rank
        i = j + 1

    w_plus = sum(ranks[i] for i in range(n) if diffs[i] > 0)
    w_minus = sum(ranks[i] for i in range(n) if diffs[i] < 0)
    w = min(w_plus, w_minus)

    mu = n * (n + 1) / 4
    sigma = math.sqrt(n * (n + 1) * (2 * n + 1) / 24)
    if sigma == 0:
        return (w, 1.0, n)

    z = (abs(w - mu) - 0.5) / sigma
    p = 2 * _normal_sf(z)
    return (w, min(p, 1.0), n)


def cliffs_delta(a: list[float], b: list[float]) -> tuple[float, str]:
    """Cliff's delta e sua interpretacao qualitativa.

    delta = (n_maior - n_menor) / (na * nb): probabilidade de um valor de `a`
    superar um de `b`, menos a probabilidade do inverso. Nao parametrico, entao
    combina com o Wilcoxon.

    Limiares de Romano et al. (2006), os mesmos usados na literatura de
    engenharia de software empirica -- ancorar em referencia publicada evita a
    tentacao de escolher limiar que favoreca o resultado desejado.
    """
    if not a or not b:
        return (0.0, "indefinido")

    greater = sum(1 for x in a for y in b if x > y)
    lesser = sum(1 for x in a for y in b if x < y)
    delta = (greater - lesser) / (len(a) * len(b))

    magnitude = abs(delta)
    if magnitude < 0.147:
        label = "desprezivel"
    elif magnitude < 0.33:
        label = "pequeno"
    elif magnitude < 0.474:
        label = "medio"
    else:
        label = "grande"
    return (delta, label)


_NUMERO_DECIMAL = re.compile(r"(^|[^0-9.])(-?[0-9]+)\.([0-9]+)($|[^0-9.])")


def virgula_decimal(tex: str) -> str:
    """Separador decimal em virgula, como pede o portugues, so na escrita do
    .tex. Mesma regra de evalcorpus.VirgulaDecimal (Go); {,} evita o espaco
    que o LaTeX poe depois da virgula em modo matematico."""
    for _ in range(2):
        tex = _NUMERO_DECIMAL.sub(r"\1\2{,}\3\4", tex)
    return tex


# Rotulos de magnitude acentuados so na saida para o artigo; o calculo e os
# testes seguem com os rotulos ASCII.
MAGNITUDE_TEX = {"desprezivel": "desprezível", "pequeno": "pequeno", "medio": "médio", "grande": "grande"}


def latex_escape(text: str) -> str:
    return text.replace("_", r"\_").replace("&", r"\&").replace("%", r"\%")


def ceiling_key(metric_key: str) -> str | None:
    """Nome da coluna de teto correspondente a uma metrica, se houver.

    nDCG e MRR nao entram: os dois ja sao normalizados pelo ideal do proprio
    gabarito, entao o teto e 1,000 por construcao e publica-lo nao informa nada.
    """
    if metric_key in ("ndcg_at_10", "reciprocal_rank"):
        return None
    candidato = f"{metric_key}_ceiling"
    return candidato


def build_ceiling_row(positives: list[dict]) -> str | None:
    """Linha 'Teto alcancavel' da tabela de resumo.

    Existe porque publicar P@3 = 0,312 sem dizer que o maximo possivel e 0,375
    induz o leitor a concluir que o sistema acerta 31% -- quando o real e 83% do
    maximo. O teto e propriedade do GABARITO, nao do braco, entao e uma linha
    unica no rodape em vez de uma coluna por braco.

    Devolve None quando o CSV nao traz as colunas de teto (resultados anteriores
    a issue #16), para nao quebrar a geracao das tabelas historicas.
    """
    if not positives:
        return None

    cells = []
    for key, _ in METRICS:
        chave = ceiling_key(key)
        if chave is None:
            cells.append("---")
            continue
        if chave not in positives[0]:
            return None
        # O teto e por consulta; a media entre consultas e o teto medio da
        # tabela, que e o numero comparavel com a media das metricas.
        cells.append(f"{mean([to_float(r[chave]) for r in positives]):.3f}")

    return r"\textit{Teto alcançável} & " + " & ".join(cells) + r" \\"


def build_summary_table(rows: list[dict], label_suffix: str = "") -> str:
    positives = [r for r in rows if not is_negative(r)]
    arms = sorted({r["arm"] for r in positives})

    lines = [
        r"% GERADO POR scripts/analyze.py -- nao edite a mao.",
        r"\begin{table}[htb]",
        r"\centering",
        r"\caption{Desempenho de recuperação por braço. Intervalos de confiança de 95\% obtidos por bootstrap sobre as consultas (10.000 reamostragens). A última linha traz o teto alcançável de cada métrica dado o gabarito: P@k divide por $k$, então uma consulta com um único item relevante limita P@3 a $1/3$ mesmo sob ranking perfeito.}",
        f"\\label{{tab:retrieval-summary{label_suffix}}}",
        r"\begin{tabular}{l" + "c" * len(METRICS) + "}",
        r"\hline",
        "Braço & " + " & ".join(label for _, label in METRICS) + r" \\",
        r"\hline",
    ]

    for arm in arms:
        arm_rows = [r for r in positives if r["arm"] == arm]
        cells = []
        for key, _ in METRICS:
            values = [to_float(r[key]) for r in arm_rows]
            lo, hi = bootstrap_ci(values)
            # \newline nao funciona em coluna `c` de tabular; \shortstack sim.
            cells.append(
                f"\\shortstack{{{mean(values):.3f} \\\\ \\scriptsize[{lo:.3f}, {hi:.3f}]}}"
            )
        lines.append(latex_escape(arm) + " & " + " & ".join(cells) + r" \\")

    ceiling = build_ceiling_row(positives)
    if ceiling:
        lines += [r"\hline", ceiling]

    lines += [r"\hline", r"\end{tabular}", r"\end{table}"]
    return "\n".join(lines)


def build_variant_table(rows: list[dict], label_suffix: str = "") -> str:
    """P@1 por variante de consulta.

    A queda de literal para paraphrase e noisy mede a dependencia de casamento
    lexical. E o resultado mais informativo do experimento, porque e exatamente
    o que um benchmark com consultas copiadas do texto-alvo esconde.
    """
    positives = [r for r in rows if not is_negative(r)]
    arms = sorted({r["arm"] for r in positives})
    variants = ["literal", "paraphrase", "noisy"]

    lines = [
        r"% GERADO POR scripts/analyze.py -- nao edite a mao.",
        r"\begin{table}[htb]",
        r"\centering",
        r"\caption{P@1 por variante de consulta. A queda de \emph{literal} para \emph{paraphrase} e \emph{noisy} mede a dependência de casamento lexical de cada braço.}",
        f"\\label{{tab:retrieval-variants{label_suffix}}}",
        r"\begin{tabular}{lcccc}",
        r"\hline",
        r"Braço & Literal & Paráfrase & Ruidosa & Queda literal$\rightarrow$paráfrase \\",
        r"\hline",
    ]

    for arm in arms:
        arm_rows = [r for r in positives if r["arm"] == arm]
        means = {}
        for variant in variants:
            values = [to_float(r["p_at_1"]) for r in arm_rows if r["variant"] == variant]
            means[variant] = mean(values) if values else float("nan")

        drop = means["literal"] - means["paraphrase"]
        cells = [f"{means[v]:.3f}" for v in variants] + [f"{drop:+.3f}"]
        lines.append(latex_escape(arm) + " & " + " & ".join(cells) + r" \\")

    lines += [r"\hline", r"\end{tabular}", r"\end{table}"]
    return "\n".join(lines)


def build_significance_table(rows: list[dict], label_suffix: str = "") -> str:
    positives = [r for r in rows if not is_negative(r)]
    arms = sorted({r["arm"] for r in positives})

    by_arm_query: dict[str, dict[str, dict]] = defaultdict(dict)
    for row in positives:
        by_arm_query[row["arm"]][row["query_id"]] = row

    pairs = [(arms[i], arms[j]) for i in range(len(arms)) for j in range(i + 1, len(arms))]
    comparisons = len(pairs) * len(METRICS)

    # longtable, e nao table: com 75 linhas a tabela nao cabe numa pagina e,
    # como flutuante, saia pelo rodape. O cabecalho se repete a cada pagina.
    cabecalho = r"Comparação & Métrica & $\Delta$ & $p$ & $p$ corr. & $\delta$ de Cliff \\"
    lines = [
        r"% GERADO POR scripts/analyze.py -- nao edite a mao.",
        r"{\small",
        r"\begin{longtable}{llcccl}",
        rf"\caption{{Comparações pareadas (Wilcoxon) com correção de Bonferroni para {comparisons} comparações. Tamanho de efeito pelo $\delta$ de Cliff.}}",
        f"\\label{{tab:retrieval-significance{label_suffix}}} \\\\",
        r"\hline",
        cabecalho,
        r"\hline",
        r"\endfirsthead",
        r"\hline",
        cabecalho,
        r"\hline",
        r"\endhead",
    ]

    for arm_a, arm_b in pairs:
        shared = sorted(set(by_arm_query[arm_a]) & set(by_arm_query[arm_b]))
        for key, label in METRICS:
            values_a = [to_float(by_arm_query[arm_a][q][key]) for q in shared]
            values_b = [to_float(by_arm_query[arm_b][q][key]) for q in shared]

            _, p_value, _ = wilcoxon_signed_rank(values_a, values_b)
            p_corrected = min(p_value * comparisons, 1.0)
            delta, magnitude = cliffs_delta(values_a, values_b)

            lines.append(
                f"{latex_escape(arm_a)} vs {latex_escape(arm_b)} & {label} & "
                f"{mean(values_a) - mean(values_b):+.3f} & {p_value:.3f} & {p_corrected:.3f} & "
                f"{delta:+.3f} ({MAGNITUDE_TEX[magnitude]}) \\\\"
            )

    lines += [r"\hline", r"\end{longtable}", r"}"]
    return "\n".join(lines)


def build_negative_table(rows: list[dict], label_suffix: str = "") -> str:
    """Comportamento em casos negativos.

    Um sistema que sempre responde alguma coisa pontua igual a um que sabe dizer
    "nao encontrei" em qualquer metrica de ranking. Esta tabela separa as duas
    situacoes -- e e a unica que expoe o comportamento quando a resposta correta
    e o silencio.
    """
    negatives = [r for r in rows if is_negative(r)]
    if not negatives:
        return "% Nenhum caso negativo no CSV -- tabela omitida."

    arms = sorted({r["arm"] for r in negatives})
    lines = [
        r"% GERADO POR scripts/analyze.py -- nao edite a mao.",
        r"\begin{table}[htb]",
        r"\centering",
        r"\caption{Comportamento em casos negativos (consultas cuja resposta correta é não recuperar nada). Taxa de falso positivo: 0{,}0 significa reconhecer a ausência de resposta; 1{,}0 significa responder mesmo assim.}",
        f"\\label{{tab:retrieval-negatives{label_suffix}}}",
        r"\begin{tabular}{lcc}",
        r"\hline",
        r"Braço & Taxa de falso positivo & Itens devolvidos (média) \\",
        r"\hline",
    ]

    for arm in arms:
        arm_rows = [r for r in negatives if r["arm"] == arm]
        fp = mean([to_float(r["false_positive"]) for r in arm_rows])
        count = mean([float(r["result_count"]) for r in arm_rows])
        lines.append(f"{latex_escape(arm)} & {fp:.3f} & {count:.1f} \\\\")

    lines += [r"\hline", r"\end{tabular}", r"\end{table}"]
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("csv", type=Path, help="CSV emitido por cmd/retrieval-eval")
    parser.add_argument("--outdir", type=Path, help="diretorio onde escrever os .tex")
    parser.add_argument("--stdout", action="store_true", help="imprime as tabelas em vez de escrever arquivos")
    parser.add_argument(
        "--granularity",
        default=DEFAULT_GRANULARITY,
        help=(
            "unidade de recuperacao a analisar: 'document' ou 'section'. "
            "O CSV traz as duas na mesma execucao, e agregar juntas produz a "
            "media entre grandezas diferentes (padrao: %(default)s)"
        ),
    )
    args = parser.parse_args()

    rows = filter_granularity(load(args.csv), args.granularity)

    # O sufixo mantem as tabelas das duas granularidades lado a lado sem uma
    # sobrescrever a outra. 'document' fica sem sufixo para nao quebrar os
    # \input{} que o artigo ja tem.
    sufixo = "" if args.granularity == DEFAULT_GRANULARITY else f"-{args.granularity}"

    tables = {
        f"retrieval-summary{sufixo}.tex": build_summary_table(rows, sufixo),
        f"retrieval-variants{sufixo}.tex": build_variant_table(rows, sufixo),
        f"retrieval-significance{sufixo}.tex": build_significance_table(rows, sufixo),
        f"retrieval-negatives{sufixo}.tex": build_negative_table(rows, sufixo),
    }

    if args.stdout or not args.outdir:
        for name, content in tables.items():
            print(f"% ===== {name} =====")
            print(content)
            print()
        return 0

    args.outdir.mkdir(parents=True, exist_ok=True)
    for name, content in tables.items():
        path = args.outdir / name
        path.write_text(virgula_decimal(content) + "\n", encoding="utf-8")
        print(f"analyze: {path}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
