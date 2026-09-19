#!/usr/bin/env bash
#
# Roda o gabarito de cada tema contra a sua coleção e consolida tudo numa tabela.
#
# Existe porque medir três temas na mão significava três comandos e um script
# Python improvisado para juntar os CSV. Agora é um comando, o resultado é
# reproduzível e os CSV ficam versionados em docs/article/results/.
#
#   ./scripts/bateria-temas.sh                  # roda tudo, 1 repetição
#   ./scripts/bateria-temas.sh -r 3             # 3 repetições
#   ./scripts/bateria-temas.sh -t licitacoes    # só um tema
#   ./scripts/bateria-temas.sh -s               # só resume os CSV que já existem
#
# Exige a stack de pé e a conta de demonstração com provedor de LLM configurado.
set -euo pipefail

RAIZ="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SAIDA="$RAIZ/docs/article/results"
EMAIL="${THINGINDEX_BATERIA_EMAIL:-aluno.demo@thingindex.local}"
SENHA="${THINGINDEX_BATERIA_SENHA:-ThingIndex2026}"
API="${THINGINDEX_BATERIA_API:-http://localhost:8000/api/v1}"

# tema:coleção:pasta-de-pdfs
TEMAS=(
  "seguranca-trabalho:147:examples/seguranca-trabalho"
  "produto:148:examples/produto"
  "licitacoes:146:examples/licitacoes"
)

REPETICOES=1
SO_RESUMO=0
FILTRO=""

while getopts "r:t:sh" opt; do
  case "$opt" in
    r) REPETICOES="$OPTARG" ;;
    t) FILTRO="$OPTARG" ;;
    s) SO_RESUMO=1 ;;
    h) sed -n '2,20p' "${BASH_SOURCE[0]}"; exit 0 ;;
    *) exit 2 ;;
  esac
done

mkdir -p "$SAIDA"

if [ "$SO_RESUMO" -eq 0 ]; then
  # A rota de chat tem limitador de 20 pedidos por minuto por usuário. O harness
  # reespera sozinho no 429, então a bateria demora mais do que a soma das
  # latências; isso é esperado e não é erro.
  for entrada in "${TEMAS[@]}"; do
    IFS=: read -r tema colecao pdfs <<< "$entrada"
    [ -n "$FILTRO" ] && [ "$FILTRO" != "$tema" ] && continue

    gabarito="$RAIZ/api/internal/evalcorpus/testdata/gabarito-$tema.json"
    [ -f "$gabarito" ] || { echo "sem gabarito para $tema, pulando"; continue; }

    echo "==> $tema (coleção $colecao, $REPETICOES repetição(ões))"
    ( cd "$RAIZ/api" && go run ./cmd/generation-eval \
        -gabarito "$gabarito" \
        -pdf-dir "$RAIZ/$pdfs" \
        -arms T -repeats "$REPETICOES" -collection "$colecao" \
        -email "$EMAIL" -password "$SENHA" -api "$API" \
        -out "$SAIDA/bateria-$tema.csv" )
  done
fi

python3 "$RAIZ/scripts/resumo-bateria.py" "$SAIDA" ${FILTRO:+--tema "$FILTRO"}
