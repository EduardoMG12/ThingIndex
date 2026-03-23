#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

GOCACHE_DIR="${THINGINDEX_GOCACHE:-/tmp/thingindex-go-cache}"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/test.sh <target>

Targets:
  unit                  Run backend unit tests and frontend unit tests
  backend-unit          Run backend unit tests
  backend-benchmark     Run the indexing benchmark unit suite
  backend-integration   Run backend integration tests
  backend-e2e           Run backend end-to-end tests
  backend-smoke         Run backend smoke tests
  frontend-unit         Run frontend unit tests
  frontend-storybook    Run the Storybook build and Storybook smoke tests
  frontend-e2e          Run frontend end-to-end tests
  frontend-smoke        Run frontend smoke tests
  frontend-e2e-install  Install the Playwright Chromium browser
  integration           Alias for backend-integration
  e2e                   Run backend and frontend end-to-end tests
  all                   Run the main full suite

Examples:
  ./scripts/test.sh unit
  ./scripts/test.sh backend-integration
  ./scripts/test.sh frontend-e2e
  make test-all
EOF
}

run_backend_unit() {
  echo "[test] backend unit"
  (
    cd api
    env GOCACHE="$GOCACHE_DIR" go test ./...
  )
}

run_backend_integration() {
  echo "[test] backend integration"
  (
    cd api
    env GOCACHE="$GOCACHE_DIR" go test -tags=integration ./...
  )
}

run_backend_benchmark() {
  echo "[test] backend indexing benchmark"
  (
    cd api
    env GOCACHE="$GOCACHE_DIR" go test ./internal/usecase -run IndexingBenchmark
  )
}

run_backend_e2e() {
  echo "[test] backend e2e"
  (
    cd api
    env GOCACHE="$GOCACHE_DIR" go test -tags=e2e ./internal/api -run RoutesE2E
  )
}

run_backend_smoke() {
  echo "[test] backend smoke"
  (
    cd api
    env GOCACHE="$GOCACHE_DIR" go test -tags=smoke ./internal/handlers -run RouteSmoke
  )
}

run_frontend_unit() {
  echo "[test] frontend unit"
  pnpm --dir frontend run test:unit
}

run_frontend_e2e_install() {
  echo "[test] frontend e2e browser install"
  pnpm --dir frontend run test:e2e:install
}

run_frontend_e2e() {
  echo "[test] frontend e2e"
  pnpm --dir frontend run test:e2e
}

run_frontend_storybook() {
  echo "[test] frontend storybook"
  pnpm --dir frontend run test:storybook
}

run_frontend_smoke() {
  echo "[test] frontend smoke"
  pnpm --dir frontend run test:smoke
}

target="${1:-}"

case "$target" in
  unit)
    run_backend_unit
    run_frontend_unit
    ;;
  backend-unit)
    run_backend_unit
    ;;
  backend-benchmark)
    run_backend_benchmark
    ;;
  backend-integration|integration)
    run_backend_integration
    ;;
  backend-e2e)
    run_backend_e2e
    ;;
  backend-smoke)
    run_backend_smoke
    ;;
  frontend-unit)
    run_frontend_unit
    ;;
  frontend-storybook)
    run_frontend_storybook
    ;;
  frontend-e2e)
    run_frontend_e2e
    ;;
  frontend-smoke)
    run_frontend_smoke
    ;;
  frontend-e2e-install)
    run_frontend_e2e_install
    ;;
  e2e)
    run_backend_e2e
    run_frontend_e2e
    ;;
  all)
    run_backend_unit
    run_frontend_unit
    run_backend_integration
    run_backend_smoke
    run_backend_e2e
    run_frontend_e2e
    ;;
  -h|--help|help|"")
    usage
    ;;
  *)
    echo "Unknown target: $target" >&2
    echo >&2
    usage
    exit 1
    ;;
esac
