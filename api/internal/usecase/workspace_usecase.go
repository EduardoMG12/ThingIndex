package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

type workspaceUseCase struct {
	highlights  domain.ParagraphHighlightRepository
	collections domain.CollectionRepository
}

func NewWorkspaceUseCase(highlights domain.ParagraphHighlightRepository, collections domain.CollectionRepository) domain.WorkspaceUseCase {
	return &workspaceUseCase{highlights: highlights, collections: collections}
}

func (u *workspaceUseCase) ListHighlights(ctx context.Context, userID, collectionID uint) ([]models.ParagraphHighlight, error) {
	if _, err := u.ownCollection(ctx, userID, collectionID); err != nil {
		return nil, err
	}
	return u.highlights.ListHighlights(ctx, userID, collectionID)
}

func (u *workspaceUseCase) SetHighlight(ctx context.Context, userID, collectionID, paragraphNodeID uint, color string) (*models.ParagraphHighlight, error) {
	color = strings.ToLower(strings.TrimSpace(color))
	if !domain.IsHighlightColor(color) {
		return nil, domain.ErrInvalidHighlightColor
	}
	return u.highlights.UpsertHighlight(ctx, userID, collectionID, paragraphNodeID, color)
}

func (u *workspaceUseCase) RemoveHighlight(ctx context.Context, userID, collectionID, paragraphNodeID uint) error {
	if _, err := u.ownCollection(ctx, userID, collectionID); err != nil {
		return err
	}
	return u.highlights.DeleteHighlight(ctx, userID, collectionID, paragraphNodeID)
}

func (u *workspaceUseCase) GetLegend(ctx context.Context, userID, collectionID uint) (domain.Legend, error) {
	collection, err := u.ownCollection(ctx, userID, collectionID)
	if err != nil {
		return nil, err
	}
	return legendFromMetadata(collection.Metadata), nil
}

// SetLegend substitui a legenda inteira: o cliente sempre manda as cinco
// chaves que quer manter, e uma chave ausente apaga o rotulo.
func (u *workspaceUseCase) SetLegend(ctx context.Context, userID, collectionID uint, legend domain.Legend) (domain.Legend, error) {
	clean := domain.Legend{}
	for color, label := range legend {
		if !domain.IsHighlightColor(color) {
			return nil, fmt.Errorf("%w: cor %q", domain.ErrInvalidLegend, color)
		}
		label = strings.TrimSpace(label)
		if utf8.RuneCountInString(label) > domain.MaxLegendLabelLength {
			return nil, fmt.Errorf("%w: rotulo de %q com mais de %d caracteres", domain.ErrInvalidLegend, color, domain.MaxLegendLabelLength)
		}
		if label != "" {
			clean[color] = label
		}
	}
	encoded, err := json.Marshal(clean)
	if err != nil {
		return nil, err
	}
	if err := u.collections.UpdateCollectionMetadataKey(ctx, userID, collectionID, domain.LegendKey, encoded); err != nil {
		return nil, err
	}
	return clean, nil
}

func (u *workspaceUseCase) ownCollection(ctx context.Context, userID, collectionID uint) (*models.Collection, error) {
	collection, err := u.collections.GetCollectionByID(ctx, collectionID, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrCollectionNotFound
	}
	if err != nil {
		return nil, err
	}
	return collection, nil
}

func legendFromMetadata(metadata map[string]any) domain.Legend {
	legend := domain.Legend{}
	raw, ok := metadata[domain.LegendKey].(map[string]any)
	if !ok {
		return legend
	}
	for color, value := range raw {
		label, isString := value.(string)
		if isString && domain.IsHighlightColor(color) && strings.TrimSpace(label) != "" {
			legend[color] = label
		}
	}
	return legend
}
