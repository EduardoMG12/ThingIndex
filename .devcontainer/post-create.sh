#!/usr/bin/env bash
set -euo pipefail

cd /workspace/api
go mod download

cd /workspace/frontend
pnpm install
