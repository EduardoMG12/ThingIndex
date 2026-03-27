package dto

import "thing-index/internal/domain"

type IndexingBenchmarkCorpusDocumentResponse struct {
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

type IndexingBenchmarkMetricsResponse struct {
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

type IndexingBenchmarkSnapshotResponse struct {
	ID                   string                           `json:"id"`
	Label                string                           `json:"label"`
	RecordedOn           string                           `json:"recorded_on"`
	Summary              string                           `json:"summary,omitempty"`
	DatasetVersion       string                           `json:"dataset_version"`
	PipelineVersion      string                           `json:"pipeline_version,omitempty"`
	PromptProfileVersion string                           `json:"prompt_profile_version,omitempty"`
	ChunkStrategyVersion string                           `json:"chunk_strategy_version,omitempty"`
	Metrics              IndexingBenchmarkMetricsResponse `json:"metrics"`
}

type IndexingBenchmarkMetricDeltaResponse struct {
	Metric        string  `json:"metric"`
	PreviousValue float64 `json:"previous_value"`
	CurrentValue  float64 `json:"current_value"`
	Delta         float64 `json:"delta"`
}

type IndexingBenchmarkComparisonResponse struct {
	ID              string                                 `json:"id"`
	Title           string                                 `json:"title"`
	RecordedOn      string                                 `json:"recorded_on"`
	Summary         string                                 `json:"summary,omitempty"`
	Before          IndexingBenchmarkSnapshotResponse      `json:"before"`
	After           IndexingBenchmarkSnapshotResponse      `json:"after"`
	KeyMetricDeltas []IndexingBenchmarkMetricDeltaResponse `json:"key_metric_deltas"`
}

type IndexingBenchmarkReferenceArtifactResponse struct {
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

type IndexingBenchmarkReferenceChecklistItemResponse struct {
	ID             string `json:"id"`
	Dimension      string `json:"dimension"`
	Label          string `json:"label"`
	Status         string `json:"status"`
	CurrentValue   string `json:"current_value"`
	ReferenceValue string `json:"reference_value"`
	Summary        string `json:"summary"`
}

type IndexingBenchmarkReferenceComparisonResponse struct {
	SourcePath     string                                            `json:"source_path"`
	LastReviewedOn string                                            `json:"last_reviewed_on"`
	Summary        string                                            `json:"summary"`
	Artifacts      []IndexingBenchmarkReferenceArtifactResponse      `json:"artifacts"`
	Checklist      []IndexingBenchmarkReferenceChecklistItemResponse `json:"checklist"`
}

type IndexingBenchmarkCaseResultResponse struct {
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

type IndexingBenchmarkReportResponse struct {
	DatasetVersion       string                                        `json:"dataset_version"`
	CorpusSourcePath     string                                        `json:"corpus_source_path"`
	LastReviewedOn       string                                        `json:"last_reviewed_on"`
	Summary              string                                        `json:"summary"`
	Methodology          []string                                      `json:"methodology"`
	PassCriteria         []string                                      `json:"pass_criteria"`
	Limitations          []string                                      `json:"limitations"`
	HeuristicQualityNote string                                        `json:"heuristic_quality_note"`
	Corpus               []IndexingBenchmarkCorpusDocumentResponse     `json:"corpus"`
	Metrics              IndexingBenchmarkMetricsResponse              `json:"metrics"`
	Cases                []IndexingBenchmarkCaseResultResponse         `json:"cases"`
	CurrentSnapshot      *IndexingBenchmarkSnapshotResponse            `json:"current_snapshot,omitempty"`
	LatestComparison     *IndexingBenchmarkComparisonResponse          `json:"latest_comparison,omitempty"`
	ComparisonHistory    []IndexingBenchmarkComparisonResponse         `json:"comparison_history,omitempty"`
	ReferenceComparison  *IndexingBenchmarkReferenceComparisonResponse `json:"reference_comparison,omitempty"`
}

func ToIndexingBenchmarkReportResponse(report *domain.IndexingBenchmarkReport) *IndexingBenchmarkReportResponse {
	if report == nil {
		return nil
	}

	corpus := make([]IndexingBenchmarkCorpusDocumentResponse, 0, len(report.Corpus))
	for _, item := range report.Corpus {
		corpus = append(corpus, IndexingBenchmarkCorpusDocumentResponse{
			ID:                    item.ID,
			Title:                 item.Title,
			MimeType:              item.MimeType,
			Focus:                 item.Focus,
			QueryCount:            item.QueryCount,
			ExpectedOutlineTitles: append([]string(nil), item.ExpectedOutlineTitles...),
			ExpectedTableCount:    item.ExpectedTableCount,
			OutlineNodeCount:      item.OutlineNodeCount,
			TableCount:            item.TableCount,
			StructurePassed:       item.StructurePassed,
			RetrievalReady:        item.RetrievalReady,
			HeuristicQualityScore: item.HeuristicQualityScore,
		})
	}

	cases := make([]IndexingBenchmarkCaseResultResponse, 0, len(report.Cases))
	for _, item := range report.Cases {
		cases = append(cases, IndexingBenchmarkCaseResultResponse{
			ID:                    item.ID,
			Scope:                 item.Scope,
			Query:                 item.Query,
			DocumentTitle:         item.DocumentTitle,
			ExpectedDocumentTitle: item.ExpectedDocumentTitle,
			ActualDocumentTitle:   item.ActualDocumentTitle,
			DocumentMatched:       item.DocumentMatched,
			DocumentInTop3:        item.DocumentInTop3,
			ExpectedNodeTitle:     item.ExpectedNodeTitle,
			ActualNodeTitle:       item.ActualNodeTitle,
			NodeMatched:           item.NodeMatched,
			NodeInTop3:            item.NodeInTop3,
			ExpectedPageNumber:    item.ExpectedPageNumber,
			ActualPageNumber:      item.ActualPageNumber,
			PageMatched:           item.PageMatched,
			PageInTop3:            item.PageInTop3,
			Passed:                item.Passed,
		})
	}

	comparisonHistory := make([]IndexingBenchmarkComparisonResponse, 0, len(report.ComparisonHistory))
	for _, item := range report.ComparisonHistory {
		comparisonHistory = append(comparisonHistory, toIndexingBenchmarkComparisonResponse(item))
	}

	return &IndexingBenchmarkReportResponse{
		DatasetVersion:       report.DatasetVersion,
		CorpusSourcePath:     report.CorpusSourcePath,
		LastReviewedOn:       report.LastReviewedOn,
		Summary:              report.Summary,
		Methodology:          append([]string(nil), report.Methodology...),
		PassCriteria:         append([]string(nil), report.PassCriteria...),
		Limitations:          append([]string(nil), report.Limitations...),
		HeuristicQualityNote: report.HeuristicQualityNote,
		Corpus:               corpus,
		Metrics: IndexingBenchmarkMetricsResponse{
			CorpusDocuments:                report.Metrics.CorpusDocuments,
			DocumentQueries:                report.Metrics.DocumentQueries,
			CollectionQueries:              report.Metrics.CollectionQueries,
			StructurePassRate:              report.Metrics.StructurePassRate,
			TreeNodeTop1Accuracy:           report.Metrics.TreeNodeTop1Accuracy,
			TreeNodeRecallAt3:              report.Metrics.TreeNodeRecallAt3,
			TreePageTop1Accuracy:           report.Metrics.TreePageTop1Accuracy,
			TreePageRecallAt3:              report.Metrics.TreePageRecallAt3,
			CollectionDocumentTop1Accuracy: report.Metrics.CollectionDocumentTop1Accuracy,
			CollectionDocumentRecallAt3:    report.Metrics.CollectionDocumentRecallAt3,
			ExactMatchAccuracy:             report.Metrics.ExactMatchAccuracy,
			AverageHeuristicScore:          report.Metrics.AverageHeuristicScore,
			MinimumHeuristicScore:          report.Metrics.MinimumHeuristicScore,
			MaximumHeuristicScore:          report.Metrics.MaximumHeuristicScore,
		},
		Cases:               cases,
		CurrentSnapshot:     toIndexingBenchmarkSnapshotResponse(report.CurrentSnapshot),
		LatestComparison:    toIndexingBenchmarkComparisonResponsePtr(report.LatestComparison),
		ComparisonHistory:   comparisonHistory,
		ReferenceComparison: toIndexingBenchmarkReferenceComparisonResponse(report.ReferenceComparison),
	}
}

func toIndexingBenchmarkMetricsResponse(metrics domain.IndexingBenchmarkMetrics) IndexingBenchmarkMetricsResponse {
	return IndexingBenchmarkMetricsResponse{
		CorpusDocuments:                metrics.CorpusDocuments,
		DocumentQueries:                metrics.DocumentQueries,
		CollectionQueries:              metrics.CollectionQueries,
		StructurePassRate:              metrics.StructurePassRate,
		TreeNodeTop1Accuracy:           metrics.TreeNodeTop1Accuracy,
		TreeNodeRecallAt3:              metrics.TreeNodeRecallAt3,
		TreePageTop1Accuracy:           metrics.TreePageTop1Accuracy,
		TreePageRecallAt3:              metrics.TreePageRecallAt3,
		CollectionDocumentTop1Accuracy: metrics.CollectionDocumentTop1Accuracy,
		CollectionDocumentRecallAt3:    metrics.CollectionDocumentRecallAt3,
		ExactMatchAccuracy:             metrics.ExactMatchAccuracy,
		AverageHeuristicScore:          metrics.AverageHeuristicScore,
		MinimumHeuristicScore:          metrics.MinimumHeuristicScore,
		MaximumHeuristicScore:          metrics.MaximumHeuristicScore,
	}
}

func toIndexingBenchmarkSnapshotResponse(snapshot *domain.IndexingBenchmarkSnapshot) *IndexingBenchmarkSnapshotResponse {
	if snapshot == nil {
		return nil
	}

	return &IndexingBenchmarkSnapshotResponse{
		ID:                   snapshot.ID,
		Label:                snapshot.Label,
		RecordedOn:           snapshot.RecordedOn,
		Summary:              snapshot.Summary,
		DatasetVersion:       snapshot.DatasetVersion,
		PipelineVersion:      snapshot.PipelineVersion,
		PromptProfileVersion: snapshot.PromptProfileVersion,
		ChunkStrategyVersion: snapshot.ChunkStrategyVersion,
		Metrics:              toIndexingBenchmarkMetricsResponse(snapshot.Metrics),
	}
}

func toIndexingBenchmarkComparisonResponse(item domain.IndexingBenchmarkComparison) IndexingBenchmarkComparisonResponse {
	deltas := make([]IndexingBenchmarkMetricDeltaResponse, 0, len(item.KeyMetricDeltas))
	for _, delta := range item.KeyMetricDeltas {
		deltas = append(deltas, IndexingBenchmarkMetricDeltaResponse{
			Metric:        delta.Metric,
			PreviousValue: delta.PreviousValue,
			CurrentValue:  delta.CurrentValue,
			Delta:         delta.Delta,
		})
	}

	before := toIndexingBenchmarkSnapshotResponse(&item.Before)
	after := toIndexingBenchmarkSnapshotResponse(&item.After)

	response := IndexingBenchmarkComparisonResponse{
		ID:              item.ID,
		Title:           item.Title,
		RecordedOn:      item.RecordedOn,
		Summary:         item.Summary,
		KeyMetricDeltas: deltas,
	}
	if before != nil {
		response.Before = *before
	}
	if after != nil {
		response.After = *after
	}
	return response
}

func toIndexingBenchmarkComparisonResponsePtr(item *domain.IndexingBenchmarkComparison) *IndexingBenchmarkComparisonResponse {
	if item == nil {
		return nil
	}

	response := toIndexingBenchmarkComparisonResponse(*item)
	return &response
}

func toIndexingBenchmarkReferenceComparisonResponse(
	item *domain.IndexingBenchmarkReferenceComparison,
) *IndexingBenchmarkReferenceComparisonResponse {
	if item == nil {
		return nil
	}

	artifacts := make([]IndexingBenchmarkReferenceArtifactResponse, 0, len(item.Artifacts))
	for _, artifact := range item.Artifacts {
		artifacts = append(artifacts, IndexingBenchmarkReferenceArtifactResponse{
			ID:                   artifact.ID,
			Label:                artifact.Label,
			SourcePath:           artifact.SourcePath,
			DocumentName:         artifact.DocumentName,
			TotalNodes:           artifact.TotalNodes,
			MaxDepth:             artifact.MaxDepth,
			SummaryCoverageRatio: artifact.SummaryCoverageRatio,
			InvalidRangeCount:    artifact.InvalidRangeCount,
			AverageTitleLength:   artifact.AverageTitleLength,
			AverageSummaryLength: artifact.AverageSummaryLength,
			Notes:                append([]string(nil), artifact.Notes...),
		})
	}

	checklist := make([]IndexingBenchmarkReferenceChecklistItemResponse, 0, len(item.Checklist))
	for _, checklistItem := range item.Checklist {
		checklist = append(checklist, IndexingBenchmarkReferenceChecklistItemResponse{
			ID:             checklistItem.ID,
			Dimension:      checklistItem.Dimension,
			Label:          checklistItem.Label,
			Status:         checklistItem.Status,
			CurrentValue:   checklistItem.CurrentValue,
			ReferenceValue: checklistItem.ReferenceValue,
			Summary:        checklistItem.Summary,
		})
	}

	return &IndexingBenchmarkReferenceComparisonResponse{
		SourcePath:     item.SourcePath,
		LastReviewedOn: item.LastReviewedOn,
		Summary:        item.Summary,
		Artifacts:      artifacts,
		Checklist:      checklist,
	}
}
