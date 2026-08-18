# Auditoria do backend Go — ThingIndex

**Escopo:** `api/` (usecase, indexing, repository, handlers, api, pkg/provider, queue). Build/vet/testes já limpos — este relatório ignora higiene de ferramenta e busca só defeito substantivo.

**Veredito honesto:** não encontrei vazamento de dados entre usuários. O escopo por `user_id` está consistentemente aplicado nas rotas expostas — toda entrada em recurso de outro usuário passa por `GetDocument`/`GetCollectionByID`/`GetConversationByID` com `WHERE ... AND user_id = ?` antes de tocar em dado derivado, e as tabelas estruturadas (que de fato não têm `user_id`) só são alcançáveis via `document_id` já autorizado. **Não há IDOR.** Também não há SQL injection (GORM parametrizado em toda parte), nem path traversal explorável (nome do objeto é gerado pelo servidor com UUID, e `Content-Disposition` é sanitizado). A criptografia de credenciais (AES-GCM com chave derivada por usuário) está correta.

O que **de fato** está quebrado é operacional: **idempotência do outbox, rate limiting e resiliência do worker**. Abaixo, 12 itens ordenados por severidade real.

---

## P0 — bug real que quebra ou duplica trabalho

### 1. O outbox "claim" não reivindica nada: o mesmo documento é publicado e indexado 2× (ou N×)

**`api/internal/repository/gorm_outbox_repository.go:22-53`**

```go
func (r *gormOutboxRepository) ClaimPendingEvents(ctx context.Context, limit int, now time.Time) ([]models.OutboxEvent, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status IN ? AND available_at <= ?", []string{domain.OutboxEventStatusPending, domain.OutboxEventStatusFailed}, now).
			...
		return tx.Model(&models.OutboxEvent{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"attempts":   gorm.Expr("attempts + 1"),   // <-- só incrementa tentativa
				"updated_at": now,                          // <-- status continua PENDING
			}).Error
	})
```

A função se chama `ClaimPendingEvents` e o `FOR UPDATE SKIP LOCKED` sugere reivindicação, mas **o `status` nunca sai de `PENDING`** e `available_at` não é adiantado. O lock morre junto com a transação, que fecha antes da publicação (`dispatchOnce` só publica depois, em `outbox_dispatcher.go:52-67`). Consequências reais:

- **Duas réplicas do `indexer`** (o `docker-compose.yml` não impede escalar) reivindicam o mesmo evento no tick seguinte → `PublishDocumentIndexRequested` duplicado → `DocumentProcessor.Process` roda 2× no mesmo documento → **custo de LLM dobrado**, `ReplaceIndexingResult` concorrente na mesma transação e status oscilando.
- Mesmo com **uma** réplica: se `MarkEventPublished` falhar (`outbox_dispatcher.go:64`, que aborta o batch), o evento já publicado continua `PENDING` e é republicado no próximo `pollInterval` (5s por padrão) — indefinidamente.

O teste de integração existente (`gorm_outbox_repository_integration_test.go:52`) só verifica filtro e incremento de `attempts`, por isso o defeito passa.

**Correção:** marcar dentro da mesma transação do claim, e só então publicar.
```go
Updates(map[string]any{
    "status":       domain.OutboxEventStatusInFlight, // novo estado
    "attempts":     gorm.Expr("attempts + 1"),
    "available_at": now.Add(visibilityTimeout), // reentrega se o worker morrer
    "updated_at":   now,
})
```
E tornar o consumidor idempotente: descartar `DocumentIndexRequested` cujo documento já esteja `INDEXED` com `indexed_at` posterior ao evento.

---

### 2. Rate limiting de auth colapsa num único balde global — brute force por conta é ilimitado e o login é trivialmente derrubável

**`api/internal/api/middleware/rate_limit.go:75`** + **`api/internal/api/routes.go:110-117`**

```go
key := c.ClientIP() + ":" + c.Request.Method + ":" + c.FullPath()
```

O frontend Next.js chama a API **server-side**, de dentro do container (`frontend/src/components/auth/actions.ts:21` → `fetchApiUpstream` → `http://api:8080/api/v1`). Logo `c.ClientIP()` é **sempre o IP do container do frontend**, para todos os usuários. Reforçando: `TRUSTED_PROXIES=` vem vazio no `.env.example:8`, então `SetTrustedProxies(nil)` faz o gin ignorar `X-Forwarded-For` e usar o peer TCP direto. Dois efeitos, ambos ruins:

- **Segurança:** o limite não é por conta nem por cliente real. Um atacante distribuindo tentativas contra `POST /auth/login` compartilha o mesmo balde com todo mundo, e nada limita tentativas *por e-mail* — não há bloqueio de conta nem backoff. A defesa contra brute force efetivamente não existe.
- **Disponibilidade:** com o padrão `AUTH_RATE_LIMIT_REQUESTS=10` / `60s` (`config.go:116-117`), **o 11º login de qualquer usuário no minuto recebe 429**. Em uma banca com poucos acessos simultâneos isso já dispara.

**Correção:** limitar por identidade além de IP (`email` normalizado do corpo para `/login` e `/forgot-password`), propagar o IP real do usuário do frontend para a API (header assinado ou `X-Forwarded-For` com `TRUSTED_PROXIES` configurado), e adicionar backoff progressivo por conta.

---

### 3. Consumidor é estritamente sequencial e sem timeout por mensagem — um PDF pesado trava o pipeline inteiro

**`api/internal/queue/rabbitmq.go:204-256`** e **`api/cmd/indexer/main.go:87`**

```go
for {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case delivery, ok := <-deliveries:
		...
		if err := handler(ctx, event); err != nil {   // ctx = contexto do processo, sem deadline
```

O `ctx` repassado ao handler é o contexto do processo inteiro (`signal.NotifyContext`), **sem nenhum deadline por mensagem**. E o loop processa uma mensagem por vez, numa única goroutine — `ConsumerPrefetchCount=10` só enche o buffer, não paraleliza. Dentro de `Process` há operações sem limite próprio: OCR página-a-página via `exec.CommandContext` (`helpers.go:328-352`), e o cliente LLM tem timeout de 45s **por requisição** (`factory.go:159`), mas o enricher faz 4+ chamadas sequenciais (`enricher.go:72-93`) sem orçamento agregado.

Resultado: um único documento escaneado grande, ou um endpoint de LLM lento, **para a fila inteira** por tempo indeterminado. Todos os outros documentos ficam `QUEUED` sem sintoma no monitoramento. Não há reaper — busquei e confirmei que nada no repositório reseta documentos presos em `PROCESSING`.

**Correção:** derivar contexto com deadline por mensagem e processar em pool limitado.
```go
msgCtx, cancel := context.WithTimeout(ctx, cfg.DocumentIndexProcessingTimeout) // ex.: 10min
err := handler(msgCtx, event)
cancel()
```
Mais um job periódico que reenfileira documentos em `PROCESSING` com `index_started_at` mais antigo que o timeout.

---

## P1 — risco sério

### 4. Stream de chat (SSE) é cortado pelo `WriteTimeout` de 30s

**`api/cmd/api/main.go:58-64`** + **`api/internal/handlers/chat.go:230-277`**

```go
server := &http.Server{
	WriteTimeout: config.AppConfig.ServerWriteTimeout,  // padrão 30s (config.go:152)
```

`WriteTimeout` no `net/http` é um **deadline absoluto** contado do início da requisição, não um idle timeout. `StreamCollectionChat` é SSE de longa duração: qualquer resposta que leve mais de 30s para terminar tem a conexão fechada no meio, e o cliente recebe uma resposta truncada. Pior: o texto parcial acumulado em `answerBuilder` **não é persistido** — `chat_usecase.go:241-263` só grava a mensagem do assistente depois que o stream termina com sucesso. O usuário perde a resposta e o histórico fica com a pergunta sem resposta.

**Correção:** aplicar o deadline só nas rotas não-streaming, ou zerá-lo no handler de SSE:
```go
if rc := http.NewResponseController(c.Writer); rc != nil {
    _ = rc.SetWriteDeadline(time.Time{}) // sem deadline para SSE
}
```
E persistir a resposta parcial quando o stream é interrompido.

---

### 5. Validação de listas é código morto: falta `dive` em todos os DTOs de coleção

**`api/internal/dto/catalog.go:16-47`**

```go
type ParagraphNodeInput struct {
	SequenceOrder int  `binding:"required,gte=0"`
	PageNumber    *int `binding:"omitempty,gte=1"`
	Content       string `binding:"required"`
}
type StoreParagraphNodesRequest struct {
	Nodes []ParagraphNodeInput `json:"nodes" binding:"required"`  // <-- sem 'dive'
}
```

Sem `dive`, o `validator` **não desce nos elementos do slice**. Confirmei empiricamente rodando gin + validator: o payload `{"nodes":[{"sequence_order":-7,"page_number":-99,"content":""}]}` retorna **200 ACEITO** hoje, e **400** com `binding:"required,dive"`. Vale para `StoreParagraphNodesRequest`, `StoreTableSectionsRequest` e `StoreMetadataPairsRequest`.

Isso permite gravar `sequence_order` negativo, `page_number` negativo e `content` vazio direto nas tabelas estruturadas — dados que a ordenação (`ORDER BY sequence_order`) e o retrieval assumem sanos. (Nota lateral: `required` em `int` também rejeita `0`, o valor legítimo do primeiro item; ao adicionar `dive` isso vira um bug visível — use `gte=0` sozinho.)

**Correção:**
```go
Nodes []ParagraphNodeInput `json:"nodes" binding:"required,dive"`
// e trocar `required,gte=0` por `gte=0` em SequenceOrder
```

---

### 6. Contexto de coleção inteira carregado em RAM, sem paginação, a cada requisição

**`api/internal/repository/gorm_collection_context_repository.go:48-90`**

```go
if len(documentIDs) > 0 {
	r.db.WithContext(ctx).Where("document_id IN ?", documentIDs).Find(&documentPages)
	r.db.WithContext(ctx).Where("document_id IN ?", documentIDs).Find(&paragraphNodes)
	r.db.WithContext(ctx).Where("document_id IN ?", documentIDs).Find(&chunks)
	... // 6 queries, nenhuma com LIMIT
}
```

Carrega **todas as páginas, parágrafos, tabelas, chunks, metadados e nós de outline de todos os documentos da coleção**, com o texto completo (`Content string` é `type:text`), para dentro do processo. É chamado em `GET /collections/:id/context` e **em toda mensagem de chat** (`chat_usecase.go:156`).

Uma coleção de 50 PDFs de 200 páginas gera dezenas de milhares de linhas com texto integral por requisição. Como não há limite de upload por usuário nem quota, um único usuário derruba o processo por OOM. O agravante é que o consumo é quase todo desperdiçado: `buildChatSources` usa no máximo 6 fontes (`chat_usecase.go:302`) e o prompt corta em 6 documentos (`chat_usecase.go:413`).

**Correção:** empurrar a seleção para o banco — filtrar por termos com índice full-text (`tsvector`) e `LIMIT`, carregando conteúdo só das páginas/chunks que entrarem no top-N, em vez de materializar a coleção toda em Go.

---

### 7. Erros de persistência de estado são engolidos no worker — documento fica "PROCESSING" para sempre, em silêncio

**`api/internal/processing/document_processor.go:47-56`**

```go
updateState := func(status, stage, detail, lastError string, completedAt *time.Time) {
	_ = p.repo.UpdateDocumentProcessingState(ctx, event.DocumentID, domain.DocumentProcessingState{...})
}
```

O `_ =` descarta o erro em **todas as ~10 transições de estágio**. Se o banco cair, ou — mais provável — se o `ctx` for cancelado no meio (SIGTERM, ou o timeout do item #3 quando existir), toda transição subsequente falha em silêncio, inclusive as que marcam `FAILED`/`ORPHANED`. O documento fica congelado no último estágio gravado e **nada no sistema o resgata** (não há reaper, ver item #3).

O mesmo padrão está no caminho de erro de `DeleteDocument` (`document_usecase.go:515`, `534`, `536`) e `DeleteCollectionDocuments` (`document_usecase.go:576`, `582`, `603`, `609`), onde a falha em gravar a auditoria de exclusão desaparece — justamente a trilha que precisaria existir para diagnosticar a exclusão parcial.

**Correção:** logar todas as falhas de `updateState` com `documentID` e estágio, e nas transições terminais (`FAILED`/`ORPHANED`) usar um contexto desacoplado do cancelamento:
```go
finalCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
defer cancel()
```

---

### 8. Corte de string em bytes quebra UTF-8 e corrompe títulos acentuados

**`api/internal/usecase/chat_usecase.go:277-279`** e **`api/internal/repository/gorm_chat_repository.go:155-157`**

```go
if len(normalized) > 255 {
	return strings.TrimSpace(normalized[:255])   // len() = bytes, não runes
}
```

```go
if len(trimmed) > 80 {
	return strings.TrimSpace(trimmed[:80]) + "..."
}
```

Confirmei rodando: um título de 200 caracteres acentuados (400 bytes) sai de `buildConversationTitle` com `utf8.ValidString == false` — o corte cai no meio de um code point de 2 bytes. Em português isso é o caso comum, não o exótico. O resultado é um título com `U+FFFD` no fim, gravado no banco e serializado em JSON. Note que o resto do código já faz certo (`clampRetrievalSnippet` em `document_usecase.go:2247-2253` converte para `[]rune`) — estes dois pontos ficaram para trás.

**Correção:** usar o helper que já existe, ou cortar por runes:
```go
r := []rune(normalized)
if len(r) > 255 { return strings.TrimSpace(string(r[:255])) }
```

---

### 9. Upload sem rate limit, sem quota e com `Content-Length` vindo do cliente

**`api/internal/api/routes.go:138-141`** e **`api/internal/usecase/document_usecase.go:363-368`**

O grupo `/documents` só tem `AuthMiddleware` — **nenhum rate limit**, ao contrário de `/auth`. E o tamanho vem do cliente:

```go
if size > u.maxUploadSizeBytes { return nil, domain.ErrFileTooLarge }
...
fileURL, err := storageClient.UploadFile(ctx, validatedReader, size, resolvedContentType, objectName)
```

`size` é `fileHeader.Size` (`handlers/document.go:70`), derivado do multipart. O `http.MaxBytesReader` em `document.go:39` limita o corpo, então não há bypass do teto de 20 MB — mas não há **nada** limitando o número de uploads. Um usuário autenticado pode enfileirar milhares de documentos, cada um disparando OCR e chamadas de LLM no worker sequencial do item #3, esgotando storage, banco e crédito de API.

**Correção:** aplicar `RateLimitMiddleware` (chaveado por `user_id`, não por IP) em `/documents/upload` e adicionar quota por usuário (contagem e bytes totais) verificada antes do `UploadFile`.

---

## P2 — melhoria

### 10. `ListProviders` faz N+1 de descriptografia e engole o erro

**`api/internal/handlers/provider.go:209-219`**

```go
for i := range providers {
	credentials, err := h.useCase.GetDecryptedCredentials(c.Request.Context(), userID, providers[i].ProviderType)
	if err != nil {
		continue   // silencioso
	}
```

Uma query `SELECT` **por provider** (`GetConfig` em `gorm_provider_repository.go:209`) — sendo que `ListUserProviders` já trouxe todos os registros, incluindo o `EncryptedData`. Além do N+1, um `continue` mudo esconde corrupção de credencial ou rotação de `MASTER_ENCRYPTION_KEY`: o usuário simplesmente vê o provider sem metadados, sem explicação. `GetProviderRuntimeOverview` (`provider_usecase.go:370`) já faz o certo, decifrando a partir da lista em memória.

**Correção:** decifrar direto de `providers[i]` (sem ida ao banco) e logar a falha com `providerType` + `userID`.

---

### 11. Login expõe oráculo de enumeração de usuários por temporização

**`api/internal/usecase/auth_usecase.go:78-89`**

```go
user, err := u.repo.GetUserByEmail(ctx, normalizeEmail(email))
if err != nil {
	return "", domain.ErrInvalidCredentials   // retorna imediatamente
}
if err := bcrypt.CompareHashAndPassword(...); err != nil {   // ~100ms
```

A mensagem é idêntica nos dois casos (bom), mas o **tempo** não: e-mail inexistente responde em ~1ms, existente em ~100ms (custo do bcrypt). Diferença trivialmente mensurável pela rede. Combinado com o item #2 (rate limit inútil), permite enumerar a base de usuários. `ForgotPassword` já trata isso corretamente (`auth_usecase.go:94-96`).

**Correção:** comparar contra um hash bcrypt dummy fixo quando o usuário não existir, igualando o tempo de resposta.

---

### 12. Dados estruturados são apagados em soft delete e ficam órfãos para sempre

**`api/internal/repository/gorm_catalog_document_repository.go:469-489`**

```go
for _, model := range []any{ &models.DocumentPage{}, &models.ParagraphNode{}, ... } {
	if err := tx.Where("document_id IN ?", documentIDs).Delete(model).Error; err != nil {
```

Sem `Unscoped()`, e todos esses modelos têm `gorm.DeletedAt` — então é **soft delete**. As linhas permanecem na tabela indefinidamente após a exclusão do documento, com o texto integral. Note a inconsistência interna: os caminhos de reindexação usam `Unscoped()` para a mesma limpeza (`gorm_structured_data_repository.go:21`, `gorm_document_processing_repository.go:69`). Para um TCC que documenta exclusão de documento como remoção efetiva, o conteúdo continua no banco.

**Correção:** usar `tx.Unscoped().Where("document_id IN ?", documentIDs).Delete(model)`, coerente com o resto do código.

---

## Onde eu procurei e não achei problema

Registro explícito para não inflar a lista:

- **Multi-tenancy / IDOR:** revisei as 86 funções de repositório. Todas as consultas alcançáveis por rota filtram por `user_id`. As tabelas estruturadas não têm `user_id`, mas o acesso é sempre precedido de verificação de posse do `document_id`/`collection_id`. `GetDocumentByIDForProcessing` (sem escopo) só é chamado pelo worker, a partir de evento do outbox — não é alcançável por HTTP.
- **SQL injection:** todas as consultas usam placeholders. `escapeLikePattern` (`audit_query_helpers.go:60`) trata `%`/`_` corretamente. `loadAuditBuckets` interpola nome de coluna, mas só recebe literais internos (`"action"`, `"status"`).
- **Path traversal:** o nome do objeto é gerado pelo servidor (`timestamp + UUID + extensão de allowlist`, `document_usecase.go:390`); o nome do cliente nunca chega ao storage. `buildSafeContentDisposition` remove separadores e caracteres de controle.
- **SSRF:** bem defendido — `netsec.StorageEndpointValidator` resolve DNS e bloqueia faixas privadas, e o `DialContext` customizado (`minio.go:97`) impede rebinding entre validação e conexão.
- **Criptografia:** AES-GCM com chave derivada por HMAC-SHA256 por usuário, nonce novo a cada operação, com testes cobrindo troca de usuário/chave e ciphertext adulterado.
- **Panics:** nenhum índice de slice sem guarda nos caminhos quentes; `allHits[0]` (`document_usecase.go:1222`) e `baseNodeHits[0]` (`:1291`) estão protegidos por checagem de tamanho acima.
- **Overflow:** `parseUintParam` usa `ParseUint(..., 32)` corretamente para `uint`.