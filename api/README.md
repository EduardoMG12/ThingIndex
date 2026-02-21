# LegisGraph API 🚀

A decoupled, fast, and modern FastAPI gateway for the LegisGraph Crawler. It uses RabbitMQ under the hood to asynchronously dispatch crawling tasks to Go workers.

## ✨ Features

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
2. Install dependencies with PDM:
   ```bash
   pdm install
   ```
3. Start the Uvicorn server:
   ```bash
   pdm run uvicorn src.main:app --host 0.0.0.0 --port 8000 --reload
   ```

## 📖 Endpoints

Access `http://localhost:8000/scalar` to interact with the API endpoints visually.

### `POST /crawl`
Enqueues a specific URL to be crawled by the Go workers.
```json
// Request Body
{
  "url": "http://planalto.gov.br/...",
  "depth": 0
}
```

### `POST /purge`
Injects a magic payload (`__PURGE__`) into the RabbitMQ queue, commanding all actively listening Go crawlers to immediately wipe the waiting queue and rest their internal caches.

