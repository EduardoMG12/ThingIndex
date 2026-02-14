# LegisGraph Legal Parser

This component processes raw Brazilian legal documents (HTML/PDF) into structured, graph-ready JSON.

## 🚀 Features

- **Document Identity**: Extracts type (Lei, Decreto, MPV), number, year, and canonical ID (e.g., `lei-federal-8078-1990`).
- **Hierarchical Parsing**: Breaks down the text into Articles, Paragraphs, Incisos, and Alíneas.
- **Enriched Cross-References**:
  - Identifies references to other laws in the text.
  - Matches references with crawler metadata to pinpoint exactly where in the source document the link appeared.
- **Incoming Context**: Preserves the context of where the document was discovered (e.g., linked from Art. 5 of CF88).

## 📦 Output Format

The parser generates standard JSON for ingestion into Neo4j:

```json
{
  "id": "lei-federal-8078-1990",
  "tipo": "Lei",
  "numero": "8078",
  "ano": "1990",
  "titulo": "LEI Nº 8.078, DE 11 DE SETEMBRO DE 1990.",
  "ementa": "Dispõe sobre a proteção do consumidor e dá outras providências.",
  "url_pai": "http://www.planalto.gov.br/ccivil_03/Constituicao/Constituicao.htm",
  "url_pai_context": {
    "artigo": "5",
    "inciso": "XXXII"
  },
  "artigos": [
    {
      "numero": "1",
      "texto": "O presente código estabelece normas de proteção e defesa do consumidor..."
    }
  ],
  "referencias": [
    {
      "id": "lei-federal-7347-1985",
      "context": {
        "artigo": "117"
      }
    }
  ]
}
```

## 🛠️ Usage

1. **Install Dependencies**:
   ```bash
   go mod download
   ```

2. **Run**:
   ```bash
   go run cmd/parser/main.go
   ```
