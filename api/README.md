# API

O backend do ThingIndex e uma API HTTP em Go responsavel por autenticacao, upload, catalogacao documental, indexacao assincrona, configuracao de providers, recuperacao de contexto relacional e chat grounded por collection.

Ele e o centro operacional do monorepo: recebe uploads, persiste metadados, publica eventos de indexacao, processa enriquecimento opcional com LLM e expõe telemetria documental para o frontend.

## Componentes

O diretório `api/` contém quatro executaveis principais:

- `cmd/api`: servidor HTTP principal;
- `cmd/indexer`: worker de indexacao e dispatcher da outbox;
- `cmd/pricing-worker`: worker de sincronizacao de precos de modelos.
- `cmd/index-import`: CLI para importar exports versionados de indice em banco externo.

## Responsabilidades do backend

- cadastro, login, reset de senha e emissao de JWT;
- CRUD de categorias e collections;
- upload de documentos e persistencia de arquivos em MinIO/S3-compatible ou Google Cloud Storage nativo;
- indexacao estrutural assincrona com RabbitMQ;
- enriquecimento opcional com LLM;
- recuperacao de contexto consolidado por collection;
- chat por collection com conversas persistidas e resposta incremental via SSE;
- exposicao de indice, chunks, monitoramento e score heuristico por documento;
- export versionado de indice por documento para reuso externo;
- agregacao de monitoramento de indexacao por usuario;
- armazenamento criptografado de credenciais de providers.

## Stack

- Go 1.25+
- Gin
- GORM
- PostgreSQL 16
- RabbitMQ
- MinIO
- Google Cloud Storage nativo no runtime e emulator local para integracao

## Estrutura relevante

```text
api/
|-- cmd/
|   |-- api/               # servidor HTTP
|   |-- indexer/           # worker de indexacao + outbox dispatcher
|   |-- pricing-worker/    # sincronizacao de precos
|   |-- index-import/      # importacao de exports de indice em banco externo
|-- db/                    # conexao e bootstrap de schema
|-- docs/                  # swagger gerado
|-- internal/
|   |-- api/               # middlewares e setup de rotas
|   |-- config/            # envs e validacao
|   |-- domain/            # contratos e erros
|   |-- dto/               # requests/responses
|   |-- handlers/          # borda HTTP
|   |-- models/            # modelos GORM
|   |-- processing/        # pipeline de processamento/indexacao
|   |-- queue/             # RabbitMQ publisher/consumer
|   |-- repository/        # persistencia
|   |-- usecase/           # regra de negocio
```

## Fluxo operacional

### Upload e indexacao

1. o usuario autentica e envia um documento;
2. a API valida tamanho, MIME e metadados obrigatorios;
3. o arquivo e persistido no storage configurado;
4. o documento e salvo no banco com status `QUEUED`;
5. um evento e gravado na outbox;
6. o dispatcher publica o evento no RabbitMQ;
7. o `indexer` consome a mensagem, processa o documento e persiste:
   - paginas;
   - paragrafos;
   - tabelas;
   - metadata pairs;
   - chunks;
   - outline nodes;
   - telemetria de pipeline e LLM.

### Pricing sync

1. fontes de preco habilitadas sao consultadas periodicamente;
2. o worker faz parse dos dados;
3. os precos atuais e o historico sao atualizados no banco.

## Dependencias externas

Para funcionamento completo, o backend depende de:

- PostgreSQL;
- MinIO ou storage S3-compatible;
- RabbitMQ;
- SMTP para `forgot-password`;
- provider de LLM configurado por usuario, quando houver enriquecimento ou chat runtime.

## Como executar

### Servidor HTTP

```bash
cd api
go run ./cmd/api
```

### Worker de indexacao

```bash
cd api
go run ./cmd/indexer
```

### Worker de precos

```bash
cd api
go run ./cmd/pricing-worker
```

### CLI de importacao de indices

```bash
cd api
go run ./cmd/index-import --database-url "postgres://postgres:postgres@localhost:5434/thingindex?sslmode=disable" --path ../tmp/index-exports
```

Esse fluxo:

- aceita um arquivo `.json` exportado ou um diretório com varios exports;
- valida o contrato `document_index_export` versionado;
- cria automaticamente o schema `thingindex_import_*` no banco alvo;
- sincroniza documentos por `source_key`, atualizando somente quando o payload muda.

## Consumo open source da API

Para explorar a API sem passar primeiro pela interface web:

- Scalar API reference: `http://localhost:8000/docs`
- Swagger UI fallback: `http://localhost:8000/swagger/index.html`
- Swagger JSON: `http://localhost:8000/swagger/doc.json`
- Runtime profile: `http://localhost:8000/api/v1/meta/runtime-profile`
- Indexing benchmark: `http://localhost:8000/api/v1/meta/indexing-benchmark`
- OpenAPI versionada no repo: `docs/swagger.yaml` e `docs/swagger.json`

O endpoint `/docs` e a referencia interativa primaria com Scalar. Quando o embed nao carregar, a propria pagina aponta para o Swagger UI local, para o JSON bruto e para os exemplos versionados do repositório.

Se a API estiver publicada, os mesmos paths continuam válidos na origin da deployment:

- `https://<sua-origin>/docs`
- `https://<sua-origin>/swagger/index.html`
- `https://<sua-origin>/swagger/doc.json`
- `https://<sua-origin>/api/v1/meta/runtime-profile`
- `https://<sua-origin>/api/v1/meta/indexing-benchmark`
- `https://<sua-origin>/api/v1`

O arquivo `../examples/http/thingindex-api.http` tambem deixa um fluxo pronto com:

- healthcheck;
- leitura do perfil publico do runtime;
- leitura do benchmark publico de indexacao;
- abertura das docs;
- registro;
- login;
- leitura de collections, documentos e runtime de providers.

Quando o alvo for uma deployment já online, basta trocar `@apiHost` no topo desse arquivo.

Para regenerar a referencia versionada depois de alterar annotations:

```bash
make docs-api
```

## Variaveis relevantes

As configuracoes completas estao documentadas no README raiz, mas as mais importantes para o backend sao:

| Variavel | Descricao |
| --- | --- |
| `APP_ENV` | Ambiente de execucao |
| `PORT` | Porta HTTP da API |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` | Conexao com PostgreSQL |
| `JWT_SECRET` | Assinatura de JWT |
| `MASTER_ENCRYPTION_KEY` | Chave mestre para criptografia de providers |
| `STORAGE_PROVIDER` | Provider de storage padrao do runtime (`MINIO` ou `GCP_STORAGE`) |
| `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET_NAME` | Bootstrap do storage MinIO/S3-compatible |
| `GCP_STORAGE_PROJECT_ID`, `GCP_STORAGE_BUCKET_NAME`, `GCP_STORAGE_CREDENTIALS_JSON`, `GCP_STORAGE_EMULATOR_HOST` | Bootstrap do storage GCS nativo |
| `RABBITMQ_URL` | Broker para indexacao assincrona |
| `DOCUMENT_INDEX_*` | Exchanges, filas e retries da indexacao |
| `OUTBOX_POLL_INTERVAL_SECONDS`, `OUTBOX_BATCH_SIZE` | Parametros do dispatcher |
| `PRICING_SYNC_INTERVAL_SECONDS` | Intervalo do pricing worker |
| `RESET_PASSWORD_URL`, `SMTP_*` | Fluxo de reset de senha |
| `ALLOW_PRIVATE_STORAGE_ENDPOINTS` | Override de desenvolvimento para storage privado |

## Principais rotas

### Publicas

- `GET /api/v1/meta/runtime-profile`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/forgot-password`
- `POST /api/v1/auth/reset-password`

### Providers

- `POST /api/v1/user/providers`
- `POST /api/v1/user/providers/test`
- `POST /api/v1/user/providers/models`
- `GET /api/v1/user/providers`
- `GET /api/v1/user/providers/runtime`
- `GET /api/v1/user/providers/audit`
- `PATCH /api/v1/user/providers/:providerType`

### Documentos

- `POST /api/v1/documents/upload`
- `GET /api/v1/documents/audit`
- `GET /api/v1/documents/`
- `GET /api/v1/documents/:id`
- `GET /api/v1/documents/:id/content`
- `GET /api/v1/documents/:id/index`
- `GET /api/v1/documents/:id/index/export`
- `GET /api/v1/documents/monitoring/summary`
- `GET /api/v1/documents/collections/:id/context`

### Chat

- `POST /api/v1/chat/conversations`
- `GET /api/v1/chat/conversations`
- `GET /api/v1/chat/conversations/:id`
- `PATCH /api/v1/chat/conversations/:id`
- `DELETE /api/v1/chat/conversations/:id`
- `POST /api/v1/chat/stream`

## Seguranca

O backend ja incorpora algumas medidas relevantes:

- JWT assinado com expiracao;
- bcrypt para senha;
- tokens de reset hashados;
- revogacao de sessoes apos reset de senha;
- criptografia AES-GCM para credenciais de provider;
- validacao de upload com limite de corpo e checagem de MIME;
- bloqueio de endpoints privados para storage por padrao;
- validacao de `base_url` customizada para providers de LLM;
- sanitizacao de erros na borda HTTP;
- rate limiting em auth;
- restricao de preview HTML inline para evitar execucao de conteudo ativo.

## Observabilidade

O backend expõe monitoramento em dois niveis:

- por documento:
  - status;
  - etapa;
  - duracao;
  - provider/modelo;
  - requests LLM;
  - tokens estimados;
  - custo estimado;
  - score heuristico de qualidade;
- agregado por usuario:
  - totais de documentos por status;
  - requests e tokens consolidados;
  - custo total estimado;
  - duracao media;
  - score medio;
  - breakdown por modelo.

Esse resumo agregado, exposto em `GET /api/v1/documents/monitoring/summary` e documentado no Swagger gerado do projeto, alimenta o painel operacional do frontend para destacar anomalias, filas travadas e falhas recentes de forma acionavel.

Tambem expõe trilhas de auditoria consultaveis para documentos e providers, com:

- busca textual;
- filtro por acao, status, actor e request id;
- recorte temporal em RFC3339;
- paginação por cursor;
- resumo agregado com breakdown por acao e status.

Na frente de providers, o backend tambem expõe:

- matriz pública de suporte real por storage em `GET /api/v1/providers/support?family=STORAGE`;
- matriz pública de suporte real por provider LLM em `GET /api/v1/providers/support?family=LLM`;
- teste de readiness com retorno estruturado por check;
- discovery de modelos para Ollama e providers OpenAI-compatible suportados no runtime;
- overview do runtime ativo para storage, indexacao e chat.

## Testes e verificacoes

Comandos simples a partir da raiz do monorepo:

```bash
./scripts/test.sh backend-unit
./scripts/test.sh backend-integration
./scripts/test.sh backend-e2e
./scripts/test.sh backend-smoke
```

Para regenerar o Swagger/Scalar depois de alterar annotations:

```bash
make docs-api
```

Ou, se preferir:

```bash
make test-backend
make test-backend-integration
make test-backend-e2e
make test-backend-smoke
```

Comandos diretos dentro de `api/`:

```bash
cd api
go test ./...
```

Se o ambiente tiver restricao de escrita no cache padrao do Go, use:

```bash
cd api
env GOCACHE=/tmp/go-build go test ./...
```

Integracao real do backend com PostgreSQL, MinIO, GCS emulator e RabbitMQ:

```bash
cd api
env GOCACHE=/tmp/go-build go test -tags=integration ./...
```

Esses testes usam `THINGINDEX_TEST_DB_*`, `THINGINDEX_TEST_MINIO_*`, `THINGINDEX_TEST_GCP_*` e `THINGINDEX_TEST_RABBITMQ_URL` quando definidos. Sem isso, tentam reutilizar a configuracao do `.env` raiz usado pelo `docker compose` local.

Integracao opcional com provider LLM real e processamento documental com IA:

```bash
cd api
env GOCACHE=/tmp/go-build go test -tags=integration ./internal/pkg/provider ./internal/processing
```

Esses testes fazem `skip` por padrao. Para habilitar rodada real, configure no `.env`:

- `THINGINDEX_TEST_LLM_ENABLED=true`
- `THINGINDEX_TEST_LLM_PROVIDER=OPENAI` ou `OLLAMA`/outro provider suportado
- `THINGINDEX_TEST_LLM_API_KEY`, `THINGINDEX_TEST_LLM_BASE_URL`, `THINGINDEX_TEST_LLM_MODEL` conforme o provider
- `THINGINDEX_TEST_LLM_RUN_PROCESSING=true` para exercitar o `DocumentProcessor` com enriquecimento real
- `THINGINDEX_TEST_LLM_RUN_MODEL_DISCOVERY=true` para exercitar listagem de modelos quando o provider suportar discovery no runtime atual, incluindo bridges OpenAI-compatible e Ollama

Os testes tambem aceitam fallbacks por provider, por exemplo `OPENAI_API_KEY`/`OPENAI_MODEL`, `GROK_API_KEY`/`GROK_MODEL` ou `OLLAMA_BASE_URL`/`OLLAMA_MODEL`, e nunca imprimem o segredo em assertions ou docs.

Esse smoke real nao roda no `pre-commit`: ele exige Postgres de pe e chamadas pagas ao provider. Rode sob demanda com `./scripts/test.sh backend-real-llm`.

`e2e` autenticado do backend cobrindo upload, retry, exclusao, documento orfao, providers e chat grounded:

```bash
cd api
env GOCACHE=/tmp/go-build go test -tags=e2e ./internal/api -run RoutesE2E
```

Smoke HTTP das rotas protegidas com middleware real:

```bash
cd api
env GOCACHE=/tmp/go-build go test -tags=smoke ./internal/handlers -run RouteSmoke
```

## Limitacoes atuais

- a telemetria de tokens e custo pode ser estimada, dependendo do provider;
- o score de qualidade da indexacao e heuristico e deve ser interpretado junto do benchmark publico em `GET /api/v1/meta/indexing-benchmark`;
- o chat runtime existe, mas depende de provider LLM configurado pelo usuario e ainda nao substitui validacao objetiva de acuracia/citacao para uso de producao;
- a infraestrutura de rate limit atual e simples e local ao processo;
- o runtime de storage agora cobre MinIO/S3-compatible e Google Cloud Storage nativo; os demais providers cloud seguem o caminho compartilhado S3-compatible e devem ser validados no deployment alvo.

## Relacao com o README raiz

Este arquivo documenta o backend de forma isolada. Para:

- arquitetura geral;
- escopo academico do TCC;
- frontend;
- docker compose completo;
- roadmap do projeto,

consulte o [README da raiz](../README.md).
