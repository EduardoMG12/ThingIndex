#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
GOCACHE_DIR="${THINGINDEX_GOCACHE:-/tmp/thingindex-go-cache}"

cd "$ROOT_DIR/api"
env GOCACHE="$GOCACHE_DIR" go run ./cmd/indexing-benchmark
