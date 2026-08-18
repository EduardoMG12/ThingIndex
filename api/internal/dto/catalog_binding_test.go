package dto_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"thing-index/internal/dto"

	"github.com/gin-gonic/gin"
)

// Regressao: os DTOs de escrita em lote declaravam apenas `binding:"required"`
// na fatia, sem `dive`. Sem `dive` o validator nao desce nos elementos, entao
// as constraints de ParagraphNodeInput/TableSectionInput/MetadataPairInput
// nunca eram avaliadas: dava para gravar sequence_order negativo, page_number
// negativo e content vazio direto nas tabelas estruturadas.
//
// Isso importa porque a ordenacao (ORDER BY sequence_order) e a proveniencia
// dependem justamente desses campos. Estes testes travam o comportamento.

func bindJSON[T any](t *testing.T, payload string) error {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")

	var target T
	return ctx.ShouldBindJSON(&target)
}

func TestStoreParagraphNodesRejectsInvalidElements(t *testing.T) {
	testCases := []struct {
		name    string
		payload string
	}{
		{
			name:    "sequence order negativa",
			payload: `{"nodes":[{"sequence_order":-7,"content":"texto"}]}`,
		},
		{
			name:    "page number negativo",
			payload: `{"nodes":[{"sequence_order":0,"page_number":-99,"content":"texto"}]}`,
		},
		{
			name:    "conteudo vazio",
			payload: `{"nodes":[{"sequence_order":0,"content":""}]}`,
		},
		{
			name:    "elemento invalido depois de um valido",
			payload: `{"nodes":[{"sequence_order":0,"content":"ok"},{"sequence_order":-1,"content":"ruim"}]}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if err := bindJSON[dto.StoreParagraphNodesRequest](t, testCase.payload); err == nil {
				t.Fatalf("payload invalido foi aceito: %s", testCase.payload)
			}
		})
	}
}

func TestStoreParagraphNodesAcceptsSequenceOrderZero(t *testing.T) {
	// sequence_order = 0 e o valor legitimo do primeiro no. A tag anterior
	// (`required,gte=0`) rejeitava zero, porque `required` trata o valor zero
	// de um int como ausente. Trocada por `gte=0` sozinha.
	payload := `{"nodes":[{"sequence_order":0,"page_number":1,"content":"primeiro paragrafo"}]}`

	if err := bindJSON[dto.StoreParagraphNodesRequest](t, payload); err != nil {
		t.Fatalf("sequence_order zero deveria ser aceito, veio: %v", err)
	}
}

func TestStoreTableSectionsRejectsInvalidElements(t *testing.T) {
	payload := `{"sections":[{"sequence_order":-3,"title":"tabela"}]}`

	if err := bindJSON[dto.StoreTableSectionsRequest](t, payload); err == nil {
		t.Fatal("sequence_order negativa em table section foi aceita")
	}
}

func TestStoreMetadataPairsRejectsInvalidElements(t *testing.T) {
	testCases := []struct {
		name    string
		payload string
	}{
		{
			name:    "chave vazia",
			payload: `{"pairs":[{"sequence_order":0,"key":"","value":"v"}]}`,
		},
		{
			name:    "valor vazio",
			payload: `{"pairs":[{"sequence_order":0,"key":"k","value":""}]}`,
		},
		{
			name:    "sequence order negativa",
			payload: `{"pairs":[{"sequence_order":-1,"key":"k","value":"v"}]}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if err := bindJSON[dto.StoreMetadataPairsRequest](t, testCase.payload); err == nil {
				t.Fatalf("payload invalido foi aceito: %s", testCase.payload)
			}
		})
	}
}

func TestStoreRequestsAcceptValidPayloads(t *testing.T) {
	// Guarda contra excesso de rigor: o `dive` nao pode passar a rejeitar
	// payload legitimo.
	nodes := dto.StoreParagraphNodesRequest{
		Nodes: []dto.ParagraphNodeInput{
			{SequenceOrder: 0, Content: "primeiro"},
			{SequenceOrder: 1, Content: "segundo"},
		},
	}
	encoded, err := json.Marshal(nodes)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := bindJSON[dto.StoreParagraphNodesRequest](t, string(encoded)); err != nil {
		t.Fatalf("payload valido foi rejeitado: %v", err)
	}
}
