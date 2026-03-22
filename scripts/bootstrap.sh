#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
ENV_FILE="$ROOT_DIR/.env"
ENV_EXAMPLE_FILE="$ROOT_DIR/.env.example"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/bootstrap.sh [command]

Commands:
  env       Create .env from .env.example when missing
  install   Install backend and frontend dependencies
  all       Prepare .env, install dependencies, and print next steps
  help      Show this help message

Examples:
  ./scripts/bootstrap.sh
  ./scripts/bootstrap.sh env
  ./scripts/bootstrap.sh install
EOF
}

require_command() {
  local command_name="$1"

  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "[bootstrap] missing required command: $command_name" >&2
    exit 1
  fi
}

ensure_env_file() {
  if [[ -f "$ENV_FILE" ]]; then
    echo "[bootstrap] using existing .env"
    return
  fi

  cp "$ENV_EXAMPLE_FILE" "$ENV_FILE"
  echo "[bootstrap] created .env from .env.example"
}

install_backend_dependencies() {
  require_command go
  echo "[bootstrap] downloading Go modules"
  (
    cd "$ROOT_DIR/api"
    go mod download
  )
}

install_frontend_dependencies() {
  require_command pnpm
  echo "[bootstrap] installing frontend dependencies"
  (
    cd "$ROOT_DIR/frontend"
    pnpm install
  )
}

print_next_steps() {
  cat <<'EOF'
[bootstrap] next steps
  1. ./scripts/stack.sh up
  2. ./scripts/doctor.sh
  3. open http://localhost:3000
  4. open http://localhost:8000/docs
  5. read docs/open-source-onboarding.md
  6. try examples/http/thingindex-api.http
EOF
}

command_name="${1:-all}"

case "$command_name" in
  env)
    ensure_env_file
    ;;
  install)
    install_backend_dependencies
    install_frontend_dependencies
    ;;
  all)
    ensure_env_file
    install_backend_dependencies
    install_frontend_dependencies
    print_next_steps
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    echo "Unknown bootstrap command: $command_name" >&2
    echo >&2
    usage
    exit 1
    ;;
esac
