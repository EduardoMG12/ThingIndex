package indexing

import (
	"strings"

	"thing-index/internal/models"
)

const (
	maxChunkContentLength   = 4000
	minChunkContentLength   = 80
	maxChunksPerOutlineNode = 5
)

type chunkBuilder struct {
	documentID uint
}

func newChunkBuilder(documentID uint) *chunkBuilder {
	return &chunkBuilder{documentID: documentID}
}

// chunkPageFor decide a pagina de origem dos chunks de um no do sumario.
//
// Precedencia, da origem mais proxima para a mais distante: pagina do primeiro
// paragrafo NAO VAZIO do intervalo do no; se nula, o StartPage do proprio no;
// se tambem nula, nulo. Nunca zero.
//
// A ordem importa. O StartPage do no e derivado dos mesmos paragrafos, entao
// usar o paragrafo direto e um passo de derivacao a menos entre a evidencia e
// a coordenada publicada. E nada aqui infere pagina por semelhanca de texto:
// heuristica de atribuicao produziria uma pagina plausivel e errada, que e o
// pior resultado possivel numa metrica de auditabilidade (principio I).
//
// O retorno e sempre um ponteiro NOVO, nunca o do paragrafo ou o do no. Devolver
// o ponteiro de origem faria o chunk compartilhar memoria com a estrutura que o
// gerou: alterar a pagina do paragrafo depois mudaria a pagina do chunk sem que
// ninguem tivesse escrito nele. Nao ha esse caminho hoje, mas o principio II
// exige que a mesma entrada produza a mesma saida, e proveniencia que muda
// sozinha e o oposto disso.
func chunkPageFor(node models.DocumentOutlineNode, paragraphs []models.ParagraphNode) *int {
	if start, end, ok := outlineParagraphRange(node); ok {
		for idx := start; idx <= end && idx < len(paragraphs); idx++ {
			if strings.TrimSpace(paragraphs[idx].Content) == "" {
				continue
			}
			if page := paragraphs[idx].PageNumber; page != nil {
				return intPtr(*page)
			}
		}
	}
	if node.StartPage != nil {
		return intPtr(*node.StartPage)
	}
	return nil
}

func (b *chunkBuilder) buildFromOutline(
	nodes []models.DocumentOutlineNode,
	paragraphs []models.ParagraphNode,
	tables []models.TableSection,
	pages []models.DocumentPage,
) []models.DocumentChunk {
	if len(nodes) == 0 {
		return b.buildFallbackFromPages(pages)
	}

	chunks := make([]models.DocumentChunk, 0, len(nodes)*2)
	for _, node := range nodes {
		nodeChunks := b.buildChunksForNode(node, paragraphs, tables)
		chunks = append(chunks, nodeChunks...)
	}

	if len(chunks) == 0 {
		return b.buildFallbackFromPages(pages)
	}

	return chunks
}

func (b *chunkBuilder) buildChunksForNode(
	node models.DocumentOutlineNode,
	paragraphs []models.ParagraphNode,
	tables []models.TableSection,
) []models.DocumentChunk {
	start, end, ok := outlineParagraphRange(node)
	if !ok {
		return nil
	}

	contentParts := make([]string, 0, 4)

	contentParts = append(contentParts, node.Title)
	if node.Summary != "" {
		contentParts = append(contentParts, node.Summary)
	}

	for idx := start; idx <= end && idx < len(paragraphs); idx++ {
		text := strings.TrimSpace(paragraphs[idx].Content)
		if text == "" {
			continue
		}
		contentParts = append(contentParts, text)
		if len(strings.Join(contentParts, "\n\n")) > maxChunkContentLength {
			break
		}
	}

	for _, table := range matchingOutlineTables(node, tables) {
		tableText := formatTableForChunk(table)
		if tableText == "" {
			continue
		}
		contentParts = append(contentParts, tableText)
		if len(strings.Join(contentParts, "\n\n")) > maxChunkContentLength {
			break
		}
	}

	fullContent := strings.Join(contentParts, "\n\n")
	if len(strings.TrimSpace(fullContent)) < minChunkContentLength {
		return nil
	}

	pageNumber := chunkPageFor(node, paragraphs)

	if len([]rune(fullContent)) > maxChunkContentLength {
		return b.splitOversizedChunk(node, fullContent, pageNumber)
	}

	return []models.DocumentChunk{
		{
			DocumentID:           b.documentID,
			SequenceOrder:        node.SequenceOrder,
			ChunkType:            "outline",
			Title:                node.Title,
			Content:              fullContent,
			SourceParagraphStart: intPtr(start),
			SourceParagraphEnd:   intPtr(end),
			PageNumber:           pageNumber,
			TokenEstimate:        estimateTextTokens(fullContent),
			Metadata: map[string]any{
				"node_id":     node.NodeID,
				"level":       node.Level,
				"section_key": metadataValue(node.Metadata, "section_key"),
			},
		},
	}
}

// splitOversizedChunk recebe a pagina ja resolvida em vez de recalcular: as
// partes de um mesmo no compartilham a pagina de inicio do no, e recalcular
// abriria espaco para as partes divergirem entre si.
func (b *chunkBuilder) splitOversizedChunk(
	node models.DocumentOutlineNode,
	content string,
	pageNumber *int,
) []models.DocumentChunk {
	sentences := splitIntoSentences(content)
	if len(sentences) == 0 {
		return nil
	}

	chunks := make([]models.DocumentChunk, 0, maxChunksPerOutlineNode)
	var current strings.Builder
	current.WriteString(node.Title + "\n\n")
	chunkSeq := 0

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		projected := current.Len() + len(sentence) + 2
		if projected > maxChunkContentLength && current.Len() > minChunkContentLength {
			chunks = append(chunks, models.DocumentChunk{
				DocumentID:    b.documentID,
				SequenceOrder: node.SequenceOrder*100 + chunkSeq,
				ChunkType:     "outline-split",
				Title:         node.Title,
				Content:       strings.TrimSpace(current.String()),
				PageNumber:    pageNumber,
				TokenEstimate: estimateTextTokens(current.String()),
				Metadata: map[string]any{
					"node_id":     node.NodeID,
					"level":       node.Level,
					"split_part":  chunkSeq + 1,
					"section_key": metadataValue(node.Metadata, "section_key"),
				},
			})
			chunkSeq++
			current.Reset()
			current.WriteString(node.Title + "\n\n")
		}
		current.WriteString(sentence)
		current.WriteString("\n\n")
	}

	if current.Len() >= minChunkContentLength {
		chunks = append(chunks, models.DocumentChunk{
			DocumentID:    b.documentID,
			SequenceOrder: node.SequenceOrder*100 + chunkSeq,
			ChunkType:     "outline-split",
			Title:         node.Title,
			Content:       strings.TrimSpace(current.String()),
			PageNumber:    pageNumber,
			TokenEstimate: estimateTextTokens(current.String()),
			Metadata: map[string]any{
				"node_id":     node.NodeID,
				"level":       node.Level,
				"split_part":  chunkSeq + 1,
				"section_key": metadataValue(node.Metadata, "section_key"),
			},
		})
	}

	if len(chunks) > maxChunksPerOutlineNode {
		chunks = chunks[:maxChunksPerOutlineNode]
	}

	return chunks
}

func (b *chunkBuilder) buildFallbackFromPages(pages []models.DocumentPage) []models.DocumentChunk {
	if len(pages) == 0 {
		return nil
	}

	chunks := make([]models.DocumentChunk, 0, len(pages))
	for idx, page := range pages {
		content := strings.TrimSpace(page.Content)
		if content == "" {
			continue
		}

		chunks = append(chunks, models.DocumentChunk{
			DocumentID:    b.documentID,
			SequenceOrder: idx,
			ChunkType:     "page",
			Title:         pageTitle(page),
			Content:       content,
			PageNumber:    intPtr(page.PageNumber),
			TokenEstimate: page.TokenEstimate,
			Metadata: map[string]any{
				"page_number": page.PageNumber,
			},
		})
	}

	return chunks
}

func splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for _, r := range text {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			s := strings.TrimSpace(current.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}

	remainder := strings.TrimSpace(current.String())
	if remainder != "" {
		sentences = append(sentences, remainder)
	}

	return sentences
}

func formatTableForChunk(table models.TableSection) string {
	var b strings.Builder

	if table.Title != "" {
		b.WriteString("Table: ")
		b.WriteString(table.Title)
		b.WriteString("\n")
	}

	if len(table.Headers) > 0 {
		b.WriteString("Columns: ")
		b.WriteString(strings.Join(table.Headers, ", "))
		b.WriteString("\n")
	}

	for rowIdx := 0; rowIdx < len(table.Rows) && rowIdx < 10; rowIdx++ {
		row := table.Rows[rowIdx]
		if len(row) == 0 {
			continue
		}
		b.WriteString("Row: ")
		b.WriteString(strings.Join(row, " | "))
		b.WriteString("\n")
	}

	return strings.TrimSpace(b.String())
}

func pageTitle(page models.DocumentPage) string {
	if page.Summary != "" {
		return firstSentence(page.Summary)
	}
	return firstSentence(page.Content)
}

func metadataValue(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	v, ok := metadata[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
