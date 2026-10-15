package indexing

import (
	"context"
	"strings"
	"testing"

	"thing-index/internal/models"
)

// Regressao: o texto de um <li>/<p> aninhado era emitido duas vezes.
func TestIndexHTMLDoesNotDuplicateNestedBlocks(t *testing.T) {
	t.Parallel()

	input := `<html><body>
<h1>Contrato</h1>
<ul>
  <li>Clausula primeira firmada entre as partes.
    <ul><li>Subitem aninhado com detalhe relevante.</li></ul>
  </li>
</ul>
<p>Paragrafo externo <span>com trecho inline preservado</span>.</p>
</body></html>`

	doc := &models.Document{ID: 1, MimeType: "text/html"}
	result, err := NewDeterministicIndexer().Index(context.Background(), doc, strings.NewReader(input))
	if err != nil {
		t.Fatalf("index: %v", err)
	}

	occurrences := 0
	for _, node := range result.ParagraphNodes {
		occurrences += strings.Count(node.Content, "Subitem aninhado com detalhe relevante.")
	}
	if occurrences != 1 {
		var got []string
		for _, node := range result.ParagraphNodes {
			got = append(got, node.Content)
		}
		t.Fatalf("subitem aparece %d vezes, quero 1; paragrafos=%q", occurrences, got)
	}

	// Texto inline (<span>) continua fazendo parte do paragrafo que o contem.
	inlineKept := false
	for _, node := range result.ParagraphNodes {
		if strings.Contains(node.Content, "com trecho inline preservado") {
			inlineKept = true
		}
	}
	if !inlineKept {
		t.Fatal("texto inline dentro de <p> foi perdido")
	}
}

// Tabelas dentro de listas continuam sendo descobertas apesar do corte de
// emissao de texto aninhado.
func TestIndexHTMLFindsTableInsideList(t *testing.T) {
	t.Parallel()

	input := `<html><body><ul><li>Itens contratados
<table><tr><th>Campo</th><th>Valor</th></tr><tr><td>Total</td><td>100</td></tr></table>
</li></ul></body></html>`

	doc := &models.Document{ID: 1, MimeType: "text/html"}
	result, err := NewDeterministicIndexer().Index(context.Background(), doc, strings.NewReader(input))
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if len(result.TableSections) != 1 {
		t.Fatalf("tabelas = %d, quero 1", len(result.TableSections))
	}
}
