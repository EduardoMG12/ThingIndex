package domain

import (
	"context"
	"errors"

	"thing-index/internal/models"
)

// HighlightColors sao as cinco chaves aceitas para uma marca. Fixas de
// proposito: a legenda da colecao da significado a cada uma, e o frontend
// tem classe de cor para cada chave.
var HighlightColors = []string{"amarelo", "verde", "azul", "vermelho", "roxo"}

// LegendKey e a chave em collections.metadata onde os rotulos das cores vivem.
const LegendKey = "workspace_legend"

// MaxLegendLabelLength limita o rotulo de cada cor.
const MaxLegendLabelLength = 60

var (
	ErrInvalidHighlightColor    = errors.New("invalid highlight color")
	ErrParagraphNotInCollection = errors.New("paragraph not in collection")
	ErrInvalidLegend            = errors.New("invalid legend")
	ErrInvalidViewState         = errors.New("invalid view state")
)

// MaxViewStateBytes limita o estado serializado; 64 KB cabem centenas de
// quadros e e o suficiente para uma colecao inteira.
const MaxViewStateBytes = 64 * 1024

// WorkspaceViewStateRepository persiste um estado por usuario e colecao.
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
	// DeleteHighlight e idempotente: apagar o que nao existe nao e erro.
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
