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
  dev [services...]    Start the stack with hot reload for api/indexer using docker-compose.dev.yml
  core                 Start only db, minio, and rabbitmq in detached mode
  down                 Stop the stack and remove orphan containers
  dev-down             Stop the dev stack and remove orphan containers
  ps                   Show container status
  dev-ps               Show container status for the dev stack
  logs [services...]   Follow logs for the whole stack or selected services
  dev-logs [services...] Follow logs for the dev stack or selected services
  restart [services...] Restart the whole stack or selected services
  dev-restart [services...] Restart the dev stack or selected services
  help                 Show this help message

Examples:
  ./scripts/stack.sh up
  ./scripts/stack.sh dev
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

resolve_go_mod_cache() {
  if [[ -n "${THINGINDEX_GO_MOD_CACHE:-}" ]]; then
    printf '%s\n' "$THINGINDEX_GO_MOD_CACHE"
    return
  fi

  if command -v go >/dev/null 2>&1; then
    (
      cd "$ROOT_DIR/api"
      go env GOMODCACHE
    )
    return
  fi

  printf '%s\n' "$ROOT_DIR/.cache/go-mod"
}

resolve_go_build_cache() {
  if [[ -n "${THINGINDEX_GO_BUILD_CACHE:-}" ]]; then
    printf '%s\n' "$THINGINDEX_GO_BUILD_CACHE"
    return
  fi

  if command -v go >/dev/null 2>&1; then
    (
      cd "$ROOT_DIR/api"
      go env GOCACHE
    )
    return
  fi

  printf '%s\n' "$ROOT_DIR/.cache/go-build"
}

run_compose_dev() {
  local mod_cache build_cache
  mod_cache="$(resolve_go_mod_cache)"
  build_cache="$(resolve_go_build_cache)"

  mkdir -p "$mod_cache" "$build_cache"

  THINGINDEX_GO_MOD_CACHE="$mod_cache" \
  THINGINDEX_GO_BUILD_CACHE="$build_cache" \
    run_compose -f docker-compose.yml -f docker-compose.dev.yml "$@"
}

prepare_go_dev_cache() {
  local mod_cache build_cache
  mod_cache="$(resolve_go_mod_cache)"
  build_cache="$(resolve_go_build_cache)"

  mkdir -p "$mod_cache" "$build_cache"

  if ! command -v go >/dev/null 2>&1; then
    echo "[stack] go is not available on the host; dev mode may still need to download modules inside Docker" >&2
    return
  fi

  (
    cd "$ROOT_DIR/api"
    GOMODCACHE="$mod_cache" GOCACHE="$build_cache" go mod download
  )
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
  dev)
    prepare_go_dev_cache
    if [[ $# -eq 0 ]]; then
      run_compose_dev up
    else
      run_compose_dev up "$@"
    fi
    ;;
  core)
    run_compose up -d db minio rabbitmq
    ;;
  down)
    run_compose down --remove-orphans
    ;;
  dev-down)
    run_compose_dev down --remove-orphans
    ;;
  ps)
    run_compose ps
    ;;
  dev-ps)
    run_compose_dev ps
    ;;
  logs)
    if [[ $# -eq 0 ]]; then
      run_compose logs -f
    else
      run_compose logs -f "$@"
    fi
    ;;
  dev-logs)
    if [[ $# -eq 0 ]]; then
      run_compose_dev logs -f
    else
      run_compose_dev logs -f "$@"
    fi
    ;;
  restart)
    if [[ $# -eq 0 ]]; then
      run_compose restart
    else
      run_compose restart "$@"
    fi
    ;;
  dev-restart)
    if [[ $# -eq 0 ]]; then
      run_compose_dev restart
    else
      run_compose_dev restart "$@"
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
