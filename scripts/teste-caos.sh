#!/usr/bin/env bash
# Testes de caos: derruba cada dependencia com o sistema em uso e verifica a
# RECUPERACAO COMPLETA -- nao apenas a reconexao.
#
# Existe porque os quatro defeitos mais graves da rodada de 2026-09 so
# apareceram quando a infraestrutura foi derrubada ou o sistema foi posto sob
# carga. Nenhum teste unitario os teria encontrado:
#
#   1. consumidor nao reconectava ao RabbitMQ (documentos presos em QUEUED)
#   2. publicador nao reconectava (eventos FAILED com o broker ja no ar)
#   3. /health respondia 200 com o Postgres fora
#   4. corrida na criacao de colecao devolvia HTTP 500 sob upload paralelo
#
# A diferenca que importa: o teste unitario prova que O SUPERVISOR RECONECTA;
# este prova que O DOCUMENTO CHEGA A INDEXED depois da queda.
#
# Uso:
#   scripts/teste-caos.sh              # todos os cenarios
#   scripts/teste-caos.sh rabbitmq     # um cenario
set -uo pipefail

RAIZ="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API="${THINGINDEX_CAOS_API:-http://localhost:8000/api/v1}"

# Por padrao usa a stack de desenvolvimento (com hot-reload); no CI, onde a
# imagem e compilada, THINGINDEX_CAOS_COMPOSE aponta so para o compose base.
#
# Importa porque o `air` do modo dev encerra o processo a cada recompilacao, e
# isso confunde a verificacao de que o worker sobreviveu a queda.
if [[ -n "${THINGINDEX_CAOS_COMPOSE:-}" ]]; then
  DC="docker compose $THINGINDEX_CAOS_COMPOSE"
else
  DC="docker compose -f $RAIZ/docker-compose.yml -f $RAIZ/docker-compose.dev.yml"
fi

cd "$RAIZ"
# shellcheck disable=SC1091
set -a; . ./.env; set +a

falhas=0
TRAB=$(mktemp -d)
trap 'rm -rf "$TRAB"' EXIT

relatar() {
  if [[ "$2" == "1" ]]; then
    printf '  ok     %-46s %s\n' "$1" "${3:-}"
  else
    printf '  FALHA  %-46s %s\n' "$1" "${3:-}"
    falhas=$((falhas + 1))
  fi
}

psql_() {
  $DC exec -T db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tAc "$1" 2>/dev/null
}

# esperar_por repete o comando ate a saida bater com o esperado.
#
# Nunca sleep fixo: em runner compartilhado o tempo varia, e sleep fixo produz
# teste intermitente -- que e pior que teste ausente, porque ensina a equipe a
# ignorar falha vermelha.
esperar_por() {
  local descricao="$1" esperado="$2" comando="$3" tentativas="${4:-40}"
  for _ in $(seq 1 "$tentativas"); do
    if [[ "$(eval "$comando" 2>/dev/null | tr -d '[:space:]')" == "$esperado" ]]; then
      return 0
    fi
    sleep 3
  done
  echo "     (timeout esperando: $descricao)" >&2
  return 1
}

autenticar() {
  local email="caos$(date +%s%N)@test.local" senha="SenhaForte#123"
  curl -s -X POST "$API/auth/register" -H 'Content-Type: application/json' \
    -d "{\"name\":\"Caos\",\"email\":\"$email\",\"password\":\"$senha\"}" >/dev/null
  curl -s -X POST "$API/auth/login" -H 'Content-Type: application/json' \
    -d "{\"email\":\"$email\",\"password\":\"$senha\"}" |
    python3 -c 'import json,sys; print(json.load(sys.stdin).get("data",{}).get("token",""))'
}

enviar_documento() {
  local token="$1" titulo="$2" colecao="$3"
  printf '<html><body><h1>1 DAS OBRIGACOES</h1><p>Prestacao continuada com multa e fatura mensal.</p></body></html>' \
    > "$TRAB/doc.html"
  curl -s -X POST "$API/documents/upload" -H "Authorization: Bearer $token" \
    -F "file=@$TRAB/doc.html;type=text/html" \
    -F "title=$titulo.html" -F "collection=$colecao" |
    python3 -c 'import json,sys; print((json.load(sys.stdin).get("data") or {}).get("id",""))' 2>/dev/null
}

status_documento() {
  psql_ "SELECT status FROM documents WHERE id=$1;"
}

# ---------------------------------------------------------------------------
# Cenario 1: RabbitMQ cai com upload em andamento
# ---------------------------------------------------------------------------
cenario_rabbitmq() {
  echo "== 1. RabbitMQ cai com upload em andamento =="
  local token; token=$(autenticar)
  [[ -n "$token" ]] || { relatar "autenticacao" 0 "sem token"; return; }

  local colecao="Caos$(date +%s)"
  local antes; antes=$(enviar_documento "$token" "AntesDaQueda" "$colecao")
  esperar_por "documento base indexado" "INDEXED" "status_documento $antes" 30 ||
    { relatar "linha de base" 0 "documento base nao indexou"; return; }
  relatar "documento indexa antes da queda" 1 "id=$antes"

  $DC stop rabbitmq >/dev/null 2>&1
  sleep 3

  # O outbox precisa ACEITAR o upload mesmo sem broker: documento e evento sao
  # gravados na mesma transacao, e o publicador envia depois.
  local durante; durante=$(enviar_documento "$token" "DuranteAQueda" "$colecao")
  if [[ -n "$durante" ]]; then
    relatar "upload aceito sem o broker (outbox)" 1 "id=$durante"
  else
    relatar "upload aceito sem o broker (outbox)" 0 "upload rejeitado"
  fi

  $DC start rabbitmq >/dev/null 2>&1

  # A prova central: o documento enviado durante a queda precisa chegar a
  # INDEXED sozinho. Antes da correcao ficava QUEUED para sempre.
  if esperar_por "documento da queda indexado" "INDEXED" "status_documento $durante" 50; then
    relatar "documento da queda chega a INDEXED" 1 "sem restart do worker"
  else
    relatar "documento da queda chega a INDEXED" 0 "preso em $(status_documento "$durante")"
  fi

  if esperar_por "outbox drenado" "0" "psql_ \"SELECT count(*) FROM outbox_events WHERE status<>'PUBLISHED';\"" 30; then
    relatar "outbox drena ate zero pendentes" 1
  else
    relatar "outbox drena ate zero pendentes" 0 "restaram eventos nao publicados"
  fi

  # O processo nao pode ter MORRIDO por falha: antes da correcao o consumidor
  # devolvia erro fatal e o binario encerrava com codigo diferente de zero.
  #
  # "Process Exit with Code 0" tambem aparece quando o air recompila apos uma
  # alteracao de codigo, e isso e normal em desenvolvimento -- a primeira versao
  # desta verificacao acusava falha por causa disso. O sinal de crash e o codigo
  # DIFERENTE de zero.
  if $DC logs indexer --since 3m 2>&1 | grep -qE "Process Exit with Code [1-9]"; then
    relatar "worker sobrevive a queda" 0 "indexer encerrou com codigo de erro"
  else
    relatar "worker sobrevive a queda" 1 "nenhuma saida por falha"
  fi

  # E a prova positiva: o supervisor precisa ter registrado a reconexao.
  if $DC logs indexer --since 3m 2>&1 | grep -q "reconectando em"; then
    relatar "supervisor registrou a reconexao" 1
  else
    relatar "supervisor registrou a reconexao" 0 "sem log de reconexao: o consumidor pode ter morrido silenciosamente"
  fi
}

# ---------------------------------------------------------------------------
# Cenario 2: Postgres cai
# ---------------------------------------------------------------------------
cenario_postgres() {
  echo "== 2. Postgres cai =="
  local codigo
  codigo=$(curl -s -o /dev/null -w '%{http_code}' --max-time 10 "$API/health")
  relatar "health responde 200 com tudo no ar" "$([[ "$codigo" == "200" ]] && echo 1 || echo 0)" "http=$codigo"

  $DC stop db >/dev/null 2>&1
  sleep 4

  codigo=$(curl -s -o /dev/null -w '%{http_code}' --max-time 12 "$API/health")
  if [[ "$codigo" == "503" ]]; then
    relatar "health reporta 503 com o banco fora" 1 "http=503"
  else
    relatar "health reporta 503 com o banco fora" 0 "http=$codigo -- orquestrador manteria a instancia no balanceador"
  fi

  # O detalhe do driver (IP interno, porta) nao pode vazar em rota publica.
  local corpo; corpo=$(curl -s --max-time 12 "$API/health")
  if grep -qE '5432|172\.[0-9]+\.[0-9]+\.[0-9]+|connection refused' <<< "$corpo"; then
    relatar "health nao vaza detalhe de infraestrutura" 0 "corpo expoe rede interna"
  else
    relatar "health nao vaza detalhe de infraestrutura" 1
  fi

  $DC start db >/dev/null 2>&1

  if esperar_por "health de volta" "200" "curl -s -o /dev/null -w '%{http_code}' --max-time 12 $API/health" 40; then
    relatar "health volta a 200 sem restart da api" 1
  else
    relatar "health volta a 200 sem restart da api" 0 "pool ficou com conexoes mortas"
  fi
}

# ---------------------------------------------------------------------------
# Cenario 3: MinIO cai
# ---------------------------------------------------------------------------
cenario_minio() {
  echo "== 3. MinIO cai =="
  local token; token=$(autenticar)
  local colecao="CaosS3$(date +%s)"
  local id; id=$(enviar_documento "$token" "Storage" "$colecao")
  esperar_por "documento indexado" "INDEXED" "status_documento $id" 30 >/dev/null

  local codigo
  codigo=$(curl -s -o /dev/null -w '%{http_code}' --max-time 15 \
    "$API/documents/$id/content" -H "Authorization: Bearer $token")
  relatar "download funciona antes da queda" "$([[ "$codigo" == "200" ]] && echo 1 || echo 0)" "http=$codigo"

  $DC stop minio >/dev/null 2>&1
  sleep 4

  codigo=$(curl -s -o /dev/null -w '%{http_code}' --max-time 20 \
    "$API/documents/$id/content" -H "Authorization: Bearer $token")
  if [[ "$codigo" == "200" ]]; then
    relatar "download falha explicitamente sem storage" 0 "http=200 sem o arquivo"
  else
    relatar "download falha explicitamente sem storage" 1 "http=$codigo"
  fi

  $DC start minio >/dev/null 2>&1

  if esperar_por "download de volta" "200" \
      "curl -s -o /dev/null -w '%{http_code}' --max-time 15 $API/documents/$id/content -H 'Authorization: Bearer $token'" 40; then
    relatar "download volta sem restart da api" 1
  else
    relatar "download volta sem restart da api" 0 "cliente ficou com conexao morta"
  fi
}

# ---------------------------------------------------------------------------
# Cenario 4: uploads simultaneos para a mesma colecao
# ---------------------------------------------------------------------------
cenario_concorrencia() {
  echo "== 4. 20 uploads simultaneos para a mesma colecao =="
  local token; token=$(autenticar)
  local colecao="CaosConc$(date +%s)"

  printf '<html><body><h1>1 DAS OBRIGACOES</h1><p>Clausula com multa e fatura.</p></body></html>' \
    > "$TRAB/conc.html"

  : > "$TRAB/codigos.txt"
  for i in $(seq 1 20); do
    curl -s -o /dev/null -w '%{http_code}\n' -X POST "$API/documents/upload" \
      -H "Authorization: Bearer $token" \
      -F "file=@$TRAB/conc.html;type=text/html" \
      -F "title=Conc $i.html" -F "collection=$colecao" >> "$TRAB/codigos.txt" &
  done
  wait

  local criados; criados=$(grep -c '^201$' "$TRAB/codigos.txt")
  if [[ "$criados" == "20" ]]; then
    relatar "20 uploads simultaneos aceitos" 1 "20 x 201"
  else
    relatar "20 uploads simultaneos aceitos" 0 "$(sort "$TRAB/codigos.txt" | uniq -c | tr '\n' ' ')"
  fi

  # Exatamente uma colecao: ids diferentes dividiriam o acervo em duas homonimas.
  local colecoes
  colecoes=$(psql_ "SELECT count(*) FROM collections WHERE name='$colecao';" | tr -d '[:space:]')
  if [[ "$colecoes" == "1" ]]; then
    relatar "exatamente uma colecao criada" 1
  else
    relatar "exatamente uma colecao criada" 0 "$colecoes colecoes homonimas"
  fi
}

restaurar_servicos() {
  $DC start db minio rabbitmq >/dev/null 2>&1
  sleep 5
}

principal() {
  local alvo="${1:-todos}"
  echo "Testes de caos -- API em $API"
  echo

  case "$alvo" in
    rabbitmq) cenario_rabbitmq ;;
    postgres) cenario_postgres ;;
    minio)    cenario_minio ;;
    concorrencia) cenario_concorrencia ;;
    todos)
      # Em sequencia, nunca em paralelo: derrubar o Postgres afeta o health e a
      # propria verificacao dos outros cenarios.
      cenario_rabbitmq; echo
      cenario_postgres; echo
      cenario_minio; echo
      cenario_concorrencia
      ;;
    *) echo "cenario desconhecido: $alvo" >&2; exit 2 ;;
  esac

  restaurar_servicos

  echo
  if [[ "$falhas" -gt 0 ]]; then
    echo "$falhas verificacao(oes) falharam."
    exit 1
  fi
  echo "Todos os cenarios de caos passaram."
}

principal "$@"
