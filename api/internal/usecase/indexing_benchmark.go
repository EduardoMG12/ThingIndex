package usecase

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"thing-index/internal/domain"
	"thing-index/internal/indexing"
	"thing-index/internal/models"
)

const indexingBenchmarkCorpusPath = "internal/usecase/testdata/indexing-benchmark/corpus.json"

//go:embed testdata/indexing-benchmark/corpus.json
var indexingBenchmarkFixtures embed.FS

var (
	indexingBenchmarkCorpusOnce sync.Once
	indexingBenchmarkCorpusData indexingBenchmarkCorpusFixture
	indexingBenchmarkCorpusErr  error
)

type indexingBenchmarkCorpusFixture struct {
	Version           string                             `json:"version"`
	LastReviewedOn    string                             `json:"last_reviewed_on"`
	Documents         []indexingBenchmarkDocumentFixture `json:"documents"`
	CollectionQueries []indexingBenchmarkCollectionQuery `json:"collection_queries"`
}

type indexingBenchmarkDocumentFixture struct {
	ID                    string                           `json:"id"`
	Title                 string                           `json:"title"`
	MimeType              string                           `json:"mime_type"`
	Focus                 string                           `json:"focus"`
	Content               string                           `json:"content"`
	ExpectedOutlineTitles []string                         `json:"expected_outline_titles"`
	ExpectedTableCount    int                              `json:"expected_table_count"`
	DocumentQueries       []indexingBenchmarkDocumentQuery `json:"document_queries"`
}

type indexingBenchmarkDocumentQuery struct {
	ID                   string `json:"id"`
	Query                string `json:"query"`
	ExpectedTopNodeTitle string `json:"expected_top_node_title"`
	ExpectedTopPage      int    `json:"expected_top_page_number"`
}

type indexingBenchmarkCollectionQuery struct {
	ID                    string `json:"id"`
	Query                 string `json:"query"`
	ExpectedTopDocumentID string `json:"expected_top_document_id"`
}

type indexingBenchmarkProvider struct {
	indexer domain.DocumentIndexer
}

func NewIndexingBenchmarkProvider() domain.IndexingBenchmarkProvider {
	return &indexingBenchmarkProvider{
		indexer: indexing.NewDeterministicIndexer(),
	}
}

func loadIndexingBenchmarkCorpus() (indexingBenchmarkCorpusFixture, error) {
	indexingBenchmarkCorpusOnce.Do(func() {
		raw, err := indexingBenchmarkFixtures.ReadFile("testdata/indexing-benchmark/corpus.json")
		if err != nil {
			indexingBenchmarkCorpusErr = fmt.Errorf("read indexing benchmark corpus: %w", err)
			return
		}

		if err := json.Unmarshal(raw, &indexingBenchmarkCorpusData); err != nil {
			indexingBenchmarkCorpusErr = fmt.Errorf("parse indexing benchmark corpus: %w", err)
			return
		}
	})

	if indexingBenchmarkCorpusErr != nil {
		return indexingBenchmarkCorpusFixture{}, indexingBenchmarkCorpusErr
	}

	return indexingBenchmarkCorpusData, nil
}

func (p *indexingBenchmarkProvider) GetPublicIndexingBenchmark(ctx context.Context) (*domain.IndexingBenchmarkReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	corpus, err := loadIndexingBenchmarkCorpus()
	if err != nil {
		return nil, err
	}

	report := &domain.IndexingBenchmarkReport{
		DatasetVersion:   corpus.Version,
		CorpusSourcePath: indexingBenchmarkCorpusPath,
		LastReviewedOn:   corpus.LastReviewedOn,
		Summary: "Controlled exact-match benchmark for the current relational indexing baseline. The corpus is embedded, " +
			"reproducible, and evaluated with the real deterministic indexer used by the main runtime.",
		Methodology: []string{
			"Run the real deterministic indexer over the embedded benchmark corpus.",
			"Check expected outline titles and table counts for each benchmark document.",
			"Run document-tree retrieval cases and validate exact top-1 node and page matches.",
			"Run collection retrieval cases and validate exact top-1 document matches.",
			"Keep the monitoring heuristic visible as an operational signal, but separate it from benchmark accuracy.",
		},
		PassCriteria: []string{
			"Document structure passes only when all expected outline titles are present and the table count matches the corpus expectation.",
			"Document-tree cases pass only when the expected node and page are both ranked first.",
			"Collection cases pass only when the expected document is ranked first.",
		},
		Limitations: []string{
			"This corpus is intentionally small and controlled; it validates the baseline, not arbitrary production corpora.",
			"The benchmark measures structural indexing and lexical retrieval on the current runtime, not grounded chat answer faithfulness.",
			"Comparative evaluation against vector retrieval and GraphRAG remains a separate research track.",
		},
		HeuristicQualityNote: "The quality score exposed in document monitoring remains a structural heuristic for operators. " +
			"It is not ground truth and should be interpreted together with this benchmark instead of replacing it.",
	}

	type benchmarkDocumentRuntime struct {
		document models.Document
		fixture  indexingBenchmarkDocumentFixture
		result   domain.IndexingResult
	}

	documentRuntimes := make([]benchmarkDocumentRuntime, 0, len(corpus.Documents))
	documentTitleByID := make(map[string]string, len(corpus.Documents))

	structurePasses := 0
	documentCasePasses := 0
	nodeTop1Passes := 0
	pageTop1Passes := 0
	documentNodeTop3Hits := 0
	documentPageTop3Hits := 0
	collectionCasePasses := 0
	collectionTop3Hits := 0
	totalDocumentQueries := 0
	totalCollectionQueries := len(corpus.CollectionQueries)
	totalExactCases := 0
	totalExactPasses := 0
	heuristicTotal := 0.0
	minHeuristic := 0.0
	maxHeuristic := 0.0

	for index, fixture := range corpus.Documents {
		doc := models.Document{
			ID:       uint(index + 1),
			Title:    fixture.Title,
			MimeType: fixture.MimeType,
		}

		result, err := p.indexer.Index(ctx, &doc, strings.NewReader(fixture.Content))
		if err != nil {
			return nil, fmt.Errorf("index benchmark document %s: %w", fixture.ID, err)
		}

		evaluation := buildIndexEvaluation(result.MetadataPairs, result.OutlineNodes)
		heuristicScore := estimateIndexQualityScore(evaluation)
		if index == 0 || heuristicScore < minHeuristic {
			minHeuristic = heuristicScore
		}
		if index == 0 || heuristicScore > maxHeuristic {
			maxHeuristic = heuristicScore
		}
		heuristicTotal += heuristicScore

		structurePassed := benchmarkDocumentStructurePassed(fixture, result)
		if structurePassed {
			structurePasses++
		}

		report.Corpus = append(report.Corpus, domain.IndexingBenchmarkCorpusDocument{
			ID:                    fixture.ID,
			Title:                 fixture.Title,
			MimeType:              fixture.MimeType,
			Focus:                 fixture.Focus,
			QueryCount:            len(fixture.DocumentQueries),
			ExpectedOutlineTitles: append([]string(nil), fixture.ExpectedOutlineTitles...),
			ExpectedTableCount:    fixture.ExpectedTableCount,
			OutlineNodeCount:      len(result.OutlineNodes),
			TableCount:            len(result.TableSections),
			StructurePassed:       structurePassed,
			RetrievalReady:        evaluation != nil && evaluation.RetrievalReady,
			HeuristicQualityScore: heuristicScore,
		})

		documentTitleByID[fixture.ID] = fixture.Title
		documentRuntimes = append(documentRuntimes, benchmarkDocumentRuntime{
			document: doc,
			fixture:  fixture,
			result:   result,
		})

		for _, query := range fixture.DocumentQueries {
			totalDocumentQueries++
			totalExactCases++

			tree := buildTreeRetrieval(query.Query, result.OutlineNodes, result.DocumentPages, result.Chunks)

			nodeMatched := false
			nodeInTop3 := false
			actualNodeTitle := ""
			if tree != nil && len(tree.NodeHits) > 0 {
				actualNodeTitle = tree.NodeHits[0].Title
				nodeMatched = tree.NodeHits[0].Title == query.ExpectedTopNodeTitle
				for _, hit := range tree.NodeHits {
					if hit.Title == query.ExpectedTopNodeTitle {
						nodeInTop3 = true
						break
					}
				}
			}

			expectedPage := query.ExpectedTopPage
			actualPageNumber := 0
			pageMatched := false
			pageInTop3 := false
			if tree != nil && len(tree.PageHits) > 0 {
				actualPageNumber = tree.PageHits[0].PageNumber
				pageMatched = tree.PageHits[0].PageNumber == query.ExpectedTopPage
				for _, hit := range tree.PageHits {
					if hit.PageNumber == query.ExpectedTopPage {
						pageInTop3 = true
						break
					}
				}
			}

			if nodeMatched && pageMatched {
				documentCasePasses++
				totalExactPasses++
			}
			if nodeMatched {
				nodeTop1Passes++
			}
			if pageMatched {
				pageTop1Passes++
			}
			if nodeInTop3 {
				documentNodeTop3Hits++
			}
			if pageInTop3 {
				documentPageTop3Hits++
			}

			report.Cases = append(report.Cases, domain.IndexingBenchmarkCaseResult{
				ID:                 query.ID,
				Scope:              "DOCUMENT_TREE",
				Query:              query.Query,
				DocumentTitle:      fixture.Title,
				NodeMatched:        nodeMatched,
				NodeInTop3:         nodeInTop3,
				ExpectedNodeTitle:  query.ExpectedTopNodeTitle,
				ActualNodeTitle:    actualNodeTitle,
				ExpectedPageNumber: intPtr(expectedPage),
				ActualPageNumber:   optionalInt(actualPageNumber),
				PageMatched:        pageMatched,
				PageInTop3:         pageInTop3,
				Passed:             nodeMatched && pageMatched,
			})
		}
	}

	allDocuments := make([]models.Document, 0, len(documentRuntimes))
	allPages := make([]models.DocumentPage, 0)
	allNodes := make([]models.DocumentOutlineNode, 0)
	allChunks := make([]models.DocumentChunk, 0)
	allTables := make([]models.TableSection, 0)
	for _, runtime := range documentRuntimes {
		allDocuments = append(allDocuments, runtime.document)
		allPages = append(allPages, runtime.result.DocumentPages...)
		allNodes = append(allNodes, runtime.result.OutlineNodes...)
		allChunks = append(allChunks, runtime.result.Chunks...)
		allTables = append(allTables, runtime.result.TableSections...)
	}

	for _, query := range corpus.CollectionQueries {
		totalExactCases++
		result := buildCollectionRetrieval(query.Query, allDocuments, allPages, allNodes, allChunks, allTables)

		expectedDocumentTitle := documentTitleByID[query.ExpectedTopDocumentID]
		actualDocumentTitle := ""
		documentMatched := false
		documentInTop3 := false
		if result != nil && len(result.DocumentHits) > 0 {
			actualDocumentTitle = result.DocumentHits[0].Title
			documentMatched = result.DocumentHits[0].Title == expectedDocumentTitle
			for index, hit := range result.DocumentHits {
				if index >= 3 {
					break
				}
				if hit.Title == expectedDocumentTitle {
					documentInTop3 = true
					break
				}
			}
		}

		if documentMatched {
			collectionCasePasses++
			totalExactPasses++
		}
		if documentInTop3 {
			collectionTop3Hits++
		}

		report.Cases = append(report.Cases, domain.IndexingBenchmarkCaseResult{
			ID:                    query.ID,
			Scope:                 "COLLECTION",
			Query:                 query.Query,
			ExpectedDocumentTitle: expectedDocumentTitle,
			ActualDocumentTitle:   actualDocumentTitle,
			DocumentMatched:       documentMatched,
			DocumentInTop3:        documentInTop3,
			Passed:                documentMatched,
		})
	}

	report.Metrics = domain.IndexingBenchmarkMetrics{
		CorpusDocuments:                len(corpus.Documents),
		DocumentQueries:                totalDocumentQueries,
		CollectionQueries:              totalCollectionQueries,
		StructurePassRate:              percentage(structurePasses, len(corpus.Documents)),
		TreeNodeTop1Accuracy:           percentage(nodeTop1Passes, totalDocumentQueries),
		TreeNodeRecallAt3:              percentage(documentNodeTop3Hits, totalDocumentQueries),
		TreePageTop1Accuracy:           percentage(pageTop1Passes, totalDocumentQueries),
		TreePageRecallAt3:              percentage(documentPageTop3Hits, totalDocumentQueries),
		CollectionDocumentTop1Accuracy: percentage(collectionCasePasses, totalCollectionQueries),
		CollectionDocumentRecallAt3:    percentage(collectionTop3Hits, totalCollectionQueries),
		ExactMatchAccuracy:             percentage(totalExactPasses, totalExactCases),
		AverageHeuristicScore:          safeAverage(heuristicTotal, len(corpus.Documents)),
		MinimumHeuristicScore:          minHeuristic,
		MaximumHeuristicScore:          maxHeuristic,
	}

	return report, nil
}

func benchmarkDocumentStructurePassed(fixture indexingBenchmarkDocumentFixture, result domain.IndexingResult) bool {
	if len(result.TableSections) != fixture.ExpectedTableCount {
		return false
	}

	if len(fixture.ExpectedOutlineTitles) == 0 {
		return true
	}

	available := make(map[string]struct{}, len(result.OutlineNodes))
	for _, node := range result.OutlineNodes {
		available[strings.TrimSpace(node.Title)] = struct{}{}
	}

	for _, title := range fixture.ExpectedOutlineTitles {
		if _, ok := available[strings.TrimSpace(title)]; !ok {
			return false
		}
	}

	return true
}

func percentage(value, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(value) / float64(total) * 100
}

func safeAverage(total float64, count int) float64 {
	if count <= 0 {
		return 0
	}
	return total / float64(count)
}

func optionalInt(value int) *int {
	if value <= 0 {
		return nil
	}
	return intPtr(value)
}

func intPtr(value int) *int {
	return &value
}
