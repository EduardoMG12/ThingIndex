#!/usr/bin/env bash
# Verifica afirmacoes do artigo que sao FALSIFICAVEIS EM SEGUNDOS numa arguicao.
#
# Nao sao erros de digitacao: cada uma e uma afirmacao verificavel que, se
# falsa, custa credibilidade justamente no momento em que ela mais importa.
#
#   1. "repositorio publico" / "open source" enquanto o repo esta privado
#   2. tempo futuro em secao que descreve trabalho JA REALIZADO
#   3. promessa de metrica na Metodologia que os Resultados nao usam
#
# O terceiro e o mais traicoeiro: a Metodologia dizia que a metrica principal
# "sera a taxa de acerto" e os Resultados reportam P@1, MRR e nDCG. Um avaliador
# que le as duas secoes na ordem encontra a contradicao sozinho.
set -euo pipefail

cd "$(dirname "$0")/.."

ARTIGO="docs/article/artigo.tex"
falhas=0

# --- 1. Afirmacao de que o repositorio JA ESTA acessivel -------------------
# Distincao importante: "plataforma open source" e declaracao de INTENCAO de
# licenciamento e e legitima com o repo ainda privado -- o artigo inclusive diz
# que "o repositorio sera mantido publico apos a conclusao do TCC". O que nao
# pode e afirmar que o codigo JA ESTA disponivel, porque isso e verificavel em
# segundos e seria encontrado falso.
repo_privado="desconhecido"
if command -v gh >/dev/null 2>&1; then
  repo_privado=$(gh repo view --json isPrivate -q '.isPrivate' 2>/dev/null || echo "desconhecido")
fi

if [[ "$repo_privado" == "true" ]]; then
  while IFS= read -r linha; do
    echo "FALHA: o repositorio esta PRIVADO, mas o artigo afirma disponibilidade atual:" >&2
    echo "  ${linha:0:160}..." >&2
    falhas=$((falhas + 1))
  done < <(grep -n "repositório público\|disponível publicamente\|publicamente disponível\|a partir do repositório público" "$ARTIGO" || true)
fi

# --- 2. Tempo futuro em secoes de trabalho realizado -----------------------
# A Proposta (TCC I) legitimamente falava no futuro. O artigo final descreve
# trabalho concluido, e frases no futuro denunciam texto nao revisado.
futuros=(
  "será realizado"
  "serão aplicados"
  "será medida"
  "será medido"
  "será verificada"
  "consistirá em"
  "A métrica principal será"
)

for termo in "${futuros[@]}"; do
  while IFS= read -r linha; do
    echo "FALHA: tempo futuro descrevendo trabalho ja realizado:" >&2
    echo "  $linha" >&2
    falhas=$((falhas + 1))
  done < <(grep -n "$termo" "$ARTIGO" || true)
done

# --- 3. Metrica prometida na Metodologia e ausente dos Resultados ----------
# A promessa original era "a metrica principal sera a taxa de acerto", enquanto
# os Resultados reportam o painel de RI. Procura-se a PROMESSA, nao a mencao:
# o texto atual cita "taxa de acerto" justamente para explicar por que ela foi
# rejeitada, e isso e argumentacao legitima.
if grep -qE "métrica principal será a taxa de acerto|A métrica principal será" "$ARTIGO"; then
  echo "FALHA: o artigo promete 'taxa de acerto' como metrica principal, mas os Resultados reportam P@1/MRR/nDCG" >&2
  echo "  (contradicao entre Metodologia e Resultados, verificavel na leitura sequencial)" >&2
  falhas=$((falhas + 1))
fi

# --- 4. Metricas de auditabilidade precisam aparecer no texto --------------
# Elas sao implementadas, testadas e citadas como contribuicao. Se sumirem do
# texto, o artigo volta a prometer auditabilidade sem medi-la.
for termo in "TPC" "TCV" "proveniência completa"; do
  if ! grep -q "$termo" "$ARTIGO"; then
    echo "FALHA: '$termo' desapareceu do artigo -- a auditabilidade voltou a ser afirmada sem medicao" >&2
    falhas=$((falhas + 1))
  fi
done

# --- Afirmacao de que o consumo de tokens e descartado ----------------------
# O campo usage passou a ser capturado (tokenUsage / TokenUsageReporter em
# api/internal/pkg/provider). Se o texto voltar a dizer que ele e descartado, a
# prosa contradiz o codigo, e o avaliador que abrir o repositorio ve a
# divergencia.
if grep -qEi "usage[^.]{0,80}(descartad|ignorad)" "$ARTIGO"; then
  echo "FALHA: o artigo afirma que o campo usage e descartado, mas ele e capturado." >&2
  echo "  Ver tokenUsage e TokenUsageReporter em api/internal/pkg/provider/." >&2
  falhas=$((falhas + 1))
fi

echo "repositorio privado: $repo_privado"

if [[ $falhas -gt 0 ]]; then
  echo >&2
  echo "$falhas afirmacao(oes) falsificavel(is) no artigo." >&2
  exit 1
fi

echo
echo "OK: nenhuma afirmacao facilmente falsificavel encontrada."
