package dto

import (
	"encoding/json"
	"testing"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

// internal/dto tinha 0,0% de cobertura -- e e a fronteira entre a API e o
// dominio, onde o modo de falha e o mais silencioso que existe: um campo
// esquecido no mapeamento NAO quebra build nem teste, ele simplesmente some da
// resposta.
//
// Ja aconteceu neste projeto: o backend enviava node_id nos nos do sumario e o
// normalizador do frontend descartava o campo. A interface exibia a estrutura
// mas nao conseguia perguntar "com o que este trecho se relaciona" -- e o
// defeito so apareceu quando a feature de conexoes precisou do campo.
//
// Os testes cobrem o que quebra tela, nao todos os campos por completude:
// ponteiro nulo, slice vazio virando null, e campo consumido pelo frontend.

func TestSliceVazioSerializaComoListaNaoComoNulo(t *testing.T) {
	// O frontend faz .map() direto no resultado. `null` em vez de `[]` estoura
	// com "Cannot read properties of null" -- erro que aparece no navegador do
	// usuario, nao no teste do backend.
	casos := map[string]any{
		"categorias":                        ToCategoriesResponse(nil),
		"colecoes":                          ToCollectionsResponse(nil),
		"eventos de auditoria de documento": ToDocumentAuditEventResponses(nil),
		"eventos de auditoria de provider":  ToProviderAuditEventResponses(nil),
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
			if string(bruto) != "[]" {
				t.Fatalf("esperava lista vazia, veio %s", bruto)
			}
		})
	}
}

func TestConversoresDevolvemNuloParaEntradaNula(t *testing.T) {
	// Entrada nula acontece em relacionamento nao carregado pelo GORM. O
	// conversor precisa devolver nil, nunca entrar em panico -- panico no
	// handler vira 500 para o usuario.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("conversor entrou em panico com entrada nula: %v", r)
		}
	}()

	if ToCategoryResponse(nil) != nil {
		t.Error("ToCategoryResponse deveria devolver nil")
	}
	if ToCollectionResponse(nil) != nil {
		t.Error("ToCollectionResponse deveria devolver nil")
	}
	if ToDocumentResponse(nil) != nil {
		t.Error("ToDocumentResponse deveria devolver nil")
	}
	if ToDocumentAuditEventResponse(nil) != nil {
		t.Error("ToDocumentAuditEventResponse deveria devolver nil")
	}
	if ToProviderAuditEventResponse(nil) != nil {
		t.Error("ToProviderAuditEventResponse deveria devolver nil")
	}
}

func TestMetadataNulaViraObjetoVazio(t *testing.T) {
	// Mesma razao do slice: o frontend acessa metadata.x sem checar nulo.
	resposta := ToCollectionResponse(&models.Collection{
		ID:       1,
		UserID:   2,
		Name:     "Contratos",
		Metadata: nil,
	})

	if resposta == nil {
		t.Fatal("conversao devolveu nil para colecao valida")
	}
	if resposta.Metadata == nil {
		t.Fatal("metadata nula deveria virar objeto vazio, nao nil")
	}

	bruto, _ := json.Marshal(resposta.Metadata)
	if string(bruto) != "{}" {
		t.Fatalf("metadata vazia serializou como %s", bruto)
	}
}

func TestColecaoSemCategoriaNaoInventaCategoria(t *testing.T) {
	// Colecao sem categoria e estado legitimo. Inventar uma categoria vazia
	// faria a tela exibir um rotulo em branco no lugar de nao exibir nada.
	resposta := ToCollectionResponse(&models.Collection{
		ID:       1,
		UserID:   2,
		Name:     "Sem categoria",
		Category: nil,
	})

	if resposta.Category != nil {
		t.Fatalf("categoria inventada: %+v", resposta.Category)
	}
	if resposta.CategoryID != nil {
		t.Fatalf("id de categoria inventado: %v", *resposta.CategoryID)
	}
}

func TestDocumentoSemColecaoNaoQuebraAConversao(t *testing.T) {
	// CollectionRef vem nulo quando o Preload nao foi pedido -- caminho comum
	// em listagem. O conversor le doc.CollectionRef.Category e precisa
	// sobreviver a isso.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panico com documento sem colecao carregada: %v", r)
		}
	}()

	resposta := ToDocumentResponse(&models.Document{
		ID:            7,
		UserID:        2,
		Title:         "Contrato.pdf",
		MimeType:      "application/pdf",
		CollectionRef: nil,
	})

	if resposta == nil {
		t.Fatal("conversao devolveu nil para documento valido")
	}
	if resposta.Title != "Contrato.pdf" {
		t.Fatalf("titulo perdido na conversao: %q", resposta.Title)
	}
}

func TestCamposQueOFrontendConsomeSobrevivemAConversao(t *testing.T) {
	// Cada campo aqui e lido por alguma tela. Remover um do mapeamento nao
	// quebraria build nem teste sem esta asserção -- so sumiria da interface.
	criadoEm := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)

	doc := &models.Document{
		ID:        42,
		UserID:    7,
		Title:     "Edital.pdf",
		FileURL:   "documentos/edital.pdf",
		Size:      2048,
		MimeType:  "application/pdf",
		Status:    "INDEXED",
		CreatedAt: criadoEm,
	}

	resposta := ToDocumentResponse(doc)
	if resposta == nil {
		t.Fatal("conversao devolveu nil")
	}

	verificacoes := []struct {
		campo    string
		obtido   any
		esperado any
	}{
		{"ID", resposta.ID, uint(42)},
		{"UserID", resposta.UserID, uint(7)},
		{"Title", resposta.Title, "Edital.pdf"},
		{"Size", resposta.Size, int64(2048)},
		{"MimeType", resposta.MimeType, "application/pdf"},
		{"Status", resposta.Status, "INDEXED"},
	}

	for _, v := range verificacoes {
		if v.obtido != v.esperado {
			t.Errorf("campo %s: esperado %v, veio %v", v.campo, v.esperado, v.obtido)
		}
	}

	// FilePath e FileURL apontam para o mesmo valor de proposito (compatibilidade
	// entre versoes do cliente); se divergirem, uma das telas quebra.
	if resposta.FilePath != resposta.FileURL {
		t.Errorf("FilePath (%q) divergiu de FileURL (%q)", resposta.FilePath, resposta.FileURL)
	}
}

func TestResultadoDeAuditoriaVazioNaoViraNulo(t *testing.T) {
	resposta := ToDocumentAuditQueryResponse(&domain.DocumentAuditQueryResult{})
	if resposta == nil {
		t.Fatal("conversao devolveu nil para resultado vazio")
	}

	bruto, err := json.Marshal(resposta.Items)
	if err != nil {
		t.Fatalf("serializacao falhou: %v", err)
	}
	if string(bruto) == "null" {
		t.Fatal("lista de eventos serializou como null: a tabela de auditoria quebra ao renderizar")
	}
}

func TestConversaoEmLotePreservaAOrdem(t *testing.T) {
	// A ordenacao vem do banco e carrega significado (mais recente primeiro).
	// Embaralhar na conversao mostraria eventos fora de ordem cronologica sem
	// nenhum erro visivel.
	categorias := []models.DocumentCategory{
		{ID: 1, Name: "Primeira"},
		{ID: 2, Name: "Segunda"},
		{ID: 3, Name: "Terceira"},
	}

	resposta := ToCategoriesResponse(categorias)
	if len(resposta) != 3 {
		t.Fatalf("esperava 3 categorias, veio %d", len(resposta))
	}

	for i, esperado := range []string{"Primeira", "Segunda", "Terceira"} {
		if resposta[i].Name != esperado {
			t.Fatalf("posicao %d: esperado %q, veio %q", i, esperado, resposta[i].Name)
		}
	}
}
