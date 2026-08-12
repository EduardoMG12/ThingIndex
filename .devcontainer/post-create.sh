#!/usr/bin/env bash
set -euo pipefail

cd /workspace
./scripts/bootstrap.sh all

cd /workspace/api
go install github.com/swaggo/swag/cmd/swag@v1.8.12

cat <<'EOF'
[devcontainer] ready
  frontend: http://localhost:3000
  api: http://localhost:8000
  api docs: http://localhost:8000/docs
  runtime profile: http://localhost:8000/api/v1/meta/runtime-profile
  indexing benchmark: http://localhost:8000/api/v1/meta/indexing-benchmark
  provider support matrix: http://localhost:8000/api/v1/providers/support?family=LLM
  onboarding guide: docs/open-source-onboarding.md
  product focus RFC: rfc/0001-product-focus.md
  academic workspace: docs/article/README.md
  benchmark runner: ./scripts/indexing-benchmark.sh
  index import CLI: go run ./api/cmd/index-import --help
  request examples: examples/http/thingindex-api.http
EOF
