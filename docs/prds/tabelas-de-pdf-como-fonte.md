# Tabelas de PDF como fonte: linhas endereçáveis por página no retrieval e no chat

**Status:** em execução (2026-09-09), branch `feat/tabelas-de-pdf` a partir de `feat/experimento-geracao` (PR #66, sem merge)
**Data:** 2026-09-09
**Origem:** veredito do conselho e resultado do experimento de geração (`docs/analise-2026-09/experimento-geracao.md`). O braço F lê o mesmo `pdftotext` que o indexador e acerta 90% das perguntas de tabela; o ThingIndex erra quase todas porque o pipeline de PDF nunca emite `table_sections` (só HTML e planilha emitem), a linha da matriz vira prosa embaralhada de página, e `buildChatSources` só conhece seção e página.

## Goals

(1) O indexador determinístico detecta tabelas em PDF a partir das caixas de palavra do `pdftotext -bbox-layout` e grava uma `TableSection` por tabela **por página**, com `page_number`, cabeçalho e linhas com células multilinha remontadas. (2) Cada linha de tabela é uma unidade de recuperação com o cabeçalho colado ("Disciplina: SIS07302 - Estrutura de Dados I; Pré-requisito: SIS07206"), pontuada por BM25 como as demais, e vira fonte de chat de primeira classe com (documento, página, tabela, linha). (3) Métrica determinística, sem LLM: Recall@6 do retrieval sobre os estratos `tabela` e `calendario` do gabarito, antes e depois, medida pelo `cmd/generation-eval -arms T` (as fontes saem antes do modelo).

Pronto quando: teste com a Matriz real (`examples/faculdade/matriz-curricular-si-palmas-2023.pdf`) recupera a linha de Estrutura de Dados I com pré-requisito SIS07206 e página 1; teste com o Calendário recupera a linha "Rematrícula dos estudantes veteranos" em fevereiro na página 1; Recall@6 de T no estrato `tabela` sobe de 42,9% (9 de 21, medido em 2026-09-09; no `calendario` já é 100% porque o documento tem 5 páginas, e a falha ali é o modelo ler a grade) e o número novo fica registrado em Discoveries.

## Requirements

Extração (`internal/indexing/pdf_tables.go`)
- [ ] O parser do bbox-layout guarda as palavras com `xMin/xMax/yMin/yMax` (hoje só o texto).
- [ ] Linhas visuais: palavras agrupadas por `yMin` com tolerância de 2 pt; células: palavras separadas por folga horizontal maior que `max(12 pt, 2,5 × largura média de caractere da linha)`.
- [ ] Tabela: sequência de 3+ linhas visuais consecutivas com 3+ células cujas bordas esquerdas caem nas mesmas colunas (tolerância 8 pt). Colunas = agrupamento das bordas esquerdas de célula; cabeçalho = primeira linha da sequência.
- [ ] Continuação: linha visual sem nenhuma célula "âncora" (célula curta numérica, código `[A-Z]{2,}\d{3,}` ou faixa `\d+ ?[ae-] ?\d+`) e com todas as células dentro de colunas já preenchidas na linha anterior é anexada à linha anterior, célula a célula.
- [ ] Título: a linha de uma célula só imediatamente acima da tabela (até 30 pt), quando houver ("Janeiro 2026", "Tabela 4 - ...").
- [ ] Uma `TableSection` por tabela por página; `page_number` preenchido; `metadata.box` com a caixa da tabela; `metadata.source = "pdf-bbox"`.
- [ ] Testes com os PDFs reais do demo (Matriz p. 1: linha de Estrutura de Dados I com `SIS07206` na coluna Pré-requisito; Calendário p. 1: linha da rematrícula; PPC p. 56: linha de Algoritmos com `SIS05101` e `120`) e um negativo: página de prosa do PPC não gera tabela.

Modelo
- [ ] `TableSection.PageNumber *int` (coluna, AutoMigrate); DBML regenerado.

Retrieval e chat
- [ ] `CollectionTableRowHit{DocumentID, TableID, RowIndex, PageNumber, Title, Snippet, Score}` em `CollectionRetrievalResult`; texto da unidade = título + `cabeçalho: célula` para cada célula; BM25 na mesma passada que páginas e seções; `keep(kind "table")` com o peso atual (×5); até 8 hits.
- [ ] `renderTableRowsForRetrieval(rows, 3)` deixa de limitar: a pontuação de documento por tabela usa a melhor linha.
- [ ] `buildChatSources`: linhas primeiro (até 3), depois seções (até 3), depois páginas até 6. Fonte de linha tem `Page` da tabela e `ID = "<doc>:<page>:table-<id>-row-<i>"`, snippet = linha renderizada.
- [ ] O corpo que vai ao prompt para fonte de linha é a própria linha renderizada com o título da tabela (não a página inteira); `ChatSource.Body` interno, não exposto no DTO.
- [ ] Testes: unidade de linha vence a página da matriz para "pré-requisito de Estrutura de Dados I"; fonte de linha aparece primeiro e com página.

Medição
- [ ] Reindexar os 5 documentos do demo (`POST /documents/:id/retry`), rodar `cmd/generation-eval -arms T -repeats 1` (sem crédito o modelo falha, mas as fontes saem), comparar Recall@6 por estrato com `generation-results-T-retrieval.csv` (antes) e registrar.

## Non-Goals

- `table_rows`/`table_cells` como tabelas próprias com FK: a linha continua em `rows` JSONB dentro de `table_sections`, agora com página. Migrar para linhas relacionais é o passo seguinte, quando a extração provar que vale (a célula ganha `bbox` aí).
- Tabelas de OCR (blocos sem caixa de palavra).
- Tabelas que atravessam páginas como uma só entidade: cada página é uma tabela; cabeçalho repetido resolve a Matriz e o Calendário.
- Mudar o prompt do chat além do corpo da fonte de linha.
- Rodar o braço T com LLM (sem crédito).

## Constraints

- `parsePDFBBoxLayout` já lê `page > flow > block > line > word`; a `<word>` traz `xMin/yMin/xMax/yMax`. O parser hoje descarta as coordenadas de palavra.
- `buildIndexingResult(documentID, headings, paragraphs, tables)` já aceita tabelas; o PDF passa `nil`.
- O resultado é persistido por `tx.Create(&result.TableSections)` no repositório de processamento; `TableSection` já está em `ManagedModels()` (AutoMigrate).
- O retrieval de coleção recebe `tables []models.TableSection` (`buildCollectionRetrieval`), e `CollectionContext` já carrega `TableSections`.
- Constituição I: fonte sem página é bug; por isso a tabela é por página. Constituição II: tudo determinístico; nenhuma LLM entra nesta rodada.
- Recall@6 "antes", medido em 2026-09-09 com o provedor fora: `generation-results-T-retrieval.csv`.

## Approach

1. `pdf_tables.go`: `detectPDFTables(page pdfBBoxPage, pageNumber int) []models.TableSection` sobre as palavras; chamado em `parsePDFBBoxLayout` (guardando em `pdfTextExtraction.PageTables`) e consumido em `indexPDF` → `buildIndexingResult(..., tables)`.
2. Coluna `page_number`; DBML.
3. Hits de linha no retrieval e fontes de linha no chat.
4. Reindexar, medir, registrar.

## Open Questions

- **Linhas de tabela também como parágrafos?** Hoje as palavras da tabela continuam nos blocos de texto da página (o texto da página não muda). Duplicação aceitável nesta rodada; remover os blocos cobertos pela caixa da tabela fica para depois de medir.

## Ambiguity Report (grill-me, modo spec, 2026-09-09)

```
Ambiguity Report:
  Goals:        0.0   ✓ clear
  Acceptance:   0.25  ✓ clear (Recall@6 e verificavel; "sobe de 4,8%" nao fixa alvo, o alvo e registro)
  Boundaries:   0.0   ✓ clear
  Alternatives: 0.25  ✓ clear (table_rows relacional adiado com razao; tabela por pagina x entidade multipagina)
  Assumptions:  0.25  ✓ clear (verificado: word com bbox, buildIndexingResult aceita tabelas, tx.Create persiste, keep por tipo; nao verificado: tolerancias de 2/8/12 pt, ajustadas pelo teste real)
  ──────────────────────────────
  Aggregate:    0.15  ✓ below threshold (0.2 spec)
```

## Discoveries

