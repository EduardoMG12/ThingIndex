package domain

import "context"

type IndexingBenchmarkCorpusDocument struct {
	ID                    string   `json:"id"`
	Title                 string   `json:"title"`
	MimeType              string   `json:"mime_type"`
	Focus                 string   `json:"focus"`
	QueryCount            int      `json:"query_count"`
	ExpectedOutlineTitles []string `json:"expected_outline_titles"`
	ExpectedTableCount    int      `json:"expected_table_count"`
	OutlineNodeCount      int      `json:"outline_node_count"`
	TableCount            int      `json:"table_count"`
	StructurePassed       bool     `json:"structure_passed"`
	RetrievalReady        bool     `json:"retrieval_ready"`
	HeuristicQualityScore float64  `json:"heuristic_quality_score"`
}

type IndexingBenchmarkMetrics struct {
	CorpusDocuments                int     `json:"corpus_documents"`
	DocumentQueries                int     `json:"document_queries"`
	CollectionQueries              int     `json:"collection_queries"`
	StructurePassRate              float64 `json:"structure_pass_rate"`
	TreeNodeTop1Accuracy           float64 `json:"tree_node_top1_accuracy"`
	TreeNodeRecallAt3              float64 `json:"tree_node_recall_at_3"`
	TreePageTop1Accuracy           float64 `json:"tree_page_top1_accuracy"`
	TreePageRecallAt3              float64 `json:"tree_page_recall_at_3"`
	CollectionDocumentTop1Accuracy float64 `json:"collection_document_top1_accuracy"`
	CollectionDocumentRecallAt3    float64 `json:"collection_document_recall_at_3"`
	ExactMatchAccuracy             float64 `json:"exact_match_accuracy"`
	AverageHeuristicScore          float64 `json:"average_heuristic_score"`
	MinimumHeuristicScore          float64 `json:"minimum_heuristic_score"`
	MaximumHeuristicScore          float64 `json:"maximum_heuristic_score"`
}

type IndexingBenchmarkSnapshot struct {
	ID                   string                   `json:"id"`
	Label                string                   `json:"label"`
	RecordedOn           string                   `json:"recorded_on"`
	Summary              string                   `json:"summary,omitempty"`
	DatasetVersion       string                   `json:"dataset_version"`
	PipelineVersion      string                   `json:"pipeline_version,omitempty"`
	PromptProfileVersion string                   `json:"prompt_profile_version,omitempty"`
	ChunkStrategyVersion string                   `json:"chunk_strategy_version,omitempty"`
	Metrics              IndexingBenchmarkMetrics `json:"metrics"`
}

type IndexingBenchmarkMetricDelta struct {
	Metric        string  `json:"metric"`
	PreviousValue float64 `json:"previous_value"`
	CurrentValue  float64 `json:"current_value"`
	Delta         float64 `json:"delta"`
}

type IndexingBenchmarkComparison struct {
	ID              string                         `json:"id"`
	Title           string                         `json:"title"`
	RecordedOn      string                         `json:"recorded_on"`
	Summary         string                         `json:"summary,omitempty"`
	Before          IndexingBenchmarkSnapshot      `json:"before"`
	After           IndexingBenchmarkSnapshot      `json:"after"`
	KeyMetricDeltas []IndexingBenchmarkMetricDelta `json:"key_metric_deltas"`
}

type IndexingBenchmarkReferenceArtifact struct {
	ID                   string   `json:"id"`
	Label                string   `json:"label"`
	SourcePath           string   `json:"source_path"`
	DocumentName         string   `json:"document_name"`
	TotalNodes           int      `json:"total_nodes"`
	MaxDepth             int      `json:"max_depth"`
	SummaryCoverageRatio float64  `json:"summary_coverage_ratio"`
	InvalidRangeCount    int      `json:"invalid_range_count"`
	AverageTitleLength   float64  `json:"average_title_length"`
	AverageSummaryLength float64  `json:"average_summary_length"`
	Notes                []string `json:"notes,omitempty"`
}

type IndexingBenchmarkReferenceChecklistItem struct {
	ID             string `json:"id"`
	Dimension      string `json:"dimension"`
	Label          string `json:"label"`
	Status         string `json:"status"`
	CurrentValue   string `json:"current_value"`
	ReferenceValue string `json:"reference_value"`
	Summary        string `json:"summary"`
}

type IndexingBenchmarkReferenceComparison struct {
	SourcePath     string                                    `json:"source_path"`
	LastReviewedOn string                                    `json:"last_reviewed_on"`
	Summary        string                                    `json:"summary"`
	Artifacts      []IndexingBenchmarkReferenceArtifact      `json:"artifacts"`
	Checklist      []IndexingBenchmarkReferenceChecklistItem `json:"checklist"`
}

type IndexingBenchmarkCaseResult struct {
	ID                    string `json:"id"`
	Scope                 string `json:"scope"`
	Query                 string `json:"query"`
	DocumentTitle         string `json:"document_title,omitempty"`
	ExpectedDocumentTitle string `json:"expected_document_title,omitempty"`
	ActualDocumentTitle   string `json:"actual_document_title,omitempty"`
	DocumentMatched       bool   `json:"document_matched"`
	DocumentInTop3        bool   `json:"document_in_top_3"`
	ExpectedNodeTitle     string `json:"expected_node_title,omitempty"`
	ActualNodeTitle       string `json:"actual_node_title,omitempty"`
	NodeMatched           bool   `json:"node_matched"`
	NodeInTop3            bool   `json:"node_in_top_3"`
	ExpectedPageNumber    *int   `json:"expected_page_number,omitempty"`
	ActualPageNumber      *int   `json:"actual_page_number,omitempty"`
	PageMatched           bool   `json:"page_matched"`
	PageInTop3            bool   `json:"page_in_top_3"`
	Passed                bool   `json:"passed"`
}

type IndexingBenchmarkReport struct {
	DatasetVersion       string                                `json:"dataset_version"`
	CorpusSourcePath     string                                `json:"corpus_source_path"`
	LastReviewedOn       string                                `json:"last_reviewed_on"`
	Summary              string                                `json:"summary"`
	Methodology          []string                              `json:"methodology"`
	PassCriteria         []string                              `json:"pass_criteria"`
	Limitations          []string                              `json:"limitations"`
	HeuristicQualityNote string                                `json:"heuristic_quality_note"`
	Corpus               []IndexingBenchmarkCorpusDocument     `json:"corpus"`
	Metrics              IndexingBenchmarkMetrics              `json:"metrics"`
	Cases                []IndexingBenchmarkCaseResult         `json:"cases"`
	CurrentSnapshot      *IndexingBenchmarkSnapshot            `json:"current_snapshot,omitempty"`
	LatestComparison     *IndexingBenchmarkComparison          `json:"latest_comparison,omitempty"`
	ComparisonHistory    []IndexingBenchmarkComparison         `json:"comparison_history,omitempty"`
	ReferenceComparison  *IndexingBenchmarkReferenceComparison `json:"reference_comparison,omitempty"`
}

type IndexingBenchmarkProvider interface {
	GetPublicIndexingBenchmark(ctx context.Context) (*IndexingBenchmarkReport, error)
}
