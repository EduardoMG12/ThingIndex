package domain

import (
	"context"
	"errors"

	"thing-index/internal/models"
)

// HighlightColors sao as cinco chaves aceitas; o frontend tem uma classe por chave.
var HighlightColors = []string{"amarelo", "verde", "azul", "vermelho", "roxo"}

// LegendKey e a chave em collections.metadata onde os rotulos das cores vivem.
const LegendKey = "workspace_legend"

const MaxLegendLabelLength = 60

var (
	ErrInvalidHighlightColor    = errors.New("invalid highlight color")
	ErrParagraphNotInCollection = errors.New("paragraph not in collection")
	ErrInvalidLegend            = errors.New("invalid legend")
	ErrInvalidViewState         = errors.New("invalid view state")
)

const MaxViewStateBytes = 64 * 1024

type WorkspaceViewStateRepository interface {
	GetViewState(ctx context.Context, userID, collectionID uint) (map[string]any, error)
	UpsertViewState(ctx context.Context, userID, collectionID uint, state map[string]any) error
}

func IsHighlightColor(color string) bool {
	for _, known := range HighlightColors {
		if known == color {
			return true
		}
	}
	return false
}

// ParagraphHighlightRepository persiste marcas por usuario. Todo metodo e
// escopado por userID e collectionID: o escopo transitivo (paragrafo ->
// documento -> colecao -> usuario) e validado no Upsert, nunca presumido.
type ParagraphHighlightRepository interface {
	ListHighlights(ctx context.Context, userID, collectionID uint) ([]models.ParagraphHighlight, error)
	// UpsertHighlight cria ou troca a cor. Devolve ErrParagraphNotInCollection
	// quando o paragrafo nao pertence a um documento de uma colecao do usuario.
	UpsertHighlight(ctx context.Context, userID, collectionID, paragraphNodeID uint, color string) (*models.ParagraphHighlight, error)
	// DeleteHighlight e idempotente.
	DeleteHighlight(ctx context.Context, userID, collectionID, paragraphNodeID uint) error
}

// Legend e o rotulo de cada cor numa colecao. Chave ausente = sem rotulo.
type Legend map[string]string

type WorkspaceUseCase interface {
	ListHighlights(ctx context.Context, userID, collectionID uint) ([]models.ParagraphHighlight, error)
	SetHighlight(ctx context.Context, userID, collectionID, paragraphNodeID uint, color string) (*models.ParagraphHighlight, error)
	RemoveHighlight(ctx context.Context, userID, collectionID, paragraphNodeID uint) error
	GetLegend(ctx context.Context, userID, collectionID uint) (Legend, error)
	SetLegend(ctx context.Context, userID, collectionID uint, legend Legend) (Legend, error)
	GetViewState(ctx context.Context, userID, collectionID uint) (map[string]any, error)
	SetViewState(ctx context.Context, userID, collectionID uint, state map[string]any) (map[string]any, error)
}
