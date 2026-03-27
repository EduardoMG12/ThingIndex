package usecase

import (
	"testing"

	"thing-index/internal/models"
)

type retrievalCorpusCase struct {
	name              string
	query             string
	nodes             []models.DocumentOutlineNode
	pages             []models.DocumentPage
	wantTopNodeID     string
	wantTopPageNumber int
}

func TestRetrievalCorpusDocumentCases(t *testing.T) {
	t.Parallel()

	pageOne := 1
	pageTwo := 2
	cases := []retrievalCorpusCase{
		{
			name:  "payment deadline",
			query: "payment deadline",
			nodes: []models.DocumentOutlineNode{
				{NodeID: "intro", Title: "Introduction", Summary: "Resumo geral", StartPage: &pageOne, EndPage: &pageOne},
				{NodeID: "payment", Title: "Payment Terms", Summary: "Deadline de pagamento em 15 dias", StartPage: &pageTwo, EndPage: &pageTwo, Metadata: map[string]any{"confidence_score": 90}},
			},
			pages: []models.DocumentPage{
				{DocumentID: 1, PageNumber: 1, Content: "Introduction and scope"},
				{DocumentID: 1, PageNumber: 2, Content: "Payment deadline is 15 days after invoice."},
			},
			wantTopNodeID:     "payment",
			wantTopPageNumber: 2,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := buildTreeRetrieval(tc.query, tc.nodes, tc.pages)
			if result == nil {
				t.Fatal("expected retrieval result")
			}
			if len(result.NodeHits) == 0 || result.NodeHits[0].NodeID != tc.wantTopNodeID {
				t.Fatalf("unexpected top node: %#v", result.NodeHits)
			}
			if len(result.PageHits) == 0 || result.PageHits[0].PageNumber != tc.wantTopPageNumber {
				t.Fatalf("unexpected top page: %#v", result.PageHits)
			}
		})
	}
}

func TestRetrievalCorpusCollectionCases(t *testing.T) {
	t.Parallel()

	pageOne := 1
	pageTwo := 2
	result := buildCollectionRetrieval(
		"invoice deadline",
		[]models.Document{
			{ID: 10, Title: "Master Agreement"},
			{ID: 11, Title: "Billing Addendum"},
		},
		[]models.DocumentPage{
			{DocumentID: 10, PageNumber: 1, Content: "General clauses."},
			{DocumentID: 11, PageNumber: 2, Content: "Invoice deadline is 10 days after receipt."},
		},
		[]models.DocumentOutlineNode{
			{DocumentID: 10, NodeID: "root-10", Title: "General Terms", Summary: "Resumo", StartPage: &pageOne, EndPage: &pageOne},
			{DocumentID: 11, NodeID: "billing-11", Title: "Billing Terms", Summary: "Invoice deadline e pagamento", StartPage: &pageTwo, EndPage: &pageTwo, Metadata: map[string]any{"confidence_score": 95}},
		},
	)

	if result == nil {
		t.Fatal("expected collection retrieval result")
	}
	if len(result.DocumentHits) == 0 || result.DocumentHits[0].DocumentID != 11 {
		t.Fatalf("unexpected top document: %#v", result.DocumentHits)
	}
	if len(result.NodeHits) == 0 || result.NodeHits[0].DocumentID != 11 {
		t.Fatalf("unexpected top node hit: %#v", result.NodeHits)
	}
	if len(result.PageHits) == 0 || result.PageHits[0].DocumentID != 11 {
		t.Fatalf("unexpected top page hit: %#v", result.PageHits)
	}
}
