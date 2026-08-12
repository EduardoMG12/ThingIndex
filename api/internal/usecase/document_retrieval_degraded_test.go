package usecase

import (
	"testing"

	"thing-index/internal/models"
)

func TestBuildTreeRetrievalUsesChunkEvidenceToRescueWeakTreeNodes(t *testing.T) {
	t.Parallel()

	pageOne := 1
	pageTwo := 2
	result := buildTreeRetrieval(
		"invoice paid within 7 days",
		[]models.DocumentOutlineNode{
			{
				NodeID:        "payment-overview",
				SequenceOrder: 0,
				Level:         1,
				Title:         "Payment",
				Summary:       "General payment overview.",
				StartPage:     &pageOne,
				EndPage:       &pageOne,
				Metadata: map[string]any{
					"confidence_score":   22,
					"confidence_band":    "low",
					"fallback_structure": true,
				},
			},
			{
				NodeID:        "schedule-b",
				SequenceOrder: 1,
				Level:         2,
				Title:         "Schedule B",
				Summary:       "Appendix.",
				StartPage:     &pageTwo,
				EndPage:       &pageTwo,
				Metadata: map[string]any{
					"parent_node_id": "payment-overview",
				},
			},
		},
		[]models.DocumentPage{
			{PageNumber: 1, Content: "General payment overview."},
			{PageNumber: 2, Content: "Appendix."},
		},
		[]models.DocumentChunk{
			{
				ChunkType: "paragraph_window",
				Title:     "Payment schedule",
				Content:   "The invoice must be paid within 7 days of receipt.",
				Metadata: map[string]any{
					"outline_node_id": "schedule-b",
					"page_start":      2,
					"page_end":        2,
				},
			},
		},
	)
	if result == nil {
		t.Fatal("expected retrieval result")
	}
	if len(result.NodeHits) == 0 || result.NodeHits[0].NodeID != "schedule-b" {
		t.Fatalf("expected chunk-backed node to rank first, got %#v", result.NodeHits)
	}
	if len(result.PageHits) == 0 || result.PageHits[0].PageNumber != 2 {
		t.Fatalf("expected chunk-backed page to rank first, got %#v", result.PageHits)
	}
	if len(result.TraversalSteps) != 2 || result.TraversalSteps[1].NodeID != "schedule-b" {
		t.Fatalf("expected traversal path to end at rescued node, got %#v", result.TraversalSteps)
	}
	if result.Strategy != "tree-degraded-direct-evidence" {
		t.Fatalf("expected degraded direct evidence strategy, got %#v", result)
	}
}

func TestBuildTreeRetrievalUsesChunkEvidenceForPageHitsWithoutLexicalPageMatch(t *testing.T) {
	t.Parallel()

	pageThree := 3
	result := buildTreeRetrieval(
		"written notice within 48 hours",
		[]models.DocumentOutlineNode{
			{
				NodeID:        "incident-response",
				SequenceOrder: 0,
				Level:         1,
				Title:         "Incident Response",
				Summary:       "Operational process.",
				StartPage:     &pageThree,
				EndPage:       &pageThree,
			},
		},
		[]models.DocumentPage{
			{PageNumber: 3, Content: "Appendix placeholder text."},
		},
		[]models.DocumentChunk{
			{
				ChunkType: "paragraph_window",
				Title:     "Escalation window",
				Content:   "Written notice must be sent within 48 hours after the incident.",
				Metadata: map[string]any{
					"outline_node_id": "incident-response",
					"page_start":      3,
					"page_end":        3,
				},
			},
		},
	)
	if result == nil {
		t.Fatal("expected retrieval result")
	}
	if len(result.PageHits) == 0 || result.PageHits[0].PageNumber != 3 {
		t.Fatalf("expected chunk evidence to create a page hit, got %#v", result.PageHits)
	}
	if result.PageHits[0].Score == 0 {
		t.Fatalf("expected positive page score, got %#v", result.PageHits[0])
	}
}
