package dto

import (
	"encoding/json"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

// Conversores de recuperacao e do indice do documento. Sao o payload da tela de
// documento e da busca -- as duas telas onde o trabalho demonstra a tese.

func TestRecuperacaoEmArvorePreservaAOrdemDoRanking(t *testing.T) {
	// A ordem E o resultado: embaralhar na conversao mostraria o quarto
	// colocado em primeiro sem nenhum erro visivel, e a demonstracao da tese
	// passaria a mentir na tela.
	resposta := ToTreeRetrievalResponse(&domain.TreeRetrievalResult{
		NodeHits: []domain.TreeNodeHit{
			{NodeID: "n1", Title: "Primeiro", Score: 30},
			{NodeID: "n2", Title: "Segundo", Score: 20},
			{NodeID: "n3", Title: "Terceiro", Score: 10},
		},
	})

	if resposta == nil {
		t.Fatal("conversao devolveu nil")
	}
	if len(resposta.NodeHits) != 3 {
		t.Fatalf("esperava 3 resultados, veio %d", len(resposta.NodeHits))
	}

	for i, esperado := range []string{"Primeiro", "Segundo", "Terceiro"} {
		if resposta.NodeHits[i].Title != esperado {
			t.Fatalf("posicao %d: %q, esperado %q", i, resposta.NodeHits[i].Title, esperado)
		}
	}

	// Score decrescente: se subir, a ordenacao foi invertida.
	if resposta.NodeHits[0].Score < resposta.NodeHits[2].Score {
		t.Fatalf("ranking invertido: %d antes de %d",
			resposta.NodeHits[0].Score, resposta.NodeHits[2].Score)
	}
}

func TestRecuperacaoSemResultadoNaoViraNula(t *testing.T) {
	// Consulta sem resultado e o caso que o trabalho mais precisa exibir com
	// clareza: os seis bracos tem taxa de falso positivo 1,000, e a tela poder
	// dizer "nao encontrei" e o contraponto.
	resposta := ToTreeRetrievalResponse(&domain.TreeRetrievalResult{})

	if resposta == nil {
		t.Fatal("conversao devolveu nil para resultado vazio")
	}

	bruto, _ := json.Marshal(resposta.NodeHits)
	if string(bruto) == "null" {
		t.Fatal("lista de resultados serializou como null: a tela nao distingue vazio de falha")
	}
}

func TestRecuperacaoNulaDevolveNulo(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panico com resultado nulo: %v", r)
		}
	}()

	if ToTreeRetrievalResponse(nil) != nil {
		t.Fatal("deveria devolver nil")
	}
}

func TestIndiceDoDocumentoAgregaTodasAsCamadas(t *testing.T) {
	// Este payload monta a tela de documento inteira. Uma camada nula quebra a
	// aba correspondente sem erro no backend.
	resposta := ToDocumentIndexResponse(&domain.DocumentIndex{
		Document: &models.Document{ID: 7, UserID: 2, Title: "Contrato.pdf"},
		OutlineNodes: []models.DocumentOutlineNode{
			{ID: 1, DocumentID: 7, NodeID: "n1", Title: "1 DAS OBRIGACOES"},
		},
		DocumentPages: []models.DocumentPage{
			{ID: 1, DocumentID: 7, PageNumber: 1, Content: "conteudo"},
		},
	})

	if resposta == nil {
		t.Fatal("conversao devolveu nil")
	}
	if resposta.Document == nil || resposta.Document.Title != "Contrato.pdf" {
		t.Fatal("documento perdido na agregacao")
	}
	if len(resposta.OutlineNodes) != 1 {
		t.Fatalf("estrutura perdida: %d nos", len(resposta.OutlineNodes))
	}
	if len(resposta.DocumentPages) != 1 {
		t.Fatalf("paginas perdidas: %d", len(resposta.DocumentPages))
	}

	// Camadas nao preenchidas precisam sair como lista, nao como null.
	bruto, err := json.Marshal(resposta)
	if err != nil {
		t.Fatalf("serializacao falhou: %v", err)
	}

	var mapa map[string]any
	_ = json.Unmarshal(bruto, &mapa)
	for _, chave := range []string{"chunks", "table_sections", "metadata_pairs"} {
		if valor, existe := mapa[chave]; existe && valor == nil {
			t.Errorf("camada %q veio null em vez de lista vazia", chave)
		}
	}
}

func TestIndiceNuloDevolveNulo(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panico com indice nulo: %v", r)
		}
	}()

	if ToDocumentIndexResponse(nil) != nil {
		t.Fatal("deveria devolver nil")
	}
}
