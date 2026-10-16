#!/usr/bin/env python3
"""Auditoria dos acertos do corretor automatico (falsos positivos).

Para cada resposta marcada correct=true nos CSV que o artigo usa, refaz a
correcao com o gabarito versionado (a mesma de api/cmd/generation-eval/
metrics.go: texto dobrado em caixa e acento, todas as answer_patterns casam e
nenhuma reject_pattern casa) e mostra, para cada padrao, que alternativa casou,
onde e dentro de que palavra.

Imprime tres listas:

  1. inventario: toda alternativa que casou como pedaco de uma palavra maior,
     com a palavra inteira e o numero de respostas. E o material para julgar
     se um radical ("revogad", "prev") pegou a palavra certa.
  2. falsos positivos por substring: respostas certas em que algum padrao so
     casa dentro de outra palavra -- o casamento comeca no meio da palavra
     ("trata" em "contrata") ou o radical pegou uma palavra de outra familia
     (FALSOS_AMIGOS: "prev" em "preventivos").
  3. respostas certas com marca de recusa ou negacao ("nao detalha", "nao ha
     informacao"...), para leitura humana: varias sao certas (a premissa
     falsa corrige a premissa negando), e o script nao decide sozinho.

Uso: python3 scripts/auditoria-padroes.py [--gabarito-dir DIR]
Sai com codigo 1 se a lista 2 nao estiver vazia.
"""
from __future__ import annotations

import argparse
import csv
import json
import re
import sys
from collections import Counter
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
RES = RAIZ / "docs" / "article" / "results"

# CSV usados pelo artigo e o gabarito de cada um.
TEMAS = ("seguranca-trabalho", "produto", "licitacoes")
ARQUIVOS = [(f"expansao/{a}-{t}.csv", t) for t in TEMAS for a in ("off", "on")]
ARQUIVOS += [(f"enriquecimento-reexecucao/braco-seguranca-trabalho-{b}.csv", "seguranca-trabalho")
             for b in ("content", "content-summary", "content-questions", "max")]
ARQUIVOS += [
    ("generation-results-T-ordem.csv", "si"),
    ("generation-results-F-document.csv", "si"),
    ("generation-results-F-document-gpt-oss-120b.csv", "si"),
    ("generation-results-T-licitacoes.csv", "legislacao"),
    ("bateria-prefeitura.csv", "prefeitura"),
]

# Mesma dobra de retrieval.FoldDiacritics + strings.ToLower.
_DOBRA = str.maketrans("áàâãäéèêëíìîïóòôõöúùûüçñ", "aaaaaeeeeiiiiooooouuuucn")


def dobra(s: str) -> str:
    return s.lower().translate(_DOBRA)


# Radicais que casam com palavra de outra familia: a alternativa foi escrita
# para um verbo/termo e pega outro. Cada entrada: alternativa -> regex da
# palavra inteira que NAO e o que a alternativa quer dizer.
FALSOS_AMIGOS = {
    "prev": r"preven\w*",          # prever/previsto, nao prevencao/preventivo
    "trata": r"tratament\w*",      # tratar de, nao tratamento
    "consta": r"constat\w*",       # constar, nao constatar
    "sim": r"sim(pl|ul|il)\w*",    # "sim", nao simplificacao/simultaneo/similar
    "exclu": r"exclusiv\w*",       # excluir, nao exclusivo
}

# Alternativas que so carregam a polaridade da resposta.
POLAR = r"nao|sim|nao precisa"

MARCA_RECUSA = re.compile(
    r"nao (ha|existe|consta|detalham?|especificam?|mencionam?|trazem?|contem|informam?|abordam?|tratam?|preveem|preve|foi possivel|e possivel|encontr|esta"
    r"|afirm\w*|declar\w*|deixam? claro|esclarecem?|fornecem?|apresentam?|indicam?|citam?|definem?|estabelecem?)"
    r"|nenhuma informac|sem informac|nao (sao|e|foi|foram|esta|estao) (informad|mencionad|especificad|previst|detalhad|encontrad)"
    r"|nada consta|nenhum (dos|documento|trecho)",
    re.A,
)


def alternativas(p: str) -> list[str]:
    """Alternativas de nivel zero de uma regex (a|b(c|d)|e -> a, b(c|d), e)."""
    out, cur, prof, i = [], "", 0, 0
    while i < len(p):
        c = p[i]
        if c == "\\":
            cur += p[i:i + 2]
            i += 2
            continue
        if c == "(":
            prof += 1
        elif c == ")":
            prof -= 1
        if c == "|" and prof == 0:
            out.append(cur)
            cur = ""
        else:
            cur += c
        i += 1
    out.append(cur)
    return out


def correta(resp: str, q: dict) -> bool:
    f = dobra(resp)
    pats = q["answer_patterns"]
    if not pats or not all(re.search(p, f, re.A) for p in pats):
        return False
    return not any(re.search(p, f, re.A) for p in q.get("reject_patterns") or [])


def casamentos(texto: str, padrao: str) -> list[dict]:
    """Cada casamento de cada alternativa, com a palavra que o contem."""
    out = []
    for alt in alternativas(padrao):
        for m in re.finditer(alt, texto, re.A):
            s, e = m.span()
            if s == e:
                continue
            ini = s
            while ini > 0 and texto[ini - 1].isalnum():
                ini -= 1
            fim = e
            while fim < len(texto) and texto[fim].isalnum():
                fim += 1
            palavra = texto[ini:fim]
            # comeca no meio de uma palavra; zeros a esquerda de um numero
            # ("04º" para "4º") nao contam.
            meio = ini < s and not (texto[s].isdigit() and set(texto[ini:s]) == {"0"})
            amigo = FALSOS_AMIGOS.get(alt)
            falso = meio or bool(amigo and re.fullmatch(amigo, palavra.split()[0] if palavra.split() else palavra, re.A))
            out.append({"alt": alt, "ini": s, "trecho": texto[s:e], "palavra": palavra,
                        "dentro": (ini, fim) != (s, e), "suspeito": falso,
                        "contexto": texto[max(0, s - 60):e + 60].replace("\n", " ")})
    return out


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--gabarito-dir", default=str(RAIZ / "api" / "internal" / "evalcorpus" / "testdata"))
    args = ap.parse_args()
    gdir = Path(args.gabarito_dir)

    inventario: Counter = Counter()
    falsos, recusas, polaridade, divergentes = [], [], [], []
    for arq, tema in ARQUIVOS:
        g = {q["id"]: q for q in json.loads((gdir / f"gabarito-{tema}.json").read_text(encoding="utf-8"))["questions"]}
        for r in csv.DictReader(open(RES / arq, encoding="utf-8")):
            q = g[r["question_id"]]
            if (r["correct"] == "true") != (r["error"] == "" and correta(r["answer"], q)):
                divergentes.append((arq, r["question_id"], r["repeat"], r.get("arm", "")))
            if r["correct"] != "true":
                continue
            texto = dobra(r["answer"])
            chave = (arq, r["question_id"], r.get("arm", ""), r["repeat"])
            so_suspeito = []
            for p in q["answer_patterns"]:
                ms = casamentos(texto, p)
                for m in ms:
                    if m["dentro"]:
                        inventario[(tema, q["id"], m["alt"], m["palavra"], m["suspeito"])] += 1
                if ms and all(m["suspeito"] for m in ms):
                    so_suspeito.append((p, ms))
            if so_suspeito:
                falsos.append((chave, q, r["answer"], so_suspeito))
            # Padrao satisfeito so por "sim"/"nao" que nao abre a resposta: o
            # "nao" pode negar o documento ("o texto nao afirma...") e nao a
            # pergunta.
            for p in q["answer_patterns"]:
                ms = casamentos(texto, p)
                if ms and q.get("targets") and all(re.fullmatch(POLAR, m["trecho"]) for m in ms) \
                        and not re.match(r"\W*(nao|sim)\b", texto):
                    polaridade.append((chave, q, r["answer"], p, ms))
            if not q.get("targets") == [] and MARCA_RECUSA.search(texto):
                ev = [(p, [m for m in casamentos(texto, p)][:3]) for p in q["answer_patterns"]]
                recusas.append((chave, q, r["answer"], ev))

    print("=" * 78)
    print("1. INVENTARIO: alternativa que casou dentro de palavra maior (respostas certas)")
    print("=" * 78)
    for (tema, qid, alt, palavra, susp), n in sorted(inventario.items()):
        print(f"{'SUSPEITO' if susp else 'ok      '} {tema:18} {qid:6} {alt[:40]!r:44} -> {palavra[:40]!r} x{n}")

    print()
    print("=" * 78)
    print("2. FALSOS POSITIVOS POR SUBSTRING (algum padrao so casa dentro de outra palavra)")
    print("=" * 78)
    for (arq, qid, arm, rep), q, resp, lista in falsos:
        print(f"\n[{arq}] {qid} braco {arm} repeticao {rep} ({q['stratum']})")
        print(f"  pergunta: {q['text']}")
        for p, ms in lista:
            print(f"  padrao {p!r}")
            for m in ms:
                print(f"    casou {m['alt']!r} em {m['palavra']!r}: ...{m['contexto']}...")
        print(f"  resposta: {resp[:400]!r}")
    print(f"\nTotal: {len(falsos)} respostas")

    print()
    print("=" * 78)
    print("3. RESPOSTAS CERTAS COM MARCA DE RECUSA OU NEGACAO (leitura humana)")
    print("=" * 78)
    for (arq, qid, arm, rep), q, resp, ev in recusas:
        print(f"\n[{arq}] {qid} braco {arm} rep {rep} ({q['stratum']}) -- {q['text']}")
        for p, ms in ev:
            print(f"  {p[:60]!r}: " + "; ".join(f"{m['alt'][:25]!r}->{m['palavra'][:30]!r}" for m in ms))
        print(f"  resposta: {resp[:500]!r}")
    print(f"\nTotal: {len(recusas)} respostas")

    print()
    print("=" * 78)
    print("4. ACERTO SO PELA PALAVRA 'SIM'/'NAO' FORA DA ABERTURA (leitura humana)")
    print("=" * 78)
    for (arq, qid, arm, rep), q, resp, p, ms in polaridade:
        print(f"\n[{arq}] {qid} braco {arm} rep {rep} ({q['stratum']}) -- {q['text']}")
        print(f"  esperado: {q['expected']}")
        print(f"  padrao {p!r}: " + " | ".join(f"...{m['contexto']}..." for m in ms[:3]))
    print(f"\nTotal: {len(polaridade)} respostas")

    if divergentes:
        print(f"\nAVISO: {len(divergentes)} linhas em que correct do CSV difere do gabarito atual: {divergentes[:10]}")
    return 1 if falsos else 0


if __name__ == "__main__":
    sys.exit(main())
