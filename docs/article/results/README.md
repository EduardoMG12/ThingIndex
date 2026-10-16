# Resultados brutos

Uma linha por pergunta (e por repetição, quando houve). Esta pasta guarda
também execuções antigas: nem todo arquivo daqui alimenta o artigo. Nenhum
arquivo foi apagado ou renomeado, porque alguns scripts ainda os leem.

## De onde vem cada número do artigo

| Seção do artigo | Arquivo | Script que calcula ou confere |
|---|---|---|
| 8.1 a 8.3, C1 (busca, negativas, estabilidade) | `results.csv` | `scripts/analyze.py`, `scripts/check-article-numbers.py` |
| 8.3, índices do pgvector | `ann-pgvector.csv` | `scripts/ann-pgvector.py` |
| 8.4, origem no índice | `docs/article/tables/provenance.tex` (gerada pelo Go) | `scripts/check-article-numbers.py` |
| 8.5, ensino: acerto, tokens, página, latência | `generation-results-T-ordem.csv` (braço T) e `generation-results-F-document.csv` (braço F); `folios-si.json` (número impresso das páginas) | `scripts/numeros-geracao.py`, `scripts/proveniencia-geracao.py` |
| 8.5, ajuste de continuação de página (uma repetição) | `generation-results-T-continuacao.csv` | -- |
| 8.6, legislação | `generation-results-T-licitacoes.csv` | `scripts/check-article-numbers.py` |
| 8.7 e 8.9, bateria sem e com expansão | `expansao/off-*.csv` (sem) e `expansao/on-*.csv` (com), um por acervo | `scripts/numeros-geracao.py`, `scripts/redacao-gabarito-original.py` |
| 8.8, enriquecimento | `enriquecimento-reexecucao/braco-seguranca-trabalho-*.csv` | `scripts/numeros-geracao.py` |
| 8.10, edital municipal | `bateria-prefeitura.csv` | `scripts/auditoria-padroes.py` |
| Ameaças, teste com outro modelo | `generation-results-F-document-gpt-oss-120b.csv` | `scripts/check-article-numbers.py` |

As correções feitas depois de medir estão em `alteracoes-pos-medicao.md`
(fonte do Anexo A8).

## Arquivos históricos (nenhum número do artigo os usa)

- `bateria-seguranca-trabalho.csv`, `bateria-produto.csv` e
  `bateria-licitacoes.csv`: primeira execução da bateria, corrigida com um
  gabarito anterior (149 de 213 respostas certas). Não foram recorrigidos
  depois (registro 8 de `alteracoes-pos-medicao.md`). Os números da bateria
  vêm de `expansao/off-*.csv`. `bateria-prefeitura.csv`, ao contrário, é a
  fonte do edital municipal.
- `resumo-bateria.md`: saída antiga de `scripts/resumo-bateria.py`, com o
  gabarito original e um bug já corrigido (o próprio arquivo explica).
- `antes-correcao-negativas/`: cópias de antes da correção do padrão das
  negativas, descrita em `correcao-negativas.md` (registro 2).
- `braco-seguranca-trabalho-*.csv` (na raiz): execução anterior do
  enriquecimento; os números vêm de `enriquecimento-reexecucao/`.
- `generation-results-T-final.csv`: execução anterior de T.
  `scripts/gerar-anexos.py` lê dele só os títulos dos documentos
  citados, para montar o Anexo A10.
- Demais `generation-results-*.csv` (`T`, `T-10docs`, `T-enriquecido`,
  `T-retrieval`, `T-retrieval-depois`, `T-produto*`, `T-seguranca*`,
  `T-licitacoes-v2`, `F-collection`): execuções de desenvolvimento ou
  exploratórias.
