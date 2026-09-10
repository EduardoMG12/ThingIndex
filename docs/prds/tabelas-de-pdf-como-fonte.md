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

- **Extração (2026-09-09).** Reindexados os 5 documentos do demo: PPC 26 tabelas / 171 linhas, Calendário 13 / 82, Matriz 4 / 62, Ementas 3 / 10, PPC Técnico 9 / 33, todas com `page_number`. A Matriz sai com cabeçalho ("Período, Disciplina, Categoria Pedagógica, Atividade, Bloco Hrs Coreq, Pré-requisito Trilha de Disciplina Hrs optativas": os rótulos de colunas vazias colam no vizinho) e as células multilinha remontadas ("SIS07104 - Introdução a Tecnologia de Informação e Comunicação"). O Calendário sai um mês por tabela, com o mês como título e "Dia(s)"/"Descrição" no cabeçalho; os dígitos da grade viram células ruidosas ("D: 15; S: 16; ...") mas não atrapalham. A Tabela 4 do PPC sai com o nome quebrado remontado, inclusive quando a quebra vem antes da linha numérica (votação "base").
- **O que a detecção ainda erra.** Rótulos girados ("Período", "1º") entram como texto da primeira coluna; cabeçalhos de várias linhas do PPC viram uma sopa; a Ementas p. 9 vem sem cabeçalho porque as linhas de rótulo não têm 2+ células alinhadas. Nada disso impede a linha de ser recuperada.
- **Primeira medição (linhas ocupando 3 das 6 vagas).** Recall nas fontes por estrato: tabela 42,9% → 85,7% (12 perguntas ganharam, 3 perderam), calendário 100% → 100%, prosa 80% → 63,3%. As perdas em prosa (p01, p18, p21, p25, p28) e em t19–t21 vieram de linhas fracas ocupando vagas de página. Correção: linhas não consomem as `ChatSourceLimit` vagas de seção e página (custam ~100 caracteres contra 4.000) e linha abaixo de metade do score da melhor fica de fora. Segunda medição abaixo.
- **Confundidor registrado:** a reindexação aconteceu sem crédito de LLM, então sinopses, resumos e a árvore da LLM caíram em fallback; os hits de seção mudaram entre "antes" e "depois" por isso, não só pelas tabelas.
- **Segunda medição (linhas fora das vagas de página, linha fraca cortada).** Recall nas fontes: todas 73,1% → **89,6%**; tabela 42,9% → **100%** (21 de 21); calendário 100% → 100%; prosa 80% → 76,7%. Ganharam 12 perguntas de tabela; perdeu só p01 (carga horária total), que agora traz a linha "TOTAL 3000" da Matriz p. 3, evidência certa numa página que o gabarito não lista. CSVs: `generation-results-T-retrieval.csv` (antes) e `generation-results-T-retrieval-depois.csv` (depois). O acerto do chat com essas fontes só será medido quando houver crédito de LLM.
- **Sumário sem células (2026-09-09).** Bloco cujo centro cai na caixa de uma tabela da página (cabeçalho incluído) não vira título. Seções no demo: Matriz 16 → 4, PPC 564 → 494, PPC Técnico 511 → 495, Calendário 18 → 13, Ementas 133 → 132. O que sobra no PPC são nomes e cargos da capa e rótulos de formulário, que não estão em tabela detectável.
- **Tabelas na tela (2026-09-09).** A API expõe `page_number` da tabela; a sidebar do workspace lista "Tabelas (N)" do documento focado (página, título ou "Tabela da p. N", número de linhas) e, ao clicar, centraliza a folha na tabela e desenha a caixa tracejada com o título. Uma fonte de linha de tabela no chat ("Ver no documento") realça a tabela em vez de só abrir a página.
- **Rodada 4 (2026-09-09).** Linha só vira fonte com 2+ termos significativos da pergunta casando por token inteiro (pergunta de um termo exige o um). Medido no gabarito com 10 documentos: nenhuma perda; linhas do calendário em perguntas de prosa caíram de 5 para 3. Reforço por título do documento e plural→singular no BM25 foram testados e descartados com número (relatório, seção 9).
- **Títulos no texto da página (2026-09-09).** Descoberto pela pergunta p24 do gabarito: tudo que o detector classifica como título saía do texto da página, e a identificação do PPC do Técnico ficava sem "Eixo Tecnológico / Informação e Comunicação". Corrigido em `buildDocumentPages`; medido com 10 documentos e 1 repetição: recuperação em prosa 23/30 → 25/30, acerto total 67/72 → 70/72. O indexador precisou de `docker compose restart indexer` para recompilar (o air dele tinha parado de observar), o que custou uma reindexação em vão.
- **Sumário numerado manda (2026-09-10).** Em documento com 8+ títulos numerados de verdade (até dois dígitos por nível e maiúscula depois: "1.APRESENTAÇÃO", "4.3.2 Plano"; "40 vagas" e "3600 h/a" não contam), só título numerado ou arquétipo legal vira seção; capa, valor de campo e fragmento em caixa alta ficam como parágrafo. PPC de SI: 492 → 144 seções, 131 → 13 de nível 1 (os oito capítulos e os apêndices). Rebaixar de nível não bastava: a normalização da árvore devolvia o primeiro título ao nível 1.
- **Herança de cabeçalho entre páginas:** implementada (tabela sem cabeçalho na página seguinte herda o da anterior quando o número de colunas bate), mas no demo não mudou nada (57 de 80 tabelas continuam com cabeçalho sintético): a continuação quase sempre cai em colunas diferentes. Fica registrado; resolver exige alinhar por posição das colunas, não por contagem.
- **Linhas curtas de tabela quebrada vencem a linha certa.** "SISOP510; Sistemas; Operacionais; e" (tabela sem cabeçalho do PPC) pontuava acima da linha da Matriz por ser curta (normalização de tamanho do BM25). A variante singular/plural não cobria "pré-requisitos" por causa do hífen; com o hífen aceito, a linha da Matriz ganha o terceiro termo ("pré-requisito", do cabeçalho) e volta ao primeiro lugar.
