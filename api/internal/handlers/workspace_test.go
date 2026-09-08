package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

type workspaceUseCaseFake struct {
	setCalls int
}

func (f *workspaceUseCaseFake) ListHighlights(context.Context, uint, uint) ([]models.ParagraphHighlight, error) {
	return nil, nil
}

func (f *workspaceUseCaseFake) SetHighlight(_ context.Context, _, _, _ uint, color string) (*models.ParagraphHighlight, error) {
	f.setCalls++
	if !domain.IsHighlightColor(color) {
		return nil, domain.ErrInvalidHighlightColor
	}
	return &models.ParagraphHighlight{ParagraphNodeID: 5, DocumentID: 2, Color: color}, nil
}

func (f *workspaceUseCaseFake) RemoveHighlight(context.Context, uint, uint, uint) error { return nil }

func (f *workspaceUseCaseFake) GetLegend(context.Context, uint, uint) (domain.Legend, error) {
	return domain.Legend{}, nil
}

func (f *workspaceUseCaseFake) SetLegend(_ context.Context, _, _ uint, legend domain.Legend) (domain.Legend, error) {
	return legend, nil
}

func (f *workspaceUseCaseFake) GetViewState(context.Context, uint, uint) (map[string]any, error) {
	return map[string]any{}, nil
}

func (f *workspaceUseCaseFake) SetViewState(_ context.Context, _, _ uint, state map[string]any) (map[string]any, error) {
	return state, nil
}

func montarRotaDeMarca(fake *workspaceUseCaseFake) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("language", "pt-BR")
	})
	h := NewWorkspaceHandler(fake)
	r.PUT("/collections/:id/highlights/:paragraphNodeId", h.SetHighlight)
	return r
}

func TestSetHighlightCorInvalidaDevolve400Traduzido(t *testing.T) {
	fake := &workspaceUseCaseFake{}
	r := montarRotaDeMarca(fake)

	req := httptest.NewRequest(http.MethodPut, "/collections/7/highlights/5", strings.NewReader(`{"color":"magenta"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, quero 400; corpo: %s", w.Code, w.Body.String())
	}
	var corpo map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo nao e JSON: %v", err)
	}
	if msg, _ := corpo["message"].(string); !strings.Contains(msg, "amarelo") {
		t.Fatalf("mensagem devia listar as cores validas, veio %q", msg)
	}
}

func TestSetHighlightSemCorNoCorpoDevolve400SemChamarOUsecase(t *testing.T) {
	fake := &workspaceUseCaseFake{}
	r := montarRotaDeMarca(fake)

	req := httptest.NewRequest(http.MethodPut, "/collections/7/highlights/5", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, quero 400", w.Code)
	}
	if fake.setCalls != 0 {
		t.Fatalf("usecase foi chamado %d vez(es) com corpo invalido", fake.setCalls)
	}
}

func TestSetHighlightCorValidaDevolveAMarca(t *testing.T) {
	r := montarRotaDeMarca(&workspaceUseCaseFake{})
	req := httptest.NewRequest(http.MethodPut, "/collections/7/highlights/5", strings.NewReader(`{"color":"verde"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, quero 200; corpo: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"color":"verde"`) || !strings.Contains(w.Body.String(), `"paragraph_node_id":5`) {
		t.Fatalf("corpo sem a marca: %s", w.Body.String())
	}
}
