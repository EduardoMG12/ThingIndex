#!/usr/bin/env bash
# Verifica a integridade das citacoes do artigo do TCC.
#
# Motivo: em 2026-08 a chave `schymik2015benefits` — citada na PRIMEIRA FRASE da
# introducao — nao existia no references.bib versionado, embora o PDF compilado
# a renderizasse corretamente. Ou seja: o .bib do repositorio estava
# dessincronizado do Overleaf, e compilar o artigo versionado produziria um
# documento com citacao quebrada.
#
# O PR #9 versionou docs/ justamente para servir de backup recuperavel. Um
# backup que nao compila nao e backup. Este script fecha esse buraco.
#
# Uso:
#   ./scripts/check-article-citations.sh
#
# Sai com 1 se houver citacao sem entrada correspondente no .bib.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARTICLE="${REPO_ROOT}/docs/article/article-overleaf-v2.md"
BIB="${REPO_ROOT}/docs/article/references.bib"

if [[ ! -f "${ARTICLE}" ]]; then
    echo "erro: artigo nao encontrado em ${ARTICLE}" >&2
    exit 1
fi

if [[ ! -f "${BIB}" ]]; then
    echo "erro: bibliografia nao encontrada em ${BIB}" >&2
    exit 1
fi

CITED="$(mktemp)"
DEFINED="$(mktemp)"
trap 'rm -f "${CITED}" "${DEFINED}"' EXIT

# Chaves citadas: \cite{a,b} e \cite{c} — ignorando linhas comentadas com %.
grep -v '^\s*%' "${ARTICLE}" \
    | grep -oP '\\cite\{\K[^}]+' \
    | tr ',' '\n' \
    | tr -d ' ' \
    | grep -v '^$' \
    | sort -u > "${CITED}"

# Chaves definidas no .bib: @article{chave, / @book{chave,
grep -oP '^@\w+\{\K[^,]+' "${BIB}" | tr -d ' ' | sort -u > "${DEFINED}"

ORPHANS="$(comm -23 "${CITED}" "${DEFINED}")"
UNUSED="$(comm -13 "${CITED}" "${DEFINED}")"

echo "citacoes no artigo:      $(wc -l < "${CITED}")"
echo "entradas no references:  $(wc -l < "${DEFINED}")"

STATUS=0

if [[ -n "${ORPHANS}" ]]; then
    echo
    echo "FALHA: chaves citadas que NAO existem no references.bib" >&2
    echo "Compilar o artigo assim produz [?] no lugar da citacao." >&2
    echo "${ORPHANS}" | sed 's/^/  - /' >&2
    echo >&2
    echo "Provavel causa: o references.bib local esta atrasado em relacao ao" >&2
    echo "Overleaf. Baixe a versao atual e commite junto." >&2
    STATUS=1
fi

if [[ -n "${UNUSED}" ]]; then
    echo
    echo "aviso: entradas no .bib que nunca sao citadas (peso morto, nao quebra):"
    echo "${UNUSED}" | sed 's/^/  - /'
fi

if [[ "${STATUS}" -eq 0 ]]; then
    echo
    echo "OK: toda citacao tem entrada correspondente."
fi

exit "${STATUS}"
