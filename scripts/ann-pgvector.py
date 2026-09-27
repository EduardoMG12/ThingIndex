#!/usr/bin/env python3
"""Estabilidade e revocacao dos indices aproximados do pgvector REAL.

A primeira versao do artigo mediu um IVF reimplementado em Go com
lists = sqrt(N) e probes = 1, e atribuiu esses valores ao pgvector. A
documentacao do pgvector recomenda outra coisa (lists = rows/1000 ate 1
milhao de linhas, probes = sqrt(lists)) e o indice padrao desde a 0.5.0 e o
HNSW. Este script mede o proprio pgvector, sem reimplementacao, nas
configuracoes recomendadas e na da versao anterior, sobre corpora
sinteticos com estrutura de cluster em 768 dimensoes.

Para cada configuracao e tamanho N:

  * revocacao@10: fracao do top-10 exato (calculado em numpy) que o indice
    devolve, media sobre consultas e construcoes;
  * reconstrucao: o MESMO dado, na MESMA ordem, com o indice apagado e
    recriado R vezes (o que acontece num REINDEX ou numa restauracao);
  * ordem: o mesmo conjunto de vetores inserido em R ordens diferentes.

"Instavel" = a consulta teve mais de um top-10 distinto entre as
construcoes. Todas as consultas forcam o uso do indice
(enable_seqscan = off).

Uso (exige Docker):
    python3 scripts/ann-pgvector.py --tex docs/article/tables/ann-pgvector.tex \
        --csv docs/article/results/ann-pgvector.csv
"""

from __future__ import annotations

import argparse
import csv
import math
import subprocess
import sys
import time
from pathlib import Path

try:  # so a medicao precisa deles; --so-tabela roda com a biblioteca padrao
    import numpy as np
    import psycopg2
except ImportError:  # pragma: no cover
    np = None
    psycopg2 = None

IMAGEM = "pgvector/pgvector:pg16"
CONTEINER = "thingindex-ann-pgvector"
PORTA = 25499
DIM = 768
CONSULTAS = 20
TOPK = 10
RECONSTRUCOES = 5
ORDENS = 5
TAMANHOS = (1000, 5000, 20000)


def corpus(n: int, seed: int = 7) -> np.ndarray:
    """8 clusters gaussianos, mesma construcao de internal/retrieval/ivf_scale_test.go."""
    rng = np.random.default_rng(seed)
    centros = rng.standard_normal((8, DIM))
    centros /= np.linalg.norm(centros, axis=1, keepdims=True)
    v = centros[np.arange(n) % 8] + 0.35 * rng.standard_normal((n, DIM)) / math.sqrt(DIM)
    return (v / np.linalg.norm(v, axis=1, keepdims=True)).astype(np.float32)


def consultas(v: np.ndarray, seed: int = 99) -> np.ndarray:
    rng = np.random.default_rng(seed)
    base = v[rng.integers(0, len(v), CONSULTAS)]
    q = base + 0.15 * rng.standard_normal(base.shape) / math.sqrt(DIM)
    return (q / np.linalg.norm(q, axis=1, keepdims=True)).astype(np.float32)


def literal(x: np.ndarray) -> str:
    return "[" + ",".join(f"{a:.6f}" for a in x) + "]"


def subir_banco() -> None:
    subprocess.run(["docker", "rm", "-f", CONTEINER], capture_output=True)
    subprocess.run(["docker", "run", "-d", "--name", CONTEINER, "-p", f"{PORTA}:5432",
                    "-e", "POSTGRES_PASSWORD=ann", "-e", "POSTGRES_DB=ann", IMAGEM,
                    "-c", "maintenance_work_mem=1GB", "-c", "shared_buffers=512MB"],
                   check=True, capture_output=True)
    for _ in range(60):
        try:
            psycopg2.connect(host="127.0.0.1", port=PORTA, user="postgres", password="ann", dbname="ann").close()
            return
        except psycopg2.OperationalError:
            time.sleep(1)
    sys.exit("postgres do pgvector nao subiu")


def conectar():
    c = psycopg2.connect(host="127.0.0.1", port=PORTA, user="postgres", password="ann", dbname="ann")
    c.autocommit = True
    return c


def carregar(cur, v: np.ndarray, ordem: np.ndarray) -> None:
    cur.execute("DROP TABLE IF EXISTS itens")
    cur.execute(f"CREATE TABLE itens (id int, v vector({DIM}))")
    lote = []
    for i in ordem:
        lote.append(cur.mogrify("(%s,%s)", (int(i), literal(v[i]))).decode())
        if len(lote) == 500:
            cur.execute("INSERT INTO itens VALUES " + ",".join(lote))
            lote = []
    if lote:
        cur.execute("INSERT INTO itens VALUES " + ",".join(lote))


def criar_indice(cur, cfg: dict, n: int) -> None:
    cur.execute("DROP INDEX IF EXISTS itens_v")
    if cfg["tipo"] == "ivfflat":
        cur.execute(f"CREATE INDEX itens_v ON itens USING ivfflat (v vector_cosine_ops) WITH (lists = {cfg['lists'](n)})")
        cur.execute(f"SET ivfflat.probes = {cfg['probes'](n)}")
    else:
        cur.execute("CREATE INDEX itens_v ON itens USING hnsw (v vector_cosine_ops) WITH (m = 16, ef_construction = 64)")
        cur.execute("SET hnsw.ef_search = 40")
    cur.execute("SET enable_seqscan = off")
    cur.execute("ANALYZE itens")


def buscar(cur, q: np.ndarray) -> tuple[int, ...]:
    cur.execute(f"SELECT id FROM itens ORDER BY v <=> %s::vector LIMIT {TOPK}", (literal(q),))
    return tuple(r[0] for r in cur.fetchall())


CONFIGS = [
    {"nome": "IVFFlat, $\\sqrt{N}$ listas, 1 sonda", "tipo": "ivfflat",
     "lists": lambda n: max(1, round(math.sqrt(n))), "probes": lambda n: 1},
    {"nome": "IVFFlat, recomendação do pgvector", "tipo": "ivfflat",
     "lists": lambda n: max(1, math.ceil(n / 1000)) if n <= 1_000_000 else round(math.sqrt(n)),
     "probes": lambda n: max(1, round(math.sqrt(max(1, math.ceil(n / 1000)))))},
    {"nome": "HNSW, padrões do pgvector", "tipo": "hnsw"},
]


def medir(cur, v, qs, exato, cfg, n, construcoes) -> tuple[float, int]:
    """construcoes: lista de ordens de insercao (None = reusar a tabela)."""
    tops = [[] for _ in qs]
    for ordem in construcoes:
        if ordem is not None:
            carregar(cur, v, ordem)
        criar_indice(cur, cfg, n)
        for i, q in enumerate(qs):
            tops[i].append(buscar(cur, q))
    rev = np.mean([len(set(t) & exato[i]) / TOPK for i, ts in enumerate(tops) for t in ts])
    instaveis = sum(len(set(ts)) > 1 for ts in tops)
    return float(rev), instaveis


def tabela_tex(linhas: list[dict]) -> str:
    """Tabela do artigo a partir das linhas medidas (ou lidas do CSV)."""
    versao = linhas[0]["pgvector"]
    tex = [
        "% GERADO POR scripts/ann-pgvector.py a partir de docs/article/results/ann-pgvector.csv -- nao edite a mao.",
        "\\begin{table}[htb]", "\\centering",
        f"\\caption{{Índices aproximados do pgvector {versao} sobre \\emph{{corpora}} sintéticos (768 dimensões, {CONSULTAS} consultas). "
        f"Revocação dos 10 primeiros frente à busca exata e número de consultas cujos 10 primeiros resultados mudaram ao reconstruir o índice {RECONSTRUCOES} vezes "
        f"sobre os mesmos dados e ao inserir os mesmos vetores em {ORDENS} ordens.}}",
        "\\label{tab:ann-pgvector}", "\\small",
        "\\begin{tabular}{lrrrr}", "\\toprule",
        "Índice & $N$ & Revocação@10 & Reconstrução & Ordem \\\\", "\\midrule",
    ]
    atual = None
    for l in linhas:
        n = int(l["n"])
        if atual is not None and n != atual:
            tex.append("\\midrule")
        atual = n
        n_br = f"{n:,}".replace(",", ".")
        rev_br = f"{100 * float(l['revocacao_at_10']):.1f}".replace(".", "{,}")
        tex.append(f"{l['indice']} & {n_br} & {rev_br}\\% & "
                   f"{l['instaveis_reconstrucao']}/{CONSULTAS} & {l['instaveis_ordem']}/{CONSULTAS} \\\\")
    tex += ["\\bottomrule", "\\end{tabular}", "\\end{table}"]
    return "\n".join(tex) + "\n"


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--tex")
    ap.add_argument("--csv")
    ap.add_argument("--so-tabela", action="store_true",
                    help="nao mede: le --csv e escreve --tex (o CI usa para conferir a tabela)")
    args = ap.parse_args()

    if args.so_tabela:
        with open(args.csv, encoding="utf-8") as f:
            linhas = list(csv.DictReader(f))
        Path(args.tex).write_text(tabela_tex(linhas), encoding="utf-8")
        return 0

    subir_banco()
    con = conectar()
    cur = con.cursor()
    cur.execute("CREATE EXTENSION IF NOT EXISTS vector")
    cur.execute("SELECT extversion FROM pg_extension WHERE extname = 'vector'")
    linha = cur.fetchone()
    if linha is None:
        sys.exit("extensao vector nao instalada")
    versao = linha[0]
    print(f"pgvector {versao}", file=sys.stderr)

    linhas = []
    for n in TAMANHOS:
        v = corpus(n)
        qs = consultas(v)
        exato = [set(np.argsort(-(v @ q))[:TOPK].tolist()) for q in qs]
        identidade = np.arange(n)
        rng = np.random.default_rng(1000)
        ordens = [rng.permutation(n) for _ in range(ORDENS)]
        for cfg in CONFIGS:
            t0 = time.time()
            carregar(cur, v, identidade)
            rev, inst_rec = medir(cur, v, qs, exato, cfg, n, [None] * RECONSTRUCOES)
            _, inst_ord = medir(cur, v, qs, exato, cfg, n, ordens)
            params = (f"lists={cfg['lists'](n)} probes={cfg['probes'](n)}" if cfg["tipo"] == "ivfflat"
                      else "m=16 ef_construction=64 ef_search=40")
            print(f"{cfg['nome']:<38} N={n:>6} {params:<34} revocacao={100 * rev:5.1f}% "
                  f"reconstrucao={inst_rec:2d}/{CONSULTAS} ordem={inst_ord:2d}/{CONSULTAS} ({time.time() - t0:.0f}s)",
                  file=sys.stderr)
            linhas.append({"indice": cfg["nome"], "tipo": cfg["tipo"], "n": n, "parametros": params,
                           "revocacao_at_10": round(rev, 4), "instaveis_reconstrucao": inst_rec,
                           "instaveis_ordem": inst_ord, "consultas": CONSULTAS, "pgvector": versao})
    subprocess.run(["docker", "rm", "-f", CONTEINER], capture_output=True)

    if args.csv:
        with open(args.csv, "w", newline="", encoding="utf-8") as f:
            w = csv.DictWriter(f, fieldnames=list(linhas[0]))
            w.writeheader()
            w.writerows(linhas)

    saida = tabela_tex(linhas)
    if args.tex:
        Path(args.tex).write_text(saida, encoding="utf-8")
    else:
        print(saida)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
