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

## 🛠️ Setup & Run

1. **Install Dependencies**:
   ```bash
   go mod download
   ```

2. **Configure**:
   Copy `.env.example` to `.env` (optional, uses defaults if missing).

3. **Run**:
   ```bash
   go run cmd/crawler/main.go
   ```
