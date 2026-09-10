# Fontes que sobrevivem à reindexação e enriquecimento por LLM em todas as páginas e seções

**Status:** entregue (2026-09-10), branch `feat/conexoes-por-token` (PR #72, sem merge)
**Origem:** o autor: "indexei, perguntei, a IA respondeu; indexei de novo e as fontes da primeira pergunta não são mais as mesmas: como arrumar?" E: "os resumos e tópicos estão curtos; quero resumo e tópicos por página, e melhorar o que faltar".

## Goals

(1) **Citação autocontida.** Cada fonte gravada numa resposta carrega, além de documento e página, a caixa do parágrafo (ou da tabela), as caixas das linhas citadas e a data de indexação do documento naquele momento. Depois de uma reindexação, o clique na fonte continua grifando o lugar certo na página (a página e as coordenadas não mudam para o mesmo arquivo), e o card avisa "o documento foi reindexado depois desta resposta" quando a data do índice atual é posterior. Se o arquivo foi substituído por outro, a caixa pode não bater: o aviso cobre isso, e o trecho gravado continua sendo o que foi citado.
(2) **Enriquecimento completo.** Resumo e tópicos para **todas** as páginas com texto (em lotes de 6 páginas por chamada), resumo para **todas** as seções (lotes de 12), resumos de até 60 palavras em vez de 40/45, sinopse até 200 palavras. Página vazia ou de menos de 200 caracteres não gasta chamada. Tudo com os fallbacks e logs que já existem; número de chamadas e páginas resumidas gravados nos metadados.
(3) Medir: reindexar os 10 documentos, contar chamadas e tokens, e rodar o gabarito (1 repetição) para ver se os resumos nas páginas ajudam ou atrapalham a recuperação.

## Requirements

Fonte autocontida
- [x] `models.SourceBox`; `ChatSource.Box`, `LineBoxes`, `IndexedAt` (domínio, modelo persistido, DTO).
- [x] `buildChatSources` preenche a caixa do parágrafo escolhido (de `metadata.box`), as caixas das linhas citadas (`metadata.lines[i].b`) e `Document.IndexedAt`; fonte de linha de tabela leva a caixa da tabela.
- [x] Frontend: `Source.box/lineBoxes/indexedAt`; `DocumentSummary.indexedAt`; ao clicar, se o parágrafo não existe mais no contexto, o canvas centraliza na caixa gravada e pinta as linhas gravadas (`citacao` no `PageOverlay`); o card mostra "reindexado depois desta resposta" quando `documento.indexedAt > source.indexedAt`.
- [x] Testes: unitário do `buildChatSources` (caixa e linhas presentes); Vitest do overlay com geometria gravada; Vitest do card com o aviso.

Enriquecimento
- [x] `generateAllPageSummaries`: lotes de 6 páginas com texto (≥ 200 caracteres), JSON `pages[{page_number, summary, topics}]`, resumo ≤ 60 palavras (clamp 600 caracteres), tópicos ≤ 5; `page.Summary` e `page.Metadata.topics`; falha de um lote não derruba os outros.
- [x] `generateAllSectionSummaries`: lotes de 12 seções, contexto = título + até 900 caracteres do texto da seção (páginas `start_page..end_page`), resumo ≤ 60 palavras.
- [x] Sinopse ≤ 200 palavras. Metadados `enrichment_llm_calls`, `enrichment_pages_summarized`, `enrichment_sections_summarized`.
- [x] Sidebar já mostra o resumo da seção nos dois primeiros níveis; página passa a ter tópicos disponíveis (uso na tela fica para depois).

## Non-Goals

- Diff entre versões do índice; migração de fontes antigas (ficam como estão: caem na página).
- Tópicos de página na interface nesta rodada.

## Constraints

- `metadata.box` e `metadata.lines` já existem nos parágrafos; `TableSection.Metadata.box` nas tabelas; `Document.IndexedAt` no modelo.
- O enriquecimento roda no contêiner do indexador; cada lote é uma chamada `GenerateText`. Custo estimado por PPC de 209 páginas: ~35 lotes de página + ~12 de seção + 2, ≈ 50 chamadas.

## Ambiguity Report (grill-me, modo spec, 2026-09-10)

```
Ambiguity Report:
  Goals:        0.0   ✓ clear
  Acceptance:   0.25  ✓ clear (efeito dos resumos na recuperacao e medida, nao criterio)
  Boundaries:   0.0   ✓ clear
  Alternatives: 0.25  ✓ clear (versionar o indice inteiro foi descartado: a caixa na pagina basta e e mais simples)
  Assumptions:  0.0   ✓ clear
  ──────────────────────────────
  Aggregate:    0.10  ✓ below threshold (0.2 spec)
```

## Discoveries

- **Seções pela metade na primeira tentativa.** Com o texto da página inteira como contexto e o prompt antigo ("se a evidência for fraca, deixe o nó de fora"), a LLM resumiu 76 das 144 seções do PPC de SI, 76/116 do Técnico e 70/132 das Ementas. Com o texto próprio de cada seção (o `retrieval_text` que o indexador já calcula, sem os filhos) e o pedido de resumir todos os nós, 144/144, 116/116, 132/132, sem lote descartado.
- **Custo.** 10 documentos, 486 páginas: 135 chamadas em lote, ≈ 616 mil tokens de entrada e 101 mil de saída (gemma4:31b, Ollama Cloud); PPC de SI: 48 chamadas, 198 mil tokens. 482 páginas com resumo e tópicos (as 4 sem são as de menos de 200 caracteres), 444/444 seções.
- **Recuperação não mudou.** Gabarito v4, 1 repetição: 98,6%, proveniência 97,0%, prosa 30/30, tabela 21/21, calendário 15/16 (c14), negativa 5/5; 4.191 tokens de prompt (antes 4.271). O resumo entra no BM25 da página e não moveu nenhuma página-alvo. Ganho é de leitura, não de retrieval; registrado em `docs/analise-2026-09/experimento-geracao.md` §12.
- **A API e o indexador compartilham o módulo Go:** editar o enricher durante uma rodada do gabarito recompilou a API (`air`) e derrubou 52 chamadas. Regra: nada de edição em Go durante medição.
- **Fonte gravada.** `buildChatSources` grava caixa do parágrafo (ou da tabela), caixas das linhas citadas e `indexed_at`; o canvas usa a caixa quando o parágrafo sumiu do contexto e o card avisa "reindexado" quando o índice atual é posterior à resposta. Fontes antigas (sem caixa) continuam caindo na página.
- **Tópicos de página na tela** (pedido depois): o contexto da coleção passa a expor `documentPages` (número, resumo, tópicos; sem o conteúdo) e a sidebar mostra "Nesta página (N)" com o resumo e os tópicos da página do trecho selecionado ou, no modo "uma página", da página atual.
- **Ordem das fontes:** linha de tabela só fica na frente com pelo menos metade dos termos da pergunta; o resto (linhas fracas, seções, páginas) ordenado por termos casados; multiplicar o BM25 pela cobertura foi medido e descartado (perdeu t18). Medido com 3 repetições numa conta nova: 216/216 (c14 passou de 0/3 para 3/3 só pela ordem no prompt; fontes iguais). Relatório §13, `generation-results-T-ordem.csv`.
