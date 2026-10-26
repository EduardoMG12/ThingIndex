#!/usr/bin/env bash
#
# Reexecuta o experimento pre-registrado de enriquecimento
# (docs/prds/pre-registro-perguntas-geradas.md) com uma API compilada do
# checkout atual, um processo por braco.
#
#   ./scripts/reexecuta-bracos-enriquecimento.sh <id-da-colecao> [saida]
#
# Existe porque a primeira execucao (scripts/experimento-bracos.sh, que troca o
# braco no .env e recria o conteiner) produziu as MESMAS fontes, pagina por
# pagina, nos quatro bracos: a troca nao chegou a valer. Aqui a variavel vai
# direto no ambiente do processo da API, e o script confere em /proc que ela
# esta la antes de medir. Nao toca no .env nem nos conteineres da stack.
#
# Exige a stack de pe (Postgres, MinIO, RabbitMQ nas portas padrao do compose)
# e a conta de demonstracao com provedor de LLM configurado.
set -euo pipefail

RAIZ="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COLECAO="${1:?informe o id da colecao de copia do experimento}"
SAIDA="${2:-$RAIZ/docs/article/results/enriquecimento-reexecucao}"
PORTA="${THINGINDEX_REEXEC_PORT:-8101}"
EMAIL="${THINGINDEX_BATERIA_EMAIL:-aluno.demo@thingindex.local}"
SENHA="${THINGINDEX_BATERIA_SENHA:-ThingIndex2026}"
mkdir -p "$SAIDA"

set -a; . "$RAIZ/.env"; set +a
export DB_HOST=localhost DB_PORT="${THINGINDEX_DB_PORT:-5434}" DB_USER="$POSTGRES_USER" DB_PASSWORD="$POSTGRES_PASSWORD" DB_NAME="$POSTGRES_DB"
export MINIO_ENDPOINT=localhost:9000
export RABBITMQ_URL="amqp://${RABBITMQ_DEFAULT_USER:-guest}:${RABBITMQ_DEFAULT_PASS:-guest}@localhost:5672/"
export PORT="$PORTA" THINGINDEX_QUERY_EXPANSION=

cd "$RAIZ/api"
BIN="$(mktemp)"
go build -o "$BIN" ./cmd/api
# O gabarito do pre-registro e o de 12 perguntas, congelado no commit do
# resultado; a versao atual do arquivo tem 23 e responde outra pergunta.
GABARITO="$(mktemp)"
git -C "$RAIZ" show f8813af:api/internal/evalcorpus/testdata/gabarito-seguranca-trabalho.json > "$GABARITO"
trap 'rm -f "$BIN" "$GABARITO"' EXIT

for braco in content content+summary content+questions "max(content,questions)"; do
  nome="$(echo "$braco" | tr -d "()," | tr "+ " "--" | sed 's/^maxcontentquestions$/max/')"
  THINGINDEX_PAGE_SEARCH_FIELDS="$braco" "$BIN" > "$SAIDA/api-$nome.log" 2>&1 &
  pid=$!
  for _ in $(seq 1 60); do curl -sf -m 3 "http://localhost:$PORTA/api/v1/health" >/dev/null 2>&1 && break; sleep 1; done
  ativo="$(tr '\0' '\n' < "/proc/$pid/environ" | sed -n 's/^THINGINDEX_PAGE_SEARCH_FIELDS=//p')"
  if [ "$ativo" != "$braco" ]; then
    kill "$pid"; echo "ERRO: processo com braco '${ativo:-vazio}', esperado '$braco'" >&2; exit 1
  fi
  echo "==> braco $braco"
  go run ./cmd/generation-eval -gabarito "$GABARITO" -pdf-dir "$RAIZ/examples/seguranca-trabalho" \
    -arms T -repeats 1 -collection "$COLECAO" -api "http://localhost:$PORTA/api/v1" \
    -email "$EMAIL" -password "$SENHA" -out "$SAIDA/braco-seguranca-trabalho-$nome.csv"
  kill "$pid"; wait "$pid" 2>/dev/null || true
  rm -f "$SAIDA/api-$nome.log"
done
