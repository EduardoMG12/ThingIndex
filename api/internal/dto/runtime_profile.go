package dto

import "thing-index/internal/domain"

type RuntimeProfileAudienceResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RuntimeProfileArtifactResponse struct {
	Label   string `json:"label"`
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Summary string `json:"summary"`
}

type RuntimeProfileResponse struct {
	ProductName      string                           `json:"product_name"`
	Tagline          string                           `json:"tagline"`
	PrimaryUseCase   string                           `json:"primary_use_case"`
	CoreProblem      string                           `json:"core_problem"`
	ValueProposition string                           `json:"value_proposition"`
	TargetUsers      []RuntimeProfileAudienceResponse `json:"target_users"`
	CurrentScope     []string                         `json:"current_scope"`
	Differentiators  []string                         `json:"differentiators"`
	NonGoals         []string                         `json:"non_goals"`
	ResearchTracks   []string                         `json:"research_tracks"`
	ArtifactRefs     []RuntimeProfileArtifactResponse `json:"artifact_refs"`
	LastReviewedOn   string                           `json:"last_reviewed_on"`
}

func ToRuntimeProfileResponse(profile domain.RuntimeProfile) RuntimeProfileResponse {
	audience := make([]RuntimeProfileAudienceResponse, 0, len(profile.TargetUsers))
	for _, item := range profile.TargetUsers {
		audience = append(audience, RuntimeProfileAudienceResponse{
			Name:        item.Name,
			Description: item.Description,
		})
	}

	artifacts := make([]RuntimeProfileArtifactResponse, 0, len(profile.ArtifactRefs))
	for _, item := range profile.ArtifactRefs {
		artifacts = append(artifacts, RuntimeProfileArtifactResponse{
			Label:   item.Label,
			Path:    item.Path,
			Kind:    item.Kind,
			Summary: item.Summary,
		})
	}

	return RuntimeProfileResponse{
		ProductName:      profile.ProductName,
		Tagline:          profile.Tagline,
		PrimaryUseCase:   profile.PrimaryUseCase,
		CoreProblem:      profile.CoreProblem,
		ValueProposition: profile.ValueProposition,
		TargetUsers:      audience,
		CurrentScope:     copiaDeTextos(profile.CurrentScope),
		Differentiators:  copiaDeTextos(profile.Differentiators),
		NonGoals:         copiaDeTextos(profile.NonGoals),
		ResearchTracks:   copiaDeTextos(profile.ResearchTracks),
		ArtifactRefs:     artifacts,
		LastReviewedOn:   profile.LastReviewedOn,
	}
}
