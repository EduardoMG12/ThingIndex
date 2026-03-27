#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/stack.sh <command> [services...]

Commands:
  up [services...]     Start the local Docker Compose stack, building images when needed
  core                 Start only db, minio, and rabbitmq in detached mode
  down                 Stop the stack and remove orphan containers
  ps                   Show container status
  logs [services...]   Follow logs for the whole stack or selected services
  restart [services...] Restart the whole stack or selected services
  help                 Show this help message

Examples:
  ./scripts/stack.sh up
  ./scripts/stack.sh up api frontend
  ./scripts/stack.sh core
  ./scripts/stack.sh logs api
EOF
}

run_compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
    return
  fi

  if command -v docker-compose >/dev/null 2>&1; then
    docker-compose "$@"
    return
  fi

  echo "[stack] docker compose is not available" >&2
  exit 1
}

command_name="${1:-help}"
shift || true

case "$command_name" in
  up)
    if [[ $# -eq 0 ]]; then
      run_compose up --build
    else
      run_compose up --build "$@"
    fi
    ;;
  core)
    run_compose up -d db minio rabbitmq
    ;;
  down)
    run_compose down --remove-orphans
    ;;
  ps)
    run_compose ps
    ;;
  logs)
    if [[ $# -eq 0 ]]; then
      run_compose logs -f
    else
      run_compose logs -f "$@"
    fi
    ;;
  restart)
    if [[ $# -eq 0 ]]; then
      run_compose restart
    else
      run_compose restart "$@"
    fi
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    echo "Unknown stack command: $command_name" >&2
    echo >&2
    usage
    exit 1
    ;;
esac
