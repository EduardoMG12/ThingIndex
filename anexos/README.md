# Anexos

Anexos do Trabalho de Conclusão de Curso *ThingIndex* (Sistemas de Informação, IFPR
Campus Palmas).

| Anexo | Arquivo | Conteúdo |
|---|---|---|
| A1 | [A1-requisitos-funcionais.pdf](A1-requisitos-funcionais.pdf) | Requisitos funcionais e não funcionais |
| A2 | [A2-casos-de-uso.pdf](A2-casos-de-uso.pdf) | Diagrama de casos de uso |
| A3 | [A3-sequencia-envio-e-pergunta.pdf](A3-sequencia-envio-e-pergunta.pdf) | Diagrama de sequência do envio de um documento e de uma pergunta à coleção |
| A4 | [A4-esquema-proveniencia.pdf](A4-esquema-proveniencia.pdf) | Recorte do esquema do banco com o caminho da proveniência |
| A5 | [A5-fluxo-de-recuperacao.pdf](A5-fluxo-de-recuperacao.pdf) | Fluxo de recuperação de contexto |
| A6 | [A6-telas-do-prototipo.pdf](A6-telas-do-prototipo.pdf) | Telas do protótipo: *workspace* de uma coleção e tela de operações |
| A7 | [A7-tabelas-completas-de-recuperacao.pdf](A7-tabelas-completas-de-recuperacao.pdf) | Resultados do teste de busca: por que o teste foi feito, o que cada análise respondeu e o que ela decidiu no projeto; ao final, a tabela completa de medidas com intervalos de confiança e os índices do pgvector |
| A8 | [A8-alteracoes-pos-medicao.pdf](A8-alteracoes-pos-medicao.pdf) | Correções feitas depois de medir: uma tabela com o que mudou, por quê e o número antes e depois, em porcentagem |
| A9 | [A9-esquema-do-banco.pdf](A9-esquema-do-banco.pdf) | Diagrama entidade-relacionamento das 25 tabelas do banco |

## Dados e código

- [`docs/article/results/`](../docs/article/results/): resultados brutos, uma linha por pergunta e repetição.
- [`docs/article/tables/`](../docs/article/tables/): tabelas LaTeX geradas dos resultados, incluindo as que não estão no PDF do A7 (perguntas sem resposta, nota mínima, estabilidade, origem e as 90 comparações estatísticas).
- [`docs/article/results/alteracoes-pos-medicao.md`](../docs/article/results/alteracoes-pos-medicao.md): registro original do Anexo A8 (o script confere que todo número do A8 existe nele).
- [`docs/prds/pre-registro-expansao-consulta.md`](../docs/prds/pre-registro-expansao-consulta.md): pré-registro do experimento de expansão de consulta.
- [`docs/prds/pre-registro-perguntas-geradas.md`](../docs/prds/pre-registro-perguntas-geradas.md): pré-registro do experimento de perguntas geradas.
- [`docs/schema/thingindex.dbml`](../docs/schema/thingindex.dbml): esquema do banco em DBML (fonte do Anexo A9).
- [`docs/article/anexos/diagramas/`](../docs/article/anexos/diagramas/): fontes PlantUML dos diagramas.
- Verificações do artigo: [`check-article-citations.sh`](../scripts/check-article-citations.sh), [`check-article-claims.sh`](../scripts/check-article-claims.sh), [`check-article-floats.py`](../scripts/check-article-floats.py), [`check-article-numbers.py`](../scripts/check-article-numbers.py).

Os PDFs são gerados por [`scripts/gerar-anexos.py`](../scripts/gerar-anexos.py), que requer
`chromium`, `docker` (imagem `plantuml/plantuml`), `rsvg-convert` e `pdftotext`.
