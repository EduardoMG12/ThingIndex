package usecase

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"thing-index/internal/models"
)

// Leitura e validacao acontecem uma vez, em NewRetrievalOptions.
func TestValorInvalidoDePageSearchFieldsAvisaUmaVezSo(t *testing.T) {
	var saida bytes.Buffer
	anterior := log.Writer()
	log.SetOutput(&saida)
	t.Cleanup(func() { log.SetOutput(anterior) })

	opts := NewRetrievalOptions("conteudo-errado", "")
	if opts.PageSearchFields != pageFieldsContentSummary {
		t.Fatalf("valor invalido deveria cair no padrao %q, veio %q", pageFieldsContentSummary, opts.PageSearchFields)
	}

	pages := make([]models.DocumentPage, 0, 10)
	for i := 1; i <= 10; i++ {
		pages = append(pages, models.DocumentPage{DocumentID: 1, PageNumber: i, Content: "multa contratual por atraso", Summary: "clausula de multa"})
	}
	for range 3 {
		buildCollectionRetrievalWith(opts, "multa por atraso", []models.Document{{ID: 1, Title: "Contrato"}}, pages, nil, nil, nil)
	}

	if avisos := strings.Count(saida.String(), "THINGINDEX_PAGE_SEARCH_FIELDS"); avisos != 1 {
		t.Fatalf("esperava 1 aviso de valor invalido, veio %d:\n%s", avisos, saida.String())
	}
}

func TestRetrievalOptionsLeOAmbienteComOsMesmosPadroes(t *testing.T) {
	casos := []struct {
		fields, expansion string
		wantFields        string
		wantExpansion     bool
	}{
		{"", "", pageFieldsContentSummary, false},
		{" content ", "on", pageFieldsContent, true},
		{"content+questions", "TRUE", pageFieldsContentQuestions, true},
		{"max(content,questions)", "1", pageFieldsMaxQuestions, true},
		{"content+summary", "off", pageFieldsContentSummary, false},
	}
	for _, caso := range casos {
		t.Setenv("THINGINDEX_PAGE_SEARCH_FIELDS", caso.fields)
		t.Setenv("THINGINDEX_QUERY_EXPANSION", caso.expansion)
		got := RetrievalOptionsFromEnv()
		if got.PageSearchFields != caso.wantFields || got.QueryExpansion != caso.wantExpansion {
			t.Errorf("(%q, %q) -> %+v, esperava fields=%q expansion=%v", caso.fields, caso.expansion, got, caso.wantFields, caso.wantExpansion)
		}
	}
}

// O valor zero da struct (casos de uso montados a mao nos testes) precisa se
// comportar como o padrao, e nao como "content".
func TestRetrievalOptionsZeroEquivaleAoPadrao(t *testing.T) {
	page := models.DocumentPage{Content: "conteudo", Summary: "resumo"}
	if got, want := pageSearchText(page, RetrievalOptions{}.pageSearchFields()), pageSearchText(page, DefaultRetrievalOptions().pageSearchFields()); got != want {
		t.Fatalf("valor zero %q difere do padrao %q", got, want)
	}
}

// A opcao precisa mudar o texto pontuado: termo so do resumo acha a pagina no
// padrao e nao com "content".
func TestRetrievalOptionsControlaOCampoPontuado(t *testing.T) {
	docs := []models.Document{{ID: 1, Title: "Contrato"}}
	pages := []models.DocumentPage{
		{DocumentID: 1, PageNumber: 1, Content: "texto sem o termo procurado", Summary: "penalidade"},
		{DocumentID: 1, PageNumber: 2, Content: "outro texto qualquer", Summary: "outro resumo"},
	}

	temPagina := func(opts RetrievalOptions) bool {
		result := buildCollectionRetrievalWith(opts, "penalidade", docs, pages, nil, nil, nil)
		return result != nil && len(result.PageHits) > 0 && result.PageHits[0].PageNumber == 1
	}

	if !temPagina(DefaultRetrievalOptions()) {
		t.Fatal("no padrao (content+summary) o termo do resumo deveria achar a pagina 1")
	}
	if temPagina(NewRetrievalOptions("content", "")) {
		t.Fatal("com PageSearchFields=content o resumo nao deveria ser pontuado")
	}
}
