package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"strings"

	"thing-index/internal/domain"
)

// Expansao da pergunta antes da busca, pre-registrada em
// docs/prds/pre-registro-expansao-consulta.md.
//
// Existe porque a queda de vocabulario e o unico efeito com significancia
// medido no trabalho (0,903 literal contra 0,677 parafrase, p = 0,021), e
// expandir o DOCUMENTO ja foi medido e deu nulo em quatro bracos. A assimetria
// que motiva tentar o outro lado: existem muitas formas leigas de perguntar e
// uma forma tecnica de nomear a coisa.
//
// A expansao ve APENAS o texto da pergunta. Nunca o acervo, a pagina-alvo nem o
// gabarito: sem isso, um termo que acerte o vocabulario do documento seria
// vazamento em vez de conhecimento de mundo do modelo.

// queryExpansionSystemPrompt esta congelado pelo pre-registro. Alterar o texto
// quebra o teste que trava o hash, de proposito: rodar varias versoes e
// reportar a melhor produz um numero favoravel e sem valor.
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

// queryExpansionEnabled le o braco do experimento. Desligado por padrao: o
// braco C do pre-registro e o comportamento atual do sistema.
func queryExpansionEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("THINGINDEX_QUERY_EXPANSION"))) {
	case "1", "true", "on":
		return true
	default:
		return false
	}
}

// chatModelName entra na chave do cache: trocar de modelo tem que invalidar as
// expansoes guardadas, senao uma medicao herdaria os termos da outra.
func chatModelName(llm domain.LLMService) string {
	if m, ok := llm.(domain.LLMMetadataProvider); ok {
		return m.ModelName()
	}
	return ""
}

type queryExpander struct {
	cache domain.EnrichmentCache
}

func newQueryExpander(cache domain.EnrichmentCache) *queryExpander {
	return &queryExpander{cache: cache}
}

// queryExpansionKey normaliza antes do hash para que a mesma pergunta com outra
// caixa ou espacamento reaproveite o cache em vez de gastar outra chamada.
func queryExpansionKey(question string) string {
	normalized := strings.Join(strings.Fields(strings.ToLower(question)), " ")
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

// sanitizeExpansion corta o que a LLM devolveu para termos utilizaveis num
// indice lexical. Descarta linha longa demais para ser lista de termos: o
// modelo as vezes ignora a instrucao e responde a pergunta, e uma resposta
// inteira colada na consulta arruinaria o ranking.
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

// Expand devolve os termos a acrescentar a pergunta, ou "" quando nao ha nada a
// acrescentar. Erro de LLM ou de cache nao e fatal: a busca segue com a
// pergunta original, que e o braco de controle.
func (e *queryExpander) Expand(ctx context.Context, llm domain.LLMService, model, question string) string {
	if e == nil || llm == nil {
		return ""
	}
	trimmed := strings.TrimSpace(question)
	if trimmed == "" {
		return ""
	}
	key := queryExpansionKey(trimmed)

	if e.cache != nil {
		if found, err := e.cache.LookupEnrichment(ctx, queryExpansionCacheKind, model, []string{key}); err == nil {
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

	// Grava tambem a expansao vazia: sem isso, toda pergunta que ja usa o
	// vocabulario formal pagaria uma chamada de LLM em cada repeticao, e a
	// medicao de custo do experimento sairia errada.
	if e.cache != nil {
		if err := e.cache.StoreEnrichment(ctx, []domain.EnrichmentCacheEntry{{
			Kind:        queryExpansionCacheKind,
			Model:       model,
			ContentHash: key,
			Summary:     terms,
		}}); err != nil {
			log.Printf("cache da expansao da pergunta: %v", err)
		}
	}
	return terms
}
