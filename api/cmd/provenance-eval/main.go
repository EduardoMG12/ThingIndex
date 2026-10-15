// Command provenance-eval mede TPC e TCV do indice sobre o corpus C1 e gera a
// tabela LaTeX, quebrada por tipo estrutural e por escopo.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/evalcorpus"
	"thing-index/internal/indexing"
	"thing-index/internal/models"
	"thing-index/internal/provenance"
)

func main() {
	corpusPath := flag.String("corpus", filepath.Join("internal", "evalcorpus", "testdata", "corpus-c1.json"),
		"corpus de avaliacao")
	texPath := flag.String("tex", "", "arquivo .tex de saida (vazio = stdout)")
	flag.Parse()

	corpus, err := evalcorpus.Load(*corpusPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "provenance-eval: %v\n", err)
		os.Exit(1)
	}

	report, citations := measure(corpus)

	tabela := buildTable(report, citations, len(corpus.Documents))

	if *texPath == "" {
		fmt.Print(tabela)
	} else {
		if err := os.WriteFile(*texPath, []byte(evalcorpus.Tabela(tabela)), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "provenance-eval: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "provenance-eval: %s\n", *texPath)
	}

	printSummary(os.Stderr, report, citations)
}

// measure indexa o corpus com o DeterministicIndexer real e avalia o resultado.
func measure(corpus *evalcorpus.Corpus) (provenance.Report, provenance.CitationReport) {
	indexer := indexing.NewDeterministicIndexer()

	var items []provenance.Item
	var citations []provenance.Citation

	for i, doc := range corpus.Documents {
		documentID := uint(i + 1)

		// Mesmo caminho de um .txt enviado pelo usuario.
		modelo := &models.Document{ID: documentID, MimeType: "text/plain"}
		result, err := indexer.Index(context.Background(), modelo, strings.NewReader(doc.Content))
		if err != nil {
			fmt.Fprintf(os.Stderr, "provenance-eval: indexar %s: %v\n", doc.ID, err)
			os.Exit(1)
		}

		items = append(items, itemsFromResult(documentID, result)...)
		citations = append(citations, citationsFromResult(doc, result)...)
	}

	return provenance.Evaluate(items), provenance.VerifyCitations(citations)
}

func itemsFromResult(documentID uint, result domain.IndexingResult) []provenance.Item {
	var items []provenance.Item

	for _, node := range result.ParagraphNodes {
		items = append(items, provenance.Item{
			Kind:          "paragraph",
			Scope:         provenance.ScopePage,
			DocumentID:    documentID,
			PageNumber:    node.PageNumber,
			SequenceOrder: node.SequenceOrder,
			Content:       node.Content,
		})
	}
	for _, page := range result.DocumentPages {
		pageNumber := page.PageNumber
		items = append(items, provenance.Item{
			Kind:          "page",
			Scope:         provenance.ScopePage,
			DocumentID:    documentID,
			PageNumber:    &pageNumber,
			SequenceOrder: page.PageNumber,
			Content:       page.Content,
		})
	}
	for _, table := range result.TableSections {
		// TableSection ganhou PageNumber quando tabelas de PDF viraram fonte do chat;
		// antes aqui ia nil.
		items = append(items, provenance.Item{
			Kind:          "table",
			Scope:         provenance.ScopePage,
			DocumentID:    documentID,
			PageNumber:    table.PageNumber,
			SequenceOrder: table.SequenceOrder,
			Content:       strings.Join(table.Headers, " "),
		})
	}
	for _, pair := range result.MetadataPairs {
		// Metadados sao estatisticas do documento inteiro ("paragraph_count",
		// "tree_builder"): escopo de documento, sem pagina.
		items = append(items, provenance.Item{
			Kind:          "metadata",
			Scope:         provenance.ScopeDocument,
			DocumentID:    documentID,
			PageNumber:    nil,
			SequenceOrder: pair.SequenceOrder,
			Content:       pair.Value,
		})
	}
	for _, chunk := range result.Chunks {
		// A pagina do chunk vem do paragrafo de origem (chunkPageFor em
		// internal/indexing/chunks.go).
		items = append(items, provenance.Item{
			Kind:          "chunk",
			Scope:         provenance.ScopePage,
			DocumentID:    documentID,
			PageNumber:    chunk.PageNumber,
			SequenceOrder: chunk.SequenceOrder,
			Content:       chunk.Content,
		})
	}
	for _, node := range result.OutlineNodes {
		items = append(items, provenance.Item{
			Kind:          "outline",
			Scope:         provenance.ScopePage,
			DocumentID:    documentID,
			PageNumber:    node.StartPage,
			SequenceOrder: node.SequenceOrder,
			Content:       node.Title,
		})
	}

	return items
}

// citationsFromResult trata cada paragrafo indexado como citacao do documento
// original, para checar se a fragmentacao preservou o texto.
func citationsFromResult(doc evalcorpus.Document, result domain.IndexingResult) []provenance.Citation {
	citations := make([]provenance.Citation, 0, len(result.ParagraphNodes))
	for _, node := range result.ParagraphNodes {
		citations = append(citations, provenance.Citation{
			Snippet:       node.Content,
			SourceContent: doc.Content,
		})
	}
	return citations
}

func buildTable(report provenance.Report, citations provenance.CitationReport, documentos int) string {
	var b strings.Builder

	b.WriteString("% GERADO POR cmd/provenance-eval -- nao edite a mao.\n")
	b.WriteString("\\begin{table}[htb]\n\\centering\n")
	b.WriteString("\\caption{Auditabilidade do índice sobre o corpus C1. ")
	b.WriteString("TPC (taxa de proveniência completa) é a fração de itens cuja origem é resolvível: ")
	b.WriteString("documento, página, posição sequencial e conteúdo, todos presentes. ")
	b.WriteString("O recorte por tipo estrutural é deliberado: uma média global esconderia um tipo com proveniência incompleta. ")
	b.WriteString("Itens de escopo de documento (pares de metadado, que são estatísticas do documento inteiro) são avaliados sem exigência de página, porque não existem em página alguma; os demais tipos permanecem sob o critério estrito.}\n")
	b.WriteString("\\label{tab:provenance}\n")
	b.WriteString("\\begin{tabular}{lrrr}\n\\hline\n")
	b.WriteString("Tipo estrutural & Itens & Completos & TPC \\\\\n\\hline\n")

	for _, kind := range report.Kinds() {
		entry := report.ByKind[kind]
		fmt.Fprintf(&b, "%s & %d & %d & %.3f \\\\\n",
			traduzirTipo(kind), entry.Total, entry.Complete, report.RateFor(kind))
	}

	b.WriteString("\\hline\n")
	fmt.Fprintf(&b, "\\textbf{Total} & \\textbf{%d} & \\textbf{%d} & \\textbf{%.3f} \\\\\n",
		report.Total, report.Complete, report.Rate())
	b.WriteString("\\hline\n\\end{tabular}\n\\end{table}\n")

	return b.String()
}

func traduzirTipo(kind string) string {
	switch kind {
	case "paragraph":
		return "Parágrafo"
	case "page":
		return "Página"
	case "table":
		return "Tabela"
	case "metadata":
		return "Metadado"
	case "chunk":
		return "Trecho"
	case "outline":
		return "Nó de sumário"
	default:
		return kind
	}
}

func printSummary(w *os.File, report provenance.Report, citations provenance.CitationReport) {
	fmt.Fprintf(w, "\nTPC global: %.3f (%d/%d itens)\n", report.Rate(), report.Complete, report.Total)

	kinds := report.Kinds()
	sort.Strings(kinds)
	for _, kind := range kinds {
		entry := report.ByKind[kind]
		fmt.Fprintf(w, "  %-14s %.3f (%d/%d)\n", kind, report.RateFor(kind), entry.Complete, entry.Total)
	}

	// Escopo vem depois do recorte por tipo, nao no lugar dele.
	fmt.Fprintf(w, "\npor escopo de origem:\n")
	for _, scope := range report.Scopes() {
		entry := report.ByScope[scope]
		fmt.Fprintf(w, "  %-14s %.3f (%d/%d)\n", scope, report.RateForScope(scope), entry.Complete, entry.Total)
	}

	if report.Complete < report.Total {
		fmt.Fprintf(w, "\ncausas de incompletude:\n")
		fmt.Fprintf(w, "  document_id ausente   %d\n", report.MissingDocumentID)
		fmt.Fprintf(w, "  page_number ausente   %d\n", report.MissingPageNumber)
		fmt.Fprintf(w, "  sequence_order invalido %d\n", report.InvalidSequenceOrder)
		fmt.Fprintf(w, "  conteudo vazio        %d\n", report.EmptyContent)
	}

	fmt.Fprintf(w, "\nTCV: %.3f (%d/%d citacoes)\n",
		citations.Rate(), citations.Verified, citations.Total)
}
