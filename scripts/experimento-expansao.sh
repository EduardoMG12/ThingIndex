#!/usr/bin/env bash
#
# Roda os dois braços do experimento pré-registrado de expansão da pergunta
# (docs/prds/pre-registro-expansao-consulta.md) sobre os três acervos.
#
#   ./scripts/experimento-expansao.sh [repeticoes]
#   BRACOS="on" ./scripts/experimento-expansao.sh 3   # só o tratamento
#
# O braço C (off) é o comportamento atual do sistema e já está medido em
# docs/article/results/bateria-*.csv; rodá-lo de novo só gasta cota. Por isso
# BRACOS existe: o padrão roda os dois, e quem já tem o controle roda só "on".
#
# O braço se troca no .env e não na linha de comando pelo mesmo motivo do
# experimento anterior: a recuperação acontece dentro do contêiner da API, não
# no harness, e exportar a variável no `go run` do harness não teria efeito
# nenhum -- produziria dois braços idênticos apresentados como dois braços.
#
# O .env é restaurado ao final, inclusive se o script for interrompido.
set -euo pipefail

RAIZ="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SAIDA="$RAIZ/docs/article/results"
REPETICOES="${1:-3}"
BRACOS="${BRACOS:-off on}"

BACKUP="$(mktemp)"
cp "$RAIZ/.env" "$BACKUP"
restaura() {
  cp "$BACKUP" "$RAIZ/.env"
  rm -f "$BACKUP"
  ( cd "$RAIZ" && docker compose up -d --force-recreate api >/dev/null 2>&1 ) || true
  echo "(.env restaurado)"
}
trap restaura EXIT

mkdir -p "$SAIDA/expansao"

for braco in $BRACOS; do
  echo "==> braço $([ "$braco" = on ] && echo "E, com expansão" || echo "C, controle")"

  grep -v '^THINGINDEX_QUERY_EXPANSION=' "$BACKUP" > "$RAIZ/.env"
  echo "THINGINDEX_QUERY_EXPANSION=$braco" >> "$RAIZ/.env"
  ( cd "$RAIZ" && docker compose up -d --force-recreate api >/dev/null 2>&1 )

  for _ in $(seq 1 60); do
    curl -sf -m 3 http://localhost:8000/api/v1/health >/dev/null 2>&1 && break
    sleep 2
  done

  # Confirma o braço ativo lendo o ambiente do contêiner, e aborta se divergir.
  ativo="$(docker compose -f "$RAIZ/docker-compose.yml" exec -T api printenv THINGINDEX_QUERY_EXPANSION 2>/dev/null | tr -d '\r' || true)"
  if [ "$ativo" != "$braco" ]; then
    echo "ERRO: a API está com THINGINDEX_QUERY_EXPANSION=${ativo:-vazio}, esperado $braco" >&2
    exit 1
  fi

  for entrada in "seguranca-trabalho:147:examples/seguranca-trabalho" \
                 "produto:148:examples/produto" \
                 "licitacoes:146:examples/licitacoes"; do
    IFS=: read -r tema colecao pdfs <<< "$entrada"
    echo "  -> $tema"
    ( cd "$RAIZ/api" && go run ./cmd/generation-eval \
        -gabarito "internal/evalcorpus/testdata/gabarito-$tema.json" \
        -pdf-dir "$RAIZ/$pdfs" -arms T -repeats "$REPETICOES" -collection "$colecao" \
        -email "${THINGINDEX_BATERIA_EMAIL:-aluno.demo@thingindex.local}" \
        -password "${THINGINDEX_BATERIA_SENHA:-ThingIndex2026}" \
        -out "$SAIDA/expansao/$braco-$tema.csv" )
  done
done
