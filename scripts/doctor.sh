#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
ENV_FILE="$ROOT_DIR/.env"

strict_mode=0
api_base_override=""
frontend_base_override=""

print_usage() {
  cat <<'EOF'
Usage:
  ./scripts/doctor.sh [--strict] [--api-base <url>] [--frontend-base <url>]

Checks:
  - required local commands for Docker Compose and manual execution
  - presence of .env
  - presence of the indexing benchmark runner and benchmark corpus
  - reachability of frontend when running locally
  - reachability of API health, Scalar docs, Swagger UI, Swagger JSON, the public runtime profile, the public indexing benchmark, and the public storage/LLM support matrices

Behavior:
  - non-strict mode reports unavailable services as warnings
  - --strict turns warnings into a non-zero exit code

Examples:
  ./scripts/doctor.sh
  ./scripts/doctor.sh --strict
  ./scripts/doctor.sh --api-base https://api.example.com
  ./scripts/doctor.sh --api-base https://api.example.com --frontend-base https://app.example.com
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --strict)
      strict_mode=1
      shift
      ;;
    --api-base)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for --api-base" >&2
        exit 1
      fi
      api_base_override="$2"
      shift 2
      ;;
    --frontend-base)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for --frontend-base" >&2
        exit 1
      fi
      frontend_base_override="$2"
      shift 2
      ;;
    -h|--help|help)
      print_usage
      exit 0
      ;;
    *)
      echo "Unknown doctor option: $1" >&2
      exit 1
      ;;
  esac
done

failures=0
warnings=0

pass() {
  echo "[ok] $1"
}

warn() {
  warnings=$((warnings + 1))
  echo "[warn] $1"
}

fail() {
  failures=$((failures + 1))
  echo "[fail] $1"
}

has_command() {
  command -v "$1" >/dev/null 2>&1
}

check_command() {
  local command_name="$1"
  local description="$2"

  if has_command "$command_name"; then
    pass "$description: $command_name"
  else
    fail "$description is missing ($command_name)"
  fi
}

check_compose() {
  if docker compose version >/dev/null 2>&1; then
    pass "Docker Compose v2 is available"
    return
  fi

  if has_command docker-compose; then
    pass "docker-compose standalone is available"
    return
  fi

  fail "Docker Compose is missing"
}

check_file() {
  local path="$1"
  local description="$2"

  if [[ -f "$ROOT_DIR/$path" ]]; then
    pass "$description present at $path"
  else
    fail "$description is missing at $path"
  fi
}

read_env_value() {
  local key="$1"

  if [[ ! -f "$ENV_FILE" ]]; then
    return
  fi

  grep -E "^${key}=" "$ENV_FILE" | head -n 1 | cut -d= -f2-
}

resolve_api_reference_base() {
  if [[ -n "$api_base_override" ]]; then
    echo "$api_base_override" | sed -E 's#/api/v1/?$##; s#/$##'
    return
  fi

  local configured_url
  configured_url="$(read_env_value "NEXT_PUBLIC_API_URL")"

  if [[ -z "$configured_url" ]]; then
    echo "http://127.0.0.1:8000"
    return
  fi

  echo "$configured_url" | sed -E 's#/api/v1/?$##'
}

resolve_frontend_base() {
  if [[ -n "$frontend_base_override" ]]; then
    echo "$frontend_base_override" | sed -E 's#/$##'
    return
  fi

  if [[ -n "$api_base_override" ]]; then
    echo ""
    return
  fi

  echo "http://127.0.0.1:3000"
}

probe_url() {
  local label="$1"
  local url="$2"

  if ! has_command curl; then
    warn "curl is not installed; skipping $label"
    return
  fi

  if curl --silent --show-error --fail --max-time 5 "$url" >/dev/null; then
    pass "$label reachable at $url"
  else
    warn "$label is not reachable at $url"
  fi
}

echo "[doctor] ThingIndex local environment"

check_command git "Git"
check_command go "Go toolchain"
check_command pnpm "pnpm"
check_command docker "Docker"
check_compose
check_file "docs/open-source-onboarding.md" "Open-source onboarding guide"
check_file "examples/http/thingindex-api.http" "HTTP request examples"
check_file "api/docs/swagger.yaml" "Versioned OpenAPI YAML"
check_file "api/docs/swagger.json" "Versioned OpenAPI JSON"
check_file "scripts/indexing-benchmark.sh" "Indexing benchmark runner"
check_file "api/internal/usecase/testdata/indexing-benchmark/corpus.json" "Indexing benchmark corpus"

if [[ -f "$ENV_FILE" ]]; then
  pass ".env is present"
else
  warn ".env is missing; run ./scripts/bootstrap.sh env"
fi

api_base="$(resolve_api_reference_base)"
frontend_base="$(resolve_frontend_base)"

if [[ -n "$frontend_base" ]]; then
  probe_url "Frontend" "$frontend_base/en"
fi
probe_url "API health" "$api_base/api/v1/health"
probe_url "Scalar API docs" "$api_base/docs"
probe_url "Swagger UI" "$api_base/swagger/index.html"
probe_url "Swagger JSON" "$api_base/swagger/doc.json"
probe_url "Runtime profile" "$api_base/api/v1/meta/runtime-profile"
probe_url "Indexing benchmark" "$api_base/api/v1/meta/indexing-benchmark"
probe_url "Storage support matrix" "$api_base/api/v1/providers/support?family=STORAGE"
probe_url "LLM support matrix" "$api_base/api/v1/providers/support?family=LLM"

echo "[doctor] failures=$failures warnings=$warnings"

if (( failures > 0 )); then
  exit 1
fi

if (( strict_mode == 1 && warnings > 0 )); then
  exit 1
fi
