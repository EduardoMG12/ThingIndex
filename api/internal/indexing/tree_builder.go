package indexing

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

const llmTreePageLimit = 12

type treeConstructionResponse struct {
	Nodes []struct {
		NodeID       string `json:"node_id"`
		ParentNodeID string `json:"parent_node_id"`
		Title        string `json:"title"`
		Level        int    `json:"level"`
		StartPage    int    `json:"start_page"`
		EndPage      int    `json:"end_page"`
		Summary      string `json:"summary"`
	} `json:"nodes"`
}

func (e *LLMEnricher) buildTree(ctx context.Context, llm domain.LLMService, doc *models.Document, result domain.IndexingResult) (domain.IndexingResult, error) {
	if llm == nil || len(result.DocumentPages) == 0 {
		return result, nil
	}

	contextBlock := buildTreeConstructionContext(doc, result)
	systemPrompt, userPrompt := buildTreeConstructionPrompt(contextBlock)
	raw, err := llm.GenerateText(ctx, systemPrompt, userPrompt)
	if err != nil {
		return result, err
	}

	var payload treeConstructionResponse
	if err := parseLLMJSON(raw, &payload); err != nil {
		return result, fmt.Errorf("parse tree construction json: %w", err)
	}

	nodes := normalizeConstructedTree(result, payload)
	if len(nodes) == 0 {
		return result, fmt.Errorf("tree construction returned no valid nodes")
	}

	updated := result
	updated.OutlineNodes = nodes
	return updated, nil
}

func buildTreeConstructionContext(doc *models.Document, result domain.IndexingResult) string {
	var builder strings.Builder
	builder.WriteString(buildDocumentIdentityBlock(doc))
	builder.WriteString("Document pages:\n")
	for idx, page := range result.DocumentPages {
		if idx >= llmTreePageLimit {
			break
		}
		builder.WriteString("- page_number=")
		builder.WriteString(strconv.Itoa(page.PageNumber))
		builder.WriteString(", content=")
		builder.WriteString(clampText(page.Content, 1800))
		builder.WriteString("\n")
	}

	if len(result.OutlineNodes) > 0 {
		builder.WriteString("\nExisting outline hints:\n")
		for idx, node := range result.OutlineNodes {
			if idx >= llmOutlineLimit {
				break
			}
			builder.WriteString("- sequence_order=")
			builder.WriteString(strconv.Itoa(node.SequenceOrder))
			builder.WriteString(", level=")
			builder.WriteString(strconv.Itoa(node.Level))
			builder.WriteString(", title=")
			builder.WriteString(node.Title)
			if node.StartPage != nil {
				builder.WriteString(", start_page=")
				builder.WriteString(strconv.Itoa(*node.StartPage))
			}
			if node.EndPage != nil {
				builder.WriteString(", end_page=")
				builder.WriteString(strconv.Itoa(*node.EndPage))
			}
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func normalizeConstructedTree(result domain.IndexingResult, payload treeConstructionResponse) []models.DocumentOutlineNode {
	if len(payload.Nodes) == 0 {
		return nil
	}

	type normalizedNode struct {
		nodeID       string
		parentNodeID string
		title        string
		level        int
		startPage    int
		endPage      int
		summary      string
	}

	pageNumbers := availablePageNumbers(result.DocumentPages)
	if len(pageNumbers) == 0 {
		return nil
	}
	minPage := pageNumbers[0]
	maxPage := pageNumbers[len(pageNumbers)-1]

	items := make([]normalizedNode, 0, len(payload.Nodes))
	seen := make(map[string]struct{}, len(payload.Nodes))
	for idx, node := range payload.Nodes {
		title := strings.TrimSpace(node.Title)
		if title == "" {
			continue
		}

		nodeID := strings.TrimSpace(node.NodeID)
		if nodeID == "" {
			nodeID = fmt.Sprintf("node-%03d-%s", idx, slugify(title))
		}
		if _, ok := seen[nodeID]; ok {
			nodeID = fmt.Sprintf("%s-%d", nodeID, idx)
		}
		seen[nodeID] = struct{}{}

		level := node.Level
		if level <= 0 {
			level = 1
		}

		startPage := clampPage(node.StartPage, minPage, maxPage)
		endPage := clampPage(node.EndPage, startPage, maxPage)
		if endPage < startPage {
			endPage = startPage
		}

		items = append(items, normalizedNode{
			nodeID:       nodeID,
			parentNodeID: strings.TrimSpace(node.ParentNodeID),
			title:        title,
			level:        level,
			startPage:    startPage,
			endPage:      endPage,
			summary:      clampText(node.Summary, 320),
		})
	}

	if len(items) == 0 {
		return nil
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].startPage == items[j].startPage {
			if items[i].endPage == items[j].endPage {
				if items[i].level == items[j].level {
					return items[i].title < items[j].title
				}
				return items[i].level < items[j].level
			}
			return items[i].endPage < items[j].endPage
		}
		return items[i].startPage < items[j].startPage
	})

	nodes := make([]models.DocumentOutlineNode, 0, len(items))
	nodeIDSet := make(map[string]struct{}, len(items))
	for _, item := range items {
		nodeIDSet[item.nodeID] = struct{}{}
	}

	for idx, item := range items {
		parentNodeID := item.parentNodeID
		if _, ok := nodeIDSet[parentNodeID]; !ok {
			parentNodeID = ""
		}

		metadata := map[string]any{}
		if parentNodeID != "" {
			metadata["parent_node_id"] = parentNodeID
		}
		metadata["builder"] = "llm"

		startPage := item.startPage
		endPage := item.endPage
		nodes = append(nodes, models.DocumentOutlineNode{
			DocumentID:    documentIDFromResult(result),
			SequenceOrder: idx,
			NodeID:        item.nodeID,
			Level:         item.level,
			Title:         item.title,
			Slug:          slugify(item.title),
			Summary:       item.summary,
			StartPage:     &startPage,
			EndPage:       &endPage,
			Metadata:      metadata,
		})
	}

	return repairConstructedTree(nodes)
}

func repairConstructedTree(nodes []models.DocumentOutlineNode) []models.DocumentOutlineNode {
	if len(nodes) == 0 {
		return nodes
	}

	indexByNodeID := make(map[string]int, len(nodes))
	for idx, node := range nodes {
		indexByNodeID[node.NodeID] = idx
	}

	for idx := range nodes {
		nodes[idx].SequenceOrder = idx
		nodes[idx].ParentID = nil

		parentNodeID, _ := nodes[idx].Metadata["parent_node_id"].(string)
		parentIndex, ok := indexByNodeID[parentNodeID]
		if !ok || parentIndex >= idx || parentNodeID == nodes[idx].NodeID {
			delete(nodes[idx].Metadata, "parent_node_id")
			nodes[idx].Level = 1
			nodes[idx].NodePath = nodes[idx].NodeID
			continue
		}

		parent := nodes[parentIndex]
		if !pageRangeContains(parent.StartPage, parent.EndPage, nodes[idx].StartPage, nodes[idx].EndPage) {
			delete(nodes[idx].Metadata, "parent_node_id")
			nodes[idx].Level = 1
			nodes[idx].NodePath = nodes[idx].NodeID
			continue
		}

		if nodes[idx].Level <= parent.Level {
			nodes[idx].Level = parent.Level + 1
		}

		parentPath := strings.TrimSpace(parent.NodePath)
		if parentPath == "" {
			parentPath = parent.NodeID
		}
		nodes[idx].NodePath = parentPath + "/" + nodes[idx].NodeID
	}

	for idx := range nodes {
		if strings.TrimSpace(nodes[idx].NodePath) == "" {
			nodes[idx].NodePath = nodes[idx].NodeID
		}
		if nodes[idx].Metadata == nil {
			nodes[idx].Metadata = map[string]any{}
		}
		nodes[idx].Metadata["validated"] = true
	}

	return nodes
}

func availablePageNumbers(pages []models.DocumentPage) []int {
	values := make([]int, 0, len(pages))
	for _, page := range pages {
		if page.PageNumber > 0 {
			values = append(values, page.PageNumber)
		}
	}
	sort.Ints(values)
	return values
}

func clampPage(value, minPage, maxPage int) int {
	if value < minPage {
		return minPage
	}
	if value > maxPage {
		return maxPage
	}
	return value
}

func pageRangeContains(parentStart, parentEnd, childStart, childEnd *int) bool {
	if parentStart == nil || parentEnd == nil || childStart == nil || childEnd == nil {
		return false
	}
	return *parentStart <= *childStart && *parentEnd >= *childEnd
}
