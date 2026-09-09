# Chat grounded com o conteúdo das páginas, seções antes de páginas, tokens e custo por resposta

**Status:** entregue em PR sem merge (2026-09-09), empilhado sobre o PR #55
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

- ~~Mudar o scorer do retrieval~~ Revisto na rodada 2 a pedido do autor: a produção adota BM25; o braço A do experimento fica intacto (ver Discoveries, #61).
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

**As três perguntas reais (usuário demo, coleção SI, Ollama Cloud `gemma4:31b`)**

| Pergunta | Antes | Depois |
|---|---|---|
| Carga horária total do curso e onde aparece | "there is no information" (em inglês); fontes PPC p. 124, 134, 180, 198, 201 e Matriz p. 3 | **Certa:** 3.600 h/aula = 3.000 h/relógio, citando Matriz p. 3 e PPC p. 54; fontes de seção primeiro (Matriz p. 3, PPC p. 54 e 134) |
| Início do 1º semestre de 2026 e dias letivos | "não há informações... menciona apenas abril e julho"; fontes Cal. p. 3, 5, 2 e PPC p. 206, 140, 207 | **Metade certa:** 205 dias letivos (105 + 100), citando p. 5; a data de início não foi lida da p. 1 (o modelo citou o ajuste de matrícula de 19 a 25/02 e os sábados letivos) |
| Período e pré-requisito de Banco de Dados I | "there is no information" (em inglês); fontes só do PPC (p. 40, 26, 176, 194, 27, 28) | **Ainda errada:** "não há informações"; as fontes continuam só do PPC (p. 30, 176, 205, 40, 26, 194). A resposta está na Matriz (doc 265) e nas Ementas (doc 266), que o retrieval não trouxe |

Tokens por resposta: 2.694 a 3.565 de prompt (estimados: o cliente Ollama nativo não reporta consumo), 44 a 150 de resposta. Custo nulo: não há preço cadastrado para o Ollama Cloud.

**Rodada 2 (CC5, #61): BM25 em produção**

O autor pediu para "fazer funcionar". O scorer entrou no escopo com uma regra: o braço A do experimento (`internal/retrieval/lexical.go`) continua intacto, medindo o sistema como foi publicado; a produção adota o braço B (BM25), que o experimento já media como melhor, com dobra de acentos e a expansão numérica do tokenizador de produção. O ranking de documentos passa a somar a melhor evidência de cada tipo (o benchmark público caía de 100% para 80% com a soma de todas as unidades). Onze perguntas na coleção de SI, `gemma4:31b`:

| Pergunta | Só trecho (início) | Conteúdo da página | + BM25 |
|---|---|---|---|
| Carga horária total do curso | errada (inglês) | certa | certa (Matriz p. 1 a 3) |
| Início do semestre e dias letivos | errada | parcial (205 dias) | parcial |
| Período e pré-requisito de Banco de Dados I | errada (inglês) | errada | parcial (3º período; pré-requisito não lido) |
| Ementa de Estrutura de Dados I | errada | errada | certa (PPC p. 63) |
| Vagas por ano e turno | errada | errada | certa (40 vagas, noturno, Ementas p. 2) |
| Disciplinas optativas | parcial | parcial | certa |
| Recesso de julho de 2026 | errada | errada | certa (férias 6 a 29/07, Cal. p. 3) |
| O que o PPC diz do TCC | certa | certa | certa |
| Carga horária de POO | certa | certa | certa (80 h/a, PPC p. 78) |
| Pré-requisitos de Sistemas Operacionais | errada | errada | errada (a Matriz p. 3 não entra: "sistemas" é onipresente) |
| Semestre de Metodologia da Pesquisa | errada | errada | certa (2º período, PPC p. 74) |
| **Total** | **2 certas** | **3 certas, 2 parciais** | **8 certas, 2 parciais, 1 errada** |

O que sobra: pré-requisitos vivem numa coluna de tabela da Matriz (p. 3), e "sistemas"/"operacionais"/"banco" competem com a mesma linha em outras páginas. É caso para as `table_sections` entrarem como fonte de chat (hoje só pontuam o documento) ou para o sumário limpo da Matriz; ambos são PRDs seguintes.

**O que a rodada ensinou**

- O conteúdo da página resolve quando o retrieval acerta a página. Quando erra (P3), nenhum conteúdo salva. O gargalo era o scorer; a rodada 2 trocou por BM25.
- A causa do erro de P3 está no `tokenizeRetrievalText`: a consulta vira termos sem remover stopwords ("em", "que", "do", "e", "qual", "o", "seu"), e `retrievalScore` conta quantos termos distintos a página contém como substring. Páginas longas do PPC contêm todas as stopwords e vencem a linha da Matriz que tem "Banco de Dados I" e o pré-requisito. Isso é o braço A do experimento, copiado fielmente em `internal/retrieval/lexical.go`. A produção passou ao braço B (BM25) na rodada 2; o braço A segue no experimento como foi publicado, e a diferença da produção (dobra de acentos, expansão numérica, melhor evidência por tipo) fica declarada aqui.
- Fontes de seção primeiro ajudou em P1 (a Matriz entrou pela seção) mas o sumário ruidoso do PPC também entra: "node-783-digitacao" e "node-823-municipal" são células de tabela viradas seção. A limpeza do sumário (outro PRD) melhora o chat de graça.
- O modelo passou a responder em português com a instrução repetida no fim dos dois prompts.
- O cliente Ollama nativo não implementa `TokenUsageReporter`; a estimativa runas/4 fica rotulada ("≈") até a interface. Para números reportados, usar o provedor OpenAI-compatível.
- Entregue em 5 issues (#56 a #59 e #61). Suítes: `go test ./...` ok, Vitest 228, Playwright mock 36, Playwright real 4.
