# Relational Context Architecture

## Objetivo

Substituir o modelo atual em que `documents.collection` e apenas uma `string` por uma estrutura relacional explicita, capaz de:

- preservar contexto entre varios arquivos sem RAG nem grafo;
- reconstruir uma collection inteira via joins previsiveis;
- manter rastreabilidade por `sequence_order`;
- evitar N+1 e montagem quadratica no backend Go.

## Schema alvo

```sql
create table categories (
    id bigserial primary key,
    user_id bigint not null references users(id) on delete cascade,
    name varchar(255) not null,
    normalized_name varchar(255) not null,
    description text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (user_id, normalized_name)
);

create table collections (
    id bigserial primary key,
    category_id bigint not null references categories(id) on delete cascade,
    user_id bigint not null references users(id) on delete cascade,
    name varchar(255) not null,
    normalized_name varchar(255) not null,
    metadata jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (user_id, normalized_name)
);

create table documents (
    id bigserial primary key,
    collection_id bigint not null references collections(id) on delete cascade,
    user_id bigint not null references users(id) on delete cascade,
    title varchar(255) not null,
    file_path varchar(1024) not null,
    mime_type varchar(100) not null,
    size_bytes bigint not null default 0,
    checksum varchar(128),
    status varchar(50) not null default 'PENDING',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table paragraph_nodes (
    id bigserial primary key,
    document_id bigint not null references documents(id) on delete cascade,
    sequence_order integer not null,
    page_number integer,
    content text not null,
    metadata jsonb not null default '{}'::jsonb,
    unique (document_id, sequence_order)
);

create table table_sections (
    id bigserial primary key,
    document_id bigint not null references documents(id) on delete cascade,
    sequence_order integer not null,
    title varchar(255),
    headers jsonb not null default '[]'::jsonb,
    rows jsonb not null default '[]'::jsonb,
    metadata jsonb not null default '{}'::jsonb,
    unique (document_id, sequence_order)
);

create table metadata_pairs (
    id bigserial primary key,
    document_id bigint not null references documents(id) on delete cascade,
    sequence_order integer not null,
    key varchar(255) not null,
    value text not null,
    unique (document_id, sequence_order, key)
);
```

## Indices minimos

```sql
create index idx_categories_user_name on categories (user_id, normalized_name);
create index idx_collections_user_name on collections (user_id, normalized_name);
create index idx_collections_category on collections (category_id, id);
create index idx_documents_collection on documents (collection_id, id);
create index idx_documents_user_collection on documents (user_id, collection_id);
create index idx_paragraph_nodes_document_order on paragraph_nodes (document_id, sequence_order);
create index idx_table_sections_document_order on table_sections (document_id, sequence_order);
create index idx_metadata_pairs_document_order on metadata_pairs (document_id, sequence_order);
create index idx_collections_metadata_gin on collections using gin (metadata);
```

## Contratos de rota

```text
POST   /api/v1/document-catalog/categories
GET    /api/v1/document-catalog/categories
POST   /api/v1/document-catalog/collections
GET    /api/v1/document-catalog/collections
GET    /api/v1/document-catalog/collections/:id
GET    /api/v1/document-catalog/collections/:id/context
POST   /api/v1/document-catalog/documents
GET    /api/v1/document-catalog/documents/:id
```

## Camadas Go recomendadas

### Models

- `Category`
- `Collection`
- `Document`
- `ParagraphNode`
- `TableSection`
- `MetadataPair`

### Repositories

- `CategoryRepository`
- `CollectionRepository`
- `DocumentRepository`
- `StructuredDataRepository`
- `CollectionContextRepository`

`CollectionContextRepository` deve encapsular a reconstrucao de contexto com numero constante de queries, sem espalhar joins por handlers.

### Use cases

- `CreateCategory`
- `CreateCollection`
- `UploadDocument`
- `StoreDocumentStructure`
- `GetCollectionContext`
- `ListCollections`

### Handlers

- apenas validacao HTTP, auth e serializacao;
- nada de regra de negocio ou montagem de contexto dentro do handler;
- todas as mensagens via chaves i18n, nunca literais hardcoded.

## Estrategia de recuperacao

Para montar o contexto de uma collection:

1. Buscar a collection por `id` e `user_id`.
2. Buscar todos os documentos da collection em uma unica query.
3. Extrair os `document_ids`.
4. Buscar `paragraph_nodes`, `table_sections` e `metadata_pairs` com `where document_id = any($1)`.
5. Montar mapas em memoria por `document_id` em uma unica passagem por tabela.

Exemplo de consulta base:

```sql
select d.id, d.title, d.file_path, d.mime_type, d.status
from documents d
where d.user_id = $1
  and d.collection_id = $2
order by d.id;

select p.*
from paragraph_nodes p
where p.document_id = any($1)
order by p.document_id, p.sequence_order;

select t.*
from table_sections t
where t.document_id = any($1)
order by t.document_id, t.sequence_order;

select m.*
from metadata_pairs m
where m.document_id = any($1)
order by m.document_id, m.sequence_order;
```

## Complexidade

O objetivo realista nao e transformar o banco inteiro em `O(n)` absoluto. O que deve ser `O(n)` e a montagem da resposta em funcao da quantidade de linhas retornadas:

- 1 query para collection;
- 1 query para documentos;
- 3 queries para estruturas tipadas;
- 3 passagens lineares para indexar os resultados em mapas por `document_id`.

Ou seja, o backend evita `N` queries por documento e faz agregacao em tempo linear sobre o conjunto carregado.

## Exemplo de montagem em Go

```go
docMap := make(map[uint]*CollectionDocumentContext, len(documents))
for i := range documents {
    doc := documents[i]
    docMap[doc.ID] = &CollectionDocumentContext{Document: doc}
}

for _, node := range paragraphNodes {
    docMap[node.DocumentID].ParagraphNodes = append(docMap[node.DocumentID].ParagraphNodes, node)
}

for _, section := range tableSections {
    docMap[section.DocumentID].TableSections = append(docMap[section.DocumentID].TableSections, section)
}

for _, pair := range metadataPairs {
    docMap[pair.DocumentID].MetadataPairs = append(docMap[pair.DocumentID].MetadataPairs, pair)
}
```

## i18n

- manter erros de dominio como codigos estaveis: `collection_not_found`, `category_duplicate`, `document_invalid_mime`;
- traduzir apenas na borda HTTP;
- aceitar `Accept-Language` e cair para `en` por padrao;
- DTOs nunca devem carregar mensagens localizadas, apenas dados.

## Migracao incremental da base atual

1. Criar `collections` e popular a partir de `documents.collection`.
2. Adicionar `collection_id` em `documents`.
3. Backfill em lote com `update documents set collection_id = ...`.
4. Trocar leitura e escrita da API para `collection_id`.
5. Remover a coluna legado `documents.collection` so depois de estabilizar os consumidores.

## Gap atual da base

Hoje a API Go ainda trata collection como `string` dentro de `documents`, o que:

- impede chave estrangeira para contexto;
- dificulta filtros de autorizacao e unicidade;
- obriga updates por nome de collection;
- nao oferece rota de reconstrucao hierarquica de contexto.

O frontend desta branch ja foi preparado para aceitar um payload de `collections` mais rico e compativel com o formato legado, reduzindo acoplamento para a refatoracao da API.
