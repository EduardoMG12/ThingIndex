# Entrega final: onde o projeto está, o que o artigo diz e o que cortar

> Análise feita em 2026-09-10, depois do merge dos PRs #55, #66, #71 e #72 na `main` (o #60 fechou ao apagar a base; seu conteúdo entrou pelo #66). Fontes: `docs/article/article-overleaf.md`, `.specify/memory/constitution.md`, `docs/analise-2026-09/experimento-geracao.md` (§1–16), o inventário do código e a stack real rodando. Cada afirmação aponta o arquivo.

## 1. Veredito

**O sistema está no caminho da tese e já entrega mais do que o artigo afirma.** A pergunta de pesquisa (`article-overleaf.md:91`: sustentar inteligência documental auditável só com modelagem relacional, sem vetor ou grafo como camada primária) tem hoje uma resposta medida: 100% de acerto e 100% de proveniência em 72 perguntas com gabarito, empate com "o documento inteiro no LLM" gastando 17× menos tokens, tudo determinístico e com o CSV versionado (`experimento-geracao.md` §13–14).

**O risco da entrega não é o software. São três coisas:**

1. **O artigo está atrasado em relação ao sistema.** Ele reporta só o experimento de recuperação (braços A–E, 5 documentos, 18 consultas) e afirma coisas que o sistema atual desmente (negativas "nenhum braço reconhece ausência", proveniência 0,329, custo "não medido"). Braços T e F, gabarito de 72 perguntas, 10 documentos e os números de custo não aparecem em lugar nenhum do `.tex`.
2. **Superfície com incerteza.** 7 tipos de LLM e 7 de storage no catálogo, com 2 clientes reais de cada; OCR por GPU; painel de custo e worker de preços fora do compose; árvore de seções por LLM que quase nunca é aceita. Nada disso é a tese, e cada um é um lugar onde a banca pode perguntar "isso funciona?" e a resposta honesta é "não medimos".
3. **O gabarito saturou** (100% em 3 repetições). Ele deixou de discriminar mudanças; a próxima medição que vale é com um gabarito v5 escrito por outra pessoa.

## 2. O que o artigo promete versus o que existe

| Promessa (artigo) | Estado no sistema | Evidência |
|---|---|---|
| RF01–RF06, RF08, RF10, RF13–RF15 (auth, categorias, coleções, upload PDF, storage, indexação assíncrona, contexto REST, provedores criptografados, auditoria, retry/órfãos, i18n) | Entregues e cobertos por testes Go e E2E | `article-overleaf.md:311-327`; 165 `_test.go`; 16 E2E com mock + 3 na stack real |
| RF07 enriquecimento opcional por LLM | Entregue, agora por escopo (documento, páginas, seções) e com cache por hash | `docs/prds/escopo-do-enriquecimento-cache-foco-e-conexoes.md` |
| RF09 chat grounded com SSE e fontes | Entregue; fontes com página, parágrafo, linhas, caixa, termos casados, continuação de página; falha do provedor persistida | `docs/prds/chat-fontes-navegaveis.md`, `fontes-autocontidas-e-enriquecimento-completo.md` |
| RF11 telemetria por documento e agregada | Entregue em `/documents` (observabilidade) | `document-observability.spec.ts` |
| RF12 benchmark público de indexação | Existe (`cmd/indexing-benchmark`, corpus C1) | `docs/analise-2026-08/benchmark.md` |
| "cada parágrafo rastreável até documento, página e posição" (`:290`) | Verdade, e além: linha e caixa, sobrevivendo à reindexação | `buildChatSources`, `SourceBox`, `metadata.lines` |
| "reproduzível com dois comandos" (`:564`) | Recuperação sim; geração exige chave de LLM e a coleção demo indexada | `cmd/generation-eval` |
| Custo operacional: "capacidade de medir, não a medição" (`:540-544`) | **Medido**: 4,2 mil tokens/pergunta em T contra 72 mil (F/documento) e 250 mil (F/coleção) | `experimento-geracao.md` §11 |
| Negativas: "nenhum braço reconhece ausência" (`:634`) | **Desmentido**: T acerta 5/5 negativas; F/documento 80% | §11, §13 |
| Proveniência TPC 0,329 (`:660`) | **Superado**: 67/67 páginas-alvo nas fontes | §14 |
| Escopo "sem vetor nem grafo como dependência" | Mantido; o braço denso existe só como comparação (`dense.go`, IVF instável) | constituição, princípio II |

**Conclusão da tabela:** nenhum requisito ficou para trás. O que falta é escrever no artigo o que já foi medido.

## 3. O que cortar (ou esconder) para entregar sem incerteza

Critério: fica o que a tese precisa e o que foi medido; sai (ou vira "experimental", fora do texto) o que existe só como possibilidade.

| Item | Hoje | Proposta | Esforço |
|---|---|---|---|
| **Provedores de LLM** | 7 tipos no catálogo (`support_matrix.go:133-233`); 2 clientes reais (`factory.go:967,979`); Gemini e Claude "experimental" via ponte OpenAI com `base_url` obrigatório; DeepSeek, Grok e OpenRouter "partial", nunca testados ao vivo | Catálogo com **2 entradas**: *Ollama* (nativo, medido, o que o experimento usou) e *OpenAI e compatíveis* (uma entrada, `base_url` opcional, cobre OpenAI, OpenRouter, DeepSeek, Grok, Gemini/Claude por ponte). Remover as 5 entradas específicas do catálogo e da matriz de suporte; o cliente compatível continua o mesmo. Artigo: "o sistema fala com qualquer API compatível com OpenAI e com Ollama; os experimentos usaram gemma4:31b via Ollama Cloud" | 1 dia (catálogo, matriz, i18n, testes de catálogo) |
| **Provedores de storage** | 7 tipos; 2 adaptadores (`storage/minio.go` serve todos os S3; `gcs.go`); R2, Spaces, Wasabi, B2 e AWS sem teste próprio | **2 entradas**: *S3 compatível* (endpoint + chaves; MinIO é o padrão do compose e tem teste de integração) e *Google Cloud Storage* (tem teste com emulador). Tirar as 5 marcas | meio dia |
| **OCR por GPU** (`docker-compose.yml:152`, perfil `ocr`, Unlimited-OCR remoto) | Fora do fluxo do demo (todos os PDFs têm camada de texto); exige NVIDIA | Manter o fallback tesseract (testado) e tirar o serviço GPU do compose e do README, ou deixá-lo só como perfil documentado como "não usado nos experimentos" | 1 hora |
| **Painel de custo** (`/cost-visibility`) + `pricing-worker` + 3 tabelas de preço | Worker não está no compose; a página depende de catálogo de preços que ninguém alimenta; o custo por resposta que importa já aparece no chat (tokens, modelo) | Cortar a página e o worker da entrega; manter tokens e modelo por resposta (é o número do artigo). Se quiser manter, rotular "experimental" e tirar do menu | meio dia (ou 10 min para esconder) |
| **Árvore de seções por LLM** (`tree_builder.go`) | Só aceita quando anchorada; no demo 8 de 10 documentos ficam com a determinística; é a parte mais cara que sobra depois do cache (≈70 mil tokens por PPC, em tentativas) | Desligar por padrão (a árvore determinística é a tese); manter sinopse, tópicos, resumos de página e seção. Ganho: custo e uma fonte de variação a menos | 1 hora + reindexar |
| **Formatos HTML e XLSX** | Código e testes existem; artigo fala só de PDF (RF04) e DOCX aparece por engano (`document_usecase.go:2531`) | Manter (custam pouco), tirar a menção a DOCX; no artigo, uma frase: "PDF é o alvo; HTML e XLSX entram pelo mesmo pipeline sem posição na página" | 20 min |
| **Página `/documentation`** | Conteúdo institucional com números do experimento antigo | Conferir os números contra §13–14 ou reduzir a um link para o artigo | 1 hora |
| **Ferramental em `cmd/`** (`index-import`, `dense-determinism`, `threshold-eval`, `provenance-eval`, `retrieval-eval`, `generation-eval`, `indexing-benchmark`, `schema-dbml`) | 8 binários de experimento | Manter; documentar em um `docs/experimentos.md` com o comando de cada um (dois deles já estão no README) | 1 hora |

O que **não** cortar, mesmo parecendo periférico: auditoria de documentos e provedores (é o princípio I), retry/órfãos (RF14), o braço denso como comparação (é o argumento contra o vetor), a legenda e as marcas do workspace (é o uso real do coordenador).

## 4. O artigo: o que reescrever

Em ordem de dano se ficar como está:

1. **Seção de resultados de geração** (nova): braços T, F/documento e F/coleção; 72 perguntas em 4 estratos; tabela da §13 (100%, 97% → 100% de proveniência, tokens); pareado T × F (vence 1, empata 71, IC −0,042 a 0,000); custo 17–62× menor. Tudo já está em `experimento-geracao.md` e nos CSVs de `docs/article/results/`.
2. **Negativas** (`:634`): trocar "nenhum braço reconhece ausência" por "no experimento de recuperação lexical nenhum braço reconhece ausência; no chat grounded, 5/5 negativas são respondidas com 'não consta'".
3. **Custo operacional** (`:538-544`): substituir a lacuna pela medição.
4. **Proveniência** (`:660`): TPC 0,329 era do experimento de recuperação por chunk; a proveniência da resposta é 67/67. Explicar a diferença de unidade.
5. **Hipóteses falseáveis**: o artigo tem a pergunta (`:91`) e "não-inferioridade" (`:601`), mas não H1/H2. Duas linhas cada: H1 não-inferioridade em acerto (medida), H2 superioridade em proveniência e custo (medida), H3 determinismo (o índice IVF instável já está em `:716-719`).
6. **Blocos comentados** (`:104-113`, `:731-854`, `:856-1013`, `:1023-1048`): decidir e apagar. Texto morto de 400 linhas num `.tex` é onde a banca acha contradição.
7. **Inconsistências pequenas**: 21 tabelas vs "14 entidades/16 classes" (`:412, :417`) — hoje são 25 tabelas (`docs/schema/thingindex.dbml`); "repositório público" (`:294`) vs README privado; `apresentacao.md:192` dizendo que "o sistema não funciona".
8. **O que entrou hoje e cabe numa frase cada**: escopo do enriquecimento, cache por hash (51 → 4 chamadas), fontes que sobrevivem à reindexação, conexões tipadas (norma, disciplina), continuação de página.

## 5. Higiene que cobra pouco e evita pergunta

- CI da `main` estava vermelha por um teste de legenda (persistência de localStorage entre testes na CI); corrigido no PR #73.
- `docs/estado-do-projeto.md` é de 2026-08-12, anterior a tudo isto; ou atualiza ou apaga (o `experimento-geracao.md` e os PRDs são o estado).
- `README.md:333,465` apontam para `docs/open-source-onboarding.md`, que não existe. Arquivo `./1` vazio na raiz.
- ESLint com 21 avisos antigos (`set-state-in-effect`, refs em render); zero erros. Ou corrige, ou registra como aceito.
- Branch `feat/chat-contexto-e-custo` sobrou no remoto (PR #60 fechado); apagar.
- Chaves de LLM: as duas contas originais estouraram o limite mensal; a entrega precisa de uma conta com crédito no dia da demo e de uma resposta pronta para "e sem LLM?": o sistema indexa, recupera e cita sem LLM; só o texto da resposta e os resumos dependem dela.

## 6. Ordem sugerida até a entrega

1. Artigo primeiro (itens 1–4 da seção 4): é o que a banca lê. Dois dias.
2. Cortes da seção 3 na ordem da tabela: provedores, storage, OCR, custo, árvore por LLM. Dois dias, com PR por item e teste.
3. Gabarito v5 com revisor externo (a saturação está registrada em `experimento-geracao.md` §15). Uma tarde de quem escreve as perguntas; a medição é automática.
4. Roteiro de demo de 10 minutos: subir a stack, indexar um PDF novo com os toggles, perguntar, clicar na fonte, mostrar a conexão "mesma norma" e a mensagem de erro quando a LLM cai. Cada passo tem um teste E2E correspondente hoje.
