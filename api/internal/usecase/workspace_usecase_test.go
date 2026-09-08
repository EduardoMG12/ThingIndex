package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

type highlightRepoFake struct {
	upserts []string
	deleted int
}

func (f *highlightRepoFake) ListHighlights(context.Context, uint, uint) ([]models.ParagraphHighlight, error) {
	return nil, nil
}

func (f *highlightRepoFake) UpsertHighlight(_ context.Context, _, _, _ uint, color string) (*models.ParagraphHighlight, error) {
	f.upserts = append(f.upserts, color)
	return &models.ParagraphHighlight{Color: color}, nil
}

func (f *highlightRepoFake) DeleteHighlight(context.Context, uint, uint, uint) error {
	f.deleted++
	return nil
}

type collectionRepoLegendFake struct {
	collectionRepoStub
	metadata map[string]any
	written  map[string][]byte
}

func (f *collectionRepoLegendFake) GetCollectionByID(_ context.Context, id uint, _ uint) (*models.Collection, error) {
	return &models.Collection{ID: id, Metadata: f.metadata}, nil
}

func (f *collectionRepoLegendFake) UpdateCollectionMetadataKey(_ context.Context, _, _ uint, key string, valueJSON []byte) error {
	if f.written == nil {
		f.written = map[string][]byte{}
	}
	f.written[key] = valueJSON
	return nil
}

type viewStateRepoFake struct {
	salvo map[string]any
}

func (f *viewStateRepoFake) GetViewState(context.Context, uint, uint) (map[string]any, error) {
	if f.salvo == nil {
		return map[string]any{}, nil
	}
	return f.salvo, nil
}

func (f *viewStateRepoFake) UpsertViewState(_ context.Context, _, _ uint, state map[string]any) error {
	f.salvo = state
	return nil
}

func TestSetViewStateRejeitaNuloEGrande(t *testing.T) {
	repo := &viewStateRepoFake{}
	uc := NewWorkspaceUseCase(&highlightRepoFake{}, &collectionRepoLegendFake{}, repo)
	if _, err := uc.SetViewState(context.Background(), 1, 7, nil); !errors.Is(err, domain.ErrInvalidViewState) {
		t.Fatalf("estado nulo: quero ErrInvalidViewState, veio %v", err)
	}
	grande := map[string]any{"x": strings.Repeat("a", domain.MaxViewStateBytes)}
	if _, err := uc.SetViewState(context.Background(), 1, 7, grande); !errors.Is(err, domain.ErrInvalidViewState) {
		t.Fatalf("estado grande: quero ErrInvalidViewState, veio %v", err)
	}
	if repo.salvo != nil {
		t.Fatalf("repositorio nao devia ter sido tocado: %v", repo.salvo)
	}
	ok := map[string]any{"viewport": map[string]any{"x": 1, "y": 2, "escala": 0.5}}
	if _, err := uc.SetViewState(context.Background(), 1, 7, ok); err != nil {
		t.Fatalf("estado valido: %v", err)
	}
	lido, _ := uc.GetViewState(context.Background(), 1, 7)
	if lido["viewport"] == nil {
		t.Fatalf("GetViewState nao devolveu o que foi salvo: %v", lido)
	}
}

func TestSetHighlightRejeitaCorForaDaPaletaSemTocarNoRepositorio(t *testing.T) {
	repo := &highlightRepoFake{}
	uc := NewWorkspaceUseCase(repo, &collectionRepoLegendFake{}, &viewStateRepoFake{})

	for _, cor := range []string{"magenta", "", "yellow", "amarelo "} {
		_, err := uc.SetHighlight(context.Background(), 1, 1, 1, cor)
		if cor == "amarelo " {
			// Espaco e caixa sao normalizados: nao e erro do usuario.
			if err != nil {
				t.Fatalf("%q devia ser aceito apos trim: %v", cor, err)
			}
			continue
		}
		if !errors.Is(err, domain.ErrInvalidHighlightColor) {
			t.Fatalf("cor %q: quero ErrInvalidHighlightColor, veio %v", cor, err)
		}
	}
	if len(repo.upserts) != 1 || repo.upserts[0] != "amarelo" {
		t.Fatalf("repositorio recebeu %v; so 'amarelo' devia chegar", repo.upserts)
	}
}

func TestSetLegendLimpaValidaEGravaSoAChaveDaLegenda(t *testing.T) {
	repo := &collectionRepoLegendFake{}
	uc := NewWorkspaceUseCase(&highlightRepoFake{}, repo, &viewStateRepoFake{})

	saida, err := uc.SetLegend(context.Background(), 1, 7, domain.Legend{"amarelo": "  prazo ", "verde": "   "})
	if err != nil {
		t.Fatalf("SetLegend: %v", err)
	}
	if saida["amarelo"] != "prazo" || len(saida) != 1 {
		t.Fatalf("legenda limpa = %v; quero so amarelo=prazo (verde vazio some)", saida)
	}
	var gravado map[string]string
	if err := json.Unmarshal(repo.written[domain.LegendKey], &gravado); err != nil {
		t.Fatalf("o que foi gravado nao e JSON: %v", err)
	}
	if gravado["amarelo"] != "prazo" {
		t.Fatalf("gravado = %v", gravado)
	}

	if _, err := uc.SetLegend(context.Background(), 1, 7, domain.Legend{"magenta": "x"}); !errors.Is(err, domain.ErrInvalidLegend) {
		t.Fatalf("cor fora da paleta na legenda: quero ErrInvalidLegend, veio %v", err)
	}
	longo := strings.Repeat("a", domain.MaxLegendLabelLength+1)
	if _, err := uc.SetLegend(context.Background(), 1, 7, domain.Legend{"azul": longo}); !errors.Is(err, domain.ErrInvalidLegend) {
		t.Fatalf("rotulo longo: quero ErrInvalidLegend, veio %v", err)
	}
}

func TestGetLegendIgnoraLixoNoMetadata(t *testing.T) {
	repo := &collectionRepoLegendFake{metadata: map[string]any{
		domain.LegendKey: map[string]any{"amarelo": "prazo", "magenta": "nao", "verde": 3, "azul": ""},
		"outra_chave":    "fica",
	}}
	uc := NewWorkspaceUseCase(&highlightRepoFake{}, repo, &viewStateRepoFake{})
	legenda, err := uc.GetLegend(context.Background(), 1, 7)
	if err != nil {
		t.Fatalf("GetLegend: %v", err)
	}
	if len(legenda) != 1 || legenda["amarelo"] != "prazo" {
		t.Fatalf("legenda = %v; quero so amarelo=prazo", legenda)
	}
}
