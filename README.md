# LegisGraph Crawler

A decoupled, recursive PDF crawler written in Go.

## Features
- **Clean Architecture**: Domain-driven design with decoupled components.
- **Recursive Crawling**: Finds links to other PDFs and crawls them.
- **Resilient Fetching**: Retries on failure with exponential backoff.
- **Swappable Storage**: Currently supports local file system storage.

## Usage

### Build
```bash
go build -o crawler cmd/crawler/main.go
```

### Run
```bash
./crawler -url "https://example.com/start.pdf" -output "./pdfs"
```

### Flags
- `-url`: (Required) Starting URL.
- `-output`: Directory to save PDFs (default: `./output_pdfs`).
- `-timeout`: HTTP timeout (default: `10s`).
- `-retries`: Number of retries (default: `3`).
