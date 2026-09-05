package dto

import (
	"encoding/json"
	"testing"

	"thing-index/internal/models"
)

// Conversores da estrutura extraida: sao os que alimentam a aba Workspace e a
// metrica de proveniencia do trabalho. Um campo perdido aqui nao quebra nada
// visivel no backend -- some da tela e da citacao.

func ponteiroInt(v int) *int { return &v }

func TestNodeIDSobreviveAConversaoDoSumario(t *testing.T) {
	// Este campo tem historia: o backend enviava node_id e o normalizador do
	// frontend descartava. A interface exibia a estrutura mas nao conseguia
	// pedir conexoes, porque a rota precisa do node_id como parametro.
	//
	// A cobertura do lado do frontend ja existe; esta e a metade do backend.
	resposta := ToOutlineNodeResponses([]models.DocumentOutlineNode{
		{ID: 1, DocumentID: 7, NodeID: "node-001-2-penalidades", Title: "2 DAS PENALIDADES", Level: 1},
	})

	if len(resposta) != 1 {
		t.Fatalf("esperava 1 no, veio %d", len(resposta))
	}
	if resposta[0].NodeID != "node-001-2-penalidades" {
		t.Fatalf("node_id perdido: %q -- a rota de conexoes deixa de funcionar", resposta[0].NodeID)
	}
	if resposta[0].Title != "2 DAS PENALIDADES" {
		t.Fatalf("titulo perdido: %q", resposta[0].Title)
	}

	// O JSON precisa carregar node_id: o cliente le a chave, nao o campo Go.
	bruto, _ := json.Marshal(resposta[0])
	var mapa map[string]any
	_ = json.Unmarshal(bruto, &mapa)
	if mapa["node_id"] != "node-001-2-penalidades" {
		t.Fatalf("node_id ausente no JSON: %s", bruto)
	}
}

func TestIntervaloDePaginaDoNoSobrevive(t *testing.T) {
	// start_page e end_page sustentam a proveniencia: sem eles a citacao nao
	// aponta para lugar nenhum, e a metrica TPC do trabalho mede zero.
	resposta := ToOutlineNodeResponses([]models.DocumentOutlineNode{
		{ID: 1, DocumentID: 7, NodeID: "n1", Title: "Secao", StartPage: ponteiroInt(3), EndPage: ponteiroInt(5)},
	})

	if resposta[0].StartPage == nil || *resposta[0].StartPage != 3 {
		t.Fatalf("pagina inicial perdida: %v", resposta[0].StartPage)
	}
	if resposta[0].EndPage == nil || *resposta[0].EndPage != 5 {
		t.Fatalf("pagina final perdida: %v", resposta[0].EndPage)
	}
}

func TestNoSemIntervaloDePaginaNaoInventaZero(t *testing.T) {
	// Nó sem página conhecida e estado legitimo (documento sem paginacao).
	// Virar 0 seria pior que nil: a tela exibiria "pagina 0" como se fosse
	// informacao real, e a metrica de proveniencia contaria como coberto.
	resposta := ToOutlineNodeResponses([]models.DocumentOutlineNode{
		{ID: 1, DocumentID: 7, NodeID: "n1", Title: "Sem pagina"},
	})

	if resposta[0].StartPage != nil {
		t.Fatalf("pagina inicial inventada: %d", *resposta[0].StartPage)
	}
	if resposta[0].EndPage != nil {
		t.Fatalf("pagina final inventada: %d", *resposta[0].EndPage)
	}

	// E o JSON precisa OMITIR o campo, nao mandar null.
	bruto, _ := json.Marshal(resposta[0])
	var mapa map[string]any
	_ = json.Unmarshal(bruto, &mapa)
	if _, existe := mapa["start_page"]; existe {
		t.Fatalf("start_page deveria ser omitido quando ausente: %s", bruto)
	}
}

func TestNumeroDePaginaSobreviveNaConversao(t *testing.T) {
	resposta := ToDocumentPageResponses([]models.DocumentPage{
		{ID: 1, DocumentID: 7, PageNumber: 4, Content: "conteudo da pagina quatro"},
	})

	if len(resposta) != 1 {
		t.Fatalf("esperava 1 pagina, veio %d", len(resposta))
	}
	if resposta[0].PageNumber != 4 {
		t.Fatalf("numero da pagina perdido: %d", resposta[0].PageNumber)
	}
	if resposta[0].Content != "conteudo da pagina quatro" {
		t.Fatalf("conteudo perdido: %q", resposta[0].Content)
	}
}

func TestMetadataNulaViraObjetoEmTodosOsConversoresDeEstrutura(t *testing.T) {
	// Todos estes sao lidos pela tela com acesso direto a metadata.x.
	casos := []struct {
		nome     string
		metadata map[string]any
	}{
		{"no de sumario", ToOutlineNodeResponses([]models.DocumentOutlineNode{{ID: 1, NodeID: "n"}})[0].Metadata},
		{"pagina", ToDocumentPageResponses([]models.DocumentPage{{ID: 1, PageNumber: 1}})[0].Metadata},
		{"chunk", ToDocumentChunkResponses([]models.DocumentChunk{{ID: 1}})[0].Metadata},
		{"paragrafo", ToParagraphNodeResponses([]models.ParagraphNode{{ID: 1}})[0].Metadata},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if caso.metadata == nil {
				t.Fatal("metadata nula: o acesso direto na tela quebra")
			}
			bruto, _ := json.Marshal(caso.metadata)
			if string(bruto) != "{}" {
				t.Fatalf("metadata vazia serializou como %s", bruto)
			}
		})
	}
}

func TestConversoresDeEstruturaDevolvemListaVaziaNaoNula(t *testing.T) {
	casos := map[string]any{
		"nos de sumario": ToOutlineNodeResponses(nil),
		"paginas":        ToDocumentPageResponses(nil),
		"chunks":         ToDocumentChunkResponses(nil),
		"paragrafos":     ToParagraphNodeResponses(nil),
		"tabelas":        ToTableSectionResponses(nil),
		"metadados":      ToMetadataPairResponses(nil),
		"documentos":     ToDocumentsResponse(nil),
	}

	for nome, valor := range casos {
		t.Run(nome, func(t *testing.T) {
			bruto, err := json.Marshal(valor)
			if err != nil {
				t.Fatalf("serializacao falhou: %v", err)
			}
			if string(bruto) == "null" {
				t.Fatalf("%s serializou como null: o .map() do frontend quebra", nome)
			}
		})
	}
}

func TestCabecalhoDeTabelaNuloViraListaVazia(t *testing.T) {
	// A tabela e renderizada em <thead> percorrendo headers. Nulo quebra a
	// tela inteira do documento, nao apenas a tabela.
	resposta := ToTableSectionResponses([]models.TableSection{
		{ID: 1, DocumentID: 7, Headers: nil},
	})

	if resposta[0].Headers == nil {
		t.Fatal("cabecalho nulo: a renderizacao da tabela quebra")
	}
	bruto, _ := json.Marshal(resposta[0].Headers)
	if string(bruto) != "[]" {
		t.Fatalf("cabecalho vazio serializou como %s", bruto)
	}
}

func TestParDeMetadadoPreservaChaveEValor(t *testing.T) {
	// ocr_engine e page_count sao lidos daqui, e a metrica de proveniencia do
	// artigo depende de conseguir distinguir os tipos.
	resposta := ToMetadataPairResponses([]models.MetadataPair{
		{ID: 1, DocumentID: 7, Key: "ocr_engine", Value: "unlimited_ocr"},
	})

	if len(resposta) != 1 {
		t.Fatalf("esperava 1 par, veio %d", len(resposta))
	}
	if resposta[0].Key != "ocr_engine" || resposta[0].Value != "unlimited_ocr" {
		t.Fatalf("par corrompido: %q=%q", resposta[0].Key, resposta[0].Value)
	}
}
