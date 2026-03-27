package usecase

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"math"
	"path"
	"strings"
	"sync"

	"thing-index/internal/domain"
	"thing-index/internal/indexing"
	"thing-index/internal/models"
	"thing-index/internal/processing"
)

const indexingBenchmarkCorpusPath = "internal/usecase/testdata/indexing-benchmark/corpus.json"
const indexingBenchmarkHistoryPath = "internal/usecase/testdata/indexing-benchmark/history.json"
const indexingBenchmarkReferencePath = "internal/usecase/testdata/indexing-benchmark/reference-workspace.json"
const indexingBenchmarkCorpusFixtureDir = "testdata/indexing-benchmark"

//go:embed testdata/indexing-benchmark/*
var indexingBenchmarkFixtures embed.FS

var (
	indexingBenchmarkCorpusOnce sync.Once
	indexingBenchmarkCorpusData indexingBenchmarkCorpusFixture
	indexingBenchmarkCorpusErr  error

	indexingBenchmarkHistoryOnce sync.Once
	indexingBenchmarkHistoryData indexingBenchmarkHistoryFixture
	indexingBenchmarkHistoryErr  error

	indexingBenchmarkReferenceOnce sync.Once
	indexingBenchmarkReferenceData indexingBenchmarkReferenceFixture
	indexingBenchmarkReferenceErr  error
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
	ContentFile           string                           `json:"content_file,omitempty"`
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

type indexingBenchmarkHistoryFixture struct {
	Snapshots []domain.IndexingBenchmarkSnapshot `json:"snapshots"`
}

type indexingBenchmarkReferenceFixture struct {
	SourcePath     string                                      `json:"source_path"`
	LastReviewedOn string                                      `json:"last_reviewed_on"`
	Summary        string                                      `json:"summary"`
	Artifacts      []indexingBenchmarkReferenceArtifactFixture `json:"artifacts"`
}

type indexingBenchmarkReferenceArtifactFixture struct {
	ID                   string   `json:"id"`
	Label                string   `json:"label"`
	SourcePath           string   `json:"source_path"`
	DocumentName         string   `json:"document_name"`
	TotalNodes           int      `json:"total_nodes"`
	MaxDepth             int      `json:"max_depth"`
	NestedNodeCount      int      `json:"nested_node_count"`
	SummarizedNodeCount  int      `json:"summarized_node_count"`
	ConciseTitleCount    int      `json:"concise_title_count"`
	InvalidRangeCount    int      `json:"invalid_range_count"`
	AverageTitleLength   float64  `json:"average_title_length"`
	AverageSummaryLength float64  `json:"average_summary_length"`
	Notes                []string `json:"notes"`
}

type indexingBenchmarkStructureStats struct {
	DocumentCount       int
	TotalNodes          int
	MaxDepth            int
	SumMaxDepthPerDoc   int
	NestedNodeCount     int
	SummarizedNodeCount int
	ConciseTitleCount   int
	InvalidRangeCount   int
	TotalTitleLength    int
	TotalSummaryLength  int
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

func loadIndexingBenchmarkDocumentData(fixture indexingBenchmarkDocumentFixture) ([]byte, error) {
	if contentFile := strings.TrimSpace(fixture.ContentFile); contentFile != "" {
		cleanPath := path.Clean(contentFile)
		if cleanPath == "." || strings.HasPrefix(cleanPath, "../") {
			return nil, fmt.Errorf("invalid benchmark content_file %q", contentFile)
		}

		data, err := indexingBenchmarkFixtures.ReadFile(path.Join(indexingBenchmarkCorpusFixtureDir, cleanPath))
		if err != nil {
			return nil, fmt.Errorf("read benchmark fixture %s: %w", contentFile, err)
		}
		return data, nil
	}

	return []byte(fixture.Content), nil
}

func loadIndexingBenchmarkHistory() (indexingBenchmarkHistoryFixture, error) {
	indexingBenchmarkHistoryOnce.Do(func() {
		raw, err := indexingBenchmarkFixtures.ReadFile("testdata/indexing-benchmark/history.json")
		if err != nil {
			indexingBenchmarkHistoryErr = fmt.Errorf("read indexing benchmark history: %w", err)
			return
		}

		if err := json.Unmarshal(raw, &indexingBenchmarkHistoryData); err != nil {
			indexingBenchmarkHistoryErr = fmt.Errorf("parse indexing benchmark history: %w", err)
			return
		}
	})

	if indexingBenchmarkHistoryErr != nil {
		return indexingBenchmarkHistoryFixture{}, indexingBenchmarkHistoryErr
	}

	return indexingBenchmarkHistoryData, nil
}

func loadIndexingBenchmarkReferenceFixture() (indexingBenchmarkReferenceFixture, error) {
	indexingBenchmarkReferenceOnce.Do(func() {
		raw, err := indexingBenchmarkFixtures.ReadFile("testdata/indexing-benchmark/reference-workspace.json")
		if err != nil {
			indexingBenchmarkReferenceErr = fmt.Errorf("read indexing benchmark reference comparison: %w", err)
			return
		}

		if err := json.Unmarshal(raw, &indexingBenchmarkReferenceData); err != nil {
			indexingBenchmarkReferenceErr = fmt.Errorf("parse indexing benchmark reference comparison: %w", err)
			return
		}
	})

	if indexingBenchmarkReferenceErr != nil {
		return indexingBenchmarkReferenceFixture{}, indexingBenchmarkReferenceErr
	}

	return indexingBenchmarkReferenceData, nil
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
			"Compare the current structural profile against the normalized pageindex workspace references on depth, coherence, coverage, and legibility.",
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
			"The workspace reference comparison uses normalized structure exports from pageindex/results as a quality bar, not as perfect ground truth.",
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
	currentStructureStats := indexingBenchmarkStructureStats{}
	pipelineSignature := processing.CurrentPipelineSignature()
	currentSnapshot := domain.IndexingBenchmarkSnapshot{
		ID:                   fmt.Sprintf("current-%s", corpus.Version),
		Label:                "Current runtime",
		RecordedOn:           corpus.LastReviewedOn,
		Summary:              "Current benchmark output for the embedded relational indexing corpus.",
		DatasetVersion:       corpus.Version,
		PipelineVersion:      pipelineSignature.PipelineVersion,
		PromptProfileVersion: pipelineSignature.PromptProfileVersion,
		ChunkStrategyVersion: pipelineSignature.ChunkStrategyVersion,
	}

	for index, fixture := range corpus.Documents {
		doc := models.Document{
			ID:       uint(index + 1),
			Title:    fixture.Title,
			MimeType: fixture.MimeType,
		}

		contentData, err := loadIndexingBenchmarkDocumentData(fixture)
		if err != nil {
			return nil, fmt.Errorf("load benchmark document %s: %w", fixture.ID, err)
		}

		result, err := p.indexer.Index(ctx, &doc, bytes.NewReader(contentData))
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
		accumulateBenchmarkStructureStats(&currentStructureStats, result)

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

	currentSnapshot.Metrics = report.Metrics
	report.CurrentSnapshot = &currentSnapshot

	history, err := loadIndexingBenchmarkHistory()
	if err != nil {
		return nil, err
	}

	report.ComparisonHistory = buildIndexingBenchmarkComparisonHistory(history.Snapshots)
	if matchedSnapshot := findMatchingBenchmarkSnapshot(currentSnapshot, history.Snapshots); matchedSnapshot != nil {
		report.CurrentSnapshot = matchedSnapshot
		if len(report.ComparisonHistory) > 0 {
			latest := report.ComparisonHistory[len(report.ComparisonHistory)-1]
			if sameBenchmarkSnapshot(latest.After, *matchedSnapshot) {
				report.LatestComparison = &latest
			}
		}
	}
	if report.LatestComparison == nil && len(history.Snapshots) > 0 {
		latestRecordedSnapshot := history.Snapshots[len(history.Snapshots)-1]
		latest := buildIndexingBenchmarkComparison(
			latestRecordedSnapshot,
			*report.CurrentSnapshot,
			"Current runtime vs last recorded snapshot",
			report.CurrentSnapshot.RecordedOn,
			"Unrecorded benchmark delta against the latest versioned snapshot in the workspace.",
		)
		report.LatestComparison = &latest
	}

	referenceFixture, err := loadIndexingBenchmarkReferenceFixture()
	if err != nil {
		return nil, err
	}
	report.ReferenceComparison = buildIndexingBenchmarkReferenceComparison(referenceFixture, currentStructureStats, report.Metrics)

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

func buildIndexingBenchmarkComparisonHistory(snapshots []domain.IndexingBenchmarkSnapshot) []domain.IndexingBenchmarkComparison {
	if len(snapshots) < 2 {
		return nil
	}

	comparisons := make([]domain.IndexingBenchmarkComparison, 0, len(snapshots)-1)
	for index := 1; index < len(snapshots); index++ {
		before := snapshots[index-1]
		after := snapshots[index]
		comparisons = append(comparisons, buildIndexingBenchmarkComparison(
			before,
			after,
			fmt.Sprintf("%s -> %s", before.Label, after.Label),
			after.RecordedOn,
			strings.TrimSpace(after.Summary),
		))
	}
	return comparisons
}

func buildIndexingBenchmarkComparison(
	before domain.IndexingBenchmarkSnapshot,
	after domain.IndexingBenchmarkSnapshot,
	title string,
	recordedOn string,
	summary string,
) domain.IndexingBenchmarkComparison {
	return domain.IndexingBenchmarkComparison{
		ID:              fmt.Sprintf("%s__to__%s", before.ID, after.ID),
		Title:           strings.TrimSpace(title),
		RecordedOn:      strings.TrimSpace(recordedOn),
		Summary:         strings.TrimSpace(summary),
		Before:          before,
		After:           after,
		KeyMetricDeltas: buildIndexingBenchmarkMetricDeltas(before.Metrics, after.Metrics),
	}
}

func buildIndexingBenchmarkMetricDeltas(
	before domain.IndexingBenchmarkMetrics,
	after domain.IndexingBenchmarkMetrics,
) []domain.IndexingBenchmarkMetricDelta {
	keyMetrics := []struct {
		name    string
		valueOf func(domain.IndexingBenchmarkMetrics) float64
	}{
		{name: "structure_pass_rate", valueOf: func(metrics domain.IndexingBenchmarkMetrics) float64 { return metrics.StructurePassRate }},
		{name: "tree_node_top1_accuracy", valueOf: func(metrics domain.IndexingBenchmarkMetrics) float64 { return metrics.TreeNodeTop1Accuracy }},
		{name: "tree_page_top1_accuracy", valueOf: func(metrics domain.IndexingBenchmarkMetrics) float64 { return metrics.TreePageTop1Accuracy }},
		{name: "collection_document_top1_accuracy", valueOf: func(metrics domain.IndexingBenchmarkMetrics) float64 { return metrics.CollectionDocumentTop1Accuracy }},
		{name: "exact_match_accuracy", valueOf: func(metrics domain.IndexingBenchmarkMetrics) float64 { return metrics.ExactMatchAccuracy }},
		{name: "average_heuristic_score", valueOf: func(metrics domain.IndexingBenchmarkMetrics) float64 { return metrics.AverageHeuristicScore }},
	}

	deltas := make([]domain.IndexingBenchmarkMetricDelta, 0, len(keyMetrics))
	for _, item := range keyMetrics {
		previousValue := item.valueOf(before)
		currentValue := item.valueOf(after)
		deltas = append(deltas, domain.IndexingBenchmarkMetricDelta{
			Metric:        item.name,
			PreviousValue: previousValue,
			CurrentValue:  currentValue,
			Delta:         currentValue - previousValue,
		})
	}
	return deltas
}

func findMatchingBenchmarkSnapshot(
	current domain.IndexingBenchmarkSnapshot,
	snapshots []domain.IndexingBenchmarkSnapshot,
) *domain.IndexingBenchmarkSnapshot {
	for index := range snapshots {
		if sameBenchmarkSnapshot(current, snapshots[index]) {
			return &snapshots[index]
		}
	}
	return nil
}

func sameBenchmarkSnapshot(a, b domain.IndexingBenchmarkSnapshot) bool {
	return a.DatasetVersion == b.DatasetVersion &&
		a.PipelineVersion == b.PipelineVersion &&
		a.PromptProfileVersion == b.PromptProfileVersion &&
		a.ChunkStrategyVersion == b.ChunkStrategyVersion &&
		sameBenchmarkMetrics(a.Metrics, b.Metrics)
}

func sameBenchmarkMetrics(a, b domain.IndexingBenchmarkMetrics) bool {
	return a.CorpusDocuments == b.CorpusDocuments &&
		a.DocumentQueries == b.DocumentQueries &&
		a.CollectionQueries == b.CollectionQueries &&
		nearlyEqualFloat(a.StructurePassRate, b.StructurePassRate) &&
		nearlyEqualFloat(a.TreeNodeTop1Accuracy, b.TreeNodeTop1Accuracy) &&
		nearlyEqualFloat(a.TreeNodeRecallAt3, b.TreeNodeRecallAt3) &&
		nearlyEqualFloat(a.TreePageTop1Accuracy, b.TreePageTop1Accuracy) &&
		nearlyEqualFloat(a.TreePageRecallAt3, b.TreePageRecallAt3) &&
		nearlyEqualFloat(a.CollectionDocumentTop1Accuracy, b.CollectionDocumentTop1Accuracy) &&
		nearlyEqualFloat(a.CollectionDocumentRecallAt3, b.CollectionDocumentRecallAt3) &&
		nearlyEqualFloat(a.ExactMatchAccuracy, b.ExactMatchAccuracy) &&
		nearlyEqualFloat(a.AverageHeuristicScore, b.AverageHeuristicScore) &&
		nearlyEqualFloat(a.MinimumHeuristicScore, b.MinimumHeuristicScore) &&
		nearlyEqualFloat(a.MaximumHeuristicScore, b.MaximumHeuristicScore)
}

func nearlyEqualFloat(a, b float64) bool {
	return math.Abs(a-b) <= 0.0001
}

func accumulateBenchmarkStructureStats(stats *indexingBenchmarkStructureStats, result domain.IndexingResult) {
	if stats == nil {
		return
	}

	stats.DocumentCount++
	docMaxDepth := 0
	for _, node := range result.OutlineNodes {
		stats.TotalNodes++
		if node.Level > stats.MaxDepth {
			stats.MaxDepth = node.Level
		}
		if node.Level > docMaxDepth {
			docMaxDepth = node.Level
		}
		if node.Level > 1 {
			stats.NestedNodeCount++
		}
		if strings.TrimSpace(node.Summary) != "" {
			stats.SummarizedNodeCount++
			stats.TotalSummaryLength += len(strings.TrimSpace(node.Summary))
		}
		titleLen := len(strings.TrimSpace(node.Title))
		stats.TotalTitleLength += titleLen
		if titleLen >= 4 && titleLen <= 80 {
			stats.ConciseTitleCount++
		}
		if node.StartPage != nil && node.EndPage != nil && *node.EndPage < *node.StartPage {
			stats.InvalidRangeCount++
		}
	}
	stats.SumMaxDepthPerDoc += docMaxDepth
}

func buildIndexingBenchmarkReferenceComparison(
	fixture indexingBenchmarkReferenceFixture,
	currentStats indexingBenchmarkStructureStats,
	metrics domain.IndexingBenchmarkMetrics,
) *domain.IndexingBenchmarkReferenceComparison {
	referenceArtifacts := make([]domain.IndexingBenchmarkReferenceArtifact, 0, len(fixture.Artifacts))
	referenceStats := indexingBenchmarkStructureStats{}
	for _, artifact := range fixture.Artifacts {
		referenceArtifacts = append(referenceArtifacts, domain.IndexingBenchmarkReferenceArtifact{
			ID:                   artifact.ID,
			Label:                artifact.Label,
			SourcePath:           artifact.SourcePath,
			DocumentName:         artifact.DocumentName,
			TotalNodes:           artifact.TotalNodes,
			MaxDepth:             artifact.MaxDepth,
			SummaryCoverageRatio: ratio(artifact.SummarizedNodeCount, artifact.TotalNodes),
			InvalidRangeCount:    artifact.InvalidRangeCount,
			AverageTitleLength:   artifact.AverageTitleLength,
			AverageSummaryLength: artifact.AverageSummaryLength,
			Notes:                append([]string(nil), artifact.Notes...),
		})

		referenceStats.DocumentCount++
		referenceStats.TotalNodes += artifact.TotalNodes
		if artifact.MaxDepth > referenceStats.MaxDepth {
			referenceStats.MaxDepth = artifact.MaxDepth
		}
		referenceStats.SumMaxDepthPerDoc += artifact.MaxDepth
		referenceStats.NestedNodeCount += artifact.NestedNodeCount
		referenceStats.SummarizedNodeCount += artifact.SummarizedNodeCount
		referenceStats.ConciseTitleCount += artifact.ConciseTitleCount
		referenceStats.InvalidRangeCount += artifact.InvalidRangeCount
		referenceStats.TotalTitleLength += int(math.Round(artifact.AverageTitleLength * float64(artifact.TotalNodes)))
		referenceStats.TotalSummaryLength += int(math.Round(artifact.AverageSummaryLength * float64(max(artifact.SummarizedNodeCount, 1))))
	}

	currentDepthValue := fmt.Sprintf("max_depth=%d; nested_nodes=%s", currentStats.MaxDepth, formatPercent(currentStats.nestedNodeRatio()))
	referenceDepthValue := fmt.Sprintf("max_depth=%d; nested_nodes=%s", referenceStats.MaxDepth, formatPercent(referenceStats.nestedNodeRatio()))
	depthStatus := "fail"
	depthSummary := "The public ThingIndex corpus remains structurally flatter than the workspace references and still lacks nested outline depth."
	if currentStats.MaxDepth >= referenceStats.MaxDepth-1 && currentStats.nestedNodeRatio() >= referenceStats.nestedNodeRatio()*0.5 {
		depthStatus = "pass"
		depthSummary = "The public ThingIndex corpus now reaches a useful nested depth close to the workspace references."
	} else if currentStats.MaxDepth >= 2 || currentStats.NestedNodeCount > 0 {
		depthStatus = "watch"
		depthSummary = "ThingIndex now shows some useful hierarchy, but the public corpus is still shallower than the workspace references."
	}

	currentCoherenceValue := fmt.Sprintf("invalid_ranges=%s; summary_gaps=%s", formatPercent(currentStats.invalidRangeRatio()), formatPercent(1-currentStats.summaryCoverageRatio()))
	referenceCoherenceValue := fmt.Sprintf("invalid_ranges=%s; summary_gaps=%s", formatPercent(referenceStats.invalidRangeRatio()), formatPercent(1-referenceStats.summaryCoverageRatio()))
	coherenceStatus := "pass"
	coherenceSummary := "ThingIndex is at least as coherent as the workspace references on span validity and summary presence."
	if currentStats.invalidRangeRatio() > referenceStats.invalidRangeRatio()*2 {
		coherenceStatus = "fail"
		coherenceSummary = "ThingIndex still introduces more invalid structural spans than the workspace references."
	} else if currentStats.invalidRangeRatio() > referenceStats.invalidRangeRatio() || currentStats.summaryCoverageRatio() < referenceStats.summaryCoverageRatio()-0.05 {
		coherenceStatus = "watch"
		coherenceSummary = "ThingIndex is mostly coherent, but still trails the workspace references on span or summary consistency."
	}

	currentCoverageValue := fmt.Sprintf("structure_pass=%s; summary_coverage=%s", formatPercent(metrics.StructurePassRate/100), formatPercent(currentStats.summaryCoverageRatio()))
	referenceCoverageValue := fmt.Sprintf("summary_coverage=%s; reference_docs=%d", formatPercent(referenceStats.summaryCoverageRatio()), referenceStats.DocumentCount)
	coverageStatus := "pass"
	coverageSummary := "ThingIndex currently covers the benchmark corpus at or above the summary coverage observed in the workspace references."
	if metrics.StructurePassRate < 60 || currentStats.summaryCoverageRatio() < referenceStats.summaryCoverageRatio()-0.1 {
		coverageStatus = "fail"
		coverageSummary = "ThingIndex still misses too much structural coverage relative to the workspace reference bar."
	} else if metrics.StructurePassRate < 80 || currentStats.summaryCoverageRatio() < referenceStats.summaryCoverageRatio() {
		coverageStatus = "watch"
		coverageSummary = "ThingIndex covers most of the benchmark corpus, but not yet with the same consistency as the workspace references."
	}

	currentLegibilityValue := fmt.Sprintf("concise_titles=%s; avg_summary=%s chars", formatPercent(currentStats.conciseTitleRatio()), formatFloat(currentStats.averageSummaryLength()))
	referenceLegibilityValue := fmt.Sprintf("concise_titles=%s; avg_summary=%s chars", formatPercent(referenceStats.conciseTitleRatio()), formatFloat(referenceStats.averageSummaryLength()))
	legibilityStatus := "pass"
	legibilitySummary := "ThingIndex keeps concise titles and much shorter section summaries than the workspace references while preserving coverage."
	if currentStats.conciseTitleRatio() < 0.9 {
		legibilityStatus = "fail"
		legibilitySummary = "ThingIndex still produces too many awkward titles to match the legibility of the workspace references."
	} else if currentStats.conciseTitleRatio() < referenceStats.conciseTitleRatio() || currentStats.averageSummaryLength() > referenceStats.averageSummaryLength() {
		legibilityStatus = "watch"
		legibilitySummary = "ThingIndex is readable, but still not as consistently legible as the workspace references."
	}

	checklist := []domain.IndexingBenchmarkReferenceChecklistItem{
		{
			ID:             "depth",
			Dimension:      "depth",
			Label:          "Useful structural depth",
			Status:         depthStatus,
			CurrentValue:   currentDepthValue,
			ReferenceValue: referenceDepthValue,
			Summary:        depthSummary,
		},
		{
			ID:             "coherence",
			Dimension:      "coherence",
			Label:          "Structural coherence",
			Status:         coherenceStatus,
			CurrentValue:   currentCoherenceValue,
			ReferenceValue: referenceCoherenceValue,
			Summary:        coherenceSummary,
		},
		{
			ID:             "coverage",
			Dimension:      "coverage",
			Label:          "Section coverage",
			Status:         coverageStatus,
			CurrentValue:   currentCoverageValue,
			ReferenceValue: referenceCoverageValue,
			Summary:        coverageSummary,
		},
		{
			ID:             "legibility",
			Dimension:      "legibility",
			Label:          "Outline legibility",
			Status:         legibilityStatus,
			CurrentValue:   currentLegibilityValue,
			ReferenceValue: referenceLegibilityValue,
			Summary:        legibilitySummary,
		},
	}

	return &domain.IndexingBenchmarkReferenceComparison{
		SourcePath:     fixture.SourcePath,
		LastReviewedOn: fixture.LastReviewedOn,
		Summary:        summarizeReferenceChecklist(checklist),
		Artifacts:      referenceArtifacts,
		Checklist:      checklist,
	}
}

func summarizeReferenceChecklist(checklist []domain.IndexingBenchmarkReferenceChecklistItem) string {
	passing := make([]string, 0, len(checklist))
	watching := make([]string, 0, len(checklist))
	failing := make([]string, 0, len(checklist))
	for _, item := range checklist {
		switch item.Status {
		case "pass":
			passing = append(passing, item.Label)
		case "watch":
			watching = append(watching, item.Label)
		case "fail":
			failing = append(failing, item.Label)
		}
	}

	switch {
	case len(failing) == 0 && len(watching) == 0:
		return "ThingIndex currently matches or exceeds the workspace reference artifacts on all checklist dimensions."
	case len(failing) == 0:
		return fmt.Sprintf(
			"ThingIndex is close to the workspace reference artifacts overall, with watch-level gaps remaining in %s.",
			joinBenchmarkChecklistLabels(watching),
		)
	default:
		summary := fmt.Sprintf(
			"ThingIndex still trails the workspace reference artifacts on %s.",
			joinBenchmarkChecklistLabels(failing),
		)
		if len(watching) > 0 {
			summary += fmt.Sprintf(" Watch-level gaps remain in %s.", joinBenchmarkChecklistLabels(watching))
		}
		if len(passing) > 0 {
			summary += fmt.Sprintf(" It is already competitive on %s.", joinBenchmarkChecklistLabels(passing))
		}
		return summary
	}
}

func joinBenchmarkChecklistLabels(labels []string) string {
	switch len(labels) {
	case 0:
		return "the checklist"
	case 1:
		return labels[0]
	case 2:
		return labels[0] + " and " + labels[1]
	default:
		return strings.Join(labels[:len(labels)-1], ", ") + ", and " + labels[len(labels)-1]
	}
}

func (stats indexingBenchmarkStructureStats) summaryCoverageRatio() float64 {
	return ratio(stats.SummarizedNodeCount, stats.TotalNodes)
}

func (stats indexingBenchmarkStructureStats) conciseTitleRatio() float64 {
	return ratio(stats.ConciseTitleCount, stats.TotalNodes)
}

func (stats indexingBenchmarkStructureStats) nestedNodeRatio() float64 {
	return ratio(stats.NestedNodeCount, stats.TotalNodes)
}

func (stats indexingBenchmarkStructureStats) invalidRangeRatio() float64 {
	return ratio(stats.InvalidRangeCount, stats.TotalNodes)
}

func (stats indexingBenchmarkStructureStats) averageSummaryLength() float64 {
	if stats.SummarizedNodeCount <= 0 {
		return 0
	}
	return float64(stats.TotalSummaryLength) / float64(stats.SummarizedNodeCount)
}

func ratio(value, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(value) / float64(total)
}

func formatPercent(value float64) string {
	return fmt.Sprintf("%.1f%%", value*100)
}

func formatFloat(value float64) string {
	return fmt.Sprintf("%.1f", value)
}
