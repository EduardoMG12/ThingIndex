# LegisGraph Crawler

A decoupled, recursive PDF crawler written in Go.

## Features
- **Clean Architecture**: Domain-driven design with decoupled components.
- **Recursive Crawling**: Finds links to other PDFs and crawls them.
- **Resilient Fetching**: Retries on failure with exponential backoff.
- **Swappable Storage**: Currently supports local file system storage.

## Usage

### Setup
1. Copy the example configuration:
   ```bash
   cp .env.example .env
   ```
2. Edit `.env` with your desired settings (Start URL, Output Directory, etc.).

### Build
```bash
go build -o crawler cmd/crawler/main.go
```

### Run
```bash
./crawler
```
