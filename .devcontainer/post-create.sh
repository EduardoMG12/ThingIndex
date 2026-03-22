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
  onboarding guide: docs/open-source-onboarding.md
  request examples: examples/http/thingindex-api.http
EOF
