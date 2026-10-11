#!/usr/bin/env python3
"""Gera docs/article/anexos/diagramas/esquema-proveniencia.puml a partir do
DBML versionado (docs/schema/thingindex.dbml).

A figura mostra o caminho da proveniencia: usuario -> colecao -> documento ->
as seis tabelas estruturais, com as colunas que dao a coordenada de origem
(pagina, ordem, caixa). As demais tabelas ficam fora da figura; o esquema
completo esta no Anexo A9. Gerar a partir do DBML impede a
figura de divergir do banco, que foi o que aconteceu com o diagrama de classes
antigo (16 classes contra 25 tabelas).
"""
from __future__ import annotations

import re
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
DBML = RAIZ / "docs" / "schema" / "thingindex.dbml"
SAIDA = RAIZ / "docs" / "article" / "anexos" / "diagramas" / "esquema-proveniencia.puml"

NUCLEO = ["users", "collections", "documents"]
ESTRUTURA = ["document_pages", "paragraph_nodes", "table_sections", "document_outline_nodes", "document_chunks", "metadata_pairs"]
# Tabelas da estrutura desenhadas na segunda linha, abaixo das demais.
SEGUNDA_LINHA = {"document_outline_nodes", "document_chunks", "metadata_pairs"}
# Colunas mostradas por tabela: identidade, vinculo e coordenada de origem.
COLUNAS = {
    "users": ["id", "email"],
    "collections": ["id", "user_id", "name"],
    "documents": ["id", "user_id", "collection_id", "title", "status"],
    "document_pages": ["id", "document_id", "page_number", "content"],
    # metadata guarda a caixa e as linhas do paragrafo (JSONB); o artigo
    # declara isso como desvio da regra de tabela tipada.
    "paragraph_nodes": ["id", "document_id", "sequence_order", "page_number", "content", "metadata"],
    "table_sections": ["id", "document_id", "sequence_order", "page_number", "headers", "rows"],
    "document_outline_nodes": ["id", "document_id", "parent_id", "level", "title", "start_page", "end_page"],
    "document_chunks": ["id", "document_id", "sequence_order", "page_number", "content"],
    "metadata_pairs": ["id", "document_id", "key", "value"],
}


def tabelas(dbml: str) -> dict[str, dict[str, str]]:
    out: dict[str, dict[str, str]] = {}
    for nome, corpo in re.findall(r"^Table (\w+) \{\n(.*?)^\}", dbml, re.S | re.M):
        cols = {}
        for linha in corpo.splitlines():
            m = re.match(r"\s+(\w+) ([\w()]+)", linha)
            if m and m.group(1) != "indexes":
                cols[m.group(1)] = m.group(2)
        out[nome] = cols
    return out


def refs(dbml: str) -> list[tuple[str, str, str]]:
    return re.findall(r"^Ref: (\w+)\.(\w+) > (\w+)\.id", dbml, re.M)


def main() -> None:
    dbml = DBML.read_text(encoding="utf-8")
    t = tabelas(dbml)
    r = refs(dbml)
    faltando = [n for n in NUCLEO + ESTRUTURA if n not in t]
    if faltando:
        raise SystemExit(f"tabelas ausentes no DBML: {faltando}")
    for tabela, colunas in COLUNAS.items():
        for c in colunas:
            if c not in t[tabela]:
                raise SystemExit(f"coluna {tabela}.{c} nao existe no DBML")
    outras = sorted(set(t) - set(NUCLEO) - set(ESTRUTURA))

    linhas = [
        "@startuml",
        "' GERADO por scripts/gera-esquema-proveniencia.py a partir do DBML. Nao edite a mao.",
        "!include estilo.iuml",
        "skinparam defaultFontSize 13",
        # polyline: cada relacao e uma linha propria; com ortho as linhas que
        # descem de documents para a segunda linha se fundiam num mesmo trilho
        "skinparam linetype polyline",
        "hide circle",
    ]
    # FK so onde o banco declara a constraint; user_id sem constraint aparece
    # como coluna comum, para a figura nao afirmar mais do que o esquema tem.
    com_fk = {(filho, col) for filho, col, _pai in r}

    def entidade(nome: str, cor: str) -> None:
        linhas.append(f'entity "{nome}" as {nome}{(" " + cor) if cor else ""} {{')
        for c in COLUNAS[nome]:
            tipo = t[nome][c]
            marca = "<b>PK</b> " if c == "id" else ("<i>FK</i> " if (nome, c) in com_fk else "")
            # Tipo com parenteses (varchar(255)) faz o PlantUML ler a linha
            # como metodo; o campo explicito {field} impede isso.
            linhas.append(f"  {{field}} {marca}{c} : {tipo}")
        linhas.append("}")
    linhas.append('package "Núcleo" {')
    for n in NUCLEO:
        entidade(n, "")
    linhas.append("}")
    linhas.append('package "Estrutura extraída (coordenada de origem)" {')
    for n in ESTRUTURA:
        entidade(n, "")
    linhas.append("}")
    mostradas = set(NUCLEO + ESTRUTURA)
    for filho, _col, pai in r:
        if filho in mostradas and pai in mostradas:
            # nucleo em linha e a estrutura extraida em duas linhas abaixo dele,
            # para a figura ter a proporcao da pagina em paisagem
            if filho in NUCLEO and pai in NUCLEO:
                seta = "-right-"
            else:
                seta = "--"
            linhas.append(f"{pai} ||{seta}o{{ {filho}")
    # Posse por user_id: filtrada em toda consulta, sem constraint no banco.
    # Tracejado para distinguir de chave estrangeira declarada.
    linhas.append("users ||.right.o{ collections")
    linhas.append("users ||.right.o{ documents")
    # cada tabela da segunda linha fica abaixo de uma da primeira
    primeira = [n for n in ESTRUTURA if n not in SEGUNDA_LINHA]
    segunda = [n for n in ESTRUTURA if n in SEGUNDA_LINHA]
    for de_cima, de_baixo in zip(primeira, segunda):
        linhas.append(f"{de_cima} -[hidden]down- {de_baixo}")
    linhas.append("@enduml")
    SAIDA.write_text("\n".join(linhas) + "\n", encoding="utf-8")
    print(f"{SAIDA.relative_to(RAIZ)}: {len(t)} tabelas no DBML, {len(r)} FKs, {len(outras)} fora da figura")


if __name__ == "__main__":
    main()
