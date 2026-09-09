# Chat grounded com o conteúdo das páginas, seções antes de páginas, tokens e custo por resposta

**Status:** rascunho
**Data:** 2026-09-09
**Base:** `docs/prds/workspace-unificado.md` (PR #55, sem merge). Branch a partir de `feat/workspace-unificado`; PR novo, também sem merge.
**Origem:** teste na stack real com o usuário demo e Ollama Cloud (`gemma4:31b`): três perguntas com resposta nos documentos voltaram "não há informações", uma em inglês.

## Goals

(1) O modelo passa a receber o conteúdo das páginas citadas, não só um trecho de 220 caracteres por fonte, com teto de caracteres explícito, testado e registrado. (2) Fontes vindas de seção do sumário (node hits) entram antes das fontes de página, porque são a unidade estrutural da tese e têm peso 3 no retrieval. (3) Cada resposta grava tokens de prompt e de resposta (reportados pelo provedor quando ele informa, estimados quando não), o modelo, o provedor e o custo estimado quando há preço, e a interface mostra isso embaixo da resposta. (4) O idioma pedido é obedecido: a instrução vai no fim do prompt do sistema e repetida no prompt do usuário.

Pronto quando: o unitário do prompt prova que o conteúdo da página está no prompt e que o teto corta; o unitário das fontes prova que o node hit vem antes; o unitário do stream prova que tokens e custo são persistidos; E2E com mock mostra a linha de tokens embaixo da resposta; e as três perguntas do teste real são refeitas com o resultado (melhorou ou não) registrado em Discoveries.

## Requirements

Prompt com conteúdo
- [ ] `buildCollectionChatPrompt` inclui, para cada fonte, o conteúdo da página (`DocumentPage.Content`) em vez do snippet. Fonte de página → a própria página; fonte de nó → `start_page` do nó. Teto por página `ChatPageCharLimit = 4000` caracteres (corte por runas, com "…"), teto total `ChatContextCharLimit = 24000`. Quando o total estoura, as fontes seguintes voltam ao snippet de 220. Constantes exportadas e travadas por teste.
- [ ] O bloco "Relevant outline hits" some (as seções relevantes já entram como fonte com conteúdo). "Document inventory" fica.
- [ ] A instrução de idioma: última linha do prompt do sistema e repetida no fim do prompt do usuário, na própria língua ("Responda em português do Brasil." / "Answer in English." / "Responde en español.").

Seções antes de páginas
- [ ] `buildChatSources`: até 3 fontes de nó primeiro (na ordem do retrieval, deduplicadas por `documento:página-inicial`), depois fontes de página até completar 6, sem repetir `documento:página`. Nó sem `start_page` é ignorado.

Tokens e custo
- [ ] `models.ChatConversationEntry` ganha `PromptTokens`, `CompletionTokens` (int), `UsageReported` (bool), `ProviderType`, `Model` (string), `EstimatedCostUSD` (*float64, nulo sem preço). Sem migração manual: `AutoMigrate` adiciona colunas.
- [ ] No `StreamCollectionChat`, depois da resposta: se o LLM implementa `domain.TokenUsageReporter` e `reported=true`, usa os números dele; senão estima com a mesma regra da indexação (runas/4), `UsageReported=false`. Custo via `domain.LLMPricingProvider` quando disponível. Persistido na mensagem do assistente (inclusive na parcial).
- [ ] O evento `done` do stream carrega `usage: {prompt_tokens, completion_tokens, usage_reported, model, provider_type, estimated_cost_usd}`; o DTO do histórico devolve os mesmos campos em `usage`.
- [ ] Frontend: `ChatWorkspaceMessage.usage?`; `ChatMessageCard` mostra uma linha discreta abaixo da resposta: "1.234 tokens · gemma4:31b" e, com custo, "· US$ 0,0004"; com `usage_reported=false`, prefixo "≈" e título "estimado". Chaves i18n nas três línguas.

Testes
- [ ] Backend unitário: prompt contém o conteúdo da página e não o snippet; corte no teto por página e no total (reversão: sem o teto, o prompt cresce); node hit antes de page hit; usage reportado vs estimado persistidos; custo nulo sem preço.
- [ ] Frontend unitário: `ChatMessageCard` com `usage` reportado e estimado. E2E mock `workspace-chat.spec.ts`: evento `done` com `usage` → linha "tokens" visível.
- [ ] Stack real: repetir as três perguntas (carga horária total; início do semestre letivo e dias letivos; período e pré-requisito de Banco de Dados I) e registrar resposta, fontes e tokens em Discoveries, antes e depois.

## Non-Goals

- Mudar o scorer do retrieval (`retrievalScore`, stopwords, IDF, BM25). O braço A do experimento em `internal/retrieval/lexical.go` é cópia fiel do scorer de produção; mudar um sem o outro invalida as tabelas do artigo. Fica registrado em Discoveries que a consulta inclui stopwords ("qual", "e", "de") e por isso páginas longas do PPC vencem páginas curtas com a resposta. É PRD do experimento, não deste.
- Enviar a coleção inteira, embeddings, reranking por LLM.
- Preço automático para Ollama Cloud (não há tabela); custo aparece só quando o provedor tem preço cadastrado.
- Contagem exata de tokens por tokenizer do modelo; a estimativa é runas/4, rotulada como estimativa (constituição VI).

## Constraints

- `CollectionContext.DocumentPages` já vem carregado no chat (é de onde saem os snippets), então incluir conteúdo não custa consulta nova.
- `TokenUsageReporter` já existe (`domain/provider.go`) e o cliente OpenAI-compatível implementa; Ollama nativo não reporta. `LLMPricingProvider` idem. A indexação já usa a mesma regra de estimativa em `processing/llm_observer.go` (runas/4, custo por milhão).
- `ChatSourceReference` é jsonb; os campos novos de uso ficam em colunas próprias da entrada, não no jsonb (constituição I: nada polimórfico).
- `buildChatSources` hoje pega até 6 page hits e só depois node hits; o retrieval devolve até 8 de cada, ordenados por score.
- O PPC tem 861 seções por ruído de sumário; a fonte de nó pode apontar para a página do sumário (TOC). Aceito nesta rodada; a limpeza do sumário é outro PRD.
- Idioma vem de `Accept-Language` no handler (`lang`), normalizado em `normalizeChatLanguage`.

## Approach

1. Backend, prompt e fontes: `buildChatSources` com nós primeiro; `buildCollectionChatPrompt` com conteúdo por fonte e tetos; idioma no fim. Testes unitários.
2. Backend, uso e custo: campos no modelo, cálculo no stream, evento `done` e DTO. Testes com o stub de LLM existente (`chat_stream_persistence_test.go`) estendido para reportar uso.
3. Frontend: contrato do evento `done` e do histórico, `usage` na mensagem, linha no cartão, i18n, unitário e E2E.
4. Stack real: as três perguntas, antes e depois, em Discoveries. PR sem merge.

## Open Questions

- Teto de 24.000 caracteres (≈ 6 mil tokens): chute inicial, documentado e travado por teste. — *decidido por padrão*; ajustar com dado.
- Três fontes de nó antes das páginas: — *decidido por padrão*; o número é constante exportada.

## Decisões tomadas sem entrevista (autor ausente, 2026-09-09)

- Conteúdo da página por fonte, não a coleção inteira nem o texto da seção completa (uma seção do PPC pode ter dezenas de páginas).
- Scorer intocado; achado das stopwords registrado para o PRD do experimento.
- Custo só com preço cadastrado; sem preço, campo nulo e interface sem valor.

## Alternativas descartadas

- **Coleção inteira no prompt:** O(n) em tokens, estoura janela com 200 páginas; é o que a análise de gap apontou como risco.
- **Texto completo da seção (start_page..end_page):** seções do PPC atravessam dezenas de páginas; o teto cortaria de forma arbitrária. A página inicial é o menor pedaço com a proveniência inteira.
- **Corrigir o scorer agora (stopwords/IDF):** mudaria o braço A do artigo sem rerodar o experimento. Vai para o PRD do experimento com as tabelas regeneradas.
- **Tokens no jsonb das fontes:** viola a constituição I; colunas tipadas.

## Ambiguity Report (grill-me, modo spec, 2026-09-09)

Verificado no código: `buildChatSources` (chat_usecase.go:319) prioriza page hits; `buildCollectionChatPrompt` (403) usa `source.Snippet`; `TokenUsageReporter` e `LLMPricingProvider` existem e `openAICompatibleLLMClient` implementa ambos; `llm_observer.go` estima runas/4; `ChatConversationEntry` não tem campos de uso; `lexical.go` declara-se cópia fiel do scorer; a tokenização não remove stopwords (`tokenizeRetrievalText`).

```
Ambiguity Report:
  Goals:        0.0   ✓ clear
  Acceptance:   0.25  ✓ clear (as tres perguntas reais sao registro, nao criterio de aprovacao)
  Boundaries:   0.0   ✓ clear
  Alternatives: 0.0   ✓ clear
  Assumptions:  0.25  ✓ clear (gemma4:31b via Ollama Cloud pode ou nao reportar uso; o codigo cobre os dois casos)
  ──────────────────────────────
  Aggregate:    0.10  ✓ below threshold (0.2 spec)
```

## Discoveries

(vazio)
