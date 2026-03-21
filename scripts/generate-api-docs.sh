#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR/api"

go run github.com/swaggo/swag/cmd/swag@v1.8.12 init -g cmd/api/main.go -o docs
