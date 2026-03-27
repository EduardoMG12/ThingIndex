#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR/api"

if command -v swag >/dev/null 2>&1; then
  swag init -g cmd/api/main.go -o docs
  exit 0
fi

GOCACHE_DIR="${THINGINDEX_GOCACHE:-/tmp/thingindex-go-cache}"
env GOCACHE="$GOCACHE_DIR" go run github.com/swaggo/swag/cmd/swag@v1.8.12 init -g cmd/api/main.go -o docs
