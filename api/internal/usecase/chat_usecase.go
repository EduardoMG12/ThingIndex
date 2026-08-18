package usecase

import (
	"context"
	"errors"
	"fmt"
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
			return err
		}
	}

	answer := strings.TrimSpace(answerBuilder.String())
	if answer != "" {
		persistedSources := make([]models.ChatSourceReference, 0, len(sources))
		for _, source := range sources {
			persistedSources = append(persistedSources, models.ChatSourceReference{
				ID:      source.ID,
				DocName: source.DocName,
				Page:    source.Page,
				Snippet: source.Snippet,
			})
		}

		if err := u.conversationRepo.AddMessage(ctx, userID, conversation.ID, &models.ChatConversationEntry{
			Role:    "assistant",
			Content: answer,
			Sources: persistedSources,
		}); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrChatConversationNotFound
			}
			return fmt.Errorf("%w: %v", domain.ErrChatConversationMessageStore, err)
		}
	}

	return onEvent(domain.ChatStreamEvent{
		Type:    chatStreamEventDone,
		Stage:   "completed",
		Message: "Answer completed.",
	})
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

func buildChatSources(
	collectionContext *domain.CollectionContext,
	retrieval *domain.CollectionRetrievalResult,
) []domain.ChatSource {
	if collectionContext == nil || retrieval == nil {
		return nil
	}

	documentTitles := make(map[uint]string, len(collectionContext.Documents))
	for _, doc := range collectionContext.Documents {
		documentTitles[doc.ID] = doc.Title
	}

	pageLookup := make(map[string]models.DocumentPage, len(collectionContext.DocumentPages))
	for _, page := range collectionContext.DocumentPages {
		key := fmt.Sprintf("%d:%d", page.DocumentID, page.PageNumber)
		pageLookup[key] = page
	}

	sources := make([]domain.ChatSource, 0, min(6, len(retrieval.PageHits)))
	seen := make(map[string]struct{})

	for _, hit := range retrieval.PageHits {
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

		sources = append(sources, domain.ChatSource{
			ID:      key,
			DocName: documentTitles[hit.DocumentID],
			Page:    hit.PageNumber,
			Snippet: clampRetrievalSnippet(snippet, 220),
		})
		seen[key] = struct{}{}

		if len(sources) == 6 {
			return sources
		}
	}

	for _, hit := range retrieval.NodeHits {
		pageNumber := 0
		if hit.StartPage != nil {
			pageNumber = *hit.StartPage
		}
		key := fmt.Sprintf("%d:%d:%s", hit.DocumentID, pageNumber, hit.NodeID)
		if _, exists := seen[key]; exists {
			continue
		}

		snippet := strings.TrimSpace(hit.Summary)
		if snippet == "" {
			snippet = hit.Title
		}

		sources = append(sources, domain.ChatSource{
			ID:      key,
			DocName: documentTitles[hit.DocumentID],
			Page:    pageNumber,
			Snippet: clampRetrievalSnippet(snippet, 220),
		})
		seen[key] = struct{}{}

		if len(sources) == 6 {
			break
		}
	}

	return sources
}

func buildCollectionChatPrompt(
	collectionContext *domain.CollectionContext,
	retrieval *domain.CollectionRetrievalResult,
	request domain.ChatRequest,
	sources []domain.ChatSource,
) (string, string) {
	language := normalizeChatLanguage(request.Language)
	systemPrompt := strings.Join([]string{
		"You are ThingIndex Assistant, a grounded document-analysis assistant.",
		"Answer only from the provided relational document context.",
		"If the evidence is insufficient, say that clearly and do not invent facts.",
		"Prefer precise, direct answers and cite the supporting document names and pages in prose when relevant.",
		"Write the answer in " + language + ".",
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
		userPrompt.WriteString("\nPrimary grounded sources:\n")
		for index, source := range sources {
			userPrompt.WriteString(fmt.Sprintf("%d. %s", index+1, source.DocName))
			if source.Page > 0 {
				userPrompt.WriteString(fmt.Sprintf(" (page %d)", source.Page))
			}
			userPrompt.WriteString(": ")
			userPrompt.WriteString(source.Snippet)
			userPrompt.WriteString("\n")
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

	if retrieval != nil && len(retrieval.NodeHits) > 0 {
		userPrompt.WriteString("\nRelevant outline hits:\n")
		for index, hit := range retrieval.NodeHits {
			if index == 5 {
				break
			}
			userPrompt.WriteString("- ")
			userPrompt.WriteString(hit.Title)
			if strings.TrimSpace(hit.Summary) != "" {
				userPrompt.WriteString(": ")
				userPrompt.WriteString(clampRetrievalSnippet(hit.Summary, 180))
			}
			userPrompt.WriteString("\n")
		}
	}

	userPrompt.WriteString("\nUser question:\n")
	userPrompt.WriteString(strings.TrimSpace(request.Prompt))

	return systemPrompt, userPrompt.String()
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
