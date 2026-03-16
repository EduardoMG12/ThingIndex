package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	llmResolver           domain.UserLLMResolver
}

func NewChatUseCase(
	collectionContextRepo domain.CollectionContextRepository,
	llmResolver domain.UserLLMResolver,
) domain.ChatUseCase {
	return &chatUseCase{
		collectionContextRepo: collectionContextRepo,
		llmResolver:           llmResolver,
	}
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

	retrieval := buildCollectionRetrieval(prompt, collectionContext.Documents, collectionContext.DocumentPages, collectionContext.OutlineNodes)
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

	systemPrompt, userPrompt := buildCollectionChatPrompt(collectionContext, retrieval, request, sources)
	if err := onEvent(domain.ChatStreamEvent{
		Type:    chatStreamEventStage,
		Stage:   "generating_answer",
		Message: "Generating grounded answer.",
	}); err != nil {
		return err
	}

	if streamer, ok := llm.(domain.LLMStreamingService); ok {
		if err := streamer.StreamText(ctx, systemPrompt, userPrompt, func(delta string) error {
			if delta == "" {
				return nil
			}
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
		if err := onEvent(domain.ChatStreamEvent{
			Type:  chatStreamEventDelta,
			Delta: strings.TrimSpace(response),
		}); err != nil {
			return err
		}
	}

	return onEvent(domain.ChatStreamEvent{
		Type:    chatStreamEventDone,
		Stage:   "completed",
		Message: "Answer completed.",
	})
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
		if _, exists := result[pair.DocumentID]; exists {
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
