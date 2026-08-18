#!/usr/bin/env bash
# Mede a cobertura de testes do backend e falha se ela cair abaixo do piso.
#
# POR QUE ISTO EXISTE
# -------------------
# O artigo afirma ter "cobertura de testes automatizados" e nao havia
# `-coverprofile` em lugar nenhum: nao existia percentual publicavel. Numero de
# cobertura e uma das primeiras coisas que a banca pergunta, e responder "nao
# medimos" e pior do que responder um numero modesto.
#
# POR QUE UM PISO, E NAO UMA META
# -------------------------------
# O piso existe para impedir REGRESSAO, nao para perseguir 100%. Cobertura alta
# nao significa teste bom -- da para cobrir 100% das linhas sem verificar uma
# unica asserçao util. O valor aqui e o percentual real medido no momento em que
# o guard foi criado, arredondado para baixo.
#
# Quando a cobertura subir de forma consistente, suba o piso junto: um piso
# muito abaixo do real deixa de proteger contra regressao.
#
# Uso:
#   scripts/check-coverage.sh          # usa o piso padrao
#   MIN_COVERAGE=50 scripts/check-coverage.sh

set -euo pipefail

# Piso: 45%. A medicao no momento da criacao deste guard deu 47.8%, entao ha
# ~3 pontos de folga para refatoracao normal sem quebrar o CI de quem so mexeu
# em codigo nao testado.
MIN_COVERAGE="${MIN_COVERAGE:-45}"

cd "$(dirname "$0")/../api"

profile="$(mktemp -t thingindex-cover.XXXXXX)"
trap 'rm -f "$profile"' EXIT

echo "[coverage] rodando a suite com -coverprofile"
go test -count=1 -coverprofile="$profile" ./... >/dev/null

total_line="$(go tool cover -func="$profile" | tail -1)"
percent="$(echo "$total_line" | grep -oE '[0-9]+\.[0-9]+%' | tr -d '%')"

if [ -z "$percent" ]; then
  echo "[coverage] ERRO: nao consegui extrair o percentual de:" >&2
  echo "  $total_line" >&2
  exit 1
fi

echo "[coverage] cobertura de instrucoes: ${percent}% (piso: ${MIN_COVERAGE}%)"

# Comparacao com awk porque o bash so faz aritmetica inteira e o percentual
# tem casa decimal.
if awk -v p="$percent" -v m="$MIN_COVERAGE" 'BEGIN { exit !(p < m) }'; then
  echo "[coverage] FALHOU: ${percent}% esta abaixo do piso de ${MIN_COVERAGE}%" >&2
  echo "[coverage] escreva teste para o codigo novo, ou justifique e ajuste MIN_COVERAGE" >&2
  exit 1
fi

echo "[coverage] OK"

# Os 10 pacotes menos cobertos, para orientar onde escrever teste em seguida.
# Informativo: nao falha o build.
echo
echo "[coverage] pacotes com menor cobertura:"
go tool cover -func="$profile" \
  | grep -v '^total:' \
  | awk '{ print $1, $NF }' \
  | sed 's|/[^/]*\.go:[0-9]*:||' \
  | awk '{ gsub("%","",$2); soma[$1] += $2; n[$1]++ }
         END { for (p in soma) printf "  %6.1f%%  %s\n", soma[p]/n[p], p }' \
  | sort -n \
  | head -10
