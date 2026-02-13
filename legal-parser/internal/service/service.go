package service

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/user/legisgraph/legal-parser/internal/domain"
)

// ParserService orchestrates the parsing pipeline.
// It depends on interfaces only — never on concrete adapters.
type ParserService struct {
	source domain.SourceProvider
	output domain.OutputProvider
}

// NewParserService creates a new ParserService with injected dependencies.
func NewParserService(source domain.SourceProvider, output domain.OutputProvider) *ParserService {
	return &ParserService{
		source: source,
		output: output,
	}
}

// ProcessAll lists all documents from the source, parses each one,
// and saves the structured JSON via the output provider.
func (ps *ParserService) ProcessAll() error {
	docs, err := ps.source.ListDocuments()
	if err != nil {
		return fmt.Errorf("failed to list documents: %w", err)
	}

	log.Printf("Found %d documents to process", len(docs))

	var processed, skipped, failed int

	for _, rawDoc := range docs {
		err := ps.processDocument(rawDoc)
		if err != nil {
			log.Printf("[SKIP] %s: %v", rawDoc.ID, err)
			if strings.Contains(err.Error(), "could not extract identity") {
				skipped++
			} else {
				failed++
			}
			continue
		}
		processed++
	}

	log.Printf("------------------------------------------")
	log.Printf("PARSING SUMMARY:")
	log.Printf("- Total documents:  %d", len(docs))
	log.Printf("- Processed:        %d", processed)
	log.Printf("- Skipped (no ID):  %d", skipped)
	log.Printf("- Failed:           %d", failed)
	log.Printf("------------------------------------------")

	return nil
}

func (ps *ParserService) processDocument(rawDoc domain.RawDocument) error {
	// 1. Read raw HTML content
	content, err := ps.source.GetContent(rawDoc.ID)
	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}

	// 2. Read crawler metadata
	meta, err := ps.source.GetMetadata(rawDoc.ID)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	// 3. Clean HTML and get goquery document
	doc, err := CleanHTML(content)
	if err != nil {
		return fmt.Errorf("failed to clean HTML: %w", err)
	}

	// 4. Extract title and ementa
	title := ExtractTitle(doc)
	ementa := ExtractEmenta(doc)

	if title == "" {
		return fmt.Errorf("could not extract title from document")
	}

	// 5. Parse identity from title
	identity, err := ExtractIdentityFromTitle(title)
	if err != nil {
		return err
	}

	// 6. Generate canonical ID
	canonicalID := GenerateCanonicalID(identity.Tipo, "federal", identity.Numero, identity.Ano)
	if canonicalID == "" {
		return fmt.Errorf("could not extract identity from title: %q", title)
	}

	// 7. Extract hierarchical structure
	articles := ExtractStructure(doc)

	// 8. Extract cross-references from the full text
	fullText := doc.Find("body").Text()
	references := ExtractReferences(fullText)

	// 9. Build the final document
	legalDoc := domain.LegalDocument{
		ID:          canonicalID,
		Tipo:        identity.Tipo,
		Esfera:      "federal",
		Numero:      identity.Numero,
		Ano:         identity.Ano,
		Titulo:      title,
		Ementa:      ementa,
		URLOrigem:   meta.URL,
		URLPai:      meta.ParentURL,
		Artigos:     articles,
		Referencias: references,
	}

	// 10. Serialize to JSON
	jsonData, err := json.MarshalIndent(legalDoc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize document: %w", err)
	}

	// 11. Save via output provider
	if err := ps.output.Save(canonicalID, jsonData); err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}

	log.Printf("[OK] %s → %s (%d artigos, %d refs)",
		rawDoc.ID, canonicalID, len(articles), len(references))

	return nil
}
