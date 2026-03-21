#!/usr/bin/env bash
set -euo pipefail

cd /workspace/api
go mod download
go install github.com/swaggo/swag/cmd/swag@v1.8.12

cd /workspace/frontend
pnpm install
