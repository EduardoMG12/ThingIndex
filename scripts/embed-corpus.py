#!/usr/bin/env python3
"""Gera os embeddings do braco C (recuperacao densa) do experimento comparativo.

POR QUE ESTE SCRIPT EXISTE, E POR QUE ELE NAO RODA NO HARNESS
-------------------------------------------------------------
O braco C precisa de um modelo de embedding (768 dimensoes, ~1,1 GB de pesos).
Carregar isso dentro do `retrieval-eval` teria tres consequencias ruins:

  1. O harness deixaria de ser hermetico -- passaria a depender de download de
     modelo, de GPU/CPU disponivel e de versao de torch. Nao roda em CI.
  2. A execucao deixaria de ser reproduzivel: versoes diferentes de
     `transformers` produzem embeddings numericamente diferentes.
  3. O experimento passaria a medir tempo de inferencia do modelo junto com
     tempo de recuperacao, contaminando a metrica de latencia.

A solucao adotada e a mesma que a literatura usa para replicabilidade: os
embeddings sao computados UMA VEZ, aqui, e versionados como artefato. O
harness em Go le o artefato e faz apenas a busca. O manifesto no topo do
arquivo (modelo, revisao, dimensao, hash do texto de cada item) permite que
qualquer pessoa verifique que os vetores correspondem ao corpus versionado --
e o teste em Go FALHA se o corpus mudar sem os embeddings serem regerados.

O PREFIXO DO E5 -- O DETALHE QUE DECIDE SE A BASELINE E HONESTA
---------------------------------------------------------------
O `multilingual-e5-base` foi treinado com prefixos assimetricos obrigatorios:
"query: " para consultas e "passage: " para documentos. Omiti-los degrada a
qualidade de forma significativa (o proprio card do modelo documenta isso).

Isso importa cientificamente: o braco C e a BASELINE CONCORRENTE da tese deste
trabalho. Configura-lo mal produziria um numero artificialmente baixo, e o
trabalho estaria vencendo um espantalho. Um baseline precisa ser implementado
na sua melhor configuracao publicada -- caso contrario a comparacao nao vale
nada, e a banca tem todo o direito de desmontar o resultado.

NORMALIZACAO
------------
Os vetores sao normalizados para norma 1 aqui, na geracao. Com norma 1, o
produto interno E a similaridade de cosseno, entao o codigo Go nao precisa
dividir por normas em tempo de busca -- menos operacao de ponto flutuante,
menos chance de divergencia numerica entre execucoes.

USO
---
    python3 scripts/embed-corpus.py \
        --corpus api/internal/evalcorpus/testdata/corpus-c1.json \
        --out    api/internal/evalcorpus/testdata/embeddings-c1.json
"""

from __future__ import annotations

import argparse
import hashlib
import json
import pathlib
import sys

MODEL_NAME = "intfloat/multilingual-e5-base"

# Dimensao esperada do modelo. Declarada explicitamente para que a troca
# silenciosa de modelo (que mudaria a dimensao) quebre aqui, na geracao, e nao
# vire um numero errado no artigo.
EXPECTED_DIM = 768


def sha256_text(text: str) -> str:
    """Hash do texto exatamente como foi enviado ao modelo.

    E este hash que o teste em Go confere. Ele responde a pergunta que decide
    se o resultado publicado e valido: "estes vetores foram gerados a partir
    DESTE corpus, ou de uma versao anterior dele?"
    """
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def build_passage(doc: dict) -> str:
    """Monta o texto do documento igual ao braco lexical: titulo + conteudo.

    Igualar a unidade indexavel entre os bracos nao e detalhe de implementacao,
    e condicao de validade: se o braco denso indexasse titulo+conteudo e o
    lexical so o conteudo, a comparacao mediria diferenca de campo indexado em
    vez de diferenca de modelo de recuperacao.

    O `buildArms` do harness Go faz `doc.Title + "\\n" + doc.Content`. Qualquer
    divergencia aqui invalida a comparacao, entao a linha abaixo e deliberada.
    """
    return f"{doc['title']}\n{doc['content']}"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--corpus", required=True, type=pathlib.Path)
    parser.add_argument("--out", required=True, type=pathlib.Path)
    parser.add_argument("--batch-size", type=int, default=8)
    args = parser.parse_args()

    corpus = json.loads(args.corpus.read_text(encoding="utf-8"))

    # Importado aqui, e nao no topo, para que `--help` funcione sem carregar
    # 1 GB de torch.
    from sentence_transformers import SentenceTransformer
    import torch

    # CPU explicito: a inferencia em GPU usa kernels nao-deterministicos e
    # produziria vetores ligeiramente diferentes a cada execucao. Como este
    # artefato e versionado e conferido por hash, ele precisa ser bit-a-bit
    # reproduzivel. O corpus e pequeno; o custo de rodar em CPU e irrelevante.
    torch.manual_seed(0)
    model = SentenceTransformer(MODEL_NAME, device="cpu")
    model.eval()

    dim = model.get_sentence_embedding_dimension()
    if dim != EXPECTED_DIM:
        print(
            f"erro: modelo devolveu dimensao {dim}, esperada {EXPECTED_DIM}",
            file=sys.stderr,
        )
        return 1

    documents = corpus["documents"]
    queries = corpus["queries"]

    passages = [build_passage(doc) for doc in documents]
    # Os prefixos do e5. Ver o bloco de docstring: sem eles a baseline fica
    # artificialmente fraca e a comparacao perde validade.
    passage_inputs = [f"passage: {text}" for text in passages]
    query_inputs = [f"query: {query['text']}" for query in queries]

    with torch.inference_mode():
        passage_vectors = model.encode(
            passage_inputs,
            batch_size=args.batch_size,
            normalize_embeddings=True,
            convert_to_numpy=True,
            show_progress_bar=False,
        )
        query_vectors = model.encode(
            query_inputs,
            batch_size=args.batch_size,
            normalize_embeddings=True,
            convert_to_numpy=True,
            show_progress_bar=False,
        )

    def encode_entries(ids, texts, vectors):
        entries = []
        for identifier, text, vector in zip(ids, texts, vectors):
            entries.append(
                {
                    "id": identifier,
                    "text_sha256": sha256_text(text),
                    # float() explicito: numpy.float32 nao serializa em JSON, e
                    # a conversao para float64 aqui e exata (float32 -> float64
                    # nao perde bit nenhum).
                    "vector": [float(value) for value in vector],
                }
            )
        return entries

    artifact = {
        "manifest": {
            "model": MODEL_NAME,
            "dimension": int(dim),
            "normalized": True,
            "query_prefix": "query: ",
            "passage_prefix": "passage: ",
            "corpus_version": corpus.get("version", ""),
            "note": (
                "Gerado por scripts/embed-corpus.py. Os hashes text_sha256 sao "
                "conferidos pelo teste Go: se o corpus mudar sem os embeddings "
                "serem regerados, o teste falha em vez de medir vetores obsoletos."
            ),
        },
        "documents": encode_entries(
            [doc["id"] for doc in documents], passages, passage_vectors
        ),
        "queries": encode_entries(
            [query["id"] for query in queries],
            [query["text"] for query in queries],
            query_vectors,
        ),
    }

    args.out.parent.mkdir(parents=True, exist_ok=True)
    # sort_keys + indent para que o diff do git seja legivel e a regeracao
    # produza sempre o mesmo arquivo byte-a-byte.
    args.out.write_text(
        json.dumps(artifact, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )

    print(
        f"embeddings: {len(documents)} documentos, {len(queries)} consultas, "
        f"dim={dim} -> {args.out}",
        file=sys.stderr,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
