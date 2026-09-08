#!/usr/bin/env bash
# Verifica a consistencia entre o texto do artigo e as tabelas geradas.
#
# Tres defeitos que o LaTeX NAO reporta como erro, e que por isso sobrevivem ate
# a defesa:
#
#   1. \ref{tab:x} sem \label correspondente  -> sai "??" no PDF
#   2. \input{tables/x} de arquivo inexistente -> falha so na compilacao final
#   3. tabela inserida e nunca referenciada     -> flutua no texto sem ancora
#
# O terceiro nao quebra nada tecnicamente, mas em texto academico e defeito de
# escrita: toda tabela precisa ser chamada pelo corpo do texto.
set -euo pipefail

cd "$(dirname "$0")/.."

ARTIGO="docs/article/article-overleaf.md"
TABELAS="docs/article/tables"

if [[ ! -f "$ARTIGO" ]]; then
  echo "erro: $ARTIGO nao encontrado" >&2
  exit 1
fi

falhas=0

# --- 1. Todo \ref{tab:...} tem \label correspondente -----------------------
# Os labels podem estar nos .tex gerados OU inline no proprio artigo.
labels_disponiveis=$(
  {
    grep -ho 'label{tab:[^}]*}' "$TABELAS"/*.tex 2>/dev/null || true
    grep -ho 'label{tab:[^}]*}' "$ARTIGO" || true
  } | sed 's/label{//; s/}//' | sort -u
)

refs_usadas=$(grep -o 'ref{tab:[^}]*}' "$ARTIGO" | sed 's/ref{//; s/}//' | sort -u)

for ref in $refs_usadas; do
  if ! grep -qx "$ref" <<<"$labels_disponiveis"; then
    echo "FALHA: \\ref{$ref} nao tem \\label correspondente (sairia '??' no PDF)" >&2
    falhas=$((falhas + 1))
  fi
done

# --- 2. Todo \input{tables/...} aponta para arquivo existente --------------
while read -r nome; do
  [[ -z "$nome" ]] && continue
  if [[ ! -f "$TABELAS/$nome.tex" ]]; then
    echo "FALHA: \\input{tables/$nome} mas $TABELAS/$nome.tex nao existe" >&2
    falhas=$((falhas + 1))
  fi
done < <(grep -o 'input{tables/[^}]*}' "$ARTIGO" | sed 's|input{tables/||; s/}//' | sort -u)

# --- 3. Toda tabela inserida e referenciada no corpo do texto --------------
# Uma tabela que aparece sem ninguem cita-la e defeito de escrita academica.
while read -r nome; do
  [[ -z "$nome" ]] && continue
  arquivo="$TABELAS/$nome.tex"
  [[ -f "$arquivo" ]] || continue

  label=$(grep -o 'label{tab:[^}]*}' "$arquivo" | head -1 | sed 's/label{//; s/}//')
  [[ -z "$label" ]] && continue

  if ! grep -q "ref{$label}" "$ARTIGO"; then
    echo "FALHA: tabela '$nome' e inserida mas nunca referenciada por \\ref{$label}" >&2
    falhas=$((falhas + 1))
  fi
done < <(grep -o 'input{tables/[^}]*}' "$ARTIGO" | sed 's|input{tables/||; s/}//' | sort -u)

# --- 4. Nenhum label duplicado entre as tabelas ----------------------------
# Duas tabelas com o mesmo label fazem todo \ref apontar para a ultima; o LaTeX
# so avisa 'multiply defined' e compila.
duplicados=$(grep -ho 'label{tab:[^}]*}' "$TABELAS"/*.tex 2>/dev/null | sort | uniq -d || true)
if [[ -n "$duplicados" ]]; then
  echo "FALHA: labels duplicados entre tabelas geradas:" >&2
  echo "$duplicados" | sed 's/^/  /' >&2
  falhas=$((falhas + 1))
fi

echo "tabelas geradas:      $(ls "$TABELAS"/*.tex 2>/dev/null | wc -l)"
echo "referencias no texto: $(wc -l <<<"$refs_usadas")"

if [[ $falhas -gt 0 ]]; then
  echo >&2
  echo "$falhas problema(s) de consistencia entre texto e tabelas." >&2
  exit 1
fi

echo
echo "OK: toda referencia tem label, todo input existe, toda tabela e citada."
