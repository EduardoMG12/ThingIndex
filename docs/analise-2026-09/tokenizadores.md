# Tokenizadores e normalizacao duplicados (parecer B6)

Estado registrado, sem correcao de codigo. Unificar os tokenizadores mudaria o
que o BM25 e a cobertura enxergam, e portanto os numeros do artigo; isso exige
um experimento proprio, com antes/depois nas tabelas, e nao cabe numa rodada de
correcoes de engenharia.

## Onde cada um vive

| Funcao | Arquivo | Separa em | Acento | Uso |
|---|---|---|---|---|
| `retrieval.Tokenize` | `api/internal/retrieval/lexical.go` | tudo que nao e letra/digito (inclui hifen) | nao dobra | dentro do BM25 (indice e consulta) |
| `retrieval.FoldDiacritics` | `api/internal/retrieval/lexical.go` | nao tokeniza | dobra | texto das paginas e consulta antes do BM25 |
| `tokenizeRetrievalText` | `api/internal/usecase/collection_retrieval.go` | espaco e `, . ; : ? !`; mantem hifen | nao dobra | consulta de producao, pontuacao por substring, `expandNumberVariants` |
| `connectionTokens` | `api/internal/usecase/node_connections.go` | `FoldDiacritics` + `tokenizeRetrievalText` + trim | dobra | conexoes entre trechos, `significantQueryTokens` (cobertura do chat) |
| `tokenizeNodeText` | `api/internal/indexing/tree_builder.go` | regra propria | propria | construcao da arvore na indexacao |
| `foldLatinRune` | `api/internal/indexing/helpers.go` | nao tokeniza | segunda implementacao de dobra | indexacao |

## Divergencia concreta (a pergunta 15 da banca)

`expandNumberVariants` gera a variante singular de termos hifenizados
("pre-requisitos" tambem vira "pre-requisito"), porque `tokenizeRetrievalText`
mantem o hifen. Em `bm25Scores` a lista expandida e juntada em texto e passa de
novo por `retrieval.Tokenize` dentro do BM25, que separa no hifen. O efeito real:
a variante hifenizada nunca chega ao BM25 como um termo so; chegam "pre" e
"requisito" (e "requisitos" do original). O singular ainda ajuda, mas pela
metade que sobra depois do hifen, e nao pela regra que o comentario descreve.

A deteccao de documento orfao tambem esta em dois lugares:
`documentCatalogUseCase.shouldMarkDocumentAsOrphaned`
(`usecase/document_usecase.go`) e `shouldMarkAsOrphaned`
(`processing/document_processor.go`).

## Correcao proposta (fora desta rodada)

1. Pacote unico `internal/textnorm` com `Fold` e `Tokenize`, usado na
   indexacao, na consulta e na cobertura, com teste de propriedade
   (`Tokenize(Fold(x))` idempotente).
2. Decidir explicitamente a regra do hifen (manter como termo, separar, ou
   indexar os dois) e medir o efeito com `cmd/retrieval-eval` e o gabarito
   antes de trocar.
3. Extrair a deteccao de orfao para uma funcao compartilhada.
