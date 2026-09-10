package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
)

const (
	chatStreamEventStage   = "stage"
	chatStreamEventDelta   = "delta"
	chatStreamEventSources = "sources"
	chatStreamEventDone    = "done"
)

type chatUseCase struct {
	collectionContextRepo domain.CollectionContextRepository
	conversationRepo      domain.ChatConversationRepository
	llmResolver           domain.UserLLMResolver
}

func NewChatUseCase(
	collectionContextRepo domain.CollectionContextRepository,
	conversationRepo domain.ChatConversationRepository,
	llmResolver domain.UserLLMResolver,
) domain.ChatUseCase {
	return &chatUseCase{
		collectionContextRepo: collectionContextRepo,
		conversationRepo:      conversationRepo,
		llmResolver:           llmResolver,
	}
}

func (u *chatUseCase) CreateConversation(ctx context.Context, userID uint, collectionID uint, title string) (*models.ChatConversation, error) {
	if collectionID == 0 {
		return nil, domain.ErrCollectionRequired
	}

	if _, err := u.collectionContextRepo.GetCollectionContext(ctx, userID, collectionID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCollectionNotFound
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrCollectionFind, err)
	}

	conversation := &models.ChatConversation{
		UserID:       userID,
		CollectionID: collectionID,
		Title:        buildConversationTitle(title),
	}
	if err := u.conversationRepo.CreateConversation(ctx, conversation); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrChatConversationCreate, err)
	}

	return conversation, nil
}

func (u *chatUseCase) ListConversations(ctx context.Context, userID uint, collectionID uint) ([]models.ChatConversation, error) {
	if collectionID == 0 {
		return nil, domain.ErrCollectionRequired
	}

	conversations, err := u.conversationRepo.ListConversationsByCollection(ctx, userID, collectionID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrChatConversationFind, err)
	}

	return conversations, nil
}

func (u *chatUseCase) GetConversationMessages(ctx context.Context, userID uint, conversationID uint) (*models.ChatConversation, []models.ChatConversationEntry, error) {
	conversation, err := u.conversationRepo.GetConversationByID(ctx, userID, conversationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, domain.ErrChatConversationNotFound
		}
		return nil, nil, fmt.Errorf("%w: %v", domain.ErrChatConversationFind, err)
	}

	messages, err := u.conversationRepo.ListMessagesByConversation(ctx, userID, conversationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, domain.ErrChatConversationNotFound
		}
		return nil, nil, fmt.Errorf("%w: %v", domain.ErrChatConversationFind, err)
	}

	return conversation, messages, nil
}

func (u *chatUseCase) UpdateConversationTitle(ctx context.Context, userID uint, conversationID uint, title string) (*models.ChatConversation, error) {
	normalizedTitle := strings.Join(strings.Fields(strings.TrimSpace(title)), " ")
	if normalizedTitle == "" {
		return nil, domain.ErrTitleRequired
	}

	conversation, err := u.conversationRepo.UpdateConversationTitle(ctx, userID, conversationID, normalizedTitle)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrChatConversationNotFound
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrChatConversationFind, err)
	}

	return conversation, nil
}

func (u *chatUseCase) DeleteConversation(ctx context.Context, userID uint, conversationID uint) error {
	if err := u.conversationRepo.DeleteConversation(ctx, userID, conversationID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrChatConversationNotFound
		}
		return fmt.Errorf("%w: %v", domain.ErrChatConversationDelete, err)
	}

	return nil
}

func (u *chatUseCase) StreamCollectionChat(
	ctx context.Context,
	userID uint,
	request domain.ChatRequest,
	onEvent func(domain.ChatStreamEvent) error,
) error {
	prompt := strings.TrimSpace(request.Prompt)
	if request.CollectionID == 0 || prompt == "" {
		return domain.ErrCollectionRequired
	}

	conversation, err := u.conversationRepo.GetConversationByID(ctx, userID, request.ConversationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrChatConversationNotFound
		}
		return fmt.Errorf("%w: %v", domain.ErrChatConversationFind, err)
	}
	if conversation.CollectionID != request.CollectionID {
		return domain.ErrChatConversationNotFound
	}

	if err := onEvent(domain.ChatStreamEvent{
		Type:    chatStreamEventStage,
		Stage:   "retrieving_context",
		Message: "Reconstructing collection context.",
	}); err != nil {
		return err
	}

	collectionContext, err := u.collectionContextRepo.GetCollectionContext(ctx, userID, request.CollectionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrCollectionNotFound
		}
		return fmt.Errorf("%w: %v", domain.ErrCollectionFind, err)
	}

	retrieval := buildCollectionRetrieval(
		prompt,
		collectionContext.Documents,
		collectionContext.DocumentPages,
		collectionContext.OutlineNodes,
		collectionContext.Chunks,
		collectionContext.TableSections,
	)
	sources := buildChatSources(collectionContext, retrieval)
	if len(sources) > 0 {
		if err := onEvent(domain.ChatStreamEvent{
			Type:    chatStreamEventStage,
			Stage:   "selecting_sources",
			Message: "Selecting grounded document evidence.",
		}); err != nil {
			return err
		}
		if err := onEvent(domain.ChatStreamEvent{
			Type:    chatStreamEventSources,
			Sources: sources,
		}); err != nil {
			return err
		}
	}

	llm, err := u.llmResolver.ResolveLLMForChat(ctx, userID)
	if err != nil {
		return err
	}

	if err := u.conversationRepo.AddMessage(ctx, userID, conversation.ID, &models.ChatConversationEntry{
		Role:    "user",
		Content: prompt,
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrChatConversationNotFound
		}
		return fmt.Errorf("%w: %v", domain.ErrChatConversationMessageStore, err)
	}

	systemPrompt, userPrompt := buildCollectionChatPrompt(collectionContext, retrieval, request, sources)
	if err := onEvent(domain.ChatStreamEvent{
		Type:    chatStreamEventStage,
		Stage:   "generating_answer",
		Message: "Generating grounded answer.",
	}); err != nil {
		return err
	}

	var answerBuilder strings.Builder

	// persistAnswer grava o que houver de resposta ate agora.
	//
	// Precisa acontecer TAMBEM no caminho de erro. Antes, um `return err` no
	// meio do streaming pulava a persistencia inteira: o usuario via o texto
	// aparecer na tela, a conexao caia, e ao recarregar a conversa a resposta
	// tinha sumido -- sem nenhum registro de que a chamada de LLM foi feita e
	// paga. Guardar o parcial e melhor que perder tudo.
	var usage *domain.ChatUsage
	persistAnswer := func() error {
		answer := strings.TrimSpace(answerBuilder.String())
		if answer == "" {
			return nil
		}
		// Medido depois da resposta (o provedor so informa consumo da ultima
		// chamada); a parcial tambem e paga, entao tambem e medida.
		measured := chatUsageFor(llm, systemPrompt, userPrompt, answer)
		usage = &measured

		persistedSources := make([]models.ChatSourceReference, 0, len(sources))
		for _, source := range sources {
			persistedSources = append(persistedSources, models.ChatSourceReference{
				ID:              source.ID,
				DocName:         source.DocName,
				Page:            source.Page,
				Snippet:         source.Snippet,
				DocumentID:      source.DocumentID,
				NodeID:          source.NodeID,
				ParagraphID:     source.ParagraphID,
				Lines:           source.Lines,
				MatchedTerms:    source.MatchedTerms,
				Box:             source.Box,
				LineBoxes:       source.LineBoxes,
				ContinuesOnPage: source.ContinuesOnPage,
				IndexedAt:       source.IndexedAt,
			})
		}

		if err := u.conversationRepo.AddMessage(ctx, userID, conversation.ID, &models.ChatConversationEntry{
			Role:             "assistant",
			Content:          answer,
			Sources:          persistedSources,
			PromptTokens:     measured.PromptTokens,
			CompletionTokens: measured.CompletionTokens,
			UsageReported:    measured.UsageReported,
			ProviderType:     measured.ProviderType,
			Model:            measured.Model,
			EstimatedCostUSD: measured.EstimatedCostUSD,
		}); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrChatConversationNotFound
			}
			return fmt.Errorf("%w: %v", domain.ErrChatConversationMessageStore, err)
		}

		return nil
	}

	if streamer, ok := llm.(domain.LLMStreamingService); ok {
		if err := streamer.StreamText(ctx, systemPrompt, userPrompt, func(delta string) error {
			if delta == "" {
				return nil
			}
			answerBuilder.WriteString(delta)
			return onEvent(domain.ChatStreamEvent{
				Type:  chatStreamEventDelta,
				Delta: delta,
			})
		}); err != nil {
			// O contexto usado para persistir e o mesmo que acabou de falhar.
			// Se ele ja estiver cancelado (cliente desconectou), a gravacao
			// tambem falharia -- por isso o erro de persistencia so e logado,
			// nunca substitui o erro original do stream, que e o que o handler
			// precisa para escolher a mensagem certa ao usuario.
			if persistErr := persistAnswer(); persistErr != nil {
				log.Printf("chat: resposta parcial da conversa %d nao pode ser persistida: %v", conversation.ID, persistErr)
			}
			return err
		}
	} else {
		response, err := llm.GenerateText(ctx, systemPrompt, userPrompt)
		if err != nil {
			return err
		}
		answerBuilder.WriteString(strings.TrimSpace(response))
		if err := onEvent(domain.ChatStreamEvent{
			Type:  chatStreamEventDelta,
			Delta: strings.TrimSpace(response),
		}); err != nil {
			if persistErr := persistAnswer(); persistErr != nil {
				log.Printf("chat: resposta da conversa %d nao pode ser persistida: %v", conversation.ID, persistErr)
			}
			return err
		}
	}

	if err := persistAnswer(); err != nil {
		return err
	}

	return onEvent(domain.ChatStreamEvent{
		Type:    chatStreamEventDone,
		Stage:   "completed",
		Message: "Answer completed.",
		Usage:   usage,
	})
}

// chatUsageFor mede o consumo de uma resposta: numeros do provedor quando ele
// os informa (TokenUsageReporter), estimativa runas/4 quando nao -- rotulada
// como estimativa. Custo so com preco cadastrado (LLMPricingProvider).
func chatUsageFor(llm domain.LLMService, systemPrompt, userPrompt, answer string) domain.ChatUsage {
	usage := domain.ChatUsage{
		PromptTokens:     estimateChatTokens(systemPrompt) + estimateChatTokens(userPrompt),
		CompletionTokens: estimateChatTokens(answer),
	}
	if reporter, ok := llm.(domain.TokenUsageReporter); ok {
		if prompt, completion, _, reported := reporter.LastTokenUsage(); reported {
			usage.PromptTokens, usage.CompletionTokens, usage.UsageReported = prompt, completion, true
		}
	}
	if metadata, ok := llm.(domain.LLMMetadataProvider); ok {
		usage.ProviderType, usage.Model = metadata.ProviderType(), metadata.ModelName()
	}
	if pricer, ok := llm.(domain.LLMPricingProvider); ok {
		if pricing, ok := pricer.Pricing(); ok {
			cost := float64(usage.PromptTokens)/1_000_000*pricing.InputCostPerMillionUSD +
				float64(usage.CompletionTokens)/1_000_000*pricing.OutputCostPerMillionUSD
			usage.EstimatedCostUSD = &cost
		}
	}
	return usage
}

// estimateChatTokens e a mesma regra da indexacao (processing.estimateTextTokens):
// uma runa a cada quatro. Copiada, nao importada, para o usecase nao depender
// do pacote de processamento.
func estimateChatTokens(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	return (len([]rune(trimmed)) + 3) / 4
}

func buildConversationTitle(title string) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(title)), " ")
	if normalized == "" {
		return fmt.Sprintf("Chat %s", time.Now().UTC().Format("2006-01-02 15:04"))
	}
	// Corte por runes, nao por bytes. `normalized[:255]` parte um code point
	// UTF-8 no meio quando o caractere 255 e acentuado -- caso comum em
	// portugues -- e grava bytes invalidos no titulo. O limite de 255 e o
	// tamanho da coluna, entao contar runes so pode gravar menos, nunca mais.
	if runes := []rune(normalized); len(runes) > 255 {
		return strings.TrimSpace(string(runes[:255]))
	}
	return normalized
}

// Limites do contexto enviado ao LLM. Constantes exportadas de proposito: o
// artigo compara custo por consulta, entao o teto tem de ser um numero
// nomeado e travado por teste, nao um literal escondido.
const (
	// ChatSourceLimit e o total de fontes citadas por resposta.
	ChatSourceLimit = 6
	// ChatNodeSourceLimit e quantas dessas fontes vem de secoes do sumario
	// (node hits), que entram ANTES das paginas por serem a unidade
	// estrutural da tese e terem peso 3 no retrieval.
	ChatNodeSourceLimit = 3
	// ChatTableRowSourceLimit: linhas de tabela que entram antes das secoes,
	// alem das ChatSourceLimit fontes de secao e pagina.
	ChatTableRowSourceLimit = 3
	// ChatPageCharLimit corta o conteudo de cada pagina enviada ao modelo.
	ChatPageCharLimit = 4000
	// ChatContextCharLimit e o teto somado de conteudo de paginas no prompt;
	// estourou, as fontes seguintes voltam ao trecho curto.
	ChatContextCharLimit = 24000
)

func buildChatSources(
	collectionContext *domain.CollectionContext,
	retrieval *domain.CollectionRetrievalResult,
) []domain.ChatSource {
	if collectionContext == nil || retrieval == nil {
		return nil
	}

	documentTitles := make(map[uint]string, len(collectionContext.Documents))
	documentIndexedAt := make(map[uint]*time.Time, len(collectionContext.Documents))
	for _, doc := range collectionContext.Documents {
		documentTitles[doc.ID] = doc.Title
		documentIndexedAt[doc.ID] = doc.IndexedAt
	}

	pageLookup := make(map[string]models.DocumentPage, len(collectionContext.DocumentPages))
	for _, page := range collectionContext.DocumentPages {
		key := fmt.Sprintf("%d:%d", page.DocumentID, page.PageNumber)
		pageLookup[key] = page
	}

	pagesWithTables := make(map[string]struct{}, len(collectionContext.TableSections))
	for _, table := range collectionContext.TableSections {
		if table.PageNumber != nil {
			pagesWithTables[fmt.Sprintf("%d:%d", table.DocumentID, *table.PageNumber)] = struct{}{}
		}
	}

	sources := make([]domain.ChatSource, 0, ChatSourceLimit)
	seen := make(map[string]struct{})

	// O paragrafo da pagina com mais termos da pergunta e o que o canvas
	// grifa ao clicar na fonte. Zero termos em comum: nenhum, e o clique cai
	// na pagina.
	significant := significantQueryTokens(retrieval.Query)
	// Fonte de secao ou de pagina precisa de uma fracao dos termos da
	// pergunta para entrar: sem isso a lista trazia "5.1.5 Acessibilidade"
	// para uma pergunta sobre estagio, por um termo solto. Linha de tabela ja
	// exige 2 termos no retrieval.
	minCoverage := chatSourceMinCoverage(len(significant))
	matched := func(text string) []string {
		tokens := tokenSet(text)
		out := make([]string, 0, len(significant))
		for _, term := range significant {
			if _, ok := tokens[term]; ok {
				out = append(out, term)
			}
		}
		return out
	}
	paragraphsByPage := make(map[string][]models.ParagraphNode)
	for _, paragraph := range collectionContext.ParagraphNodes {
		if paragraph.PageNumber == nil {
			continue
		}
		key := fmt.Sprintf("%d:%d", paragraph.DocumentID, *paragraph.PageNumber)
		paragraphsByPage[key] = append(paragraphsByPage[key], paragraph)
	}
	// Texto da pagina para contar termos: o conteudo da pagina, ou os
	// paragrafos dela quando o contexto veio sem paginas.
	// continuesOn: a pagina seguinte, quando o texto desta termina sem
	// pontuacao final (frase quebrada na virada) e a seguinte existe.
	// Pagina com tabela nao continua: termina sem ponto por ser tabela, e o
	// comeco da seguinte e outra tabela (o calendario de outubro colado ao de
	// setembro fez o modelo errar c12). As linhas ja entram como fonte.
	continuesOn := func(documentID uint, page int) int {
		key := fmt.Sprintf("%d:%d", documentID, page)
		if _, hasTable := pagesWithTables[key]; hasTable || !pageEndsMidSentence(pageLookup[key].Content) {
			return 0
		}
		if _, ok := pageLookup[fmt.Sprintf("%d:%d", documentID, page+1)]; !ok {
			return 0
		}
		return page + 1
	}
	pageText := func(key string) string {
		if content := pageLookup[key].Content; content != "" {
			return content
		}
		parts := make([]string, 0)
		for _, paragraph := range paragraphsByPage[key] {
			parts = append(parts, paragraph.Content)
		}
		return strings.Join(parts, " ")
	}
	// bestParagraph devolve o paragrafo com mais termos da pergunta, as linhas
	// dele que os contem (grifo fino) e o texto dessas linhas (trecho).
	type paragraphPick struct {
		id        uint
		lines     []int
		snippet   string
		box       *models.SourceBox
		lineBoxes [][4]float64
	}
	bestParagraph := func(documentID uint, page int) paragraphPick {
		var best *models.ParagraphNode
		bestCoverage := 0
		for i, paragraph := range paragraphsByPage[fmt.Sprintf("%d:%d", documentID, page)] {
			if coverage := queryTermCoverage(paragraph.Content, significant); coverage > bestCoverage {
				best, bestCoverage = &paragraphsByPage[fmt.Sprintf("%d:%d", documentID, page)][i], coverage
			}
		}
		if best == nil {
			return paragraphPick{}
		}
		lines, snippet := matchingLines(best.Metadata, significant)
		return paragraphPick{id: best.ID, lines: lines, snippet: snippet, box: sourceBoxFromMetadata(best.Metadata), lineBoxes: lineBoxesFromMetadata(best.Metadata, lines)}
	}

	tableBoxes := make(map[uint]*models.SourceBox, len(collectionContext.TableSections))
	for _, table := range collectionContext.TableSections {
		tableBoxes[table.ID] = sourceBoxFromMetadata(table.Metadata)
	}

	// Linhas de tabela primeiro: e a fonte mais precisa (documento, pagina,
	// tabela, linha) e a mais barata no prompt. A pagina inteira da tabela
	// ainda pode entrar depois pelos hits de pagina.
	// As linhas nao consomem as vagas das paginas: custam ~100 caracteres
	// contra 4.000 de uma pagina, e uma pergunta de prosa perdia a pagina
	// certa para uma linha fraca. Linha fraca (abaixo de metade da melhor)
	// tambem fica de fora.
	rowSources := 0
	for _, hit := range retrieval.TableRowHits {
		if rowSources == ChatTableRowSourceLimit || hit.Score*2 < retrieval.TableRowHits[0].Score {
			break
		}
		id := fmt.Sprintf("%d:%d:table-%d-row-%d", hit.DocumentID, hit.PageNumber, hit.TableID, hit.RowIndex)
		if _, exists := seen[id]; exists {
			continue
		}
		body := hit.Text
		if title := strings.TrimSpace(hit.Title); title != "" {
			body = title + " | " + body
		}
		sources = append(sources, domain.ChatSource{
			ID:           id,
			DocName:      documentTitles[hit.DocumentID],
			Page:         hit.PageNumber,
			Snippet:      clampRetrievalSnippet(body, 220),
			DocumentID:   hit.DocumentID,
			Body:         body,
			MatchedTerms: matched(body),
			Box:          tableBoxes[hit.TableID],
			IndexedAt:    documentIndexedAt[hit.DocumentID],
		})
		seen[id] = struct{}{}
		rowSources++
	}

	// Secoes depois: a pagina inicial da secao e o menor pedaco com a
	// proveniencia inteira (documento, secao, pagina).
	for _, hit := range retrieval.NodeHits {
		if len(sources)-rowSources == ChatNodeSourceLimit {
			break
		}
		if hit.StartPage == nil || *hit.StartPage <= 0 {
			continue
		}
		pageNumber := *hit.StartPage
		key := fmt.Sprintf("%d:%d", hit.DocumentID, pageNumber)
		if _, exists := seen[key]; exists {
			continue
		}

		snippet := strings.TrimSpace(hit.Summary)
		if snippet == "" {
			snippet = hit.Title
		}
		pick := bestParagraph(hit.DocumentID, pageNumber)
		if pick.snippet != "" {
			snippet = pick.snippet
		}
		terms := matched(hit.Title + " " + hit.Summary + " " + pageText(key))
		if len(terms) < minCoverage {
			continue
		}

		sources = append(sources, domain.ChatSource{
			ID:              fmt.Sprintf("%s:%s", key, hit.NodeID),
			DocName:         documentTitles[hit.DocumentID],
			Page:            pageNumber,
			Snippet:         clampRetrievalSnippet(snippet, 220),
			DocumentID:      hit.DocumentID,
			NodeID:          hit.NodeID,
			ParagraphID:     pick.id,
			Lines:           pick.lines,
			MatchedTerms:    terms,
			ContinuesOnPage: continuesOn(hit.DocumentID, pageNumber),
			Box:             pick.box,
			LineBoxes:       pick.lineBoxes,
			IndexedAt:       documentIndexedAt[hit.DocumentID],
		})
		seen[key] = struct{}{}
	}

	for _, hit := range retrieval.PageHits {
		if len(sources)-rowSources == ChatSourceLimit {
			break
		}
		key := fmt.Sprintf("%d:%d", hit.DocumentID, hit.PageNumber)
		if _, exists := seen[key]; exists {
			continue
		}

		page := pageLookup[key]
		snippet := strings.TrimSpace(hit.Snippet)
		if snippet == "" {
			snippet = page.Summary
		}
		if snippet == "" {
			snippet = page.Content
		}

		pick := bestParagraph(hit.DocumentID, hit.PageNumber)
		if pick.snippet != "" {
			snippet = pick.snippet
		}
		terms := matched(pageText(key))
		if len(terms) < minCoverage {
			continue
		}
		sources = append(sources, domain.ChatSource{
			ID:              key,
			DocName:         documentTitles[hit.DocumentID],
			Page:            hit.PageNumber,
			Snippet:         clampRetrievalSnippet(snippet, 220),
			DocumentID:      hit.DocumentID,
			ParagraphID:     pick.id,
			Lines:           pick.lines,
			MatchedTerms:    terms,
			ContinuesOnPage: continuesOn(hit.DocumentID, hit.PageNumber),
			Box:             pick.box,
			LineBoxes:       pick.lineBoxes,
			IndexedAt:       documentIndexedAt[hit.DocumentID],
		})
		seen[key] = struct{}{}
	}

	return orderChatSources(sources, rowSources, len(significant))
}

// orderChatSources decide a ordem final. Linha de tabela so fica na frente
// quando responde a pelo menos metade dos termos da pergunta (pergunta de
// tabela: "pre-requisito de Estrutura de Dados I" casa quase tudo na linha);
// as demais linhas entram na ordenacao comum, junto com secoes e paginas,
// por numero de termos casados (estavel: empate mantem linha, secao, pagina).
// Antes a ordem era fixa "linhas, secoes, paginas", e a pagina da Lei com 8
// termos ficava em 4o atras de tres linhas de calendario com 2 termos
// genericos. O conjunto de fontes nao muda, so a ordem no card e no prompt.
// (Multiplicar o BM25 pela cobertura foi medido e perdeu uma pagina-alvo,
// t18; nao entrou.)
func orderChatSources(sources []domain.ChatSource, rowSources int, significantTerms int) []domain.ChatSource {
	threshold := (significantTerms + 1) / 2
	strong := make([]domain.ChatSource, 0, rowSources)
	rest := make([]domain.ChatSource, 0, len(sources))
	for i, source := range sources {
		if i < rowSources && len(source.MatchedTerms) >= threshold {
			strong = append(strong, source)
			continue
		}
		rest = append(rest, source)
	}
	sort.SliceStable(rest, func(i, j int) bool {
		return len(rest[i].MatchedTerms) > len(rest[j].MatchedTerms)
	})
	return append(strong, rest...)
}

// sourceBoxFromMetadata le metadata.box (JSONB: x_min...) de um paragrafo ou
// tabela. Nulo quando nao ha caixa (OCR sem posicao, HTML).
func sourceBoxFromMetadata(metadata map[string]any) *models.SourceBox {
	raw, ok := metadata["box"].(map[string]any)
	if !ok {
		return nil
	}
	num := func(key string) (float64, bool) {
		v, ok := raw[key].(float64)
		return v, ok
	}
	xMin, ok1 := num("x_min")
	yMin, ok2 := num("y_min")
	xMax, ok3 := num("x_max")
	yMax, ok4 := num("y_max")
	pw, ok5 := num("page_width")
	ph, ok6 := num("page_height")
	if !(ok1 && ok2 && ok3 && ok4 && ok5 && ok6) || pw <= 0 || ph <= 0 {
		return nil
	}
	return &models.SourceBox{XMin: xMin, YMin: yMin, XMax: xMax, YMax: yMax, PageWidth: pw, PageHeight: ph}
}

// lineBoxesFromMetadata devolve as caixas [xMin, yMin, xMax, yMax] das linhas
// escolhidas, na ordem de `lines`.
func lineBoxesFromMetadata(metadata map[string]any, lines []int) [][4]float64 {
	raw, ok := metadata["lines"].([]any)
	if !ok || len(lines) == 0 {
		return nil
	}
	out := make([][4]float64, 0, len(lines))
	for _, i := range lines {
		if i < 0 || i >= len(raw) {
			continue
		}
		line, _ := raw[i].(map[string]any)
		b, _ := line["b"].([]any)
		if len(b) != 4 {
			continue
		}
		var box [4]float64
		okAll := true
		for k := range b {
			v, ok := b[k].(float64)
			if !ok {
				okAll = false
				break
			}
			box[k] = v
		}
		if okAll {
			out = append(out, box)
		}
	}
	return out
}

// chatSourceMinCoverage e quantos termos significativos da pergunta uma
// fonte de secao ou pagina precisa conter: 2 (pergunta de um termo exige
// esse um). Foi 40% dos termos e perdeu 3 paginas-alvo no gabarito (t18, t20,
// p21): pergunta longa tem muito termo acessorio, e a pagina certa casa so
// os dois que importam.
func chatSourceMinCoverage(significant int) int {
	if significant <= 1 {
		return significant
	}
	return 2
}

// matchingLines le metadata.lines do paragrafo ([{t, b}]) e devolve os indices
// das linhas que contem termos da pergunta -- a janela contigua em torno da
// linha com mais termos -- e o texto dessas linhas. Sem linhas (paragrafo de
// uma linha, ou texto sem caixa), devolve nada e o grifo e o paragrafo.
func matchingLines(metadata map[string]any, terms []string) ([]int, string) {
	raw, ok := metadata["lines"].([]any)
	if !ok || len(raw) < 2 {
		return nil, ""
	}
	coverage := make([]int, len(raw))
	texts := make([]string, len(raw))
	best, bestCoverage := -1, 0
	for i, item := range raw {
		line, _ := item.(map[string]any)
		text, _ := line["t"].(string)
		texts[i] = text
		coverage[i] = queryTermCoverage(text, terms)
		if coverage[i] > bestCoverage {
			best, bestCoverage = i, coverage[i]
		}
	}
	if best < 0 {
		return nil, ""
	}
	start, end := best, best
	for start > 0 && coverage[start-1] > 0 {
		start--
	}
	for end < len(raw)-1 && coverage[end+1] > 0 {
		end++
	}
	lines := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		lines = append(lines, i)
	}
	return lines, strings.Join(texts[start:end+1], " ")
}

func buildCollectionChatPrompt(
	collectionContext *domain.CollectionContext,
	retrieval *domain.CollectionRetrievalResult,
	request domain.ChatRequest,
	sources []domain.ChatSource,
) (string, string) {
	language := normalizeChatLanguage(request.Language)
	// A instrucao de idioma vai por ultimo no system prompt e repetida no fim
	// do user prompt, na propria lingua: com o prompt todo em ingles e
	// evidencia longa, o modelo respondia em ingles a pergunta em portugues.
	systemPrompt := strings.Join([]string{
		"You are ThingIndex Assistant, a grounded document-analysis assistant.",
		"Answer only from the provided relational document context.",
		"If the evidence is insufficient, say that clearly and do not invent facts.",
		// "Start with the direct answer in one sentence" foi testado em
		// 2026-09-09 e descartado: o modelo afirmava antes de ler ("coluna de
		// pre-requisito vazia" com o SIS07201 na fonte) e a resposta ficou
		// mais longa (311 -> 358 caracteres), nao mais curta.
		"Prefer precise, direct answers and cite the supporting document names and pages in prose when relevant.",
		"Write the answer in " + language + ". " + languageReminder(language),
	}, "\n")

	var userPrompt strings.Builder
	userPrompt.WriteString("Collection:\n")
	userPrompt.WriteString("- Name: ")
	userPrompt.WriteString(strings.TrimSpace(collectionContext.Collection.Name))
	userPrompt.WriteString("\n")

	if len(request.History) > 0 {
		userPrompt.WriteString("\nRecent conversation:\n")
		for _, message := range limitChatHistory(request.History, 6) {
			userPrompt.WriteString("- ")
			userPrompt.WriteString(strings.ToUpper(message.Role))
			userPrompt.WriteString(": ")
			userPrompt.WriteString(clampRetrievalSnippet(message.Content, 500))
			userPrompt.WriteString("\n")
		}
	}

	if len(sources) > 0 {
		// O conteudo da pagina citada, nao so um trecho de 220 caracteres:
		// com o trecho o modelo dizia "nao ha informacoes" com a resposta na
		// pagina. Teto por pagina e teto total, ambos constantes testadas.
		pageContent := make(map[string]string, len(collectionContext.DocumentPages))
		for _, page := range collectionContext.DocumentPages {
			pageContent[fmt.Sprintf("%d:%d", page.DocumentID, page.PageNumber)] = page.Content
		}
		remaining := ChatContextCharLimit
		userPrompt.WriteString("\nPrimary grounded sources (table rows and full page content):\n")
		for index, source := range sources {
			userPrompt.WriteString(fmt.Sprintf("%d. %s", index+1, source.DocName))
			if source.Page > 0 {
				userPrompt.WriteString(fmt.Sprintf(" (page %d)", source.Page))
			}
			userPrompt.WriteString(":\n")
			body, used := chatSourceBody(source, pageContent, remaining)
			remaining -= used
			userPrompt.WriteString(body)
			userPrompt.WriteString("\n\n")
		}
	}

	documentSummaries := buildDocumentSummaryMap(collectionContext.MetadataPairs)
	if len(collectionContext.Documents) > 0 {
		userPrompt.WriteString("\nDocument inventory:\n")
		for index, doc := range collectionContext.Documents {
			if index == 6 {
				break
			}
			userPrompt.WriteString("- ")
			userPrompt.WriteString(doc.Title)
			if summary := strings.TrimSpace(documentSummaries[doc.ID]); summary != "" {
				userPrompt.WriteString(": ")
				userPrompt.WriteString(clampRetrievalSnippet(summary, 220))
			}
			userPrompt.WriteString("\n")
		}
	}

	userPrompt.WriteString("\nUser question:\n")
	userPrompt.WriteString(strings.TrimSpace(request.Prompt))
	userPrompt.WriteString("\n\n")
	userPrompt.WriteString(languageReminder(language))

	return systemPrompt, userPrompt.String()
}

// chatSourceBody devolve o conteudo da pagina da fonte (cortado por runas em
// ChatPageCharLimit) e quantas runas consumiu do teto total. Sem espaco no
// teto, ou sem a pagina no contexto, volta ao trecho curto e consome zero.
func chatSourceBody(source domain.ChatSource, pageContent map[string]string, remaining int) (string, int) {
	content := strings.Join(strings.Fields(pageContent[fmt.Sprintf("%d:%d", source.DocumentID, source.Page)]), " ")
	if source.Body != "" {
		// Fonte menor que a pagina (linha de tabela): vai ela, nao a pagina.
		content = strings.Join(strings.Fields(source.Body), " ")
	} else if source.ContinuesOnPage > 0 && content != "" {
		// A frase quebrou na virada: o comeco da pagina seguinte vai junto,
		// marcado, para o modelo ler o paragrafo inteiro.
		if next := pageContinuation(pageContent[fmt.Sprintf("%d:%d", source.DocumentID, source.ContinuesOnPage)]); next != "" {
			content += fmt.Sprintf(" [continua na p. %d] %s", source.ContinuesOnPage, next)
		}
	}
	if content == "" {
		return source.Snippet, 0
	}
	runes := []rune(content)
	if len(runes) > ChatPageCharLimit {
		runes = runes[:ChatPageCharLimit]
		content = strings.TrimSpace(string(runes)) + "…"
	}
	if len(runes) > remaining {
		return source.Snippet, 0
	}
	return content, len(runes)
}

// ChatContinuationCharLimit e quanto da pagina seguinte acompanha uma fonte
// cuja frase quebrou na virada.
const ChatContinuationCharLimit = 600

// pageEndsMidSentence: o texto termina sem ponto final. ";" e ":" contam
// como continuacao, nao como fim: em texto legal e academico sao item de
// lista ("II - conceito B ...;") ou abertura de lista, e o resto esta na
// pagina seguinte ("IV - conceito D" na p. 126 do PPC).
func pageEndsMidSentence(content string) bool {
	trimmed := strings.TrimRight(strings.TrimSpace(content), "\"'»)]”’")
	if trimmed == "" {
		return false
	}
	last := trimmed[len(trimmed)-1]
	return !strings.ContainsRune(".!?", rune(last))
}

// pageContinuation devolve o comeco da pagina seguinte ate o fim da ultima
// frase que couber em ChatContinuationCharLimit.
func pageContinuation(content string) string {
	runes := []rune(strings.Join(strings.Fields(content), " "))
	if len(runes) == 0 {
		return ""
	}
	if len(runes) <= ChatContinuationCharLimit {
		return string(runes)
	}
	cut := ChatContinuationCharLimit
	for i := cut - 1; i > cut/2; i-- {
		if strings.ContainsRune(".;:!?", runes[i]) {
			cut = i + 1
			break
		}
	}
	return strings.TrimSpace(string(runes[:cut]))
}

// languageReminder e a instrucao de idioma na propria lingua.
func languageReminder(language string) string {
	switch language {
	case "English":
		return "Answer in English."
	case "Spanish":
		return "Responde en español."
	default:
		return "Responda em português do Brasil."
	}
}

func buildDocumentSummaryMap(pairs []models.MetadataPair) map[uint]string {
	result := make(map[uint]string)
	for _, pair := range pairs {
		if pair.Key != "document_summary" {
			continue
		}
		result[pair.DocumentID] = pair.Value
	}
	return result
}

func limitChatHistory(history []domain.ChatMessage, limit int) []domain.ChatMessage {
	if len(history) <= limit {
		return history
	}
	return history[len(history)-limit:]
}

func normalizeChatLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "en", "en-us":
		return "English"
	case "es", "es-es":
		return "Spanish"
	default:
		return "Brazilian Portuguese"
	}
}
