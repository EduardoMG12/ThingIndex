#!/usr/bin/env bash
# Conta os testes do repositorio e escreve docs/article/tables/contagem-testes.tex,
# que o artigo inclui na Secao de verificacao. A banca achou 131/653 no texto
# quando o repositorio tinha 148/694: numero escrito a mao envelhece. Com o
# arquivo gerado, o job 'artigo' do CI regenera e compara (diff), entao um
# teste novo sem regenerar quebra o build.
#
# Criterios (os mesmos que o texto declara):
#   arquivos de teste Go   = *_test.go em api/
#   funcoes de teste Go    = linhas '^func Test' nesses arquivos
#   integracao             = arquivos com build tag 'integration'
#   integracao no CI       = arquivos de integracao dos pacotes que o job
#                            backend-integration de .github/workflows/ci.yml roda
#   frontend unitario      = *.test.ts(x) em frontend/src
#   interface em navegador = arquivos *.spec.ts em frontend/tests/e2e
set -euo pipefail
cd "$(dirname "$0")/.."

go_arquivos=$(git ls-files 'api/*_test.go' | wc -l)
go_funcoes=$(git ls-files 'api/*_test.go' | xargs grep -h '^func Test' | wc -l)
integracao=$(git ls-files 'api/*_test.go' | xargs grep -l '^//go:build.*integration' | wc -l)

# Pacotes que o CI roda com -tags=integration (extraidos do proprio ci.yml).
pacotes_ci=$(grep -oE 'go test[^|]*-tags=integration[^|]*' .github/workflows/ci.yml \
  | grep -oE '\./[A-Za-z0-9_/.-]+' | sed 's#^\./##; s#/\.\.\.$##; s#/$##' | sort -u)
integracao_ci=0
for f in $(git ls-files 'api/*_test.go' | xargs grep -l '^//go:build.*integration'); do
  dir=$(dirname "${f#api/}")
  if grep -qxF "$dir" <<<"$pacotes_ci"; then
    integracao_ci=$((integracao_ci + 1))
  fi
done

front_unit=$(git ls-files 'frontend/src/*.test.ts' 'frontend/src/*.test.tsx' | wc -l)
front_nav=$(git ls-files 'frontend/tests/e2e/*.spec.ts' | wc -l)
front_nav_casos=$(git ls-files 'frontend/tests/e2e/*.spec.ts' | xargs grep -hcE '^\s*test\(' | paste -sd+ | bc)

milhar() { printf "%s" "$1" | sed ':a;s/\B[0-9]\{3\}\>/.&/;ta'; }

cat > docs/article/tables/contagem-testes.tex <<EOF
% GERADO POR scripts/contagem-testes.sh -- nao edite a mao.
\\newcommand{\\testesGoArquivos}{$(milhar "$go_arquivos")}
\\newcommand{\\testesGoFuncoes}{$(milhar "$go_funcoes")}
\\newcommand{\\testesIntegracao}{$integracao}
\\newcommand{\\testesIntegracaoCI}{$integracao_ci}
\\newcommand{\\testesFrontUnit}{$front_unit}
\\newcommand{\\testesFrontNav}{$front_nav}
\\newcommand{\\testesFrontNavCasos}{$front_nav_casos}
EOF
cat docs/article/tables/contagem-testes.tex
