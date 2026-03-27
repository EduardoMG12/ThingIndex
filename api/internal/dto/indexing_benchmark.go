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
	DatasetVersion       string                                    `json:"dataset_version"`
	CorpusSourcePath     string                                    `json:"corpus_source_path"`
	LastReviewedOn       string                                    `json:"last_reviewed_on"`
	Summary              string                                    `json:"summary"`
	Methodology          []string                                  `json:"methodology"`
	PassCriteria         []string                                  `json:"pass_criteria"`
	Limitations          []string                                  `json:"limitations"`
	HeuristicQualityNote string                                    `json:"heuristic_quality_note"`
	Corpus               []IndexingBenchmarkCorpusDocumentResponse `json:"corpus"`
	Metrics              IndexingBenchmarkMetricsResponse          `json:"metrics"`
	Cases                []IndexingBenchmarkCaseResultResponse     `json:"cases"`
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
		Cases: cases,
	}
}
