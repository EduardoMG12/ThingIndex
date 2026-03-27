package indexing

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

const llmTreePageLimit = 12
const minimumTreeCoverageRatio = 0.6
const maxPagesPerNode = 10
const maxTokensPerNode = 20000
const maxRecursiveExpansions = 2
const maxPhysicalFixRetries = 2

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

type treeVerificationStats struct {
	Builder            string
	TotalNodes         int
	TotalPages         int
	CoveredPages       int
	CoverageRatio      float64
	MaxDepth           int
	RepairCount        int
	DetachedNodeCount  int
	LowConfidenceNodes int
	VerificationPassed bool
	FallbackReason     string
	ExpandedNodes      int
	OversizedNodes     int
	LowTextPages       int
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

	nodes, stats := verifyAndScoreTree(result.DocumentPages, nodes, "llm")
	if !stats.VerificationPassed {
		fixedNodes, fixedStats, fixErr := e.fixTreeWithRetries(ctx, llm, doc, result, nodes, stats)
		if fixErr != nil || !fixedStats.VerificationPassed {
			return result, fmt.Errorf("tree verification failed: %s", stats.FallbackReason)
		}
		nodes = fixedNodes
		stats = fixedStats
	}

	expandedNodes, expandedStats, expandErr := e.expandOversizedNodes(ctx, llm, doc, result.DocumentPages, nodes, maxRecursiveExpansions)
	if expandErr == nil && len(expandedNodes) > 0 {
		nodes = expandedNodes
		stats = mergeTreeVerificationStats(stats, expandedStats)
	}

	updated := result
	updated.OutlineNodes = nodes
	appendTreeVerificationMetadata(&updated, documentIDFromResult(result), stats)
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
			nodeID = buildOutlineNodeID(idx, title)
		} else {
			nodeID = clampIdentifierWithHash(nodeID, maxOutlineNodeIDLength)
		}
		if _, ok := seen[nodeID]; ok {
			nodeID = buildOutlineNodeID(idx, fmt.Sprintf("%s-%d", title, idx))
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
			Slug:          buildOutlineSlug(item.title),
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

func verifyAndScoreTree(pages []models.DocumentPage, nodes []models.DocumentOutlineNode, builder string) ([]models.DocumentOutlineNode, treeVerificationStats) {
	stats := treeVerificationStats{
		Builder:    strings.TrimSpace(builder),
		TotalNodes: len(nodes),
		TotalPages: len(pages),
	}
	if stats.Builder == "" {
		stats.Builder = "unknown"
	}

	if len(nodes) == 0 || len(pages) == 0 {
		stats.FallbackReason = "missing_nodes_or_pages"
		return nodes, stats
	}

	pageContent := make(map[int]string, len(pages))
	pageCovered := make(map[int]struct{}, len(pages))
	for _, page := range pages {
		pageContent[page.PageNumber] = page.Content
		if estimateTextTokens(page.Content) < 20 {
			stats.LowTextPages++
		}
	}

	for idx := range nodes {
		if nodes[idx].Metadata == nil {
			nodes[idx].Metadata = map[string]any{}
		}

		repaired := false
		if nodes[idx].StartPage == nil || nodes[idx].EndPage == nil {
			inferNodePageRange(&nodes[idx], pageNumbersForBuilder(pages))
			repaired = true
		}
		if nodes[idx].StartPage != nil && nodes[idx].EndPage != nil && *nodes[idx].EndPage < *nodes[idx].StartPage {
			end := *nodes[idx].StartPage
			nodes[idx].EndPage = &end
			repaired = true
		}

		parentNodeID, _ := nodes[idx].Metadata["parent_node_id"].(string)
		if parentNodeID == "" {
			nodes[idx].Level = 1
		}
		if repaired {
			stats.RepairCount++
		}

		confidence := nodeConfidenceScore(nodes[idx], pageContent)
		nodes[idx].Metadata["confidence_score"] = confidence
		nodes[idx].Metadata["confidence_band"] = confidenceBand(confidence)
		nodes[idx].Metadata["builder"] = stats.Builder
		nodes[idx].Metadata["verification_status"] = "verified"
		if confidence < 55 {
			stats.LowConfidenceNodes++
		}

		if nodes[idx].StartPage != nil && nodes[idx].EndPage != nil {
			for pageNumber := *nodes[idx].StartPage; pageNumber <= *nodes[idx].EndPage; pageNumber++ {
				if _, ok := pageContent[pageNumber]; ok {
					pageCovered[pageNumber] = struct{}{}
				}
			}
		}
		if nodes[idx].Level > stats.MaxDepth {
			stats.MaxDepth = nodes[idx].Level
		}
	}

	stats.CoveredPages = len(pageCovered)
	if stats.TotalPages > 0 {
		stats.CoverageRatio = float64(stats.CoveredPages) / float64(stats.TotalPages)
	}
	stats.VerificationPassed = stats.CoverageRatio >= minimumTreeCoverageRatio && stats.TotalNodes > 0
	if !stats.VerificationPassed {
		if stats.CoverageRatio < minimumTreeCoverageRatio {
			stats.FallbackReason = "low_page_coverage"
		} else {
			stats.FallbackReason = "verification_failed"
		}
	}

	return nodes, stats
}

func appendTreeVerificationMetadata(result *domain.IndexingResult, documentID uint, stats treeVerificationStats) {
	if result == nil || documentID == 0 {
		return
	}

	appendMetadata(result, documentID, "tree_builder", stats.Builder)
	appendMetadata(result, documentID, "tree_total_nodes", strconv.Itoa(stats.TotalNodes))
	appendMetadata(result, documentID, "tree_total_pages", strconv.Itoa(stats.TotalPages))
	appendMetadata(result, documentID, "tree_covered_pages", strconv.Itoa(stats.CoveredPages))
	appendMetadata(result, documentID, "tree_coverage_ratio", fmt.Sprintf("%.2f", stats.CoverageRatio))
	appendMetadata(result, documentID, "tree_max_depth", strconv.Itoa(stats.MaxDepth))
	appendMetadata(result, documentID, "tree_repair_count", strconv.Itoa(stats.RepairCount))
	appendMetadata(result, documentID, "tree_low_confidence_nodes", strconv.Itoa(stats.LowConfidenceNodes))
	appendMetadata(result, documentID, "tree_expanded_nodes", strconv.Itoa(stats.ExpandedNodes))
	appendMetadata(result, documentID, "tree_oversized_nodes", strconv.Itoa(stats.OversizedNodes))
	appendMetadata(result, documentID, "tree_low_text_pages", strconv.Itoa(stats.LowTextPages))
	appendMetadata(result, documentID, "tree_summary_coverage_ratio", fmt.Sprintf("%.2f", outlineSummaryCoverageRatio(result.OutlineNodes)))
	appendMetadata(result, documentID, "tree_verification_passed", strconv.FormatBool(stats.VerificationPassed))
	if strings.TrimSpace(stats.FallbackReason) != "" {
		appendMetadata(result, documentID, "tree_fallback_reason", stats.FallbackReason)
	}
}

func (e *LLMEnricher) fixTreeWithRetries(
	ctx context.Context,
	llm domain.LLMService,
	doc *models.Document,
	result domain.IndexingResult,
	nodes []models.DocumentOutlineNode,
	stats treeVerificationStats,
) ([]models.DocumentOutlineNode, treeVerificationStats, error) {
	if llm == nil {
		return nodes, stats, nil
	}

	currentNodes := nodes
	currentStats := stats
	for attempt := 1; attempt <= maxPhysicalFixRetries; attempt++ {
		contextBlock := buildTreeFixContext(doc, result.DocumentPages, currentNodes, currentStats)
		systemPrompt, userPrompt := buildTreeFixPrompt(contextBlock)
		raw, err := llm.GenerateText(ctx, systemPrompt, userPrompt)
		if err != nil {
			return currentNodes, currentStats, err
		}

		var payload treeConstructionResponse
		if err := parseLLMJSON(raw, &payload); err != nil {
			return currentNodes, currentStats, err
		}

		fixedNodes := normalizeConstructedTree(result, payload)
		if len(fixedNodes) == 0 {
			continue
		}

		fixedNodes, fixedStats := verifyAndScoreTree(result.DocumentPages, fixedNodes, "llm-fixed")
		if fixedStats.VerificationPassed {
			return fixedNodes, fixedStats, nil
		}

		currentNodes = fixedNodes
		currentStats = fixedStats
	}

	return currentNodes, currentStats, fmt.Errorf("tree fix retries exhausted")
}

func buildTreeFixContext(doc *models.Document, pages []models.DocumentPage, nodes []models.DocumentOutlineNode, stats treeVerificationStats) string {
	var builder strings.Builder
	builder.WriteString(buildDocumentIdentityBlock(doc))
	builder.WriteString("Current verification status:\n")
	builder.WriteString("- coverage_ratio=")
	builder.WriteString(fmt.Sprintf("%.2f", stats.CoverageRatio))
	builder.WriteString("\n- low_confidence_nodes=")
	builder.WriteString(strconv.Itoa(stats.LowConfidenceNodes))
	builder.WriteString("\n- fallback_reason=")
	builder.WriteString(stats.FallbackReason)
	builder.WriteString("\n\nCurrent nodes:\n")
	for _, node := range nodes {
		builder.WriteString("- node_id=")
		builder.WriteString(node.NodeID)
		builder.WriteString(", title=")
		builder.WriteString(node.Title)
		builder.WriteString(", level=")
		builder.WriteString(strconv.Itoa(node.Level))
		if node.StartPage != nil {
			builder.WriteString(", start_page=")
			builder.WriteString(strconv.Itoa(*node.StartPage))
		}
		if node.EndPage != nil {
			builder.WriteString(", end_page=")
			builder.WriteString(strconv.Itoa(*node.EndPage))
		}
		builder.WriteString(", summary=")
		builder.WriteString(clampText(node.Summary, 120))
		builder.WriteString("\n")
	}
	builder.WriteString("\nPages:\n")
	for _, page := range pages {
		builder.WriteString("- page_number=")
		builder.WriteString(strconv.Itoa(page.PageNumber))
		builder.WriteString(", content=")
		builder.WriteString(clampText(page.Content, 1200))
		builder.WriteString("\n")
	}
	return builder.String()
}

func (e *LLMEnricher) expandOversizedNodes(
	ctx context.Context,
	llm domain.LLMService,
	doc *models.Document,
	pages []models.DocumentPage,
	nodes []models.DocumentOutlineNode,
	depth int,
) ([]models.DocumentOutlineNode, treeVerificationStats, error) {
	stats := treeVerificationStats{Builder: "llm-recursive"}
	if llm == nil || depth <= 0 || len(nodes) == 0 {
		return nodes, stats, nil
	}

	pageIndex := make(map[int]models.DocumentPage, len(pages))
	for _, page := range pages {
		pageIndex[page.PageNumber] = page
	}

	expanded := make([]models.DocumentOutlineNode, 0, len(nodes))
	for _, node := range nodes {
		expanded = append(expanded, node)
		if !isOversizedNode(node, pageIndex) {
			continue
		}

		stats.OversizedNodes++
		children, err := e.expandNode(ctx, llm, doc, node, pages)
		if err != nil || len(children) == 0 {
			continue
		}

		subsetPages := pagesForRange(pages, *node.StartPage, *node.EndPage)
		children, childStats := verifyAndScoreTree(subsetPages, children, "llm-recursive")
		if !childStats.VerificationPassed {
			continue
		}

		stats = mergeTreeVerificationStats(stats, childStats)
		stats.ExpandedNodes += len(children)
		expanded = append(expanded, children...)
	}

	expanded = repairConstructedTree(sortTreeNodes(expanded))
	if depth > 1 {
		return e.expandOversizedNodes(ctx, llm, doc, pages, expanded, depth-1)
	}

	return expanded, stats, nil
}

func (e *LLMEnricher) expandNode(
	ctx context.Context,
	llm domain.LLMService,
	doc *models.Document,
	parent models.DocumentOutlineNode,
	pages []models.DocumentPage,
) ([]models.DocumentOutlineNode, error) {
	if parent.StartPage == nil || parent.EndPage == nil {
		return nil, nil
	}

	subset := pagesForRange(pages, *parent.StartPage, *parent.EndPage)
	if len(subset) <= 1 {
		return nil, nil
	}

	var builder strings.Builder
	builder.WriteString(buildDocumentIdentityBlock(doc))
	builder.WriteString("Parent node:\n- node_id=")
	builder.WriteString(parent.NodeID)
	builder.WriteString(", title=")
	builder.WriteString(parent.Title)
	builder.WriteString(", level=")
	builder.WriteString(strconv.Itoa(parent.Level))
	builder.WriteString(", start_page=")
	builder.WriteString(strconv.Itoa(*parent.StartPage))
	builder.WriteString(", end_page=")
	builder.WriteString(strconv.Itoa(*parent.EndPage))
	builder.WriteString("\n\nSubset pages:\n")
	for _, page := range subset {
		builder.WriteString("- page_number=")
		builder.WriteString(strconv.Itoa(page.PageNumber))
		builder.WriteString(", content=")
		builder.WriteString(clampText(page.Content, 1600))
		builder.WriteString("\n")
	}

	systemPrompt, userPrompt := buildSubtreeExpansionPrompt(builder.String())
	raw, err := llm.GenerateText(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var payload treeConstructionResponse
	if err := parseLLMJSON(raw, &payload); err != nil {
		return nil, err
	}

	normalized := normalizeConstructedTree(domain.IndexingResult{
		DocumentPages: subset,
		OutlineNodes:  []models.DocumentOutlineNode{parent},
	}, payload)
	if len(normalized) == 0 {
		return nil, nil
	}

	children := make([]models.DocumentOutlineNode, 0, len(normalized))
	for _, node := range normalized {
		if node.NodeID == parent.NodeID {
			continue
		}
		if node.Metadata == nil {
			node.Metadata = map[string]any{}
		}
		node.Metadata["parent_node_id"] = parent.NodeID
		if node.Level <= parent.Level {
			node.Level = parent.Level + 1
		}
		children = append(children, node)
	}

	return children, nil
}

func isOversizedNode(node models.DocumentOutlineNode, pages map[int]models.DocumentPage) bool {
	if node.StartPage == nil || node.EndPage == nil {
		return false
	}

	if (*node.EndPage - *node.StartPage + 1) > maxPagesPerNode {
		return true
	}

	tokenCount := 0
	for pageNumber := *node.StartPage; pageNumber <= *node.EndPage; pageNumber++ {
		page, ok := pages[pageNumber]
		if !ok {
			continue
		}
		tokenCount += page.TokenEstimate
		if tokenCount > maxTokensPerNode {
			return true
		}
	}

	return false
}

func pagesForRange(pages []models.DocumentPage, startPage, endPage int) []models.DocumentPage {
	items := make([]models.DocumentPage, 0, endPage-startPage+1)
	for _, page := range pages {
		if page.PageNumber >= startPage && page.PageNumber <= endPage {
			items = append(items, page)
		}
	}
	return items
}

func sortTreeNodes(nodes []models.DocumentOutlineNode) []models.DocumentOutlineNode {
	sort.SliceStable(nodes, func(i, j int) bool {
		leftStart := 0
		rightStart := 0
		if nodes[i].StartPage != nil {
			leftStart = *nodes[i].StartPage
		}
		if nodes[j].StartPage != nil {
			rightStart = *nodes[j].StartPage
		}
		if leftStart == rightStart {
			if nodes[i].Level == nodes[j].Level {
				return nodes[i].Title < nodes[j].Title
			}
			return nodes[i].Level < nodes[j].Level
		}
		return leftStart < rightStart
	})
	return nodes
}

func mergeTreeVerificationStats(left, right treeVerificationStats) treeVerificationStats {
	merged := left
	merged.TotalNodes = max(merged.TotalNodes, right.TotalNodes)
	merged.TotalPages = max(merged.TotalPages, right.TotalPages)
	merged.CoveredPages = max(merged.CoveredPages, right.CoveredPages)
	merged.CoverageRatio = math.Max(merged.CoverageRatio, right.CoverageRatio)
	merged.MaxDepth = max(merged.MaxDepth, right.MaxDepth)
	merged.RepairCount += right.RepairCount
	merged.DetachedNodeCount += right.DetachedNodeCount
	merged.LowConfidenceNodes += right.LowConfidenceNodes
	merged.ExpandedNodes += right.ExpandedNodes
	merged.OversizedNodes += right.OversizedNodes
	merged.VerificationPassed = merged.VerificationPassed || right.VerificationPassed
	if strings.TrimSpace(merged.FallbackReason) == "" {
		merged.FallbackReason = right.FallbackReason
	}
	return merged
}

func inferNodePageRange(node *models.DocumentOutlineNode, pageNumbers []int) {
	if node == nil || len(pageNumbers) == 0 {
		return
	}
	start := pageNumbers[0]
	end := pageNumbers[len(pageNumbers)-1]
	node.StartPage = &start
	node.EndPage = &end
}

func pageNumbersForBuilder(pages []models.DocumentPage) []int {
	return availablePageNumbers(pages)
}

func nodeConfidenceScore(node models.DocumentOutlineNode, pageContent map[int]string) int {
	score := 100
	titleTerms := tokenizeNodeText(node.Title)
	summaryTerms := tokenizeNodeText(node.Summary)

	if strings.TrimSpace(node.Summary) == "" {
		score -= 10
	}
	if node.StartPage == nil || node.EndPage == nil {
		score -= 20
	}
	if node.StartPage != nil && node.EndPage != nil {
		span := *node.EndPage - *node.StartPage + 1
		if span > 10 {
			score -= min(20, (span-10)*2)
		}
	}

	coverageMatches := 0
	if node.StartPage != nil && node.EndPage != nil {
		for pageNumber := *node.StartPage; pageNumber <= *node.EndPage; pageNumber++ {
			content, ok := pageContent[pageNumber]
			if !ok {
				continue
			}
			if lexicalOverlapScore(content, titleTerms)+lexicalOverlapScore(content, summaryTerms) > 0 {
				coverageMatches++
			}
		}
	}
	if coverageMatches == 0 {
		score -= 30
	} else if coverageMatches == 1 {
		score -= 10
	}

	parentNodeID, _ := node.Metadata["parent_node_id"].(string)
	if parentNodeID == "" && node.Level > 1 {
		score -= 15
	}

	return clampConfidence(score)
}

func tokenizeNodeText(value string) []string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return nil
	}
	replacer := strings.NewReplacer(",", " ", ".", " ", ";", " ", ":", " ", "\n", " ", "\t", " ", "(", " ", ")", " ")
	parts := strings.Fields(replacer.Replace(normalized))
	seen := make(map[string]struct{}, len(parts))
	tokens := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) < 3 {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		tokens = append(tokens, part)
	}
	return tokens
}

func lexicalOverlapScore(content string, terms []string) int {
	normalized := strings.ToLower(content)
	score := 0
	for _, term := range terms {
		if strings.Contains(normalized, term) {
			score++
		}
	}
	return score
}

func confidenceBand(score int) string {
	switch {
	case score >= 85:
		return "high"
	case score >= 65:
		return "medium"
	default:
		return "low"
	}
}

func clampConfidence(score int) int {
	return int(math.Max(0, math.Min(100, float64(score))))
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
