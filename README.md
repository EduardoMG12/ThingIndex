<div align="center">
  <img src="assets/palmas-horizontal-1.png" alt="IFPR Campus Palmas" width="300" />
</div>

<div align="center">
  <h1>ThingIndex</h1>
  <p><strong>Relational document intelligence platform for contextual organization, structured extraction, and recoverable knowledge.</strong></p>
  <p>Trabalho de Conclusao de Curso (TCC) - Sistemas de Informacao - IFPR Campus Palmas</p>
  <p>Autor: Charles Eduardo Mello Guimaraes</p>
</div>

<div align="center">
  <img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white" alt="Go badge" />
  <img src="https://img.shields.io/badge/Next.js-16-000000?logo=next.js&logoColor=white" alt="Next.js badge" />
  <img src="https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql&logoColor=white" alt="PostgreSQL badge" />
  <img src="https://img.shields.io/badge/MinIO-S3%20Compatible-C72E49" alt="MinIO badge" />
  <img src="https://img.shields.io/badge/Architecture-Relational%20Context-2F855A" alt="Architecture badge" />
  <img src="https://img.shields.io/badge/Status-Research%20Prototype-orange" alt="Status badge" />
</div>

## English Summary

ThingIndex is an academic and engineering project focused on document ingestion, categorization, structured extraction, and context reconstruction over a **pure relational architecture**. The current backend does **not** depend on RAG, vector databases, or graph databases to organize context. Instead, it uses PostgreSQL, hierarchical joins, and typed structured-data tables to keep documents, collections, categories, and extracted content connected in a traceable and performant way.

Although the broader research line still explores GraphRAG and legal knowledge systems, the code currently available in this repository is centered on a **document catalog with relational context recovery**, JWT authentication, i18n, secure provider credential storage, MinIO-compatible object storage, asynchronous indexing with RabbitMQ, and a Next.js interface.

## Sumario

- [1. Visao Geral](#1-visao-geral)
- [2. Escopo Atual](#2-escopo-atual)
- [3. Problema e Motivacao](#3-problema-e-motivacao)
- [4. Arquitetura](#4-arquitetura)
- [5. Modelo Relacional de Contexto](#5-modelo-relacional-de-contexto)
- [6. Funcionalidades Implementadas](#6-funcionalidades-implementadas)
- [7. Estrutura do Monorepo](#7-estrutura-do-monorepo)
- [8. Stack Tecnologica](#8-stack-tecnologica)
- [9. Como Executar com Docker Compose](#9-como-executar-com-docker-compose)
- [10. Como Executar Manualmente](#10-como-executar-manualmente)
- [11. Variaveis de Ambiente](#11-variaveis-de-ambiente)
- [12. Visao Geral da API](#12-visao-geral-da-api)
- [13. Fluxo de Uso Recomendado](#13-fluxo-de-uso-recomendado)
- [14. Seguranca e Boas Praticas](#14-seguranca-e-boas-praticas)
- [15. Qualidade de Engenharia](#15-qualidade-de-engenharia)
- [16. Contexto Academico do TCC](#16-contexto-academico-do-tcc)
- [17. Limitacoes Atuais](#17-limitacoes-atuais)
- [18. Roadmap](#18-roadmap)
- [19. Contribuicao](#19-contribuicao)
- [20. Licenca](#20-licenca)

## 1. Visao Geral

O **ThingIndex** e uma plataforma para ingestao e organizacao de documentos com foco em:

- catalogacao por usuario;
- agrupamento logico por `collections`;
- categorizacao opcional por `categories`;
- armazenamento de arquivos em object storage compativel com S3;
- persistencia de estruturas extraidas em tabelas relacionais tipadas;
- reconstrucao de contexto com joins previsiveis e agregacao linear;
- camada HTTP com autenticacao, i18n e respostas padronizadas.

O projeto nasceu em contexto academico, mas foi estruturado com preocupacoes reais de engenharia: separacao de responsabilidades, criptografia de credenciais, fail-fast de configuracao, scoping por usuario, sanitizacao de erros, validacao de upload, e mitigacoes de abuso e SSRF.

## 2. Escopo Atual

### O que o projeto e hoje

- Uma aplicacao full-stack com backend em Go e frontend em Next.js.
- Um catalogo relacional de documentos com `categories`, `collections`, `documents`, `paragraph_nodes`, `table_sections` e `metadata_pairs`.
- Um backend que reconstrui contexto de uma collection por consultas hierarquicas ao PostgreSQL.
- Um painel web para autenticacao, upload e navegacao do catalogo.
- Uma base pronta para evoluir para pipelines de processamento documental e assistentes baseados em IA.

### O que o projeto nao e hoje

- Nao e um sistema de GraphRAG em execucao.
- Nao depende de banco de grafos para recuperar contexto.
- Ja possui pipeline assincrono funcional com outbox, RabbitMQ e worker dedicado para indexacao estrutural.
- Ja possui enriquecimento opcional com LLM para indexacao, incluindo telemetria estimada de tokens, custo e qualidade heuristica por documento.
- Nao e um assistente juridico/QA pronto para producao; o chat runtime atual e documental, grounded por collection e ainda experimental no escopo do TCC.
- O runtime agora suporta **MinIO/S3-compatible** e **Google Cloud Storage nativo**, com matriz pública de suporte para separar caminhos prontos, compartilhados e incrementais.

### Importante para quem vier de documentacao antiga

Partes do material de pesquisa do repositorio ainda discutem Neo4j, GraphRAG e comparacoes com abordagens vetoriais. Isso faz parte da **linha de pesquisa e do roadmap**, mas **na implementacao atual do backend o contexto e reconstruido estritamente por banco relacional**.

### Foco do produto hoje

O foco declarado do runtime atual e:

- organizar collections documentais auditaveis para times operacionais, juridicos, academicos e de compliance;
- preservar contexto, estrutura e proveniencia no banco relacional;
- manter comparacoes com vetores e GraphRAG como trilhas academicas separadas do caminho funcional principal.

Os artefatos versionados que registram esse posicionamento agora sao:

- `docs/article/artigo.tex`
- `docs/estado-do-projeto.md`
- `frontend/architecture/relational-context-architecture.md`
- `api/internal/usecase/testdata/indexing-benchmark/corpus.json`

## 3. Problema e Motivacao

Sistemas documentais costumam falhar em um dos dois extremos:

- armazenam arquivos sem preservar bem o contexto entre eles;
- ou terceirizam quase toda a inteligencia de recuperacao para embeddings, grafos ou pipelines externos.

O ThingIndex parte de uma premissa diferente:

> Se a estrutura semantica minima estiver bem modelada no banco relacional, entao grande parte da organizacao, rastreabilidade e recuperacao de contexto pode ser feita com SQL, chaves estrangeiras, ordenacao explicita e agregacao linear no backend.

Essa decisao traz vantagens importantes:

- rastreabilidade deterministica;
- menor acoplamento com componentes experimentais;
- facilidade de auditoria;
- previsibilidade de custo e operacao;
- clareza arquitetural para fins academicos e manutencao futura.

## 4. Arquitetura

```mermaid
flowchart TD
    UI[Next.js Frontend] -->|HTTP JSON / multipart| API[Go API - Gin]
    API -->|ORM + SQL| PG[(PostgreSQL)]
    API -->|Object Storage| MINIO[(MinIO / S3-compatible)]
    API -->|Password reset email| SMTP[SMTP]

    API --> AUTH[JWT + Bcrypt]
    API --> I18N[i18n PT/EN]
    API --> SEC[Credential Encryption]
    API --> RATE[Rate Limiting]
    API --> VALID[Validation + Safe Error Handling]

    PG --> CAT[categories]
    PG --> COL[collections]
    PG --> DOC[documents]
    PG --> PAR[paragraph_nodes]
    PG --> TAB[table_sections]
    PG --> META[metadata_pairs]

    API --> OUTBOX[outbox events]
    OUTBOX --> RABBIT[(RabbitMQ)]
    RABBIT --> INDEXER[indexer worker]
    INDEXER --> PG
    INDEXER --> MINIO
```

### Principios arquiteturais adotados

- **Camadas desacopladas**: `handlers -> usecases -> repositories -> models`.
- **Dominio como fronteira**: erros de negocio e contratos vivem em `api/internal/domain`.
- **Repositorio para persistencia**: regras de consulta e joins nao ficam espalhadas em handlers.
- **Borda HTTP fina**: validacao, auth, i18n e serializacao ficam nos handlers; regra de negocio fica no use case.
- **Relational first**: o contexto do dominio e preservado por FK, `sequence_order` e consultas hierarquicas.
- **Agregacao O(n)**: o backend evita `N+1` e monta resposta percorrendo o conjunto retornado uma unica vez por tabela.

## 5. Modelo Relacional de Contexto

```mermaid
erDiagram
    USER ||--o{ DOCUMENT_CATEGORY : owns
    USER ||--o{ COLLECTION : owns
    USER ||--o{ DOCUMENT : owns
    USER ||--o{ USER_PROVIDER_CONFIG : owns

    DOCUMENT_CATEGORY ||--o{ COLLECTION : groups
    COLLECTION ||--o{ DOCUMENT : contains
    DOCUMENT ||--o{ PARAGRAPH_NODE : extracts
    DOCUMENT ||--o{ TABLE_SECTION : extracts
    DOCUMENT ||--o{ METADATA_PAIR : extracts
```

### Entidades principais

| Entidade | Papel |
| --- | --- |
| `document_categories` | Filtro macro opcional de navegacao por usuario |
| `collections` | Vinculo logico entre varios documentos que pertencem ao mesmo contexto |
| `documents` | Arquivos brutos e seus metadados de armazenamento |
| `paragraph_nodes` | Paragrafos ou blocos textuais ordenados por `sequence_order` |
| `table_sections` | Tabelas estruturadas ordenadas e rastreaveis |
| `metadata_pairs` | Metadados tipados em pares chave/valor por documento |

### Regras importantes do modelo

- Todo `document` precisa pertencer a uma `collection`.
- Uma `collection` pode ou nao pertencer a uma `category`.
- A ordem do conteudo e explicita no banco via `sequence_order`.
- O backend pode reconstruir o contexto completo de uma collection em numero constante de queries.

### Estrategia de recuperacao

Para obter o contexto consolidado de uma collection, o backend:

1. carrega a collection pelo `id` e `user_id`;
2. busca todos os documentos daquela collection;
3. extrai os `document_ids`;
4. busca `paragraph_nodes`, `table_sections` e `metadata_pairs` por lote;
5. agrega os resultados em memoria por `document_id`.

Essa abordagem evita consultas por documento e mantem a montagem da resposta em **tempo linear sobre o numero de linhas retornadas**.

## 6. Funcionalidades Implementadas

### Backend

- Cadastro, login e reset de senha com JWT.
- Revogacao de sessoes antigas apos reset de senha.
- Upload de documentos com validacao de tamanho e MIME.
- Persistencia de uploads em MinIO/S3-compatible e Google Cloud Storage nativo.
- CRUD parcial de categorias e collections.
- Persistencia de documentos com vinculacao obrigatoria a `collection`.
- Persistencia de estruturas extraidas em tabelas tipadas.
- Pipeline assincrono com outbox, RabbitMQ e worker de indexacao.
- Endpoint de indice por documento com `tree_retrieval`, `index_evaluation` e `monitoring`.
- Resumo agregado de monitoramento por usuario com custo estimado, tokens, requests, duracao media, score medio e breakdown por modelo.
- Recuperacao de contexto consolidado de uma collection.
- Fluxo de chat por collection com conversas persistidas, selecao de fontes e resposta incremental por SSE.
- Internacionalizacao de mensagens HTTP em portugues e ingles.
- Criptografia de credenciais de providers por usuario.
- Provider de storage customizado com resolucao segura de endpoint.
- Validacao de `base_url` customizada para providers de LLM, com bloqueio de endpoints privados fora de overrides de desenvolvimento.

### Frontend

- Landing page institucional.
- Dashboard para autenticacao e operacao do catalogo.
- Painel de documentos com preview lateral, telemetria de indexacao por documento e resumo agregado de monitoramento.
- Tela de chat com historico de conversas, referencias de fontes e streaming incremental de resposta grounded por collection.
- Documentacao visual da proposta e da arquitetura.
- Configuracao de providers de storage e LLM com papeis separados para indexacao e chat.

### Infraestrutura local

- PostgreSQL para persistencia transacional.
- MinIO para object storage.
- RabbitMQ para orquestracao assincrona da indexacao.
- Docker Compose para reproducao do ambiente.
- Worker de indexacao e pricing worker como processos separados.

## 7. Estrutura do Monorepo

```text
.
|-- api/
|   |-- cmd/api/                 # ponto de entrada da API
|   |-- db/                      # conexao, bootstrap e introspecao de schema
|   |-- internal/
|   |   |-- api/                 # roteamento e middlewares
|   |   |-- config/              # carregamento e validacao de envs
|   |   |-- domain/              # contratos e erros de negocio
|   |   |-- dto/                 # requests e responses
|   |   |-- handlers/            # camada HTTP
|   |   |-- mailer/              # envio de email
|   |   |-- models/              # modelos GORM
|   |   |-- pkg/                 # crypto, netsec, storage, provider, i18n
|   |   |-- repository/          # persistencia
|   |   |-- usecase/             # regra de negocio
|   |-- docs/                    # swagger gerado
|-- frontend/
|   |-- src/app/                 # rotas Next.js
|   |-- src/components/          # UI
|   |-- architecture/            # documentos de arquitetura e modelagem
|-- docs/
|   |-- roadmap/                 # caminhos de evolucao academica
|-- docker-compose.yml
|-- .env.example
```

## 8. Stack Tecnologica

| Camada | Tecnologia | Papel no projeto |
| --- | --- | --- |
| Backend | Go 1.25 + Gin | API HTTP, auth, upload, catalogo documental |
| Persistencia | PostgreSQL 16 | Modelo relacional e recuperacao de contexto |
| ORM | GORM | Mapeamento e queries |
| Storage | MinIO | Armazenamento de arquivos |
| Frontend | Next.js 16 + React 19 | Interface web |
| Validacao | go-playground/validator | Validacao de payload |
| Auth | JWT + bcrypt | Sessao stateless e hash de senha |
| i18n | dicionario interno PT/EN | Mensagens da borda HTTP |
| Docs | Swagger + Scalar | Referencia de API |
| Containers | Docker Compose | Ambiente local reproduzivel |

## 9. Como Executar com Docker Compose

### Pre-requisitos

- Docker
- Docker Compose v2

### Passo a passo

**Modo produção-like (build completo, sem hot-reload):**

```bash
./scripts/bootstrap.sh
./scripts/stack.sh up
./scripts/doctor.sh
```

**Modo desenvolvimento (hot-reload em tudo — api, indexer e frontend):**

```bash
./scripts/bootstrap.sh
./scripts/stack.sh dev
./scripts/doctor.sh
```

No modo `dev`:
- `api` e `indexer` usam `air` para reload automático do Go;
- `frontend` reflete mudanças instantaneamente via Next.js dev server;
- o script monta o cache Go do host para evitar downloads repetidos dentro do container;
- os serviços só sobem depois que `db`, `minio` e `rabbitmq` estiverem realmente prontos (healthchecks).

Se preferir `make`:

```bash
# Produção-like
make bootstrap
make stack-up
make doctor

# Desenvolvimento com hot-reload
make bootstrap
make dev
make doctor
```

O guia curto de onboarding open source tambem ficou centralizado em `docs/open-source-onboarding.md`.

### Endpoints locais

| Servico | URL |
| --- | --- |
| Frontend | `http://localhost:3000` |
| API | `http://localhost:8000` |
| API docs | `http://localhost:8000/docs` |
| Swagger JSON | `http://localhost:8000/swagger/doc.json` |
| MinIO API | `http://localhost:9000` |
| MinIO Console | `http://localhost:9001` |
| GCS Emulator | `http://localhost:4443` |
| PostgreSQL | `localhost:5434` |
| RabbitMQ Management | `http://localhost:15672` |

### Observacoes importantes

- O `docker-compose.yml` sobe RabbitMQ e suporta o pipeline assincrono de indexacao.
- O modo `dev` combina `docker-compose.yml` com `docker-compose.dev.yml` e ativa hot reload para **api, indexer e frontend**.
- Servicos dependentes (`api`, `indexer`, `frontend`) aguardam `db`, `minio` e `rabbitmq` ficarem saudaveis antes de iniciar, eliminando race conditions no boot.
- O modo `dev` resolve e monta o `GOMODCACHE` e o `GOCACHE` do host para evitar depender de download de modulos no boot da API.
- O fluxo de `forgot-password` exige um servidor SMTP acessivel. O compose atual nao sobe Mailpit/Mailhog por padrao.
- O frontend consome a API por `NEXT_PUBLIC_API_URL`.
- Para requests prontas de API, use `examples/http/thingindex-api.http`.

## 10. Como Executar Manualmente

### Opcao A: subir apenas a infraestrutura com Docker e rodar app localmente

```bash
./scripts/bootstrap.sh
./scripts/stack.sh core
```

Depois, em terminais separados:

```bash
cd api
go run ./cmd/api
```

```bash
cd frontend
pnpm install
pnpm dev
```

### Ajustes necessarios para execucao manual no host

Se voce rodar a API fora do container, normalmente vai querer ajustar no `.env`:

```bash
PORT=8000
DB_HOST=localhost
DB_PORT=5434
MINIO_ENDPOINT=localhost:9000
API_INTERNAL_URL=http://localhost:8000/api/v1
NEXT_PUBLIC_API_URL=http://localhost:8000/api/v1
```

### Pre-requisitos para execucao manual

- Go `1.25+`
- Node.js `20+`
- `pnpm`
- PostgreSQL
- MinIO, outro endpoint S3-compatible ou Google Cloud Storage

### Opcao B: abrir no Dev Container

O repositorio agora inclui configuracao em `.devcontainer/devcontainer.json` para abrir o monorepo com Go, Node.js 20 e `pnpm`, reaproveitando o `docker-compose` existente.

Ao abrir o projeto no Dev Container:

- `db`, `minio`, `rabbitmq`, `api` e `indexer` sobem pelo compose;
- o `postCreateCommand` executa `./scripts/bootstrap.sh all` e instala o binario `swag` em `api/`;
- o frontend pode ser iniciado de dentro do container com `cd frontend && pnpm dev`;
- o Storybook do frontend pode ser iniciado no container com `cd frontend && pnpm storybook`, com a porta `6006` ja encaminhada;
- a instalacao do browser do Playwright continua explicita via `cd frontend && pnpm run test:e2e:install`.
- os testes reais opcionais com Ollama do host podem usar `OLLAMA_BASE_URL=http://host.docker.internal:11434/v1`, porque o devcontainer ja expõe `host.docker.internal`.

### Hooks locais

O repositorio agora possui `pre-commit` em `.husky/pre-commit`.

Ele executa checks seletivos antes do commit:

- para mudancas em `api/`: `gofmt` nos arquivos Go staged, depois `go vet ./...` e `go test ./...`;
- para mudancas relevantes em `frontend/`: `pnpm lint`, `pnpm run typecheck` e `pnpm run test:unit`.

O hook foi pensado para proteger integridade basica do codigo sem tentar rodar `integration` ou `e2e` a cada commit.

O `pre-commit` roda apenas o que e rapido e offline. O smoke real de indexacao com LLM exige Postgres de pe e chamadas pagas ao provider, entao fica sob demanda:

```bash
./scripts/test.sh backend-real-llm
```

### Comandos simples de teste

O repositorio agora expõe comandos curtos na raiz:

```bash
./scripts/bootstrap.sh
./scripts/stack.sh up
./scripts/doctor.sh
./scripts/test.sh unit
./scripts/test.sh backend-integration
./scripts/test.sh backend-real-llm
./scripts/test.sh backend-e2e
./scripts/test.sh frontend-storybook
./scripts/test.sh frontend-e2e
./scripts/test.sh all
```

Se preferir `make`, os atalhos equivalentes tambem existem:

```bash
make bootstrap
make doctor
make stack-up
make stack-dev
make test-unit
make test-backend-integration
make test-backend-e2e
make test-frontend-storybook
make test-frontend-e2e
make test-all
```

Para instalar o browser gerenciado pelo Playwright manualmente:

```bash
make test-frontend-e2e-install
```

## Open Source Quickstart

O projeto agora assume explicitamente dois modos de entrada.

### Self-hosted a partir do GitHub

Se a ideia for subir o stack completo localmente:

1. rode `./scripts/bootstrap.sh`;
2. suba a stack com `./scripts/stack.sh up`;
3. confirme o ambiente com `./scripts/doctor.sh`;
4. abra `http://localhost:3000`;
5. use `http://localhost:8000/docs` ou `http://localhost:8000/swagger/index.html` para a API.

### Consumir uma API ja publicada

Se a API ja estiver online e voce nao precisar subir o monorepo:

1. abra `https://<sua-origin>/docs`;
2. use `https://<sua-origin>/api/v1` como base HTTP;
3. use `https://<sua-origin>/swagger/index.html` e `https://<sua-origin>/swagger/doc.json` como fallbacks;
4. se tiver clonado o repo, altere `@apiHost` em `examples/http/thingindex-api.http`;
5. opcionalmente valide a deployment com `./scripts/doctor.sh --api-base https://<sua-origin>`.

Materiais de apoio diretos:

- guia curto: `docs/open-source-onboarding.md`;
- exemplos de request: `examples/http/thingindex-api.http`;
- contrato OpenAPI: `api/docs/swagger.yaml` e `api/docs/swagger.json`.

## 11. Variaveis de Ambiente

O arquivo `.env.example` foi pensado para **desenvolvimento local**. Em producao, a API faz validacoes adicionais e rejeita configuracoes inseguras.

### Variaveis principais

| Variavel | Exemplo local | Descricao |
| --- | --- | --- |
| `APP_ENV` | `development` | Ambiente de execucao |
| `PORT` | `8080` | Porta HTTP da API |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000,http://127.0.0.1:3000` | Origins permitidas |
| `DB_HOST` | `localhost` ou `db` | Host do PostgreSQL |
| `DB_PORT` | `5434` ou `5432` | Porta do PostgreSQL |
| `DB_USER` | `postgres` | Usuario da API no banco |
| `DB_PASSWORD` | `postgres` | Senha da API no banco |
| `DB_NAME` | `thingindex` | Nome do banco |
| `DB_SSLMODE` | `disable` | SSL mode do driver Postgres |
| `JWT_SECRET` | valor local forte | Chave de assinatura JWT |
| `MASTER_ENCRYPTION_KEY` | valor local forte | Chave mestre para criptografia de credenciais |
| `STORAGE_PROVIDER` | `MINIO` | Provider de storage padrão do runtime (`MINIO` ou `GCP_STORAGE`) |
| `MINIO_ENDPOINT` | `minio:9000` ou `localhost:9000` | Endpoint do storage |
| `MINIO_ACCESS_KEY` | `thingindex-local-access-key` | Access key do storage padrao |
| `MINIO_SECRET_KEY` | `thingindex-local-secret-key` | Secret key do storage padrao |
| `MINIO_USE_SSL` | `false` | HTTPS no storage padrao |
| `MINIO_BUCKET_NAME` | `thingindex-docs` | Bucket utilizado pela aplicacao |
| `GCP_STORAGE_PROJECT_ID` | `thingindex-dev` | Projeto usado no bootstrap nativo de GCS |
| `GCP_STORAGE_BUCKET_NAME` | `thingindex-docs` | Bucket usado no bootstrap nativo de GCS |
| `GCP_STORAGE_LOCATION` | `us-central1` | Location opcional do bucket GCS |
| `GCP_STORAGE_CREDENTIALS_JSON` | vazio | JSON de service account para buckets GCS reais |
| `GCP_STORAGE_EMULATOR_HOST` | vazio ou `127.0.0.1:4443` | Host do emulator GCS para desenvolvimento |
| `THINGINDEX_TEST_DB_*` | vazio ou overrides locais | Overrides opcionais para testes de integracao/e2e do backend |
| `THINGINDEX_TEST_MINIO_*` | vazio ou overrides locais | Overrides opcionais para testes reais com storage |
| `THINGINDEX_TEST_GCP_*` | vazio ou overrides locais | Overrides opcionais para testes de integracao do GCS |
| `THINGINDEX_TEST_RABBITMQ_URL` | vazio ou override local | Override opcional para os testes reais com fila |
| `THINGINDEX_TEST_LLM_*` | desabilitado por padrao | Habilita testes reais de provider/processing com IA |
| `OPENAI_*`, `OPENROUTER_*`, `DEEPSEEK_*`, `GROK_*`, `GEMINI_*`, `CLAUDE_*`, `OLLAMA_*` | provider-specific | Fallbacks de credenciais/modelo para os testes reais de LLM |
| `MAX_UPLOAD_SIZE_BYTES` | `20971520` | Limite maximo de upload |
| `RESET_PASSWORD_URL` | `http://localhost:3000/reset-password` | URL base usada no email de reset |
| `SMTP_SECURITY_MODE` | `none` em dev | `none`, `starttls` ou `tls` |
| `SMTP_HOST` | `localhost` | Host do SMTP |
| `SMTP_PORT` | `1025` | Porta do SMTP |
| `SMTP_USER` | vazio | Usuario SMTP |
| `SMTP_PASS` | vazio | Senha SMTP |
| `SMTP_FROM` | `no-reply@thingindex.com` | Remetente padrao |
| `AUTH_RATE_LIMIT_REQUESTS` | `10` | Limite de requests de auth por janela |
| `AUTH_RATE_LIMIT_WINDOW_SECONDS` | `60` | Janela do rate limit |
| `ALLOW_PRIVATE_STORAGE_ENDPOINTS` | `false` | Override de desenvolvimento para endpoints privados |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | URL de conexao do RabbitMQ |
| `DOCUMENT_INDEX_EXCHANGE` | `thingindex.document.index` | Exchange usada na indexacao assincrona |
| `DOCUMENT_INDEX_QUEUE` | `thingindex.document.index.queue` | Fila principal de indexacao |
| `DOCUMENT_INDEX_ROUTING_KEY` | `document.index.requested` | Routing key da fila principal |
| `DOCUMENT_INDEX_RETRY_QUEUE` | `thingindex.document.index.retry` | Fila de retry da indexacao |
| `DOCUMENT_INDEX_DLQ` | `thingindex.document.index.dlq` | Dead-letter queue da indexacao |
| `OUTBOX_POLL_INTERVAL_SECONDS` | `5` | Intervalo de polling do dispatcher da outbox |
| `OUTBOX_BATCH_SIZE` | `20` | Quantidade maxima de eventos por lote |
| `CONSUMER_PREFETCH_COUNT` | `10` | Prefetch do consumer de indexacao |
| `DOCUMENT_INDEX_RETRY_DELAY_SECONDS` | `30` | Delay entre tentativas de reprocessamento |
| `DOCUMENT_INDEX_MAX_RETRIES` | `5` | Numero maximo de tentativas de indexacao |
| `PRICING_SYNC_INTERVAL_SECONDS` | `86400` | Intervalo do worker de sincronizacao de precos |

### Variaveis de frontend

| Variavel | Exemplo | Descricao |
| --- | --- | --- |
| `NEXT_PUBLIC_API_URL` | `http://localhost:8000/api/v1` | URL publica consumida pelo navegador |
| `API_INTERNAL_URL` | `http://api:8080/api/v1` | URL interna para ambiente conteinerizado |

### Variaveis de compose

| Variavel | Uso |
| --- | --- |
| `POSTGRES_USER` | Usuario do container Postgres |
| `POSTGRES_PASSWORD` | Senha do container Postgres |
| `POSTGRES_DB` | Banco inicial do container |
| `RABBITMQ_DEFAULT_USER` | Usuario do RabbitMQ |
| `RABBITMQ_DEFAULT_PASS` | Senha do RabbitMQ |

## 12. Visao Geral da API

### Envelope de resposta

Todas as respostas seguem, em geral, o formato:

```json
{
  "status": 200,
  "type": "success",
  "message": "Documents retrieved successfully",
  "data": {}
}
```

Erros de validacao retornam `errors` com mensagens sanitizadas e localizadas.

### Rotas publicas

| Metodo | Rota | Descricao |
| --- | --- | --- |
| `GET` | `/api/v1/health` | Health check simples |
| `GET` | `/api/v1/meta/runtime-profile` | Perfil publico do foco atual do produto, limites declarados e referencias academicas |
| `GET` | `/api/v1/providers/support` | Matriz publica do suporte real aos providers LLM |
| `POST` | `/api/v1/auth/register` | Cadastro de usuario |
| `POST` | `/api/v1/auth/login` | Login e emissao de JWT |
| `POST` | `/api/v1/auth/forgot-password` | Solicita reset de senha |
| `POST` | `/api/v1/auth/reset-password` | Conclui reset de senha |

### Rotas autenticadas de provider

| Metodo | Rota | Descricao |
| --- | --- | --- |
| `POST` | `/api/v1/user/providers` | Salva credenciais por usuario |
| `POST` | `/api/v1/user/providers/test` | Executa readiness do provider com checks estruturados |
| `POST` | `/api/v1/user/providers/models` | Descobre modelos disponiveis para providers LLM suportados |
| `GET` | `/api/v1/user/providers` | Lista configuracoes de provider do usuario |
| `GET` | `/api/v1/user/providers/runtime` | Resume os providers ativos para storage, indexacao e chat |
| `GET` | `/api/v1/user/providers/audit` | Consulta auditoria de providers com filtros, resumo e cursor |
| `PATCH` | `/api/v1/user/providers/:providerType` | Ativa/desativa provider ou papel LLM |

### Rotas autenticadas de documentos

| Metodo | Rota | Descricao |
| --- | --- | --- |
| `POST` | `/api/v1/documents/upload` | Upload de arquivo com `multipart/form-data` |
| `GET` | `/api/v1/documents/audit` | Consulta auditoria documental com filtros, resumo e cursor |
| `GET` | `/api/v1/documents/` | Lista documentos do usuario |
| `GET` | `/api/v1/documents/:id` | Busca documento por ID |
| `GET` | `/api/v1/documents/:id/content` | Recupera o arquivo bruto do documento |
| `GET` | `/api/v1/documents/:id/index` | Recupera indice, chunks e monitoramento do documento |
| `GET` | `/api/v1/documents/monitoring/summary` | Resume uso, custo e qualidade da indexacao do usuario |
| `GET` | `/api/v1/documents/category/:id` | Lista documentos por categoria |
| `POST` | `/api/v1/documents/categories` | Cria categoria |
| `GET` | `/api/v1/documents/categories` | Lista categorias |
| `PATCH` | `/api/v1/documents/categories/:id` | Atualiza categoria |
| `POST` | `/api/v1/documents/collections` | Cria collection |
| `GET` | `/api/v1/documents/collections` | Lista collections |
| `GET` | `/api/v1/documents/collections/:id` | Busca collection |
| `GET` | `/api/v1/documents/collections/:id/context` | Recupera contexto consolidado da collection |
| `PATCH` | `/api/v1/documents/collections/:collection/category` | Atualiza categoria da collection |
| `POST` | `/api/v1/documents/:id/paragraph-nodes` | Persiste paragrafos estruturados |
| `POST` | `/api/v1/documents/:id/table-sections` | Persiste tabelas estruturadas |
| `POST` | `/api/v1/documents/:id/metadata-pairs` | Persiste pares chave/valor |

### Upload

Campos aceitos em `multipart/form-data`:

| Campo | Obrigatorio | Observacao |
| --- | --- | --- |
| `title` | sim | Titulo do documento |
| `file` | sim | Arquivo bruto |
| `collection` | sim | Collection obrigatoria |
| `category` | nao | Categoria opcional |

Tipos de arquivo aceitos hoje:

- `application/pdf`
- `text/html`
- `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`

Limite padrao de upload:

- `20 MB`

### Documentacao interativa

- Scalar UI primária: `http://localhost:8000/docs`
- Swagger UI fallback: `http://localhost:8000/swagger/index.html`
- Swagger JSON: `http://localhost:8000/swagger/doc.json`
- Perfil publico do runtime: `http://localhost:8000/api/v1/meta/runtime-profile`
- Matriz publica de storage: `http://localhost:8000/api/v1/providers/support?family=STORAGE`
- Matriz publica de LLM: `http://localhost:8000/api/v1/providers/support?family=LLM`
- Contrato versionado no repositório: `api/docs/swagger.yaml` e `api/docs/swagger.json`

Em deployments publicadas, mantenha os mesmos paths na origin publicada:

- `https://<sua-origin>/docs`
- `https://<sua-origin>/swagger/index.html`
- `https://<sua-origin>/swagger/doc.json`
- `https://<sua-origin>/api/v1/meta/runtime-profile`
- `https://<sua-origin>/api/v1/providers/support?family=STORAGE`
- `https://<sua-origin>/api/v1/providers/support?family=LLM`

Use as matrizes públicas de providers para distinguir:

- o que está pronto para storage nativo ou S3-compatible;
- o que está pronto para o runtime principal;
- o que usa o caminho compartilhado OpenAI-compatible;
- o que ainda depende de bridge experimental com `base_url` e `model` explícitos.

Para regenerar a referencia da API depois de alterar annotations:

```bash
make docs-api
```

## 13. Fluxo de Uso Recomendado

Todos os exemplos abaixo funcionam tanto localmente quanto em uma deployment publicada.
Troque `http://localhost:8000` pela origin publicada quando estiver consumindo a API online.

### 1. Registrar usuario

```bash
curl -X POST http://localhost:8000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Charles",
    "email": "charles@example.com",
    "password": "123456"
  }'
```

### 2. Fazer login

```bash
curl -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "charles@example.com",
    "password": "123456"
  }'
```

Guarde o token retornado:

```bash
export TOKEN="seu-jwt-aqui"
```

### 3. Criar uma collection

```bash
curl -X POST http://localhost:8000/api/v1/documents/collections \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Caso Silva vs Estado",
    "category": "Processos",
    "metadata": {
      "origin": "manual",
      "area": "juridico"
    }
  }'
```

### 4. Enviar um documento

```bash
curl -X POST http://localhost:8000/api/v1/documents/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "title=Peticao Inicial" \
  -F "collection=Caso Silva vs Estado" \
  -F "category=Processos" \
  -F "file=@./exemplos/peticao.pdf"
```

### 5. Persistir dados estruturados do documento

```bash
curl -X POST http://localhost:8000/api/v1/documents/1/paragraph-nodes \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": [
      {
        "sequence_order": 0,
        "page_number": 1,
        "content": "Conteudo do primeiro paragrafo",
        "metadata": {
          "source": "ocr"
        }
      }
    ]
  }'
```

### 6. Recuperar o contexto consolidado da collection

```bash
curl -X GET http://localhost:8000/api/v1/documents/collections/1/context \
  -H "Authorization: Bearer $TOKEN"
```

## 14. Seguranca e Boas Praticas

### Ja implementado

- **JWT assinado** para autenticacao stateless.
- **bcrypt** para hash de senhas.
- **Tokens de reset hashados** no banco.
- **Revogacao de sessao apos reset de senha** para invalidar JWTs anteriores.
- **Criptografia AES-GCM** para credenciais de provider, com chave derivada por usuario.
- **Rate limit** nas rotas de autenticacao.
- **CORS configuravel** sem wildcard com credenciais.
- **Sanitizacao de erros** para nao vazar detalhes internos nas respostas.
- **Validacao de uploads** com limite de corpo e checagem de MIME.
- **Mitigacao de SSRF** para storage customizado.
- **Validacao de `base_url` para LLM** com restricao de esquema e bloqueio de hosts privados.
- **Bloqueio de endpoints privados** por padrao para providers de storage.
- **Resolucao segura de hostname** no dial do storage para evitar bypass por DNS rebinding.
- **Preview HTML desativado** para evitar execucao de conteudo ativo enviado pelo usuario.
- **Validacoes de configuracao insegura** fora de ambiente de desenvolvimento.

### Decisoes de seguranca relevantes

- O projeto aceita `Accept-Language` e localiza mensagens na borda HTTP.
- O backend falha cedo se a configuracao essencial estiver invalida.
- As credenciais de provider nao sao devolvidas nas respostas da API.
- Em ambientes nao-dev, `RESET_PASSWORD_URL` precisa ser seguro e o transporte SMTP nao pode ficar em modo inseguro.

### Observacao de escopo

O projeto ainda esta em fase de desenvolvimento local. Isso significa que algumas decisoes de infraestrutura continuam orientadas a produtividade de dev, nao a hardening completo de producao distribuida.

Checklist versionada desta frente: `SECURITY.md`.

## 15. Qualidade de Engenharia

### Convencoes adotadas

- handlers finos;
- use cases com regra de negocio;
- repositorios focados em persistencia;
- DTOs separados de models;
- mensagens i18n por chave;
- validacao centralizada;
- erro de dominio sem vazamento de detalhe interno.

### Comandos uteis

Backend:

```bash
./scripts/test.sh backend-unit
./scripts/test.sh backend-integration
./scripts/test.sh backend-e2e
./scripts/test.sh backend-smoke
go vet ./...
```

Frontend:

```bash
pnpm --dir frontend run lint
pnpm --dir frontend run typecheck
./scripts/test.sh frontend-unit
./scripts/test.sh frontend-e2e
./scripts/test.sh frontend-smoke
```

### Observacao honesta

O projeto ja possui validacao estrutural, testes unitarios no frontend e backend, smoke `e2e` no frontend, smoke HTTP no backend, integracao real no backend com PostgreSQL, MinIO e RabbitMQ, cenarios opcionais com provider LLM real e uma bateria `e2e` autenticada do backend cobrindo upload, retry, documento orfao, exclusao, configuracao de provider e chat com stream grounded. A frente de testes deixou de ser uma lacuna estrutural; o trabalho restante aqui e expandir cenarios incrementalmente e manter a documentacao alinhada ao runtime atual.

## 16. Contexto Academico do TCC

Este repositorio representa uma etapa importante do TCC por alguns motivos:

- transforma uma ideia de organizacao documental em uma arquitetura executavel;
- demonstra uma alternativa relacional para preservacao de contexto;
- explicita trade-offs entre simplicidade operacional e flexibilidade futura;
- prepara terreno para comparacoes mais avancadas com pipelines baseados em grafos, embeddings e assistentes.

Em outras palavras, o valor academico aqui nao esta apenas no produto final, mas tambem na **decisao arquitetural defendida**:

> contexto pode ser preservado e reconstituido por modelagem relacional rigorosa, sem depender obrigatoriamente de RAG ou bancos de grafos desde o primeiro momento.

Os materiais em `docs/` e `frontend/architecture/` registram essa evolucao e tambem apontam caminhos futuros de pesquisa.

Os artefatos academicos versionados desta fase agora ficam em:

- `docs/article/artigo.tex` para o texto do artigo;
- `docs/article/references.bib` para as referencias bibliograficas;
- `docs/article/anexos/` para as figuras e anexos;
- `docs/article/Apresentação de TCC ThingIndex Slides.pdf` para os slides da defesa;
- `frontend/architecture/relational-context-architecture.md` para o registro da decisao arquitetural;
- `api/internal/usecase/testdata/indexing-benchmark/corpus.json` para o corpus controlado da baseline de indexacao.

## 17. Limitacoes Atuais

- O storage padrão do runtime continua simples por design, mas agora pode ser inicializado com `MINIO` ou `GCP_STORAGE`.
- O chat runtime existe com streaming incremental e grounding por collection, mas ainda nao substitui avaliacao juridica, benchmark de citacao ou validacao de acuracia para uso de producao.
- A estimativa de custo e tokens de LLM e heuristica quando o provider nao devolve usage real.
- O score de qualidade da indexacao e heuristico; ele agora deve ser lido junto do benchmark publico em `GET /api/v1/meta/indexing-benchmark`, nao como ground truth isolado.
- Nao ha arquivo `LICENSE` publicado ainda.
- Algumas docs antigas do repositorio ainda refletem a fase de pesquisa em grafos e precisam ser lidas com contexto.

## 18. Roadmap

### Curto prazo

- melhorar a operacionalizacao de retry, falhas e DLQ;
- alinhar toda a documentacao secundaria do monorepo;
- explicitar melhor o que e prototipo, funcional e roadmap;
- fechar checklist objetivo de seguranca e hardening.

### Medio prazo

- ampliar observabilidade e telemetria operacional;
- tornar auditoria mais consultavel;
- fechar i18n inconsistente;
- ampliar o pipeline assincrono de parsing e extracao;
- suporte real a mais providers de storage;
- expandir o benchmark de indexacao para corpus maiores e cenarios de citacao grounded.

### Longo prazo / pesquisa

- comparacoes com GraphRAG e RAG vetorial;
- assistente juridico/documental com rastreabilidade;
- benchmark de recuperacao e citacao;
- integracao com camadas de analise semantica mais avancadas.

## 19. Contribuicao

Contribuicoes sao bem-vindas, especialmente em:

- melhoria da documentacao;
- testes automatizados;
- observabilidade e DX;
- tratamento de documentos e extracao estruturada;
- refinamento do frontend;
- consolidacao do fluxo de pesquisa do TCC.

### Sugestao de fluxo

1. abra uma issue descrevendo problema ou proposta;
2. alinhe escopo antes de grandes refatoracoes;
3. envie PRs pequenos, com motivacao tecnica clara;
4. evite misturar mudancas de arquitetura, estilo e feature no mesmo PR.

## 20. Licenca

O projeto e distribuido sob a **Licenca MIT** — ver [`LICENSE`](LICENSE).

Isso permite uso, copia, modificacao e redistribuicao, inclusive comercial, desde que o aviso de copyright e a licenca sejam mantidos. A escolha favorece a citabilidade e o reuso academico do trabalho.

---

Se voce chegou ate aqui para avaliar o projeto como TCC, a recomendacao e observar tres aspectos em conjunto:

- a clareza da modelagem relacional;
- a preocupacao com seguranca e separacao de responsabilidades;
- a honestidade entre o que ja esta implementado e o que ainda pertence ao roadmap.
