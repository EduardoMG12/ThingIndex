package dto

import (
	"thing-index/internal/domain"
	"thing-index/internal/models"
)

type SetHighlightRequest struct {
	Color string `json:"color" binding:"required"`
}

type HighlightResponse struct {
	ParagraphNodeID uint   `json:"paragraph_node_id"`
	DocumentID      uint   `json:"document_id"`
	Color           string `json:"color"`
}

type HighlightsResponse struct {
	Highlights []HighlightResponse `json:"highlights"`
}

func ToHighlightResponse(h models.ParagraphHighlight) HighlightResponse {
	return HighlightResponse{ParagraphNodeID: h.ParagraphNodeID, DocumentID: h.DocumentID, Color: h.Color}
}

func ToHighlightsResponse(items []models.ParagraphHighlight) HighlightsResponse {
	out := make([]HighlightResponse, 0, len(items))
	for _, item := range items {
		out = append(out, ToHighlightResponse(item))
	}
	return HighlightsResponse{Highlights: out}
}

type LegendRequest struct {
	Labels map[string]string `json:"labels"`
}

type LegendResponse struct {
	Labels map[string]string `json:"labels"`
	Colors []string          `json:"colors"`
}

func ToLegendResponse(legend domain.Legend) LegendResponse {
	labels := map[string]string{}
	for color, label := range legend {
		labels[color] = label
	}
	return LegendResponse{Labels: labels, Colors: domain.HighlightColors}
}
