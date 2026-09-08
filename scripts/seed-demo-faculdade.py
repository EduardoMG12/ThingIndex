#!/usr/bin/env python3
"""Semeia o corpus de demonstracao "faculdade" numa stack ThingIndex de pe.

Documentos publicos reais do IFPR (ver examples/faculdade/FONTES.md), enviados
pela API como um usuario normal faria: categoria + collection + arquivo. Serve
para validar de ponta a ponta que categorias, collections, indexacao com
bounding box e conexoes derivadas funcionam com documentos que nao foram
escritos para o teste.

Uso:
    ./scripts/seed-demo-faculdade.py
    THINGINDEX_API=http://localhost:8000/api/v1 ./scripts/seed-demo-faculdade.py

Sai com codigo 1 se algum documento terminar FAILED, ficar sem nenhuma bounding
box, ou se alguma collection nao tiver conexao entre documentos diferentes.
So stdlib + curl (para o multipart).
"""

import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from collections import defaultdict
from pathlib import Path

API = os.environ.get("THINGINDEX_API", "http://localhost:8000/api/v1")
EMAIL = os.environ.get("THINGINDEX_DEMO_EMAIL", "demo-faculdade@thingindex.local")
SENHA = os.environ.get("THINGINDEX_DEMO_PASSWORD", "DemoFaculdade#2026")
PASTA = Path(__file__).resolve().parent.parent / "examples" / "faculdade"

# (arquivo, titulo, categoria, collection). Duas collections com dois documentos
# e uma com PPC + calendario: cada uma precisa render pelo menos uma conexao
# entre documentos diferentes, senao o workspace nao tem o que desenhar.
MANIFESTO = [
    ("edital-56-2026-ps-2027-licenciatura.pdf", "Edital 56/2026 - PS 2027 Licenciatura", "Editais", "Processo Seletivo 2027"),
    ("edital-57-2026-ps-2027-tecnicos-subsequentes.pdf", "Edital 57/2026 - PS 2027 Tecnicos Subsequentes", "Editais", "Processo Seletivo 2027"),
    ("regimento-geral-ifpr.pdf", "Regimento Geral do IFPR", "Normas", "Regimento e resolucoes"),
    ("regimento-interno-comum-campi-res-08-2014.pdf", "Regimento Interno Comum aos Campi (Res. 08/2014)", "Normas", "Regimento e resolucoes"),
    ("ppc-sistemas-de-informacao-palmas-2019.pdf", "PPC Sistemas de Informacao - Palmas (2019)", "Cursos", "Sistemas de Informacao - Palmas"),
    ("calendario-academico-palmas-2026.pdf", "Calendario Academico Palmas 2026 - Graduacao", "Cursos", "Sistemas de Informacao - Palmas"),
    # Documentos que conversam com o PPC: a matriz lista as mesmas disciplinas
    # e as ementas detalham cada uma.
    ("matriz-curricular-si-palmas-2023.pdf", "Matriz Curricular SI - Palmas (2023)", "Cursos", "Sistemas de Informacao - Palmas"),
    ("ementas-si-palmas-ppc-2023.pdf", "Ementas SI - Palmas (PPC 2023)", "Cursos", "Sistemas de Informacao - Palmas"),
]


def chamar(metodo, caminho, token=None, corpo=None):
    dados = json.dumps(corpo).encode() if corpo is not None else None
    req = urllib.request.Request(f"{API}{caminho}", data=dados, method=metodo)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", f"Bearer {token}")
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            return resp.status, json.load(resp)
    except urllib.error.HTTPError as erro:
        try:
            return erro.code, json.load(erro)
        except Exception:
            return erro.code, {}


def autenticar():
    status, _ = chamar("POST", "/auth/register", corpo={"name": "Demo Faculdade", "email": EMAIL, "password": SENHA})
    if status not in (201, 400, 409):
        sys.exit(f"registro devolveu {status}; a stack esta de pe em {API}?")
    status, corpo = chamar("POST", "/auth/login", corpo={"email": EMAIL, "password": SENHA})
    token = (corpo.get("data") or {}).get("token")
    if status != 200 or not token:
        sys.exit(f"login devolveu {status}")
    return token


def enviar(token, arquivo, titulo, categoria, collection):
    saida = subprocess.run(
        [
            "curl", "-s", "-m", "120", "-X", "POST", f"{API}/documents/upload",
            "-H", f"Authorization: Bearer {token}",
            "-F", f"file=@{PASTA / arquivo}", "-F", f"title={titulo}",
            "-F", f"category={categoria}", "-F", f"collection={collection}",
        ],
        capture_output=True, text=True, check=True,
    )
    corpo = json.loads(saida.stdout)
    doc_id = (corpo.get("data") or {}).get("id")
    if not doc_id:
        sys.exit(f"upload de {arquivo} sem id: {saida.stdout[:200]}")
    return doc_id


def esperar_indexacao(token, ids, limite_s=1500):
    inicio = time.time()
    pendentes = dict(ids)
    while pendentes and time.time() - inicio < limite_s:
        for doc_id, arquivo in list(pendentes.items()):
            _, corpo = chamar("GET", f"/documents/{doc_id}", token)
            estado = (corpo.get("data") or {}).get("status")
            if estado == "INDEXED":
                print(f"  INDEXED  {arquivo} ({int(time.time() - inicio)}s)")
                del pendentes[doc_id]
            elif estado in ("FAILED", "ORPHANED"):
                sys.exit(f"  {estado}   {arquivo}: {(corpo.get('data') or {}).get('last_processing_error')}")
        time.sleep(5)
    if pendentes:
        sys.exit(f"tempo esgotado esperando: {list(pendentes.values())}")


def relatorio(token):
    status, corpo = chamar("GET", "/documents/collections", token)
    collections = corpo.get("data") or []
    if status != 200 or not isinstance(collections, list):
        sys.exit(f"listar collections devolveu {status}: {str(corpo)[:200]}")
    esperadas = {m[3] for m in MANIFESTO}
    encontradas = {c.get("name") for c in collections} & esperadas
    if encontradas != esperadas:
        sys.exit(f"collections ausentes: {sorted(esperadas - encontradas)}")
    falhas = []
    for col in collections:
        nome = col.get("name")
        if nome not in {m[3] for m in MANIFESTO}:
            continue
        _, ctx = chamar("GET", f"/documents/collections/{col['id']}/context", token)
        dados = ctx.get("data") or {}
        docs = {d["id"]: d.get("title") for d in dados.get("documents", [])}
        nos = dados.get("paragraph_nodes", [])
        sumario = dados.get("outline_nodes", [])
        por_doc = defaultdict(lambda: {"paragrafos": 0, "com_box": 0, "paginas": 0})
        for no in nos:
            item = por_doc[no["document_id"]]
            item["paragrafos"] += 1
            item["com_box"] += 1 if (no.get("metadata") or {}).get("box") else 0
            item["paginas"] = max(item["paginas"], no.get("page_number") or 0)

        print(f"\n[{col.get('category_name') or col.get('category', {}).get('name', '?')}] {nome}")
        print(f"  {'documento':<52} {'pags':>4} {'parag':>6} {'c/box':>6}")
        for doc_id, titulo in docs.items():
            item = por_doc[doc_id]
            print(f"  {titulo[:52]:<52} {item['paginas']:>4} {item['paragrafos']:>6} {item['com_box']:>6}")
            if item["com_box"] == 0:
                falhas.append(f"{titulo}: nenhum paragrafo com box")

        # Uma conexao entre documentos diferentes por collection e o minimo
        # para o workspace ter o que desenhar. Testa o primeiro no de sumario
        # de nivel mais alto de cada documento.
        conexoes_cruzadas = 0
        testados = 0
        for doc_id in docs:
            candidatos = [n for n in sumario if n.get("document_id") == doc_id]
            candidatos.sort(key=lambda n: (n.get("level", 9), n.get("sequence_order", 0)))
            for no in candidatos[:5]:
                testados += 1
                _, res = chamar("GET", f"/documents/collections/{col['id']}/connections?node_id={urllib.parse.quote(no['node_id'])}", token)
                for c in (res.get("data") or {}).get("connections", []):
                    if c.get("document_id") != doc_id and c.get("score", 0) >= 2:
                        conexoes_cruzadas += 1
        print(f"  conexoes entre documentos (score >= 2), {testados} nos testados: {conexoes_cruzadas}")
        if conexoes_cruzadas == 0:
            falhas.append(f"{nome}: nenhuma conexao entre documentos diferentes")
    return falhas


def main():
    for arquivo, *_ in MANIFESTO:
        if not (PASTA / arquivo).exists():
            sys.exit(f"faltando {PASTA / arquivo}; veja examples/faculdade/FONTES.md")
    token = autenticar()
    print(f"usuario {EMAIL} autenticado em {API}")
    # --relatorio: so mede o que ja esta no banco (util depois de um retry).
    if "--relatorio" not in sys.argv:
        # Idempotente: o que ja esta na conta (mesmo titulo) nao sobe de novo.
        _, lista = chamar("GET", "/documents", token)
        dados = lista.get("data") or []
        if isinstance(dados, dict):
            dados = dados.get("documents") or dados.get("items") or []
        existentes = {d.get("title") for d in dados if isinstance(d, dict)}
        ids = {}
        for arquivo, titulo, categoria, collection in MANIFESTO:
            if titulo in existentes:
                print(f"  ja existe {arquivo}")
                continue
            ids[enviar(token, arquivo, titulo, categoria, collection)] = arquivo
            print(f"  enviado  {arquivo} -> {categoria} / {collection}")
        print("aguardando indexacao...")
        esperar_indexacao(token, ids)
    falhas = relatorio(token)
    if falhas:
        print("\nFALHOU:")
        for f in falhas:
            print(f"  - {f}")
        sys.exit(1)
    print("\nOK: corpus da faculdade indexado com box e conexoes em todas as collections.")


if __name__ == "__main__":
    main()
