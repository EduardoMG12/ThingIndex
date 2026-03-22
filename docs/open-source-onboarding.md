# Open Source Onboarding

Este guia existe para reduzir a friccao de quem chega ao repositório pela primeira vez e quer:

- subir o runtime local;
- consumir uma API ja publicada;
- inspecionar a API;
- navegar pela interface;
- validar se o ambiente esta saudavel;
- executar exemplos simples sem ler todo o monorepo antes.

## Entrada A - Self-hosted a partir do GitHub

Na raiz do projeto:

```bash
./scripts/bootstrap.sh
./scripts/stack.sh up
./scripts/doctor.sh
```

Esse fluxo:

- cria `.env` a partir de `.env.example` quando necessario;
- baixa dependencias de `api/` e `frontend/`;
- sobe a stack local com Docker Compose;
- verifica frontend, health da API e docs de referencia.

## URLs locais esperadas

| Recurso | URL |
| --- | --- |
| Frontend | `http://localhost:3000` |
| API | `http://localhost:8000` |
| Scalar API reference | `http://localhost:8000/docs` |
| Swagger UI | `http://localhost:8000/swagger/index.html` |
| Swagger JSON | `http://localhost:8000/swagger/doc.json` |
| Provider support matrix | `http://localhost:8000/api/v1/providers/support?family=LLM` |
| MinIO Console | `http://localhost:9001` |
| RabbitMQ Management | `http://localhost:15672` |

## Entrada B - Consumir uma API ja publicada

Se a API já estiver online, mantenha os mesmos paths na origin publicada:

| Recurso | Path |
| --- | --- |
| Scalar API reference | `https://<sua-origin>/docs` |
| Swagger UI | `https://<sua-origin>/swagger/index.html` |
| Swagger JSON | `https://<sua-origin>/swagger/doc.json` |
| Provider support matrix | `https://<sua-origin>/api/v1/providers/support?family=LLM` |
| API base | `https://<sua-origin>/api/v1` |

Fluxo curto:

1. abra `https://<sua-origin>/docs`;
2. use `https://<sua-origin>/api/v1` como base nas chamadas HTTP;
3. se tiver clonado o repo, altere `@apiHost` em `examples/http/thingindex-api.http`;
4. opcionalmente valide a deployment com `./scripts/doctor.sh --api-base https://<sua-origin>`.

## Wrappers uteis

### Bootstrap

```bash
./scripts/bootstrap.sh env
./scripts/bootstrap.sh install
./scripts/bootstrap.sh all
```

### Stack local

```bash
./scripts/stack.sh core
./scripts/stack.sh up
./scripts/stack.sh ps
./scripts/stack.sh logs api
./scripts/stack.sh down
```

### Diagnostico rapido

```bash
./scripts/doctor.sh
./scripts/doctor.sh --strict
./scripts/doctor.sh --api-base https://<sua-origin>
```

## Exemplos prontos

- `examples/http/thingindex-api.http` agrupa requests de health, docs, matriz publica de suporte a providers, auth e consulta autenticada ao catalogo.
- A referencia oficial do contrato continua em `api/docs/swagger.yaml`, `api/docs/swagger.json` e nas rotas `/docs` e `/swagger/index.html`.
- Para regenerar os artefatos versionados depois de mudar annotations do backend, use `make docs-api`.

## Troubleshooting direto

### A API docs nao carregou

- abra `http://localhost:8000/docs`;
- se o Scalar nao renderizar, use `Swagger UI` e `Swagger JSON` na mesma origin da API que voce esta consumindo.

### O frontend abriu, mas nada autenticado funciona

- confirme `NEXT_PUBLIC_API_URL` e `API_INTERNAL_URL` em `.env`;
- rode `./scripts/doctor.sh` para checar health e docs da API.

### O backend local nao conecta no banco

- em execucao manual no host, use `DB_HOST=localhost` e `DB_PORT=5434`, que e a porta publicada pelo Compose;
- no Docker Compose, a API usa `DB_HOST=db` e `DB_PORT=5432`.

### O storage nao responde

- no Compose, mantenha `MINIO_ENDPOINT=minio:9000`;
- no host, use `MINIO_ENDPOINT=localhost:9000`.

### Quero consumir a API sem abrir a UI

- use `<origin>/docs` para referencia navegavel;
- use `<origin>/api/v1/providers/support?family=LLM` para consultar o suporte real por provider, inclusive bridge requirements e notas operacionais;
- use `examples/http/thingindex-api.http` para requests prontas, trocando `@apiHost` quando necessario;
- use `<origin>/swagger/doc.json` quando precisar integrar ferramentas de OpenAPI.
