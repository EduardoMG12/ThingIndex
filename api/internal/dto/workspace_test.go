package dto

import (
	"encoding/json"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

func TestConexaoDerivadaPreservaAJustificativa(t *testing.T) {
	resposta := ToNodeConnectionsResponse("node-001-2-penalidades", 2, []domain.NodeConnection{
		{
			DocumentID:   9,
			NodeID:       "node-002-2-das-penalidades",
			Title:        "2 DAS PENALIDADES",
			Score:        11,
			MatchedTerms: []string{"multa", "fatura", "descumprimento"},
		},
	})

	if resposta.NodeID != "node-001-2-penalidades" {
		t.Fatalf("no de origem perdido: %q", resposta.NodeID)
	}
	if resposta.MinScore != 2 {
		t.Fatalf("limiar perdido: %d -- o cliente nao sabe com que corte a lista foi produzida", resposta.MinScore)
	}
	if len(resposta.Connections) != 1 {
		t.Fatalf("esperava 1 conexao, veio %d", len(resposta.Connections))
	}

	conexao := resposta.Connections[0]
	if conexao.Score != 11 {
		t.Fatalf("score perdido: %d", conexao.Score)
	}
	if len(conexao.MatchedTerms) != 3 {
		t.Fatalf("justificativa perdida: %v -- a aresta deixa de ser auditavel", conexao.MatchedTerms)
	}
	if conexao.MatchedTerms[0] != "multa" {
		t.Fatalf("termos corrompidos: %v", conexao.MatchedTerms)
	}
}

func TestListaDeConexoesVaziaNaoViraNula(t *testing.T) {
	// Sem conexao a tela mostra "nenhuma conexao acima do limiar"; `null` quebraria antes.
	resposta := ToNodeConnectionsResponse("node-001", 99, nil)

	if resposta.Connections == nil {
		t.Fatal("lista nula: a tela nao consegue distinguir 'nenhuma conexao' de 'falha ao carregar'")
	}

	bruto, _ := json.Marshal(resposta.Connections)
	if string(bruto) != "[]" {
		t.Fatalf("lista vazia serializou como %s", bruto)
	}
}

func TestContextoDaColecaoNuloDevolveNulo(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panico com contexto nulo: %v", r)
		}
	}()

	if ToCollectionContextResponse(nil) != nil {
		t.Fatal("deveria devolver nil para contexto nulo")
	}
}

func TestContextoDaColecaoVazioNaoQuebraATela(t *testing.T) {
	// Colecao sem documento indexado: null em qualquer lista quebra o Workspace.
	resposta := ToCollectionContextResponse(&domain.CollectionContext{
		Collection: &models.Collection{ID: 1, UserID: 2, Name: "Vazia"},
	})

	if resposta == nil {
		t.Fatal("conversao devolveu nil para contexto valido")
	}

	bruto, err := json.Marshal(resposta)
	if err != nil {
		t.Fatalf("serializacao falhou: %v", err)
	}

	var mapa map[string]any
	if err := json.Unmarshal(bruto, &mapa); err != nil {
		t.Fatalf("JSON invalido: %v", err)
	}

	// Os campos que o Workspace percorre com .map.
	for _, chave := range []string{"documents", "document_pages", "outline_nodes"} {
		valor, existe := mapa[chave]
		if !existe {
			continue // omitempty e aceitavel; null nao e
		}
		if valor == nil {
			t.Errorf("campo %q veio null: o .map() do Workspace quebra", chave)
		}
	}
}

func TestContextoDaColecaoPreservaOsNosDoSumario(t *testing.T) {
	resposta := ToCollectionContextResponse(&domain.CollectionContext{
		Collection: &models.Collection{ID: 1, UserID: 2, Name: "Contratos"},
		OutlineNodes: []models.DocumentOutlineNode{
			{ID: 10, DocumentID: 7, NodeID: "node-001-obrigacoes", Title: "1 DAS OBRIGACOES"},
			{ID: 11, DocumentID: 7, NodeID: "node-002-penalidades", Title: "2 DAS PENALIDADES"},
		},
	})

	if len(resposta.OutlineNodes) != 2 {
		t.Fatalf("esperava 2 nos, veio %d", len(resposta.OutlineNodes))
	}
	for i, esperado := range []string{"node-001-obrigacoes", "node-002-penalidades"} {
		if resposta.OutlineNodes[i].NodeID != esperado {
			t.Fatalf("posicao %d: node_id %q, esperado %q",
				i, resposta.OutlineNodes[i].NodeID, esperado)
		}
	}
}
