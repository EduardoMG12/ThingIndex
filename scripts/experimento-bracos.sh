#!/usr/bin/env bash
#
# Roda os três braços do experimento pré-registrado de enriquecimento
# (docs/prds/pre-registro-perguntas-geradas.md) sobre o MESMO índice.
#
#   ./scripts/experimento-bracos.sh <id-da-coleção> <nome-do-gabarito> <pasta-dos-pdfs>
#
# Por que o braço se troca no .env e não na linha de comando: a recuperação
# acontece dentro do contêiner da API, não no harness. Uma tentativa anterior
# exportou a variável no `go run` do harness e ela não teve efeito nenhum, o que
# teria produzido três medições idênticas apresentadas como três braços.
#
# O .env é restaurado ao final, inclusive se o script for interrompido.
set -euo pipefail

RAIZ="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SAIDA="$RAIZ/docs/article/results"
COLECAO="${1:?informe o id da coleção}"
GABARITO="${2:?informe o nome do gabarito, por exemplo seguranca-trabalho}"
PDFS="${3:?informe a pasta dos PDFs}"
EMAIL="${THINGINDEX_BATERIA_EMAIL:-aluno.demo@thingindex.local}"
SENHA="${THINGINDEX_BATERIA_SENHA:-ThingIndex2026}"

BACKUP="$(mktemp)"
cp "$RAIZ/.env" "$BACKUP"
restaura() {
  cp "$BACKUP" "$RAIZ/.env"
  rm -f "$BACKUP"
  ( cd "$RAIZ" && docker compose restart api >/dev/null 2>&1 ) || true
  echo "(.env restaurado)"
}
trap restaura EXIT

mkdir -p "$SAIDA"

for braco in content content+summary content+questions "max(content,questions)"; do
  nome="$(echo "$braco" | tr -d "()," | tr "+ " "--")"
  echo "==> braço $braco"

  grep -v '^THINGINDEX_PAGE_SEARCH_FIELDS=' "$BACKUP" > "$RAIZ/.env"
  echo "THINGINDEX_PAGE_SEARCH_FIELDS=$braco" >> "$RAIZ/.env"
  ( cd "$RAIZ" && docker compose up -d --force-recreate api >/dev/null 2>&1 )

  # Espera a API voltar: sem isso o primeiro lote de perguntas bate em conexão
  # recusada e o braço inteiro sai com erro em vez de resultado.
  for _ in $(seq 1 60); do
    if curl -sf -m 3 http://localhost:8000/api/v1/health >/dev/null 2>&1; then break; fi
    sleep 2
  done

  # Confirma que o braço ativo é mesmo o pedido, lendo o ambiente do contêiner.
  ativo="$(docker compose -f "$RAIZ/docker-compose.yml" exec -T api printenv THINGINDEX_PAGE_SEARCH_FIELDS 2>/dev/null | tr -d '\r' || true)"
  if [ "$ativo" != "$braco" ]; then
    echo "ERRO: a API está com THINGINDEX_PAGE_SEARCH_FIELDS=${ativo:-vazio}, esperado $braco" >&2
    exit 1
  fi

  ( cd "$RAIZ/api" && go run ./cmd/generation-eval \
      -gabarito "internal/evalcorpus/testdata/gabarito-$GABARITO.json" \
      -pdf-dir "$RAIZ/$PDFS" -arms T -repeats 1 -collection "$COLECAO" \
      -email "$EMAIL" -password "$SENHA" \
      -out "$SAIDA/braco-$GABARITO-$nome.csv" )
done
