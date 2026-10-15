package dto

import (
	"encoding/json"
	"testing"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

func TestEventoDeAuditoriaPreservaCorrelacaoEStatus(t *testing.T) {
	// correlation_id liga o upload a indexacao e ao erro.
	quando := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)

	resposta := ToDocumentAuditEventResponse(&models.DocumentAuditEvent{
		ID:            1,
		UserID:        7,
		Action:        "DOCUMENT_UPLOAD",
		Status:        "SUCCESS",
		CorrelationID: "req-abc-123",
		CreatedAt:     quando,
	})

	if resposta == nil {
		t.Fatal("conversao devolveu nil para evento valido")
	}
	if resposta.CorrelationID != "req-abc-123" {
		t.Fatalf("correlacao perdida: %q -- a trilha deixa de ligar os eventos", resposta.CorrelationID)
	}
	if resposta.Action != "DOCUMENT_UPLOAD" || resposta.Status != "SUCCESS" {
		t.Fatalf("acao/status corrompidos: %q/%q", resposta.Action, resposta.Status)
	}
}

func TestResumoDeAuditoriaDeProviderSobrevive(t *testing.T) {
	resposta := ToProviderAuditQueryResponse(&domain.ProviderAuditQueryResult{
		Summary: domain.ProviderAuditSummary{
			TotalEvents:     42,
			UniqueProviders: 3,
			UniqueRequests:  17,
		},
	})

	if resposta == nil {
		t.Fatal("conversao devolveu nil")
	}
	if resposta.Summary.TotalEvents != 42 {
		t.Fatalf("total de eventos perdido: %d", resposta.Summary.TotalEvents)
	}
	if resposta.Summary.UniqueProviders != 3 {
		t.Fatalf("contagem de providers perdida: %d", resposta.Summary.UniqueProviders)
	}

	// Os contadores alimentam cartoes na tela; null quebra a renderizacao.
	bruto, _ := json.Marshal(resposta.Items)
	if string(bruto) == "null" {
		t.Fatal("lista de eventos serializou como null")
	}
}

func TestPaginacaoDeAuditoriaPreservaOCursor(t *testing.T) {
	// Sem next_cursor o "carregar mais" nao funciona.
	resposta := ToDocumentAuditQueryResponse(&domain.DocumentAuditQueryResult{
		PageInfo: domain.AuditPageInfo{
			Limit:      12,
			Returned:   12,
			HasNext:    true,
			NextCursor: "cursor-xyz",
		},
	})

	if !resposta.PageInfo.HasNext {
		t.Fatal("has_next perdido: o botao de carregar mais some")
	}
	if resposta.PageInfo.NextCursor != "cursor-xyz" {
		t.Fatalf("cursor perdido: %q", resposta.PageInfo.NextCursor)
	}
	if resposta.PageInfo.Limit != 12 {
		t.Fatalf("limite perdido: %d", resposta.PageInfo.Limit)
	}
}

func TestConversaDeChatNulaDevolveNulo(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panico com conversa nula: %v", r)
		}
	}()

	if ToChatConversationResponse(nil) != nil {
		t.Fatal("deveria devolver nil")
	}
}

func TestConversaDeChatPreservaAColecaoEOTitulo(t *testing.T) {
	resposta := ToChatConversationResponse(&models.ChatConversation{
		ID:           5,
		CollectionID: 9,
		Title:        "Duvidas sobre o contrato",
	})

	if resposta.CollectionID != 9 {
		t.Fatalf("colecao perdida: %d -- a conversa responderia sobre outro acervo", resposta.CollectionID)
	}
	if resposta.Title != "Duvidas sobre o contrato" {
		t.Fatalf("titulo perdido: %q", resposta.Title)
	}
}

func TestListasDeChatVaziasNaoViramNulas(t *testing.T) {
	casos := map[string]any{
		"conversas": ToChatConversationResponses(nil),
		"mensagens": ToChatConversationMessageResponses(nil),
	}

	for nome, valor := range casos {
		t.Run(nome, func(t *testing.T) {
			bruto, err := json.Marshal(valor)
			if err != nil {
				t.Fatalf("serializacao falhou: %v", err)
			}
			if string(bruto) == "null" {
				t.Fatalf("%s serializou como null: a tela de chat quebra ao renderizar", nome)
			}
		})
	}
}

func TestOrdemDasConversasEhPreservada(t *testing.T) {
	// A ordenacao vem do banco (mais recente primeiro) e carrega significado.
	resposta := ToChatConversationResponses([]models.ChatConversation{
		{ID: 3, Title: "Mais recente"},
		{ID: 2, Title: "Do meio"},
		{ID: 1, Title: "Mais antiga"},
	})

	if len(resposta) != 3 {
		t.Fatalf("esperava 3 conversas, veio %d", len(resposta))
	}
	for i, esperado := range []string{"Mais recente", "Do meio", "Mais antiga"} {
		if resposta[i].Title != esperado {
			t.Fatalf("posicao %d: %q, esperado %q", i, resposta[i].Title, esperado)
		}
	}
}
