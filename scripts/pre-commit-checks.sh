#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

mapfile -t STAGED_FILES < <(git diff --cached --name-only --diff-filter=ACMR)

if [[ ${#STAGED_FILES[@]} -eq 0 ]]; then
  exit 0
fi

HAS_FRONTEND_CHANGES=0
HAS_BACKEND_CHANGES=0
declare -a STAGED_GO_FILES=()

for file in "${STAGED_FILES[@]}"; do
  case "$file" in
    api/*.go)
      HAS_BACKEND_CHANGES=1
      STAGED_GO_FILES+=("$file")
      ;;
    api/go.mod|api/go.sum)
      HAS_BACKEND_CHANGES=1
      ;;
  esac

  case "$file" in
    frontend/src/*|frontend/package.json|frontend/pnpm-lock.yaml|frontend/eslint.config.mjs|frontend/tsconfig.json|frontend/vitest.config.ts|frontend/vitest.setup.tsx|frontend/playwright.config.ts|frontend/next.config.ts|frontend/tailwind.config.js|frontend/postcss.config.js)
      HAS_FRONTEND_CHANGES=1
      ;;
  esac
done

if [[ ${#STAGED_GO_FILES[@]} -gt 0 ]]; then
  echo "[pre-commit] formatando arquivos Go staged"
  gofmt -w "${STAGED_GO_FILES[@]}"
  git add -u "${STAGED_GO_FILES[@]}"
fi

if [[ $HAS_BACKEND_CHANGES -eq 1 ]]; then
  echo "[pre-commit] validando backend/api"
  (
    cd api
    env GOCACHE=/tmp/thingindex-go-cache go vet ./...
  )
  "$ROOT_DIR/scripts/test.sh" backend-unit
fi

if [[ $HAS_FRONTEND_CHANGES -eq 1 ]]; then
  echo "[pre-commit] validando frontend"
  pnpm --dir frontend run lint
  pnpm --dir frontend run typecheck
  "$ROOT_DIR/scripts/test.sh" frontend-unit
fi
