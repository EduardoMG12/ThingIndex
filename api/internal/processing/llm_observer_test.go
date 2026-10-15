package processing

import (
	"context"
	"errors"
	"testing"

	"thing-index/internal/domain"
)

type stubLLMService struct {
	err   error
	calls int
}

func (s *stubLLMService) GenerateText(_ context.Context, _, _ string) (string, error) {
	s.calls++
	if s.err != nil {
		return "", s.err
	}
	return "resposta do modelo", nil
}

// Regressao: chamadas que falham tambem sao contadas.
func TestObservedLLMCountsFailedRequests(t *testing.T) {
	t.Parallel()

	base := &stubLLMService{err: errors.New("modelo aposentado")}
	observer := newObservedLLMService(base)

	for range 3 {
		if _, err := observer.GenerateText(context.Background(), "sistema", "usuario"); err == nil {
			t.Fatal("esperava erro do provider")
		}
	}

	usage := observer.Snapshot()
	if usage.RequestCount != 0 {
		t.Fatalf("RequestCount = %d, quero 0 (nenhuma completou)", usage.RequestCount)
	}
	if usage.FailedRequestCount != 3 {
		t.Fatalf("FailedRequestCount = %d, quero 3", usage.FailedRequestCount)
	}
	if usage.EstimatedTotalTokens != 0 {
		t.Fatalf("chamada falha nao pode estimar token: %d", usage.EstimatedTotalTokens)
	}
}

func TestObservedLLMCountsSuccessfulRequests(t *testing.T) {
	t.Parallel()

	observer := newObservedLLMService(&stubLLMService{})
	if _, err := observer.GenerateText(context.Background(), "sistema", "usuario"); err != nil {
		t.Fatalf("GenerateText: %v", err)
	}

	usage := observer.Snapshot()
	if usage.RequestCount != 1 || usage.FailedRequestCount != 0 {
		t.Fatalf("usage = %+v", usage)
	}
	if usage.EstimatedTotalTokens <= 0 {
		t.Fatal("esperava estimativa de tokens numa chamada bem-sucedida")
	}
}

var _ domain.LLMService = (*stubLLMService)(nil)
