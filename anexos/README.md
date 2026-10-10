# Anexos do artigo

Anexos do artigo de TCC *ThingIndex* (Charles Eduardo Mello Guimarães, IFPR
Palmas). Seguindo a orientação do colegiado, os diagramas, as telas do protótipo e
as tabelas completas ficam neste repositório, e o artigo cita o link de cada um.
Todos os PDFs estão em português e foram gerados a partir de arquivos do próprio
repositório pelo script `scripts/gerar-anexos.py`.

| Anexo | Arquivo | O que contém | Citado em |
|---|---|---|---|
| 1 | [A1-requisitos-funcionais.pdf](A1-requisitos-funcionais.pdf) | Requisitos funcionais RF01–RF12 e onde cada um está implementado; requisitos não funcionais | 6.1 Requisitos |
| 2 | [A2-casos-de-uso.pdf](A2-casos-de-uso.pdf) | Diagrama de casos de uso, derivado das rotas da API | 6.1 Requisitos |
| 3 | [A3-sequencia-envio-e-pergunta.pdf](A3-sequencia-envio-e-pergunta.pdf) | Diagrama de sequência do envio e indexação de um documento e de uma pergunta à coleção | 6.2 Arquitetura |
| 4 | [A4-esquema-proveniencia.pdf](A4-esquema-proveniencia.pdf) | Recorte do esquema, do usuário até as seis tabelas de estrutura extraída (linha contínua: chave estrangeira; tracejada: posse por `user_id`) | 6.3 Modelo de dados e proveniência |
| 5 | [A5-fluxo-de-recuperacao.pdf](A5-fluxo-de-recuperacao.pdf) | Fluxo de recuperação: carga em número fixo de consultas, BM25 por tipo de unidade e seleção de fontes | 6.5 Recuperação e chat |
| 6 | [A6-telas-do-prototipo.pdf](A6-telas-do-prototipo.pdf) | Telas do protótipo: *workspace* de uma coleção e tela de operações | 6.6 Interface |
| 7 | [A7-tabelas-completas-de-recuperacao.pdf](A7-tabelas-completas-de-recuperacao.pdf) | Tabelas completas de recuperação: desempenho com intervalos de confiança, variantes de consulta, consultas negativas, limiar, as comparações pareadas, determinismo do denso, índices aproximados e proveniência | 8.1–8.3 |
| 8 | [A8-alteracoes-pos-medicao.pdf](A8-alteracoes-pos-medicao.pdf) | Registro de toda alteração de gabarito ou critério feita depois de medir, com data, motivo e efeito nos números | 5 |
| 9 | [A9-esquema-do-banco.pdf](A9-esquema-do-banco.pdf) | Esquema completo do banco (25 tabelas), com colunas, tipos e referências, gerado dos modelos da aplicação | 6.3 Modelo de dados e proveniência |

## Dados e código

- [`docs/article/results/`](../docs/article/results/) — resultados brutos, uma linha por pergunta e repetição.
- [`docs/article/tables/`](../docs/article/tables/) — tabelas LaTeX geradas dos CSV (fonte do Anexo 7).
- [`docs/prds/pre-registro-expansao-consulta.md`](../docs/prds/pre-registro-expansao-consulta.md) — pré-registro do experimento de expansão de consulta.
- [`docs/prds/pre-registro-perguntas-geradas.md`](../docs/prds/pre-registro-perguntas-geradas.md) — pré-registro do experimento de perguntas geradas.
- [`docs/schema/thingindex.dbml`](../docs/schema/thingindex.dbml) — esquema completo em DBML (fonte do Anexo 9).
- Verificações do artigo: [`check-article-citations.sh`](../scripts/check-article-citations.sh), [`check-article-claims.sh`](../scripts/check-article-claims.sh), [`check-article-floats.py`](../scripts/check-article-floats.py), [`check-article-numbers.py`](../scripts/check-article-numbers.py).

## Como regerar

```sh
python3 scripts/gerar-anexos.py
```

Requer `chromium` no `PATH`. As imagens de origem ficam em
`docs/article/anexos/diagramas/` e `docs/article/anexos/telas/`.
