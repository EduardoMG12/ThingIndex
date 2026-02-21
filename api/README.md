<div align="center">
  <img src="../assets/palmas-horizontal-1.png" alt="IFPR Campus Palmas" width="300" />
</div>

<div align="center">
  <h1>LegisGraph: FastAPI Gateway</h1>
  <p><b>Course / Curso:</b> Sistemas de Informação</p>
  <p><b>Author / Autor:</b> Charles Eduardo Mello Guimarães</p>
  <p><b>Trabalho de Conclusão de Curso (TCC)</b></p>
</div>

<br />

<details>
<summary>🇺🇸 <b>English Version / Versão em Inglês</b></summary>

## ✨ Features
A decoupled, fast, and modern FastAPI gateway for the LegisGraph Crawler. It uses RabbitMQ under the hood to asynchronously dispatch crawling tasks to Go workers.
- **Asynchronous Dispatch**: Enqueues crawling tasks instantly without blocking HTTP responses.
- **Pydantic Validation**: Strict and reliable request/response schemas.
- **Queue Purging**: Ability to clean the mensageria queue and reset the crawler's memory.
- **Interactive Documentation**: Replaces standard Swagger with the beautiful **Scalar UI**.
- **Modern Python**: Managed by [PDM](https://pdm-project.org/) for reproducible builds.

## 🛠️ Stack
- **Framework**: FastAPI
- **Mensageria**: RabbitMQ (via `pika`)
- **Package Manager**: PDM
- **Docs**: Scalar
- **Config**: Pydantic Settings

## 🚦 Getting Started
The API is fully integrated with Docker Compose. To run it together with the rest of the application ecosystem:

```bash
docker compose up api -d
```

### Local Development
1. Ensure RabbitMQ is running (`docker compose up rabbitmq -d`).
2. Install dependencies with PDM: `pdm install`
3. Start the Uvicorn server: `pdm run uvicorn src.main:app --host 0.0.0.0 --port 8000 --reload`

## 📖 Endpoints
Access `http://localhost:8000/scalar` to interact with the API endpoints visually.

### `POST /crawl`
Enqueues a specific URL to be crawled by the Go workers.

### `POST /purge`
Injects a magic payload (`__PURGE__`) commanding Go crawlers to immediately wipe the waiting queue.
</details>

<details>
<summary>🇧🇷 <b>Portuguese Version / Versão em Português</b></summary>

## ✨ Funcionalidades
Um gateway em FastAPI moderno e desacoplado para a orquestração do LegisGraph. Ele opera se comunicando com o RabbitMQ, jogando tarefas assíncronas para os workers robustos em Go trabalharem no plano de fundo.
- **Despacho Assíncrono**: Responde instantaneamente para o usuário Front-End sem ficar aguardando o processamento das páginas governamentais lidas.
- **Validação Pydantic**: Contratos de comunicação (`Schemas`) puros e muito bem tipados.
- **Documentação Interativa (Scalar)**: Abandonando o velho visual do Swagger por uma UI de referência moderna (Scalar HTML).
- **Gerenciamento PDM**: Builds determinísticos pra que o container rode liso.

## 🛠️ Tecnologias
- **Framework REST**: FastAPI
- **Mensageria (AMQP)**: RabbitMQ (via `pika`)
- **Gestão de Dependências**: PDM
- **Tipagem Fina**: Pydantic (e `pydantic-settings`)

## 🚦 Primeiros Passos
Não quebre a cabeça, essa belezura já foi configurada para subir em conjunto através do Docker:

```bash
docker compose up api -d
```

### 📖 Endpoints Principais
Acesse `http://localhost:8000/scalar` para o playground dos endpoints.
- `/crawl`: Joga um alvo (URL) novo pro Go Crawler atacar em background.
- `/purge`: Manda ele esquecer todos os sites pendentes e resetar a cabeça.
</details>

