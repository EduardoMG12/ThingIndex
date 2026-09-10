# Análise de GAP — Artigo TCC I × Sistema Implementado (ThingIndex)

> Auditoria feita sobre `docs/article/article-overleaf.md` (885 linhas), `docs/estado-do-projeto.md`, `docs/analise-para-correcao/*` e o código em `api/` e `frontend/`.
> Tom deliberadamente duro: o objetivo é antecipar o que a banca vai atacar.

---

## 0. Veredito em uma página

**A boa notícia:** o artefato existe e é sólido. 53 arquivos de teste Go, 59 arquivos de teste/spec no frontend, CI com três jobs (backend, backend-integration com Postgres 16, frontend), 14/14 E2E Playwright, zero TODO/FIXME no código. Praticamente todos os RFs têm rota e código correspondentes. Isso é bem acima da média de um TCC de graduação.

**A má notícia, e é grave:** o trabalho **ainda não é um trabalho empírico**. Ele é um artefato de engenharia bem executado com uma pergunta de pesquisa colada por cima. Especificamente:

1. **Não existe baseline comparativa.** O artigo afirma superioridade em auditabilidade, determinismo e custo sobre RAG vetorial e **nunca compara com nada**. O único “comparativo” do código (`indexing_benchmark.go`, variantes `flat-page-control` e `flat-hybrid-control`) compara o sistema com **variações do próprio sistema** — é auto-referente. O próprio código admite isso em `indexing_benchmark.go:808-810`: *“This repository does not ship a native vector index in the current runtime”*. **Atualização 2026-09-09:** o braço "documento inteiro no LLM" foi medido em `docs/analise-2026-09/experimento-geracao.md`: ThingIndex 92,6% de acerto contra 98,6% do documento inteiro no LLM (72 perguntas × 3 repetições, gabarito v2), com 18 a 62 vezes menos tokens, proveniência estrutural (100% em tabela contra 25% do LLM com a coleção) e 100% nas negativas contra 80%.
2. **A pergunta de pesquisa não é falseável.** “É possível sustentar…?” tem resposta trivial: sim, o sistema está de pé. Nenhuma banca aceita isso como hipótese.
3. **O corpus de avaliação tem 5 documentos e 10 consultas** (`api/internal/usecase/testdata/indexing-benchmark/corpus.json`: 5 documentos, 5 `collection_queries`, ~1 `document_query` por documento). Isso não sustenta nenhuma afirmação estatística.
4. **As três dimensões prometidas (precisão, auditabilidade, custo) não têm nenhuma delas medida com número publicável.** Precisão: só top-1 exato em n=10. Auditabilidade: nenhuma métrica existe. Custo: `TotalEstimatedCostUSD` é *estimativa*, não medição, e nunca foi comparada com nada.
5. **Há uma contradição arquitetural não declarada que pode derrubar a tese na defesa** (detalhada na Seção 2.4): o chat envia **o contexto inteiro da coleção** ao LLM, enquanto RAG vetorial envia top-k. Ou seja, na dimensão que o artigo mais alardeia — custo — o ThingIndex pode ser **pior**, e cresce O(n). Se a banca perceber isso antes de você, o trabalho perde o eixo.

**O que o TCC II precisa entregar para virar defensável:** uma hipótese de **não-inferioridade** testável, um experimento pareado de 4 braços (relacional / BM25 / pgvector denso / híbrido RRF) sobre corpus de escala real, e uma tabela de resultados com intervalo de confiança nas 3 dimensões. O resto é polimento.

---

## 1. Mapeamento RF01–RF15 contra o código

Legenda: ✅ implementado · 🟡 parcial · ❌ ausente

| RF | Enunciado (resumido) | Status | Evidência |
|---|---|---|---|
| RF01 | cadastro, login, **logout**, perfil | 🟡 | `routes.go:113-116` (`/auth/register`, `/login`, `/forgot-password`, `/reset-password`), `routes.go:122-124` (`/user/me` GET/PUT, avatar), `middleware/auth.go`, `usecase/auth_usecase.go`. **Não existe rota de logout nem revogação de token server-side** — o JWT continua válido até expirar. O artigo promete logout; o backend não o implementa. Ataque fácil da banca em uma seção que fala de auditabilidade. |
| RF02 | criar, editar, listar, **excluir** categorias | 🟡 | `routes.go:149-152`: `POST /categories`, `GET /categories`, `PATCH /categories/:id`, `GET /category/:id`. **Não há DELETE de categoria.** |
| RF03 | criar, editar, listar, **excluir** coleções | 🟡 | `routes.go:143-148`: `POST/GET /collections`, `GET /collections/:id`, `PATCH /collections/:collection/category`, `DELETE /collections/:id/documents`. O DELETE apaga **os documentos**, não a coleção; não há rename de coleção. Dois verbos do RF não existem. |
| RF04 | upload PDF com validação de tipo | ✅ | `routes.go:141` `POST /documents/upload`; `handlers.NewDocumentHandler(..., cfg.UploadRequestLimitBytes())`; teste `usecase/document_upload_validation_test.go`. |
| RF05 | object storage | ✅ | `pkg/storage`, `pkg/provider` (`newDefaultStorageProvider`, `routes.go:46`), MinIO + fake-gcs-server no `docker-compose.yml`. Fallback `newUnavailableStorage` degrada sem derrubar a API. |
| RF06 | indexação assíncrona (parágrafos, tabelas, metadados, páginas, chunks, outline) | ✅ | `cmd/indexer`, `internal/processing`, `internal/queue`, outbox (`repository/gorm_outbox_repository.go` + teste de integração). Modelos: `ParagraphNode`, `TableSection`, `MetadataPair`, `DocumentPage`, `DocumentChunk`, `DocumentOutlineNode` (`models/document.go`). Todos os 6 tipos prometidos existem. |
| RF07 | enriquecimento opcional por LLM | ✅ | `usecase/provider_usecase.go`, `pkg/provider/factory.go`, alvo `./scripts/test.sh backend-real-llm`. |
| RF08 | recuperação de contexto completa via REST | ✅ | `routes.go:145` `GET /collections/:id/context` → `repository/gorm_collection_context_repository.go`. Coberto por 6 testes de integração, incluindo `TestCollectionContextQueryCountDoesNotGrowWithDocuments` com `collectionContextQueryBudget = 9`. **É o teste mais valioso do repositório para a tese.** |
| RF09 | chat grounded por coleção, SSE, citação de fontes | 🟡 | `routes.go:174` `POST /chat/stream`; `usecase/chat_usecase.go:172-183` emite evento SSE `sources`; `buildChatSources` limita a **6 fontes** (`min(6, len(retrieval.PageHits))`, linha ~302) e persiste `models.ChatSourceReference`. Funciona, mas: o limite 6 é mágico e não justificado, e o prompt (`buildCollectionChatPrompt`) recebe o **contexto inteiro da coleção**, não só as 6 fontes. Ver Seção 2.4. |
| RF10 | provedores externos com credenciais criptografadas | ✅ | `routes.go:125-131` (`/user/providers` POST/GET, `/test`, `/models`, `/runtime`, `/audit`, PATCH). `pkg/crypto/aes.go`: AES-GCM com chave derivada por usuário via HMAC-SHA256; 6 testes, incluindo o de isolamento (credencial do user 1 não decifra sob o user 2). |
| RF11 | painel de monitoramento com telemetria por doc e agregada | ✅ | `routes.go:159` `GET /documents/monitoring/summary` → `domain.DocumentMonitoringSummary` com `TotalLLMRequests`, `TotalInputTokens`, `TotalOutputTokens`, `TotalEstimatedCostUSD`, `AverageDurationMs`, `AverageQualityScore`, `ModelBreakdown[]`. |
| RF12 | benchmark público com corpus controlado | 🟡 | `routes.go:107` `GET /meta/indexing-benchmark` — **público de verdade** (fora do grupo autenticado). Provider em `usecase/indexing_benchmark.go` (60 KB), corpus embarcado via `go:embed`. **Mas o corpus é minúsculo**: 5 documentos (3 texto sintético + `court-order.pdf` 2 KB + `supplier-master-order-form.pdf` 2,8 KB), 5 `collection_queries`, ~5 `document_queries`. n≈10. É uma proteção anti-regressão excelente e uma evidência científica nula. |
| RF13 | trilha de auditoria de documentos e provedores | ✅ | `routes.go:142` `GET /documents/audit`, `routes.go:130` `GET /user/providers/audit`; modelos `DocumentAuditEvent`, `ProviderAuditEvent`; testes de integração em `gorm_catalog_document_repository_integration_test.go` e `gorm_provider_repository_integration_test.go` (filtro + cursor). |
| RF14 | retry manual e detecção de órfãos | ✅ | `routes.go:157` `POST /documents/:id/retry`; `domain.DocumentStatusOrphaned` (`domain/indexing.go:17`), `MarkDocumentOrphaned` (`domain/catalog.go:213`), E2E `TestRoutesE2EContentMissingMarksDocumentAsOrphaned` (`routes_e2e_test.go:339`). |
| RF15 | i18n PT-BR e EN-US nas respostas da API | 🟡 | `routes.go:36` `middleware.I18nMiddleware()`, `pkg/i18n/messages.go`. Funciona. **Risco de inconsistência:** `docs/estado-do-projeto.md` menciona componentes com “tradução nos três idiomas”. Se o sistema tem 3 locales e o artigo promete 2, corrija o artigo — a banca lê os dois. |

**Placar RF: 9 ✅ · 6 🟡 · 0 ❌.** Os parciais são todos por *verbos CRUD faltando* (delete de categoria/coleção, logout) — baratos de fechar e caros de deixar, porque são o tipo de coisa que a banca testa ao vivo.

---

## 2. Mapeamento RNF01–RNF08

| RNF | Enunciado | Status | Evidência / ressalva |
|---|---|---|---|
| RNF01 | interface responsiva, navegação clara | 🟡 | Next.js App Router, 133 componentes, 50 arquivos de teste, Storybook, 14/14 E2E. **“Responsiva” e “clara” não têm instrumento de medida.** Não há teste de viewport, nem SUS, nem auditoria de acessibilidade — e o `estado-do-projeto.md` registra que botões de conversa eram ícones sem `aria-label`. Ou você mede (SUS n=8–12, Lighthouse a11y) ou remove o adjetivo do RNF. |
| RNF02 | integridade referencial + ACID | 🟡 | FKs e `uniqueIndex` compostos em `models/document.go` (ex.: `idx_paragraph_nodes_document_sequence`); outbox gravado na mesma transação. **Duas ressalvas sérias:** (a) o schema é gerido por `AutoMigrate` do GORM, **não há migrations versionadas** — o que é uma contradição direta com “auditabilidade” e “reprodutibilidade”; (b) `gorm.DeletedAt` (soft delete) em todas as tabelas significa que “integridade referencial” convive com registros logicamente apagados mas fisicamente presentes. A banca de banco de dados vai perguntar. |
| RNF03 | centenas de documentos **sem degradação significativa** | ❌ | **Nunca medido.** É o RNF com maior distância entre promessa e evidência. Existe apenas `TestCollectionContextQueryCountDoesNotGrowWithDocuments` (1 vs 20 documentos, 9 consultas fixas) — que mede **contagem de consultas**, não latência, não memória, não bytes. Não há teste de carga, nem curva latência × n. “Sem degradação significativa” é literalmente uma afirmação não verificada. |
| RNF04 | camadas e fronteiras de domínio explícitas | 🟡 | `handlers/` → `usecase/` → `repository/` → `domain/` com interfaces (`domain.ProviderUseCase`, etc.) e injeção em `SetupRoutes`. **Contra-evidência que a banca pode citar:** `usecase/document_usecase.go` 71 KB, `usecase/indexing_benchmark.go` 60 KB, `usecase/provider_usecase.go` 35 KB, `pkg/provider/factory.go` 33 KB. Arquivo de 71 KB não é fronteira explícita; é God Object com nome bonito. |
| RNF05 | hash, tokens, criptografia, rate limiting, sanitização | 🟡 | bcrypt, JWT (`middleware/auth.go` + `auth_test.go`), AES-GCM por usuário, `SetTrustedProxies` (`routes.go:31`), CORS explícito, `pkg/netsec`, `pkg/validation`. **Rate limiting existe apenas em `/auth`** (`routes.go:111`). Upload, chat SSE e `/collections/:id/context` — as rotas caras — estão sem limite. Isso é um vetor de DoS e, pior para a tese, um furo em “custo previsível”. |
| RNF06 | reproduzível via Docker Compose (Linux/macOS/WSL) | 🟡 | `docker-compose.yml` + `docker-compose.dev.yml` com healthchecks, RabbitMQ, MinIO, fake-gcs, Postgres 16. **Ressalva:** o profile `ocr` sobe `lmsysorg/sglang` com modelo 3B em BF16 e **exige GPU NVIDIA**; e `THINGINDEX_OCR_BOX_SCALE` nunca foi validado contra execução real (registrado no `estado-do-projeto.md`). Reprodutibilidade total ≠ reprodutibilidade condicional a GPU. Declare isso como limitação antes que perguntem. |
| RNF07 | testes unitário, integração e e2e | ✅ | 53 `*_test.go` no backend, 59 test/spec no frontend, `routes_e2e_test.go` com 15 `func Test`, CI com job `backend-integration` (service container Postgres 16), Playwright 14/14 com `workers: 1`. **Falta o número que a banca pede:** não há `-coverprofile` no CI, então não existe percentual de cobertura publicável. |
| RNF08 | documentação automática via Swagger/OpenAPI | 🟡 | swaggo montado em `routes.go:90` (`/swagger/*any`) + página `/docs` própria. **Mas há 28 anotações `@Summary` para ~50 rotas registradas** — cobertura de documentação em torno de 55%. Um `swag`/lint gate no CI fecha isso. |

**Placar RNF: 1 ✅ · 6 🟡 · 1 ❌.** O ❌ (RNF03) é justamente o que sustenta a promessa de escalabilidade.

---

## 3. Afirmações do artigo que o código NÃO sustenta (superfície de ataque da banca)

Ordenadas por letalidade.

### 3.1 “Custo previsível, independente de vector DB” — a mais perigosa

O artigo repete isso quatro vezes (Trabalhos Relacionados, Proposta, Avaliação, Considerações Finais). O código conta outra história.

Em `chat_usecase.go`, `buildCollectionChatPrompt(collectionContext, retrieval, request, sources)` recebe o **contexto consolidado da coleção inteira** — é exatamente o que a Seção 6 do artigo descreve como virtude (“o contexto consolidado será enviado ao LLM junto com a pergunta”). Consequência aritmética inescapável:

- **RAG vetorial:** tokens de prompt ≈ k × tamanho_do_chunk → **O(1)** no tamanho do corpus.
- **ThingIndex:** tokens de prompt ≈ soma de todo o conteúdo da coleção → **O(n)** no tamanho do corpus.

Como o custo de LLM é linear em tokens de entrada, **o custo por consulta do ThingIndex cresce com a coleção e o do concorrente não**. Uma coleção de 200 documentos estoura a janela de contexto de qualquer modelo comercial antes de estourar o orçamento. A frase “custo previsível” é tecnicamente verdadeira (é previsível: previsivelmente cresce) e retoricamente desastrosa.

> **Isto precisa ser medido e declarado no TCC II.** Se você medir e apresentar como limitação conhecida com mitigação (envio apenas dos top-k nós do outline em vez do contexto inteiro), vira ponto forte de honestidade metodológica. Se a banca descobrir sozinha, vira colapso da tese.

### 3.2 “Auditabilidade total / cada parágrafo pode ser rastreado até documento, página e posição”

Não há **nenhuma métrica de auditabilidade** no código. E há uma falha de schema que contradiz a afirmação literal:

```go
// models/document.go
type ParagraphNode struct {
    PageNumber *int `json:"page_number,omitempty"`  // ← ponteiro: PODE SER NULL
}
```

O campo que sustenta “página de origem” é **nulável**. Some-se a isso que blocos vindos do tesseract entram **sem `Box`** (documentado no `estado-do-projeto.md`, item 1.3) — ou seja, justamente em documento digitalizado, o caso que o artigo usa como motivação (processo judicial), a âncora posicional pode não existir.

Pergunta que a banca fará e que hoje não tem resposta numérica: *“em que percentual dos itens recuperados a proveniência está completa?”* Você precisa de uma métrica **taxa de proveniência completa** e do número real (que pode muito bem ser < 100%, e tudo bem, desde que medido).

### 3.3 “Agregação linear” / “número constante de consultas”

Esta é a **única** afirmação com evidência real: `collectionContextQueryBudget = 9`, verificado com 1 e 20 documentos. Excelente. Mas é insuficiente para o que o texto sugere:

- Consultas constantes ≠ custo constante. O **payload** cresce O(n): bytes transferidos, memória de agregação, tempo de serialização JSON.
- `GET /collections/:id/context` **não tem paginação nem limite**. Com 500 documentos, é um JSON de dezenas de MB montado em memória.
- 20 documentos não é “centenas” (RNF03).

O que falta: curva de **latência p50/p95 e bytes de resposta × n ∈ {1, 10, 50, 100, 500, 1000}**. Com essa curva você troca uma afirmação retórica por um gráfico — e um gráfico é nota.

### 3.4 “Prioriza determinismo sobre RAG vetorial”

Determinismo nunca foi medido, nem no ThingIndex nem no comparador (que não existe). É, aliás, a métrica **mais fácil de vencer**: rode a mesma consulta 30 vezes e compare o hash do resultado. O relacional deve dar 100% idêntico; o denso com HNSW varia conforme `ef_search` e ordem de inserção. É uma vitória barata e mensurável que o trabalho está deixando na mesa.

### 3.5 O benchmark é auto-referente

`indexing_benchmark.go` define três variantes: `relational-tree-baseline` (baseline), `flat-hybrid-control` e `flat-page-control`. As três são implementações **do próprio ThingIndex**. Comparar-se consigo mesmo e declarar vitória é a definição de benchmark viciado. O próprio arquivo reconhece a lacuna nas `Limitations` (linha ~269) e nas notas (linhas 808-810). A banca vai ler o endpoint público `/meta/indexing-benchmark` — está tudo lá, escrito por você.

### 3.6 “Open source” + repositório privado

O artigo faz da abertura um argumento metodológico (“compromisso com a replicabilidade”, “o repositório será mantido público”). O `estado-do-projeto.md` (item 1.4) registra que **o repositório é privado**. Já existe `LICENSE` MIT. Ou torna público antes da defesa (com DOI no Zenodo, o que ainda fortalece o argumento DSR de replicabilidade), ou retira a afirmação.

### 3.7 Inconsistências numéricas que denunciam texto desatualizado

| Onde | Número | Realidade |
|---|---|---|
| Artigo, Modelagem de Dados | “20 tabelas” | `AutoMigrate` do E2E lista **21 modelos** (`routes_e2e_test.go:1355-1377`) |
| Artigo, Fig. classes | “14 classes” | idem — 21 |
| Artigo, RF15 | 2 idiomas | possivelmente 3 no frontend |
| Artigo, cronograma | “junho a dezembro de 2026” | o sistema já está implementado e com CI verde |

São detalhes, mas banca boa usa detalhe para calibrar confiança no resto.

### 3.8 O score heurístico de qualidade é circular

`AverageQualityScore` / `estimateIndexQualityScore` mede **completude estrutural observada pelo próprio sistema**. Usar a saída do sistema para avaliar o sistema é circularidade. O artigo já reconhece isso em comentário (“não será ground truth absoluto”), mas o RF11 e o painel expõem o número como se fosse qualidade. Ou você valida o score contra julgamento humano (Spearman sobre 50 documentos anotados) ou o rebaixa explicitamente a “indicador operacional”.

---

## 4. O experimento que falta: relacional × vetorial (executável, com pgvector)

Este é o núcleo do TCC II. Abaixo, um protocolo que cabe no semestre e produz tabela publicável.

### 4.1 Hipótese reformulada (falseável, de não-inferioridade)

Substitua “é possível sustentar…?” por:

> **H1 (precisão, não-inferioridade):** em corpora documentais estruturados de até 1.000 documentos, a recuperação relacional do ThingIndex é **não-inferior** à recuperação densa com pgvector em nDCG@10, com margem δ = 0,05.
>
> **H2 (auditabilidade, superioridade):** a recuperação relacional atinge taxa de proveniência completa ≥ 99% e determinismo de 100% em 30 repetições, enquanto a densa fica abaixo em ambas.
>
> **H3 (custo, direcional):** o custo total por 1.000 consultas do braço relacional é menor que o do denso **em coleções de até M documentos**, e maior acima de M — com M estimado empiricamente.

H3 escrita assim é o movimento mais inteligente disponível: transforma a fraqueza da Seção 3.1 em **achado de pesquisa** (“existe um ponto de inversão M”). Banca adora ponto de inversão.

### 4.2 Braços do experimento (4 obrigatórios + 1 opcional)

| ID | Braço | Implementação |
|---|---|---|
| **A** | ThingIndex relacional (outline + score lexical estrutural) | já existe: `relational-tree-baseline` |
| **B** | BM25 puro no Postgres | `tsvector` + `ts_rank_cd` com dicionário `portuguese`, ou extensão `pg_bm25`. **Controle lexical honesto** — hoje o `flat-hybrid-control` não é BM25 de verdade, e a banca de RI vai cobrar BM25 nominalmente (o artigo cita Robertson & Zaragoza e depois não usa) |
| **C** | Denso com pgvector | `pgvector/pgvector:pg16` (drop-in no compose), `CREATE EXTENSION vector`, embeddings `intfloat/multilingual-e5-base` (768d, roda local, sem custo de API, reprodutível) ou `text-embedding-3-small` (1536d) para ter custo em USD real. Índice HNSW (`m=16, ef_construction=64, ef_search=40`) |
| **D** | Híbrido BM25 + denso com RRF | Reciprocal Rank Fusion, k=60 (Cormack et al., 2009). É o estado da prática atual; sem ele a comparação está datada |
| E (opcional) | Reranker cross-encoder / ColBERT | Declare como **fora de escopo com justificativa** (custo computacional). Citar é obrigatório; implementar, não |

**Controle de chunking (fator ortogonal, 2 níveis):**
- (i) unidades estruturais do ThingIndex (`paragraph_nodes` / `outline_nodes`);
- (ii) chunking cego 512 tokens, overlap 64.

Rodar C e D nos dois regimes **isola a contribuição da estrutura** — permite afirmar “a estrutura ajuda” com evidência, em vez de por retórica. Isso sozinho já é uma contribuição publicável.

### 4.3 Corpus (três camadas)

| Camada | Composição | Papel | Ground truth |
|---|---|---|---|
| **C1 — controlado** | expandir o corpus embarcado de 5 → **30 documentos**, **150 consultas** rotuladas (5 por documento: factual, tabular, seccional, negativa, ambígua) | validade interna, regressão no CI | perfeito, por construção |
| **C2 — real, público, PT-BR** | **200 documentos** de fonte pública brasileira: acórdãos/decisões de TJ ou DOU/Diário Oficial (PDF nativo + digitalizados) + **200 PDFs** acadêmicos (SciELO/arXiv) | validade externa, é o que dá credibilidade | **100 consultas com qrels graduados (0/1/2)**, anotadas por você + 1 revisor, com **Kappa de Cohen reportado** |
| **C3 — escala** | replicação sintética de C2 até **1.000 documentos** | curva de custo/latência, RNF03 | não se aplica (só custo) |

> Se o tempo apertar, **C2 é inegociável** e C3 pode virar 500. Um trabalho com corpus só sintético não passa como empírico.
> Alternativa de menor esforço para comparabilidade com a literatura: acrescentar um subconjunto de **BEIR** (SciFact ou NFCorpus, qrels prontos). Ganha comparabilidade internacional, perde o argumento estrutural (é texto plano, não PDF). Idealmente rode os dois e reporte separado.

### 4.4 Métricas, por dimensão do artigo

**(i) Precisão na recuperação**
- P@1, P@5, Recall@10, MRR, **nDCG@10** (métrica principal — é o que a literatura de RI usa)
- por tipo de consulta (factual / tabular / seccional / negativa / ambígua) — a tabela por tipo é onde o relacional vai brilhar (tabular, seccional) e apanhar (ambígua/semântica). Mostrar isso é honestidade que rende nota.

**(ii) Auditabilidade** (métricas novas, que você define — é contribuição)
- **Taxa de proveniência completa (TPC):** % de itens retornados com `document_id`, `page_number`, `sequence_order` **todos não nulos e resolvíveis** por SELECT no banco. Mede diretamente o furo da Seção 3.2.
- **Taxa de citação verificável (TCV):** % de fontes citadas na resposta do chat cujo span aparece **literalmente** no parágrafo apontado (match exato ou Levenshtein ≤ 5%).
- **Determinismo:** % de execuções com resultado byte-idêntico em **30 repetições** da mesma consulta.
- **Explicabilidade do critério de match:** binário/ordinal — existe expressão auditável (SQL + score textual) que explique o ranking? (A: sim; B: sim; C: não, é distância em espaço latente; D: parcial.)
- **Custo de replay:** tempo (ms) e nº de consultas para reconstruir integralmente o contexto de uma resposta a partir do log de auditoria.

**(iii) Custo operacional**
- Tempo de indexação por documento: **p50 / p95**
- **USD de embedding** por 1.000 documentos (API) e/ou GPU-hora equivalente (local)
- **Bytes por documento**: tabelas relacionais vs. índice HNSW/IVFFlat (`pg_total_relation_size`)
- Latência de consulta **p50 / p95 / p99**
- **Tokens de prompt por consulta** e **USD por 1.000 consultas** — *a métrica que expõe o ponto de inversão M da H3*

### 4.5 Protocolo estatístico

- **5 execuções independentes** por braço (2 cold, 3 warm), hardware fixo e documentado (CPU, RAM, versão do Postgres, `shared_buffers`, `work_mem`).
- **30 repetições por consulta** para latência; descartar as 3 primeiras (warm-up).
- **Teste pareado por consulta:** Wilcoxon signed-rank (não assuma normalidade em nDCG), α = 0,05, **correção de Bonferroni** para as 6 comparações par-a-par entre 4 braços.
- **IC 95% por bootstrap** (1.000 reamostragens) em todas as médias.
- **Tamanho de efeito:** Cliff's delta (não-paramétrico), porque significância sem efeito não diz nada com n grande.
- Tudo com **seed fixa** e script único reexecutável (`make experiment`).

### 4.6 Esqueleto da tabela de resultados (o produto final)

**Tabela 1 — Precisão (corpus C2, 100 consultas, IC95% por bootstrap)**

| Braço | P@1 | P@5 | Recall@10 | MRR | nDCG@10 | p (vs A) |
|---|---|---|---|---|---|---|
| A — Relacional (ThingIndex) | | | | | | — |
| B — BM25 (tsvector) | | | | | | |
| C — Denso (pgvector, e5-base) | | | | | | |
| D — Híbrido RRF (B+C) | | | | | | |

**Tabela 2 — Auditabilidade**

| Braço | TPC (%) | TCV (%) | Determinismo (%) | Critério explicável | Replay (ms) |
|---|---|---|---|---|---|
| A | | | | SQL + score textual | |
| B | | | | ts_rank | |
| C | | | | distância latente | |
| D | | | | parcial | |

**Tabela 3 — Custo (por 1.000 documentos e por 1.000 consultas)**

| Braço | Indexação p50/p95 (s/doc) | USD indexação /1k docs | Armazenamento (MB/1k docs) | Consulta p50/p95/p99 (ms) | Tokens prompt/consulta | USD /1k consultas |
|---|---|---|---|---|---|---|
| A | | 0 (sem embeddings) | | | **← cresce com n** | |
| B | | 0 | | | | |
| C | | | | | | |
| D | | | | | | |

**Figura 1 — Ponto de inversão:** USD por 1.000 consultas × tamanho da coleção (n = 10…1000), quatro curvas. O cruzamento entre A e C é **o achado central do trabalho**.

### 4.7 Como implementar no repositório atual (caminho de menor atrito)

A boa notícia: **a abstração já existe**. `indexing_benchmark.go:734-769` define variantes de recuperação (`indexingBenchmarkRetrievalVariantDefinition`) com `ID`, `Label`, `Category`, `Status`, `Metrics`. Basta acrescentar variantes.

1. **Compose:** trocar `image: postgres:16` (linha 185 do `docker-compose.yml`) por `pgvector/pgvector:pg16`. Drop-in, mesma API. `CREATE EXTENSION IF NOT EXISTS vector;` no bootstrap.
2. **Schema:** nova tabela `document_chunk_embeddings (id, document_chunk_id FK, model text, dim int, embedding vector(768), created_at)` + índice HNSW. Aproveita `DocumentChunk`, que já existe com `SourceParagraphStart/End` — **a proveniência do braço vetorial sai de graça**, o que torna a comparação de auditabilidade honesta em vez de espantalho.
3. **Serviço de embedding:** container com `sentence-transformers` expondo `/embed`, ou usar o provider de LLM já existente (`pkg/provider/factory.go` já abstrai provedores).
4. **Novas variantes:** `vector-dense-hnsw`, `bm25-tsvector`, `hybrid-rrf` em `retrievalVariantDefinitions`, com `Status: "experimental"`. Isso faz o endpoint público `/meta/indexing-benchmark` **passar a publicar a comparação** — e o endpoint vira anexo do TCC, citável por URL.
5. **Novo comando:** `api/cmd/retrieval-eval/main.go` que carrega um *evalpack* (`corpus.json` + `qrels.json`), roda os 4 braços × 2 regimes de chunking × 5 execuções e emite `results.csv` + `results.json`. Segue o padrão dos comandos existentes (`cmd/indexing-benchmark` já é o molde).
6. **Análise:** script `scripts/analyze.py` (pandas + scipy) que consome o CSV e gera as 3 tabelas em LaTeX prontas para o artigo. Zero transcrição manual, zero erro de digitação na defesa.

**Esforço estimado:** 3–4 semanas de trabalho focado para 1–5; 1 semana para 6.

---

## 5. O que falta medir e onde instrumentar

| # | Métrica ausente | Onde instrumentar | Custo |
|---|---|---|---|
| 1 | Latência de `GET /collections/:id/context` × n docs | `usecase/document_usecase.GetCollectionContext`; devolver headers `X-Elapsed-Ms` e `X-Query-Count` | baixo |
| 2 | Contagem de consultas SQL **em produção** (hoje só em teste) | promover `newQueryCountingDB` (callback GORM em `gorm_collection_context_repository_integration_test.go:299`) a middleware opcional ativado por env | baixo |
| 3 | Bytes do payload de contexto × n | `Content-Length` já existe; basta coletar no script de carga | trivial |
| 4 | **Tokens de prompt por consulta de chat** | `buildCollectionChatPrompt` não registra tamanho; persistir `prompt_tokens` em `ChatConversationEntry` | baixo — **crítico para H3** |
| 5 | **Taxa de proveniência completa** | novo checador no benchmark que percorre o JSON de contexto e conta `PageNumber == nil`, `SequenceOrder` ausente, FK não resolvível | baixo — **crítico para H2** |
| 6 | Custo **medido** vs **estimado** | `DocumentMonitoringSummary.TotalEstimatedCostUSD` mistura os dois; acrescentar flag `cost_is_measured` por registro e reportar só o medido no artigo | baixo — evita afirmar número falso na banca |
| 7 | Determinismo | teste que roda a mesma consulta 30× e exige hash SHA-256 idêntico do resultado serializado | baixo |
| 8 | **Cobertura de testes (%)** | `go test -coverprofile` + `go tool cover` no CI; badge no README | trivial — é número que a banca pede |
| 9 | **Curva de escala (RNF03)** | script de carga (Go bench, k6 ou vegeta) com n ∈ {10, 50, 100, 500, 1000} | médio — fecha o único ❌ dos RNFs |
| 10 | Latência por **etapa** do pipeline | `DocumentProcessingAttempt` já existe; acrescentar colunas `parse_ms`, `extract_ms`, `llm_ms`, `persist_ms` | médio |
| 11 | Validação do score heurístico | anotar 50 documentos manualmente, correlacionar (Spearman) com `estimateIndexQualityScore` | médio — remove a circularidade da Seção 3.8 |
| 12 | Migrations versionadas | substituir `AutoMigrate` por golang-migrate/goose | médio — resolve a contradição do RNF02 |

---

## 6. Fraquezas metodológicas e de referencial teórico

### 6.1 Trabalhos relacionados são **produtos**, não pesquisa — e é o defeito mais visível

A seção inteira cobre Notion, LangChain e LlamaIndex. Nenhum deles é um trabalho acadêmico; nenhum tem paper revisado por pares. Uma seção de trabalhos relacionados sem trabalhos relacionados é o tipo de coisa que faz um avaliador rigoroso descontar antes mesmo de olhar o resto.

**Referências que precisam entrar (mínimo 10–15):**

| Área | Trabalho | Por quê |
|---|---|---|
| RAG estruturado | **GraphRAG** — Edge et al. (2024, Microsoft) | é o concorrente direto da tese (estrutura > vetor puro). O artigo cita “grafos” genericamente e nunca o trabalho canônico |
| RAG adaptativo | **Self-RAG** — Asai et al. (2024, ICLR) | resposta moderna à opacidade que o artigo aponta como lacuna |
| Retrieval denso | **DPR** — Karpukhin et al. (2020, EMNLP) | fundamento de tudo que o artigo chama de “RAG convencional”; hoje só há Lewis et al. |
| Late interaction | **ColBERT / ColBERTv2** — Khattab & Zaharia (2020); Santhanam et al. (2022) | a banca vai perguntar “por que não ColBERT?”. Você precisa ter a resposta citada |
| Esparso neural | **SPLADE** — Formal et al. (2021) | mostra que léxico ≠ ultrapassado — **argumento a favor da sua tese**, hoje desperdiçado |
| Fusão | **RRF** — Cormack, Clarke & Büttcher (2009) | base do braço D |
| Avaliação | **BEIR** — Thakur et al. (2021, NeurIPS) | benchmark padrão; dá comparabilidade |
| Avaliação de RAG | **RAGAS** — Es et al. (2023) | métricas de faithfulness/groundedness — exatamente o que “auditabilidade” quer capturar |
| Proveniência | **Cheney, Chiticariu & Tan (2009)** — *Provenance in Databases: Why, How, and Where* | Buneman (2001) sozinho é raso para uma tese que **se chama** auditável. O tripé why/how/where é o vocabulário técnico que falta |
| Proveniência | **W3C PROV-DM** (2013) | padrão de modelagem — dá vocabulário formal à trilha de auditoria |
| NL→SQL | **Spider** — Yu et al. (2018) | “recuperação por SQL determinístico” tem literatura própria e o artigo a ignora |
| Layout/documento | **LayoutLMv3** — Huang et al. (2022) | o artigo cita LayoutLM (2020), desatualizado |
| Estrutura de PDF | **Nougat** — Blecher et al. (2023) / **PageIndex** | o código **já compara** com exports do `pageindex` (`reference-workspace.json`), e o artigo não menciona. Evidência jogada fora |
| Avaliação em RI | **Voorhees & Harman** — paradigma Cranfield/TREC | justifica formalmente qrels + pooling |

### 6.2 A avaliação DSR está incompleta segundo o próprio Hevner

O artigo cita Hevner et al. (2004) como método, mas Hevner define cinco métodos de *Design Evaluation* (observacional, analítico, experimental, teste, descritivo) e exige que o artefato seja avaliado **contra alternativas**. O trabalho hoje faz apenas “teste” (funcional). Declare explicitamente: **experimental controlado (Tabelas 1–3) + analítico (complexidade de consultas: 9 constantes) + observacional (estudo de caso com corpus jurídico real)**. Isso alinha método e execução e desarma a pergunta “onde está a avaliação DSR?”.

### 6.3 Não existe seção de Ameaças à Validade

Obrigatória em trabalho empírico. Deve cobrir:
- **Interna:** o autor construiu o sistema **e** anotou os qrels → viés de anotação. Mitigação: segundo anotador + Kappa de Cohen reportado.
- **Externa:** corpus de domínio restrito; não generaliza para domínio aberto — e o artigo **já admite isso em comentário** (linha 844, comentada!). Descomente e transforme em seção.
- **De construto:** “auditabilidade” é operacionalizada por TPC/TCV/determinismo — métricas propostas pelo autor, não validadas pela comunidade. Declare.
- **De conclusão:** n=100 consultas, poder estatístico limitado; reportar tamanho de efeito junto com p-valor.

### 6.4 Ausência de avaliação com usuários

RNF01 fala em “navegação clara” sem instrumento. Ou aplique **SUS (System Usability Scale) com 8–12 participantes** — barato, uma tarde, e rende uma tabela — ou remova o adjetivo. Deixar como está é convite para a pergunta “clara segundo quem?”.

### 6.5 Tempo verbal e estrutura: reescrita obrigatória

O artigo inteiro está em futuro (“será implementado”, “espera-se”, “o desenvolvimento será realizado entre junho e dezembro de 2026”) enquanto o sistema está pronto, testado e com CI verde. No TCC II isso precisa virar:
- Seções 6.x (Proposta/Arquitetura) → **pretérito/presente**, descrevendo o que foi construído, com métricas reais (LOC, nº de rotas, nº de tabelas, cobertura).
- Seção “Avaliação da Qualidade” → dividir em **Metodologia Experimental** (protocolo) e **Resultados** (tabelas) — hoje as duas coisas estão fundidas em um plano.
- Cronograma → **execução realizada** + o que ficou fora.
- Considerações Finais → substituir “espera-se que venha a representar contribuição” por a contribuição, com número.
- Descomentar e promover a Seção 10 (Observabilidade, Benchmark, Testes) — está toda comentada no `.md` e é **o melhor material do trabalho**, escondido.

### 6.6 Delimitação honesta que falta

O artigo nunca diz **onde a abordagem relacional perde**. Existe um parágrafo perfeito sobre isso — comentado, linha 844 (“em domínios abertos e corpus massivos, embeddings provavelmente superarão joins hierárquicos”). Traga para o corpo. Trabalho que declara seu limite é lido como maduro; trabalho que só se elogia é lido como propaganda.

---

## 7. Plano do TCC II, priorizado por impacto na nota

| Fase | Duração | Entrega | Impacto na nota |
|---|---|---|---|
| **F0 — Higiene do texto** | 1 sem | Tempo verbal corrigido; hipóteses H1–H3 falseáveis; números (21 tabelas, idiomas) conferidos; seção de Ameaças à Validade; limitação da Seção 3.1 declarada | 🔴 **Máximo** — sem isso nada mais importa |
| **F1 — Referencial** | 1 sem | +12 referências acadêmicas (6.1); Trabalhos Relacionados reescrito com **tabela comparativa** ThingIndex × GraphRAG × ColBERT × híbrido × LlamaIndex | 🔴 **Máximo** — hoje é o buraco mais visível |
| **F2 — Instrumentação** | 2 sem | Itens 1–8 da Seção 5 (latência, tokens de prompt, TPC, determinismo, cobertura) | 🔴 **Máximo** — sem instrumento não há dado |
| **F3 — Corpus + qrels** | 2 sem | C1 (30 docs/150 queries) + C2 (200+200 docs, 100 queries com qrels e Kappa) | 🔴 **Máximo** |
| **F4 — pgvector + BM25 + RRF** | 2 sem | Braços B, C, D como variantes; `cmd/retrieval-eval`; compose com `pgvector/pgvector:pg16` | 🔴 **Máximo** — é a baseline ausente |
| **F5 — Execução** | 1 sem | 4 braços × 2 chunkings × 5 execuções; `results.csv`; curva de escala n até 1.000 (fecha RNF03) | 🔴 **Máximo** |
| **F6 — Análise + escrita** | 2 sem | Tabelas 1–3 + Figura do ponto de inversão, com Wilcoxon + Bonferroni + IC bootstrap + Cliff's delta; seções Resultados e Discussão | 🔴 **Máximo** |
| **F7 — Fechar RFs parciais** | 3 dias | logout, DELETE de categoria/coleção, rename de coleção, rate limit em upload/chat, `@Summary` nas rotas faltantes | 🟠 Médio — evita constrangimento em demo ao vivo |
| **F8 — Extras** | 1 sem | Migrations versionadas; repo público + DOI Zenodo; SUS n=10; validação do score heurístico (Spearman) | 🟡 Bônus |

**Caminho crítico:** F0 → F1 → F2 → F3 → F4 → F5 → F6 ≈ **11 semanas**. F7/F8 em paralelo.

**Se só houver tempo para uma coisa:** F4 + F5 + F6 (a baseline comparativa). Um TCC com sistema mediano e experimento rigoroso defende melhor que um TCC com sistema excelente e zero comparação — que é exatamente a situação atual.

---

## 8. Perguntas que a banca vai fazer — e o estado atual da resposta

| Pergunta | Resposta hoje | Resposta necessária |
|---|---|---|
| “Comparou com RAG vetorial?” | Não | Tabelas 1–3, 4 braços |
| “Quanto custa uma consulta comparada a um vector DB?” | Desconhecido — e provavelmente **mais**, porque envia a coleção inteira | Figura do ponto de inversão M |
| “Qual o percentual de proveniência completa?” | Não medido; `PageNumber` é nulável | TPC ≥ 99% medida, com as exceções nomeadas |
| “Centenas de documentos sem degradação — comprovou?” | Só 1 vs 20 documentos, contando consultas | Curva p50/p95 até n=1.000 |
| “Qual a cobertura de testes?” | Não publicada | `-coverprofile` no CI + badge |
| “Por que não GraphRAG / ColBERT / híbrido?” | Não citados | Citados + tabela comparativa + braço D implementado |
| “Onde sua abordagem perde?” | Só em comentário LaTeX | Seção de limitações no corpo, com dado |
| “O repositório é aberto?” | É privado; o artigo diz que é aberto | Público + MIT + DOI |
| “Quantas tabelas são?” | 20 no artigo, 14 no diagrama, 21 no código | Um número só |

---

## 9. Fecho

O ThingIndex tem hoje **um artefato de engenharia forte e uma pesquisa fraca**. Não é falta de trabalho — é falta de comparação. Tudo que sustenta a tese (proveniência granular, 9 consultas constantes, telemetria de custo, trilha de auditoria, benchmark reproduzível embarcado) **já está no código**; o que falta é apontar esses instrumentos para um adversário real e publicar o número, inclusive quando o número for desfavorável.

O risco maior não é o experimento dar “errado”. Um resultado do tipo *“o relacional empata em precisão estrutural, ganha por larga margem em auditabilidade e determinismo, e perde em custo acima de M documentos”* é **um resultado melhor** do que uma vitória em todas as frentes — é crível, é específico, e delimita exatamente quando usar a abordagem. O risco real é chegar na defesa com as mesmas três afirmações não medidas que o TCC I já fazia.
