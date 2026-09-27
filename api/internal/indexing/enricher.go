package indexing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

const llmOutlineLimit = 12

type LLMEnricher struct {
	// cache (opcional): resumos por hash do texto, para reindexar sem pagar.
	cache domain.EnrichmentCache
}

// WithCache liga o cache de resumos; sem ele, cada reindexacao chama a LLM.
func (e *LLMEnricher) WithCache(cache domain.EnrichmentCache) *LLMEnricher {
	e.cache = cache
	return e
}

// enrichmentPromptVersion entra no hash do cache: mudar o prompt invalida
// os resumos guardados.
const enrichmentPromptVersion = "2026-09-10a"

func enrichmentHash(text string) string {
	sum := sha256.Sum256([]byte(enrichmentPromptVersion + "\n" + strings.TrimSpace(text)))
	return hex.EncodeToString(sum[:])
}

type documentSynopsisResponse struct {
	DocumentSummary string   `json:"document_summary"`
	Keywords        []string `json:"keywords"`
	DocumentPurpose string   `json:"document_purpose"`
}

type sectionSummaryResponse struct {
	Nodes []struct {
		SequenceOrder int    `json:"sequence_order"`
		Summary       string `json:"summary"`
	} `json:"nodes"`
}

type pageSummaryItem struct {
	PageNumber int      `json:"page_number"`
	Summary    string   `json:"summary"`
	Topics     []string `json:"topics"`
}

type pageSummaryResponse struct {
	Pages []pageSummaryItem `json:"pages"`
}

// pageQuestionsResponse e a saida do gerador de perguntas de leigo. Formato
// separado do resumo de proposito: os dois coexistem para que a medicao possa
// comparar um contra o outro.
type pageQuestionsResponse struct {
	Pages []pageQuestionsItem `json:"pages"`
}

type pageQuestionsItem struct {
	PageNumber int      `json:"page_number"`
	Questions  []string `json:"questions"`
}

type retrievalSignalsResponse struct {
	Topics   []string `json:"topics"`
	Entities []string `json:"entities"`
	Dates    []string `json:"dates"`
}

func NewLLMEnricher() *LLMEnricher {
	return &LLMEnricher{}
}

func (e *LLMEnricher) Enrich(ctx context.Context, llm domain.LLMService, doc *models.Document, result domain.IndexingResult) (domain.IndexingResult, error) {
	return e.EnrichWith(ctx, llm, doc, result, domain.DefaultIndexingPreferences())
}

// EnrichWith respeita o escopo escolhido pelo usuario: documento (arvore,
// sinopse, sinais), paginas (resumo e topicos) e secoes (resumo). O que esta
// desligado cai no fallback deterministico, sem chamada.
func (e *LLMEnricher) EnrichWith(ctx context.Context, llm domain.LLMService, doc *models.Document, result domain.IndexingResult, prefs domain.IndexingPreferences) (domain.IndexingResult, error) {
	if llm == nil {
		return result, nil
	}

	updated := result
	documentID := documentIDFromResult(result)
	scope := make([]string, 0, 3)
	if prefs.DocumentSynopsis {
		scope = append(scope, "document")
	}
	if prefs.PageSummaries {
		scope = append(scope, "pages")
	}
	if prefs.SectionSummaries {
		scope = append(scope, "sections")
	}
	appendMetadata(&updated, documentID, "enrichment_scope", strings.Join(scope, ","))

	treeBuilt, treeErr := domain.IndexingResult{}, errors.New("document scope disabled")
	if prefs.DocumentSynopsis {
		treeBuilt, treeErr = e.buildTree(ctx, llm, doc, updated)
	}
	if treeErr == nil {
		updated = treeBuilt
	} else {
		if len(updated.DocumentPages) > 0 {
			verifiedNodes, stats := verifyAndScoreTree(updated, updated.OutlineNodes, "deterministic")
			updated.OutlineNodes = verifiedNodes
			if shouldFallback, reason := shouldUseConservativeTreeFallback(stats); shouldFallback {
				updated, stats = applyConservativeTreeFallback(updated, reason)
			}
			appendTreeVerificationMetadata(&updated, documentID, stats)
		} else {
			appendMetadata(&updated, documentID, "tree_builder", "deterministic")
			appendMetadata(&updated, documentID, "tree_verification_passed", "false")
		}
	}

	contextBlock := buildPromptContext(doc, updated)

	// Fallback nunca e silencioso: sem o log, cinco chamadas devolvendo um
	// token cada passaram meses como "enrichment_status=applied".
	calls := 0
	var synopsis documentSynopsisResponse
	if !prefs.DocumentSynopsis {
		synopsis = buildFallbackDocumentSynopsis(doc, updated)
		appendMetadata(&updated, documentID, "synopsis_source", "disabled")
	} else if got, err := e.generateDocumentSynopsis(ctx, llm, contextBlock); err != nil {
		calls++
		log.Printf("enricher: sinopse do documento %d caiu no fallback: %v", documentID, err)
		synopsis = buildFallbackDocumentSynopsis(doc, updated)
		appendMetadata(&updated, documentID, "synopsis_source", "fallback")
	} else {
		calls++
		synopsis = got
		appendMetadata(&updated, documentID, "synopsis_source", "llm")
	}
	applyDocumentSynopsis(&updated, documentID, synopsis)
	// Resumos para TODAS as secoes e TODAS as paginas com texto, em lotes:
	// antes eram 12 secoes e 8 paginas por documento, e o PPC de 209 paginas
	// ficava com 200 paginas em fallback (os 320 primeiros caracteres).
	// O texto proprio de cada secao (paragrafos dela, sem os filhos) e o que
	// vai ao prompt; com o texto da pagina inteira a LLM pulava metade dos nos.
	populateOutlineRetrievalText(&updated)
	hits := enrichmentCacheHits{}
	var sections sectionSummaryResponse
	if prefs.SectionSummaries {
		sections = e.generateAllSectionSummaries(ctx, llm, doc, updated, &calls, &hits)
		applyOutlineNodeSummaries(&updated, sections)
	}
	var pages pageSummaryResponse
	if prefs.PageSummaries {
		pages = e.generateAllPageSummaries(ctx, llm, doc, updated, &calls, &hits)
		applyPageSummaries(&updated, pages)
		// As perguntas de leigo saem da MESMA preferencia do resumo, para que o
		// experimento compare os dois campos gerados sobre o mesmo conjunto de
		// paginas. Sem cache: o cache e chaveado por (escopo, tipo, hash do
		// texto) e guarda resumo, entao reaproveita-lo aqui devolveria resumo
		// no lugar de pergunta.
		applyPageQuestions(updated, e.generateAllPageQuestions(ctx, llm, doc, updated, &calls))
	}
	fillMissingPageSummaries(&updated)
	populateOutlineRetrievalText(&updated)

	if prefs.DocumentSynopsis {
		calls++
		retrievalSignals, err := e.generateRetrievalSignals(ctx, llm, contextBlock)
		if err == nil {
			applyRetrievalSignals(&updated, documentID, retrievalSignals)
		} else {
			log.Printf("enricher: sinais de recuperacao do documento %d descartados: %v", documentID, err)
		}
	}
	// enrichment_llm_calls conta sinopse, sinais e lotes (a arvore tem o seu
	// proprio contador em tree_*); llm_request_count, do observador, conta tudo.
	appendMetadata(&updated, documentID, "enrichment_llm_calls", strconv.Itoa(calls))
	appendMetadata(&updated, documentID, "enrichment_sections_summarized", strconv.Itoa(len(sections.Nodes)))
	appendMetadata(&updated, documentID, "enrichment_pages_summarized", strconv.Itoa(len(pages.Pages)))
	// Os acertos de cache so contam entradas do proprio dono do documento
	// (escopo por user_id), entao nao revelam o que outro usuario indexou.
	appendMetadata(&updated, documentID, "enrichment_cache_hits_pages", strconv.Itoa(hits.pages))
	appendMetadata(&updated, documentID, "enrichment_cache_hits_sections", strconv.Itoa(hits.sections))

	return updated, nil
}

func (e *LLMEnricher) generateDocumentSynopsis(ctx context.Context, llm domain.LLMService, contextBlock string) (documentSynopsisResponse, error) {
	systemPrompt, userPrompt := buildDocumentSynopsisPrompt(contextBlock)
	raw, err := llm.GenerateText(ctx, systemPrompt, userPrompt)
	if err != nil {
		return documentSynopsisResponse{}, err
	}

	var payload documentSynopsisResponse
	if err := parseLLMJSON(raw, &payload); err != nil {
		return documentSynopsisResponse{}, fmt.Errorf("parse synopsis enrichment json: %w", err)
	}

	payload.DocumentSummary = clampText(payload.DocumentSummary, 1400)
	payload.DocumentPurpose = clampText(payload.DocumentPurpose, 400)
	payload.Keywords = sanitizeStringList(payload.Keywords, 8, 80)

	return payload, nil
}

func (e *LLMEnricher) generateRetrievalSignals(ctx context.Context, llm domain.LLMService, contextBlock string) (retrievalSignalsResponse, error) {
	systemPrompt, userPrompt := buildRetrievalSignalsPrompt(contextBlock)
	raw, err := llm.GenerateText(ctx, systemPrompt, userPrompt)
	if err != nil {
		return retrievalSignalsResponse{}, err
	}

	var payload retrievalSignalsResponse
	if err := parseLLMJSON(raw, &payload); err != nil {
		return retrievalSignalsResponse{}, fmt.Errorf("parse retrieval signal json: %w", err)
	}

	payload.Topics = sanitizeStringList(payload.Topics, 8, 80)
	payload.Entities = sanitizeStringList(payload.Entities, 12, 80)
	payload.Dates = sanitizeStringList(payload.Dates, 10, 40)

	return payload, nil
}

const (
	enrichmentPageBatch     = 6
	enrichmentSectionBatch  = 12
	enrichmentMinPageChars  = 200
	enrichmentPageChars     = 3000
	enrichmentSectionChars  = 900
	enrichmentMaxBatchCalls = 80
)

// generateAllPageSummaries resume todas as paginas com texto, em lotes de
// enrichmentPageBatch por chamada. Lote que falha e logado e pulado; as
// paginas dele caem no fallback de fillMissingPageSummaries.
type enrichmentCacheHits struct {
	pages    int
	sections int
}

// enrichmentCacheScope e o escopo do cache para este documento: o dono do
// documento e o LLM que ele configurou. Ver domain.EnrichmentCacheScope.
func enrichmentCacheScope(llm domain.LLMService, doc *models.Document) domain.EnrichmentCacheScope {
	var userID uint
	if doc != nil {
		userID = doc.UserID
	}
	return domain.LLMCacheScope(userID, llm)
}

// lookupCache consulta o cache por (escopo, tipo, hash); sem cache ou com
// erro, devolve mapa vazio e segue para a LLM.
func (e *LLMEnricher) lookupCache(ctx context.Context, scope domain.EnrichmentCacheScope, kind string, hashes []string) map[string]domain.EnrichmentCacheEntry {
	if e.cache == nil || len(hashes) == 0 {
		return map[string]domain.EnrichmentCacheEntry{}
	}
	found, err := e.cache.LookupEnrichment(ctx, scope, kind, hashes)
	if err != nil {
		log.Printf("enricher: cache de %s indisponivel, seguindo sem ele: %v", kind, err)
		return map[string]domain.EnrichmentCacheEntry{}
	}
	return found
}

func (e *LLMEnricher) storeCache(ctx context.Context, scope domain.EnrichmentCacheScope, entries []domain.EnrichmentCacheEntry) {
	if e.cache == nil || len(entries) == 0 {
		return
	}
	if err := e.cache.StoreEnrichment(ctx, scope, entries); err != nil {
		log.Printf("enricher: falha ao gravar cache de resumos: %v", err)
	}
}

func (e *LLMEnricher) generateAllPageSummaries(ctx context.Context, llm domain.LLMService, doc *models.Document, result domain.IndexingResult, calls *int, hits *enrichmentCacheHits) pageSummaryResponse {
	var all pageSummaryResponse
	scope := enrichmentCacheScope(llm, doc)
	hashByPage := make(map[int]string, len(result.DocumentPages))
	hashes := make([]string, 0, len(result.DocumentPages))
	for _, page := range result.DocumentPages {
		if len(strings.TrimSpace(page.Content)) < enrichmentMinPageChars {
			continue
		}
		h := enrichmentHash(page.Content)
		hashByPage[page.PageNumber] = h
		hashes = append(hashes, h)
	}
	cached := e.lookupCache(ctx, scope, "page", hashes)
	var batch []models.DocumentPage
	flush := func() {
		if len(batch) == 0 || *calls >= enrichmentMaxBatchCalls {
			batch = nil
			return
		}
		*calls++
		systemPrompt, userPrompt := buildPageSummaryPrompt(buildPageBatchContext(doc, batch))
		raw, err := llm.GenerateText(ctx, systemPrompt, userPrompt)
		var payload pageSummaryResponse
		if err == nil {
			err = parseLLMJSON(raw, &payload)
		}
		if err != nil {
			log.Printf("enricher: lote de paginas %d-%d do documento %d descartado: %v", batch[0].PageNumber, batch[len(batch)-1].PageNumber, doc.ID, err)
		} else {
			items := sanitizePageSummaries(payload.Pages, len(batch))
			all.Pages = append(all.Pages, items...)
			entries := make([]domain.EnrichmentCacheEntry, 0, len(items))
			for _, item := range items {
				if h := hashByPage[item.PageNumber]; h != "" {
					entries = append(entries, domain.EnrichmentCacheEntry{Kind: "page", ContentHash: h, Summary: item.Summary, Topics: strings.Join(item.Topics, ", ")})
				}
			}
			e.storeCache(ctx, scope, entries)
		}
		batch = nil
	}
	for _, page := range result.DocumentPages {
		h, eligible := hashByPage[page.PageNumber]
		if !eligible {
			continue
		}
		if entry, ok := cached[h]; ok {
			all.Pages = append(all.Pages, pageSummaryItem{PageNumber: page.PageNumber, Summary: entry.Summary, Topics: splitTopics(entry.Topics)})
			hits.pages++
			continue
		}
		batch = append(batch, page)
		if len(batch) == enrichmentPageBatch {
			flush()
		}
	}
	flush()
	return all
}

// generateAllPageQuestions gera, para cada pagina com texto, as perguntas que
// ela responde em linguagem de leigo. Mesma estrutura em lotes do resumo, e o
// resultado vai para uma COLUNA PROPRIA: os dois campos coexistem porque o
// experimento pre-registrado compara um contra o outro.
//
// Lote que falha e pulado, como no resumo: perder as perguntas de uma pagina
// degrada a busca daquela pagina, nao quebra a indexacao.
func (e *LLMEnricher) generateAllPageQuestions(ctx context.Context, llm domain.LLMService, doc *models.Document, result domain.IndexingResult, calls *int) pageQuestionsResponse {
	var all pageQuestionsResponse
	var batch []models.DocumentPage
	flush := func() {
		if len(batch) == 0 || *calls >= enrichmentMaxBatchCalls {
			batch = nil
			return
		}
		*calls++
		systemPrompt, userPrompt := buildPageQuestionsPrompt(buildPageBatchContext(doc, batch))
		raw, err := llm.GenerateText(ctx, systemPrompt, userPrompt)
		var payload pageQuestionsResponse
		if err == nil {
			err = parseLLMJSON(raw, &payload)
		}
		if err != nil {
			log.Printf("enricher: lote de perguntas das paginas %d-%d do documento %d descartado: %v",
				batch[0].PageNumber, batch[len(batch)-1].PageNumber, doc.ID, err)
		} else {
			all.Pages = append(all.Pages, payload.Pages...)
		}
		batch = nil
	}
	for _, page := range result.DocumentPages {
		if len(strings.TrimSpace(page.Content)) < enrichmentMinPageChars {
			continue
		}
		batch = append(batch, page)
		if len(batch) == enrichmentPageBatch {
			flush()
		}
	}
	flush()
	return all
}

// applyPageQuestions grava as perguntas na coluna propria, uma por linha.
// Uma por linha, e nao separadas por virgula, porque o BM25 tokeniza por
// espaco e a quebra de linha evita colar a ultima palavra de uma pergunta na
// primeira da seguinte.
func applyPageQuestions(result domain.IndexingResult, payload pageQuestionsResponse) {
	for _, page := range payload.Pages {
		if page.PageNumber <= 0 {
			continue
		}
		perguntas := sanitizeStringList(page.Questions, 5, 200)
		if len(perguntas) == 0 {
			continue
		}
		for idx := range result.DocumentPages {
			if result.DocumentPages[idx].PageNumber == page.PageNumber {
				result.DocumentPages[idx].GeneratedQuestions = strings.Join(perguntas, "\n")
				break
			}
		}
	}
}

func buildPageBatchContext(doc *models.Document, pages []models.DocumentPage) string {
	var b strings.Builder
	b.WriteString(buildDocumentIdentityBlock(doc))
	b.WriteString("Pages:\n")
	for _, page := range pages {
		b.WriteString("- page_number=")
		b.WriteString(strconv.Itoa(page.PageNumber))
		b.WriteString(", content=")
		b.WriteString(clampText(page.Content, enrichmentPageChars))
		b.WriteString("\n")
	}
	return b.String()
}

// generateAllSectionSummaries resume todas as secoes, em lotes de
// enrichmentSectionBatch, com o texto das paginas da secao como contexto.
func (e *LLMEnricher) generateAllSectionSummaries(ctx context.Context, llm domain.LLMService, doc *models.Document, result domain.IndexingResult, calls *int, hits *enrichmentCacheHits) sectionSummaryResponse {
	var all sectionSummaryResponse
	pageContent := make(map[int]string, len(result.DocumentPages))
	for _, page := range result.DocumentPages {
		pageContent[page.PageNumber] = page.Content
	}
	scope := enrichmentCacheScope(llm, doc)
	hashByOrder := make(map[int]string, len(result.OutlineNodes))
	hashes := make([]string, 0, len(result.OutlineNodes))
	for _, node := range result.OutlineNodes {
		h := enrichmentHash(node.Title + "\n" + sectionOwnText(node, pageContent))
		hashByOrder[node.SequenceOrder] = h
		hashes = append(hashes, h)
	}
	cached := e.lookupCache(ctx, scope, "section", hashes)
	var batch []models.DocumentOutlineNode
	flush := func() {
		if len(batch) == 0 || *calls >= enrichmentMaxBatchCalls {
			batch = nil
			return
		}
		*calls++
		systemPrompt, userPrompt := buildOutlineNodeSummaryPrompt(buildSectionBatchContext(doc, batch, pageContent))
		raw, err := llm.GenerateText(ctx, systemPrompt, userPrompt)
		var payload sectionSummaryResponse
		if err == nil {
			err = parseLLMJSON(raw, &payload)
		}
		if err != nil {
			log.Printf("enricher: lote de secoes do documento %d descartado: %v", doc.ID, err)
		} else {
			items := sanitizeSectionSummaries(payload.Nodes, len(batch))
			all.Nodes = append(all.Nodes, items...)
			entries := make([]domain.EnrichmentCacheEntry, 0, len(items))
			for _, item := range items {
				if h := hashByOrder[item.SequenceOrder]; h != "" {
					entries = append(entries, domain.EnrichmentCacheEntry{Kind: "section", ContentHash: h, Summary: item.Summary})
				}
			}
			e.storeCache(ctx, scope, entries)
		}
		batch = nil
	}
	for _, node := range result.OutlineNodes {
		if entry, ok := cached[hashByOrder[node.SequenceOrder]]; ok {
			all.Nodes = append(all.Nodes, struct {
				SequenceOrder int    `json:"sequence_order"`
				Summary       string `json:"summary"`
			}{SequenceOrder: node.SequenceOrder, Summary: entry.Summary})
			hits.sections++
			continue
		}
		batch = append(batch, node)
		if len(batch) == enrichmentSectionBatch {
			flush()
		}
	}
	flush()
	return all
}

func buildSectionBatchContext(doc *models.Document, nodes []models.DocumentOutlineNode, pageContent map[int]string) string {
	var b strings.Builder
	b.WriteString(buildDocumentIdentityBlock(doc))
	b.WriteString("Sections:\n")
	for _, node := range nodes {
		b.WriteString("- sequence_order=")
		b.WriteString(strconv.Itoa(node.SequenceOrder))
		b.WriteString(", level=")
		b.WriteString(strconv.Itoa(node.Level))
		b.WriteString(", title=")
		b.WriteString(node.Title)
		if text := sectionOwnText(node, pageContent); text != "" {
			b.WriteString(", text=")
			b.WriteString(text)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// sectionOwnText e o texto que representa a secao no prompt e no hash do
// cache: o retrieval_text (paragrafos dela, sem os filhos) ou, sem ele, o
// texto das paginas start..end. Sempre limitado a enrichmentSectionChars.
func sectionOwnText(node models.DocumentOutlineNode, pageContent map[int]string) string {
	if own, _ := metadataString(node.Metadata, "retrieval_text"); strings.TrimSpace(own) != "" {
		return clampText(own, enrichmentSectionChars)
	}
	if node.StartPage == nil {
		return ""
	}
	text := ""
	end := *node.StartPage
	if node.EndPage != nil && *node.EndPage > end {
		end = *node.EndPage
	}
	for page := *node.StartPage; page <= end && len(text) < enrichmentSectionChars; page++ {
		text += " " + pageContent[page]
	}
	return clampText(text, enrichmentSectionChars)
}

func splitTopics(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return sanitizeStringList(strings.Split(value, ","), 5, 60)
}

func parseLLMJSON(raw string, target any) error {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	if err := json.Unmarshal([]byte(cleaned), target); err == nil {
		return nil
	}

	extracted := extractJSONEnvelope(cleaned)
	if extracted == "" {
		return json.Unmarshal([]byte(cleaned), target)
	}

	return json.Unmarshal([]byte(extracted), target)
}

func extractJSONEnvelope(raw string) string {
	start := -1
	for idx, r := range raw {
		if r == '{' || r == '[' {
			start = idx
			break
		}
	}
	if start < 0 {
		return ""
	}

	var (
		depth       int
		inString    bool
		escapeNext  bool
		openingRune rune
		closingRune rune
	)

	for idx, r := range raw[start:] {
		absoluteIdx := start + idx
		if openingRune == 0 {
			openingRune = r
			if openingRune == '{' {
				closingRune = '}'
			} else {
				closingRune = ']'
			}
		}

		if inString {
			if escapeNext {
				escapeNext = false
				continue
			}
			if r == '\\' {
				escapeNext = true
				continue
			}
			if r == '"' {
				inString = false
			}
			continue
		}

		switch r {
		case '"':
			inString = true
		case openingRune:
			depth++
		case closingRune:
			depth--
			if depth == 0 {
				return strings.TrimSpace(raw[start : absoluteIdx+1])
			}
		}
	}

	return ""
}

func applyDocumentSynopsis(result *domain.IndexingResult, documentID uint, payload documentSynopsisResponse) {
	if result == nil {
		return
	}

	appendMetadataIfPresent(result, documentID, "document_summary", payload.DocumentSummary)

	if keywords := normalizeKeywords(payload.Keywords); keywords != "" {
		appendMetadata(result, documentID, "keywords", keywords)
	}

	appendMetadataIfPresent(result, documentID, "document_purpose", payload.DocumentPurpose)
}

func buildFallbackDocumentSynopsis(doc *models.Document, result domain.IndexingResult) documentSynopsisResponse {
	summary := firstNonEmptyNonBlank(
		firstNonEmptyPageContent(result.DocumentPages),
		firstNonEmptyParagraphContent(result.ParagraphNodes),
		titleFromDocument(doc),
	)
	purpose := firstNonEmptyNonBlank(
		firstNonEmptyOutlineTitle(result.OutlineNodes),
		summary,
		titleFromDocument(doc),
	)

	return documentSynopsisResponse{
		DocumentSummary: clampText(summary, 800),
		DocumentPurpose: clampText(purpose, 400),
		Keywords:        fallbackSynopsisKeywords(doc, result),
	}
}

func applyOutlineNodeSummaries(result *domain.IndexingResult, payload sectionSummaryResponse) {
	if result == nil {
		return
	}

	for _, section := range payload.Nodes {
		if section.SequenceOrder < 0 || section.SequenceOrder >= len(result.OutlineNodes) {
			continue
		}
		summary := strings.TrimSpace(section.Summary)
		if summary == "" {
			continue
		}
		result.OutlineNodes[section.SequenceOrder].Summary = summary
	}
}

func applyPageSummaries(result *domain.IndexingResult, payload pageSummaryResponse) {
	if result == nil {
		return
	}

	for _, page := range payload.Pages {
		if page.PageNumber <= 0 {
			continue
		}
		for idx := range result.DocumentPages {
			if result.DocumentPages[idx].PageNumber != page.PageNumber {
				continue
			}
			result.DocumentPages[idx].Summary = strings.TrimSpace(page.Summary)
			if topics := sanitizeStringList(page.Topics, 5, 60); len(topics) > 0 {
				if result.DocumentPages[idx].Metadata == nil {
					result.DocumentPages[idx].Metadata = map[string]any{}
				}
				result.DocumentPages[idx].Metadata["topics"] = strings.Join(topics, ", ")
			}
			break
		}
	}
}

func fillMissingPageSummaries(result *domain.IndexingResult) {
	if result == nil {
		return
	}

	for idx := range result.DocumentPages {
		if strings.TrimSpace(result.DocumentPages[idx].Summary) != "" {
			continue
		}
		result.DocumentPages[idx].Summary = fallbackPageSummary(result.DocumentPages[idx].Content)
	}
}

func fallbackPageSummary(content string) string {
	return clampText(content, 320)
}

func fallbackSynopsisKeywords(doc *models.Document, result domain.IndexingResult) []string {
	candidates := []string{}

	if title := strings.TrimSpace(titleFromDocument(doc)); title != "" {
		candidates = append(candidates, title)
	}
	for _, node := range result.OutlineNodes {
		if title := strings.TrimSpace(node.Title); title != "" {
			candidates = append(candidates, title)
		}
	}

	return sanitizeStringList(candidates, 8, 80)
}

func firstNonEmptyPageContent(pages []models.DocumentPage) string {
	for _, page := range pages {
		if value := strings.TrimSpace(page.Content); value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyParagraphContent(paragraphs []models.ParagraphNode) string {
	for _, paragraph := range paragraphs {
		if value := strings.TrimSpace(paragraph.Content); value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyOutlineTitle(nodes []models.DocumentOutlineNode) string {
	for _, node := range nodes {
		if value := strings.TrimSpace(node.Title); value != "" {
			return value
		}
	}
	return ""
}

func titleFromDocument(doc *models.Document) string {
	if doc == nil {
		return ""
	}
	return strings.TrimSpace(doc.Title)
}

func firstNonEmptyNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func applyRetrievalSignals(result *domain.IndexingResult, documentID uint, payload retrievalSignalsResponse) {
	if result == nil {
		return
	}

	appendMetadataIfPresent(result, documentID, "topics", normalizeKeywords(payload.Topics))
	appendMetadataIfPresent(result, documentID, "entities", normalizeKeywords(payload.Entities))
	appendMetadataIfPresent(result, documentID, "dates", normalizeKeywords(payload.Dates))
}

func appendMetadataIfPresent(result *domain.IndexingResult, documentID uint, key, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	appendMetadata(result, documentID, key, value)
}

func appendMetadata(result *domain.IndexingResult, documentID uint, key, value string) {
	result.MetadataPairs = append(result.MetadataPairs, models.MetadataPair{
		DocumentID:    documentID,
		SequenceOrder: nextIndexingMetadataSequence(result.MetadataPairs),
		Key:           key,
		Value:         strings.TrimSpace(value),
	})
}

func nextIndexingMetadataSequence(pairs []models.MetadataPair) int {
	maxSequence := -1
	for _, pair := range pairs {
		if pair.SequenceOrder > maxSequence {
			maxSequence = pair.SequenceOrder
		}
	}
	return maxSequence + 1
}

func documentIDFromResult(result domain.IndexingResult) uint {
	if len(result.DocumentPages) > 0 {
		return result.DocumentPages[0].DocumentID
	}
	if len(result.ParagraphNodes) > 0 {
		return result.ParagraphNodes[0].DocumentID
	}
	if len(result.TableSections) > 0 {
		return result.TableSections[0].DocumentID
	}
	if len(result.MetadataPairs) > 0 {
		return result.MetadataPairs[0].DocumentID
	}
	if len(result.OutlineNodes) > 0 {
		return result.OutlineNodes[0].DocumentID
	}
	if len(result.Chunks) > 0 {
		return result.Chunks[0].DocumentID
	}
	return 0
}

func normalizeKeywords(keywords []string) string {
	items := make([]string, 0, min(8, len(keywords)))
	seen := make(map[string]struct{}, len(keywords))
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}
		normalizedKey := strings.ToLower(keyword)
		if _, ok := seen[normalizedKey]; ok {
			continue
		}
		seen[normalizedKey] = struct{}{}
		items = append(items, keyword)
		if len(items) == 8 {
			break
		}
	}
	return strings.Join(items, ", ")
}

func sanitizeStringList(values []string, limit, maxLen int) []string {
	items := make([]string, 0, min(limit, len(values)))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		cleaned := clampText(value, maxLen)
		if cleaned == "" {
			continue
		}

		normalizedKey := strings.ToLower(cleaned)
		if _, ok := seen[normalizedKey]; ok {
			continue
		}
		seen[normalizedKey] = struct{}{}
		items = append(items, cleaned)
		if len(items) == limit {
			break
		}
	}

	return items
}

func sanitizeSectionSummaries(
	sections []struct {
		SequenceOrder int    `json:"sequence_order"`
		Summary       string `json:"summary"`
	},
	limit int,
) []struct {
	SequenceOrder int    `json:"sequence_order"`
	Summary       string `json:"summary"`
} {
	items := make([]struct {
		SequenceOrder int    `json:"sequence_order"`
		Summary       string `json:"summary"`
	}, 0, min(limit, len(sections)))
	seen := make(map[int]struct{}, len(sections))
	for _, section := range sections {
		if section.SequenceOrder < 0 {
			continue
		}
		if _, ok := seen[section.SequenceOrder]; ok {
			continue
		}

		summary := clampText(section.Summary, 500)
		if summary == "" {
			continue
		}

		seen[section.SequenceOrder] = struct{}{}
		items = append(items, struct {
			SequenceOrder int    `json:"sequence_order"`
			Summary       string `json:"summary"`
		}{
			SequenceOrder: section.SequenceOrder,
			Summary:       summary,
		})
		if len(items) == limit {
			break
		}
	}

	return items
}

func sanitizePageSummaries(pages []pageSummaryItem, limit int) []pageSummaryItem {
	items := make([]pageSummaryItem, 0, min(limit, len(pages)))
	seen := make(map[int]struct{}, len(pages))
	for _, page := range pages {
		if page.PageNumber <= 0 {
			continue
		}
		if _, ok := seen[page.PageNumber]; ok {
			continue
		}
		summary := clampText(page.Summary, 600)
		if summary == "" {
			continue
		}
		seen[page.PageNumber] = struct{}{}
		items = append(items, pageSummaryItem{PageNumber: page.PageNumber, Summary: summary, Topics: sanitizeStringList(page.Topics, 5, 60)})
		if len(items) == limit {
			break
		}
	}
	return items
}

func clampText(value string, maxLen int) string {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if cleaned == "" || maxLen <= 0 {
		return ""
	}

	runes := []rune(cleaned)
	if len(runes) <= maxLen {
		return cleaned
	}

	truncated := string(runes[:maxLen])
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > 0 {
		return truncated[:lastSpace]
	}
	return strings.TrimSpace(truncated)
}
