// Command provenance-eval mede a auditabilidade do indice sobre o corpus C1 e
// emite a tabela do artigo.
//
// Existe porque a Metodologia promete avaliar TRES dimensoes -- precisao,
// auditabilidade e custo operacional -- e os Resultados so entregavam precisao
// e determinismo. As metricas de auditabilidade (TPC e TCV) ja estavam
// implementadas e testadas em internal/provenance, mas nenhum numero delas
// aparecia no texto.
//
// A tabela e reportada POR TIPO ESTRUTURAL, nao so na media global. A razao e
// concreta: `ParagraphNode.PageNumber` e nulavel e blocos de OCR entram sem
// caixa delimitadora, entao um tipo pode ter proveniencia muito pior que os
// outros. A media esconderia isso; a quebra por tipo expoe -- e expor uma
// limitacao medida vale mais que publicar uma media favoravel.
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
		if err := os.WriteFile(*texPath, []byte(tabela), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "provenance-eval: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "provenance-eval: %s\n", *texPath)
	}

	printSummary(os.Stderr, report, citations)
}

// measure indexa o corpus e avalia a proveniencia do resultado.
//
// Usa o indexador REAL (DeterministicIndexer), nao uma simulacao: medir uma
// reimplementacao mediria a reimplementacao. O corpus e texto puro, entao passa
// pelo mesmo caminho de um .txt enviado pelo usuario.
func measure(corpus *evalcorpus.Corpus) (provenance.Report, provenance.CitationReport) {
	indexer := indexing.NewDeterministicIndexer()

	var items []provenance.Item
	var citations []provenance.Citation

	for i, doc := range corpus.Documents {
		documentID := uint(i + 1)

		// Passa pelo caminho publico do indexador, o mesmo de um .txt enviado
		// pelo usuario. Medir uma reimplementacao mediria a reimplementacao.
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

// itemsFromResult converte o indice em itens avaliaveis, um por estrutura.
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
		// Historia desta linha, porque ela explica um erro de medicao: durante
		// um periodo TableSection de fato nao tinha PageNumber, e passar nil
		// aqui era a leitura honesta do esquema. A coluna passou a existir
		// quando as tabelas de PDF viraram fonte do chat, e o nil ficou para
		// tras -- o instrumento seguiu reportando um furo que o sistema ja
		// tinha fechado. Ressalva para quem for ler o artigo: isso NAO muda o
		// numero do corpus C1, que e texto puro e nao produz tabela nenhuma;
		// muda a medicao de qualquer corpus com tabela daqui para frente.
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
		// MetadataPair nao carrega pagina, e aqui isso NAO e furo: os pares que
		// o indexador gera sao estatisticas do documento inteiro
		// ("paragraph_count", "tree_builder", "tree_coverage_ratio"). Elas nao
		// estao na pagina 3 nem em nenhuma outra. Dar pagina a elas seria
		// fabricar proveniencia, que o principio I proibe; conta-las como
		// falha permanente afundaria a media com um defeito inexistente. Por
		// isso entram com escopo de documento: continuam visiveis na conta,
		// com o criterio que cabe a elas.
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
		// Chunk e evidencia citavel: entra na recuperacao da colecao e vira
		// fonte da resposta. Ate 2026-09-19 ele so apontava os paragrafos de
		// origem, e a proveniencia era indireta -- resolvia-se via paragrafo,
		// num passo a mais. A TPC contava isso como incompleto com razao,
		// porque auditoria e apontar a pagina sem intermediario. A coluna
		// passou a existir e a ser preenchida a partir do paragrafo de origem
		// (ver internal/indexing/chunks.go, chunkPageFor).
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

// citationsFromResult monta as citacoes verificaveis.
//
// Cada paragrafo indexado e tratado como um trecho que uma resposta poderia
// citar, e a fonte e o documento original. A TCV mede quantos desses trechos
// REALMENTE existem no texto-fonte -- ou seja, se a fragmentacao preservou o
// conteudo ou o corrompeu.
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
		return "Chunk"
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

	// O recorte por escopo vem DEPOIS do recorte por tipo, e nao no lugar dele.
	// A ordem e deliberada: o leitor ve primeiro o numero sem nenhuma
	// qualificacao, depois a decomposicao por estrutura, e so entao a
	// separacao entre "item que deveria ter pagina" e "item que nunca teria".
	// Publicar so o recorte por escopo seria o caminho facil para um numero
	// bonito; publicar os tres deixa a distincao conferivel.
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
