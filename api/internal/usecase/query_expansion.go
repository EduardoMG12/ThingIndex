package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"strings"

	"thing-index/internal/domain"
)

// Expansao da pergunta antes da busca. A LLM ve apenas o texto da pergunta,
// nunca o acervo, para os termos nao vazarem o vocabulario do documento.

// queryExpansionSystemPrompt e travado por hash no teste.
const queryExpansionSystemPrompt = `You expand a search query for a lexical (BM25) index over Portuguese documents.
Return only a single line of terms separated by spaces.
Do not answer the question.
Do not explain.
Write the technical or formal terms that the question describes without naming them.
Include the singular form of each term.
At most 8 terms.
If the question already uses the formal vocabulary, return an empty line.`

const (
	queryExpansionCacheKind = "query"
	queryExpansionMaxTerms  = 8
)

// parseQueryExpansion: desligado por padrao. Chamada uma vez, em
// NewRetrievalOptions.
func parseQueryExpansion(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "on":
		return true
	default:
		return false
	}
}

type queryExpander struct {
	cache domain.EnrichmentCache
}

func newQueryExpander(cache domain.EnrichmentCache) *queryExpander {
	return &queryExpander{cache: cache}
}

func queryExpansionKey(question string) string {
	normalized := strings.Join(strings.Fields(strings.ToLower(question)), " ")
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

// sanitizeExpansion descarta linhas longas: as vezes o modelo responde a
// pergunta em vez de listar termos.
func sanitizeExpansion(raw string) string {
	line := strings.TrimSpace(raw)
	if idx := strings.IndexAny(line, "\n\r"); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}
	line = strings.Trim(line, "\"'`")
	if line == "" {
		return ""
	}
	fields := strings.Fields(line)
	if len(fields) > queryExpansionMaxTerms*3 {
		return ""
	}
	seen := make(map[string]struct{}, len(fields))
	terms := make([]string, 0, queryExpansionMaxTerms)
	for _, field := range fields {
		term := strings.ToLower(strings.Trim(field, ".,;:()[]\"'"))
		if len([]rune(term)) < 2 {
			continue
		}
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
		if len(terms) == queryExpansionMaxTerms {
			break
		}
	}
	return strings.Join(terms, " ")
}

// Expand devolve os termos a acrescentar, ou "". Erro de LLM ou cache nao e
// fatal: segue com a pergunta original. O escopo do cache inclui usuario e
// modelo.
func (e *queryExpander) Expand(ctx context.Context, llm domain.LLMService, scope domain.EnrichmentCacheScope, question string) string {
	if e == nil || llm == nil {
		return ""
	}
	trimmed := strings.TrimSpace(question)
	if trimmed == "" {
		return ""
	}
	key := queryExpansionKey(trimmed)

	if e.cache != nil {
		if found, err := e.cache.LookupEnrichment(ctx, scope, queryExpansionCacheKind, []string{key}); err == nil {
			if entry, ok := found[key]; ok {
				return entry.Summary
			}
		}
	}

	raw, err := llm.GenerateText(ctx, queryExpansionSystemPrompt, "Pergunta: "+trimmed+"\nTermos:")
	if err != nil {
		log.Printf("expansao da pergunta: %v", err)
		return ""
	}
	terms := sanitizeExpansion(raw)

	// Grava tambem a expansao vazia, senao repetir a pergunta chamaria a LLM de novo.
	if e.cache != nil {
		if err := e.cache.StoreEnrichment(ctx, scope, []domain.EnrichmentCacheEntry{{
			Kind:        queryExpansionCacheKind,
			ContentHash: key,
			Summary:     terms,
		}}); err != nil {
			log.Printf("cache da expansao da pergunta: %v", err)
		}
	}
	return terms
}
