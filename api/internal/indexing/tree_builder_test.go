package indexing

import (
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

func TestNormalizeConstructedTreeRepairsInvalidParentRanges(t *testing.T) {
	t.Parallel()

	result := domain.IndexingResult{
		DocumentPages: []models.DocumentPage{
			{DocumentID: 1, PageNumber: 1, Content: "Intro"},
			{DocumentID: 1, PageNumber: 2, Content: "Payment terms"},
			{DocumentID: 1, PageNumber: 3, Content: "Appendix"},
		},
	}

	payload := treeConstructionResponse{
		Nodes: []struct {
			NodeID       string `json:"node_id"`
			ParentNodeID string `json:"parent_node_id"`
			Title        string `json:"title"`
			Level        int    `json:"level"`
			StartPage    int    `json:"start_page"`
			EndPage      int    `json:"end_page"`
			Summary      string `json:"summary"`
		}{
			{NodeID: "root", Title: "Contrato", Level: 1, StartPage: 1, EndPage: 2, Summary: "Resumo"},
			{NodeID: "child", ParentNodeID: "root", Title: "Anexo", Level: 2, StartPage: 3, EndPage: 3, Summary: "Fora do range"},
		},
	}

	nodes := normalizeConstructedTree(result, payload)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if _, ok := nodes[1].Metadata["parent_node_id"]; ok {
		t.Fatalf("expected invalid parent to be removed: %#v", nodes[1].Metadata)
	}
	if nodes[1].Level != 1 {
		t.Fatalf("expected repaired node level 1, got %d", nodes[1].Level)
	}
	if nodes[1].NodePath != nodes[1].NodeID {
		t.Fatalf("expected repaired node path to reset, got %q", nodes[1].NodePath)
	}
}

func TestNormalizeConstructedTreeKeepsValidHierarchy(t *testing.T) {
	t.Parallel()

	result := domain.IndexingResult{
		DocumentPages: []models.DocumentPage{
			{DocumentID: 1, PageNumber: 1, Content: "Intro"},
			{DocumentID: 1, PageNumber: 2, Content: "Payment terms"},
		},
	}

	payload := treeConstructionResponse{
		Nodes: []struct {
			NodeID       string `json:"node_id"`
			ParentNodeID string `json:"parent_node_id"`
			Title        string `json:"title"`
			Level        int    `json:"level"`
			StartPage    int    `json:"start_page"`
			EndPage      int    `json:"end_page"`
			Summary      string `json:"summary"`
		}{
			{NodeID: "root", Title: "Contrato", Level: 1, StartPage: 1, EndPage: 2, Summary: "Resumo"},
			{NodeID: "child", ParentNodeID: "root", Title: "Pagamento", Level: 2, StartPage: 2, EndPage: 2, Summary: "Prazo"},
		},
	}

	nodes := normalizeConstructedTree(result, payload)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if got := nodes[1].Metadata["parent_node_id"]; got != "root" {
		t.Fatalf("unexpected parent metadata: %#v", got)
	}
	if nodes[1].NodePath != "root/child" {
		t.Fatalf("unexpected node path: %q", nodes[1].NodePath)
	}
	if nodes[1].Level != 2 {
		t.Fatalf("unexpected child level: %d", nodes[1].Level)
	}
}
