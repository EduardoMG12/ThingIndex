# LegisGraph Crawler

This component is responsible for crawling Brazilian federal legislation from the Planalto.gov.br portal.

## 🚀 Features

- **Format Handling**: Downloads both HTML and PDF versions of laws.
- **Hierarchical Context Tracking**:
  - **Outgoing Context**: Tracks exactly where a link was found (e.g., "Link to Law X found inside Art. 5, § 2 of Constitution").
  - **Incoming Context**: Propagates the context of the link that led to the current page (e.g., "This decree was crawled because it was linked from Art. 84 of CF88").
- **Smart Filtering**: Configurable to target specific laws or ignore irrelevant pages.
- **Deduplication**: Uses content hashing to avoid re-downloading identical files.

## 📦 Data Structure

For each document, the crawler saves:
1. **Raw Content**: The original `.htm` or `.pdf` file.
2. **Metadata (`.json`)**: Contains URL, parent URL, and enriched link context.

### Example Metadata (`.json`)
```json
{
  "url": "http://www.planalto.gov.br/ccivil_03/Leis/L8078.htm",
  "parent_url": "http://www.planalto.gov.br/ccivil_03/Constituicao/Constituicao.htm",
  "parent_context": {
    "artigo": "5",
    "inciso": "XXXII"
  },
  "links_context": [
    {
      "url": "http://www.planalto.gov.br/.../L7347.htm",
      "text": "Lei n° 7.347, de 24 de julho de 1985",
      "context": {
        "artigo": "117"
      }
    }
  ]
}
```

## 🐇 Queue & Triggering (RabbitMQ)

The Crawler operates purely as a background worker (consumer) using **RabbitMQ**. It listens to the `crawler_tasks` queue. 

To start crawling a website, you must publish a JSON payload to the queue matching the `CrawlTask` structure.

**Example Payload to start crawling:**
```json
{
  "url": "http://www.planalto.gov.br/ccivil_03/leis/2003/L10.741.htm",
  "parent": "",
  "description": "Start URL",
  "depth": 0
}
```

### 🧹 The `__PURGE__` Command
If you need to stop the crawler, wipe the current queue, and clean the internal memory cache (Bloom Filter) without restarting the container, you can send a magic payload with the URL set to `__PURGE__`:
```json
{
  "url": "__PURGE__",
  "parent": "",
  "description": "",
  "depth": 0
}
```
The crawler will immediately purge all pending messages in RabbitMQ and reset its state.

## 🛠️ Setup & Run

1. **Start dependencies** (RabbitMQ and Neo4j) via the root `docker-compose.yml`:
   ```bash
   go mod download
   ```

   Configured either by the `.env` file at the root of the project or Docker parameters.

3. **Run**:
   ```bash
   docker compose up crawler -d
   ```
   Or locally (ensuring RabbitMQ is running): 
   ```bash
   go run cmd/crawler/main.go
   ```
