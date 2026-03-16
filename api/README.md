# API

O backend do ThingIndex e uma API HTTP em Go responsavel por autenticacao, upload, catalogacao documental, indexacao assincrona, configuracao de providers e recuperacao de contexto relacional.

Ele e o centro operacional do monorepo: recebe uploads, persiste metadados, publica eventos de indexacao, processa enriquecimento opcional com LLM e expõe telemetria documental para o frontend.

## Componentes

O diretório `api/` contém tres executaveis principais:

- `cmd/api`: servidor HTTP principal;
- `cmd/indexer`: worker de indexacao e dispatcher da outbox;
- `cmd/pricing-worker`: worker de sincronizacao de precos de modelos.

## Responsabilidades do backend

- cadastro, login, reset de senha e emissao de JWT;
- CRUD de categorias e collections;
- upload de documentos e persistencia de arquivos em MinIO/S3-compatible;
- indexacao estrutural assincrona com RabbitMQ;
- enriquecimento opcional com LLM;
- recuperacao de contexto consolidado por collection;
- exposicao de indice, chunks, monitoramento e score heuristico por documento;
- agregacao de monitoramento de indexacao por usuario;
- armazenamento criptografado de credenciais de providers.

## Stack

- Go 1.25+
- Gin
- GORM
- PostgreSQL 16
- RabbitMQ
- MinIO

## Estrutura relevante

```text
api/
|-- cmd/
|   |-- api/               # servidor HTTP
|   |-- indexer/           # worker de indexacao + outbox dispatcher
|   |-- pricing-worker/    # sincronizacao de precos
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
- provider de LLM configurado por usuario, quando houver enriquecimento.

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

## Variaveis relevantes

As configuracoes completas estao documentadas no README raiz, mas as mais importantes para o backend sao:

| Variavel | Descricao |
| --- | --- |
| `APP_ENV` | Ambiente de execucao |
| `PORT` | Porta HTTP da API |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` | Conexao com PostgreSQL |
| `JWT_SECRET` | Assinatura de JWT |
| `MASTER_ENCRYPTION_KEY` | Chave mestre para criptografia de providers |
| `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET_NAME` | Storage padrao |
| `RABBITMQ_URL` | Broker para indexacao assincrona |
| `DOCUMENT_INDEX_*` | Exchanges, filas e retries da indexacao |
| `OUTBOX_POLL_INTERVAL_SECONDS`, `OUTBOX_BATCH_SIZE` | Parametros do dispatcher |
| `PRICING_SYNC_INTERVAL_SECONDS` | Intervalo do pricing worker |
| `RESET_PASSWORD_URL`, `SMTP_*` | Fluxo de reset de senha |
| `ALLOW_PRIVATE_STORAGE_ENDPOINTS` | Override de desenvolvimento para storage privado |

## Principais rotas

### Publicas

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/forgot-password`
- `POST /api/v1/auth/reset-password`

### Providers

- `POST /api/v1/user/providers`
- `GET /api/v1/user/providers`
- `PATCH /api/v1/user/providers/:providerType`

### Documentos

- `POST /api/v1/documents/upload`
- `GET /api/v1/documents/`
- `GET /api/v1/documents/:id`
- `GET /api/v1/documents/:id/content`
- `GET /api/v1/documents/:id/index`
- `GET /api/v1/documents/monitoring/summary`
- `GET /api/v1/documents/collections/:id/context`

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

## Testes e verificacoes

```bash
cd api
go test ./...
```

Se o ambiente tiver restricao de escrita no cache padrao do Go, use:

```bash
cd api
env GOCACHE=/tmp/go-build go test ./...
```

## Limitacoes atuais

- a telemetria de tokens e custo pode ser estimada, dependendo do provider;
- o score de qualidade da indexacao e heuristico;
- o chat runtime ainda nao esta implementado no backend;
- a infraestrutura de rate limit atual e simples e local ao processo;
- MinIO/S3-compatible e o storage efetivamente consolidado no fluxo atual.

## Relacao com o README raiz

Este arquivo documenta o backend de forma isolada. Para:

- arquitetura geral;
- escopo academico do TCC;
- frontend;
- docker compose completo;
- roadmap do projeto,

consulte o [README da raiz](../README.md).
