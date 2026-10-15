package indexing

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

func TestParseLLMJSONExtractsFencedPayload(t *testing.T) {
	t.Parallel()

	raw := "```json\n{\"document_summary\":\"Resumo\",\"keywords\":[\"contrato\"],\"document_purpose\":\"Registrar obrigacoes.\"}\n```"

	var payload documentSynopsisResponse
	if err := parseLLMJSON(raw, &payload); err != nil {
		t.Fatalf("parseLLMJSON returned error: %v", err)
	}

	if payload.DocumentSummary != "Resumo" {
		t.Fatalf("unexpected document summary: %q", payload.DocumentSummary)
	}
	if len(payload.Keywords) != 1 || payload.Keywords[0] != "contrato" {
		t.Fatalf("unexpected keywords: %#v", payload.Keywords)
	}
}

func TestParseLLMJSONExtractsJSONAmidWrapperText(t *testing.T) {
	t.Parallel()

	raw := "Aqui esta o resultado solicitado:\n{\"pages\":[{\"page_number\":1,\"summary\":\"Resumo da pagina.\"}]}\nUse apenas o objeto."

	var payload pageSummaryResponse
	if err := parseLLMJSON(raw, &payload); err != nil {
		t.Fatalf("parseLLMJSON returned error: %v", err)
	}

	if len(payload.Pages) != 1 {
		t.Fatalf("expected one page summary, got %d", len(payload.Pages))
	}
	if payload.Pages[0].PageNumber != 1 {
		t.Fatalf("unexpected page number: %d", payload.Pages[0].PageNumber)
	}
	if payload.Pages[0].Summary != "Resumo da pagina." {
		t.Fatalf("unexpected page summary: %q", payload.Pages[0].Summary)
	}
}

func TestFillMissingPageSummariesUsesContentFallback(t *testing.T) {
	t.Parallel()

	result := domain.IndexingResult{
		DocumentPages: []models.DocumentPage{
			{PageNumber: 1, Content: "Contrato de manutencao predial com atendimento emergencial em ate 4 horas."},
			{PageNumber: 2, Content: "Pagina com resumo existente.", Summary: "Ja preenchido."},
		},
	}

	fillMissingPageSummaries(&result)

	if result.DocumentPages[0].Summary == "" {
		t.Fatal("expected fallback summary for page without llm summary")
	}
	if result.DocumentPages[1].Summary != "Ja preenchido." {
		t.Fatalf("expected existing summary to be preserved, got %q", result.DocumentPages[1].Summary)
	}
}

func TestBuildFallbackDocumentSynopsisUsesIndexedContent(t *testing.T) {
	t.Parallel()

	synopsis := buildFallbackDocumentSynopsis(&models.Document{Title: "contrato-servicos.html"}, domain.IndexingResult{
		DocumentPages: []models.DocumentPage{
			{PageNumber: 1, Content: "Contrato de prestacao de servicos com manutencao preventiva mensal e atendimento emergencial em ate 4 horas."},
		},
		OutlineNodes: []models.DocumentOutlineNode{
			{Title: "Escopo"},
		},
	})

	if synopsis.DocumentSummary == "" {
		t.Fatal("expected fallback document summary")
	}
	if synopsis.DocumentPurpose == "" {
		t.Fatal("expected fallback document purpose")
	}
	if len(synopsis.Keywords) == 0 {
		t.Fatal("expected fallback keywords")
	}
}

func TestLLMEnricherFallsBackToPageSegmentsWhenTreeBuildFails(t *testing.T) {
	t.Parallel()

	startPage := 1
	endPage := 3
	result, err := NewLLMEnricher().Enrich(
		context.Background(),
		stubFailingLLM{},
		&models.Document{ID: 1, Title: "contract.pdf"},
		domain.IndexingResult{
			DocumentPages: []models.DocumentPage{
				{DocumentID: 1, PageNumber: 1, Content: "First page explains the contract overview."},
				{DocumentID: 1, PageNumber: 2, Content: "Second page adds operational details."},
				{DocumentID: 1, PageNumber: 3, Content: "Third page closes with appendix notes."},
			},
			OutlineNodes: []models.DocumentOutlineNode{
				{
					DocumentID:    1,
					SequenceOrder: 0,
					NodeID:        "root",
					Level:         1,
					Title:         "Contract",
					Summary:       "General overview",
					StartPage:     &startPage,
					EndPage:       &endPage,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Enrich() returned error: %v", err)
	}

	if len(result.OutlineNodes) != 3 {
		t.Fatalf("expected fallback outline nodes per page, got %#v", result.OutlineNodes)
	}
	if result.OutlineNodes[0].Title != "Page 1" || result.OutlineNodes[1].Title != "Page 2" {
		t.Fatalf("unexpected fallback titles: %#v", result.OutlineNodes)
	}
	if got := lastMetadataValue(result.MetadataPairs, "tree_builder"); got != conservativeTreeFallbackBuilder {
		t.Fatalf("expected fallback builder metadata, got %q", got)
	}
	if got := lastMetadataValue(result.MetadataPairs, "tree_fallback_reason"); got != "shallow_tree" {
		t.Fatalf("expected fallback reason shallow_tree, got %q", got)
	}
	if got := lastMetadataValue(result.MetadataPairs, "tree_fallback_applied"); got != "true" {
		t.Fatalf("expected fallback applied metadata, got %q", got)
	}
}

func lastMetadataValue(pairs []models.MetadataPair, key string) string {
	value := ""
	for _, pair := range pairs {
		if pair.Key == key {
			value = pair.Value
		}
	}
	return value
}

// stubBatchLLM responde resumos para todas as paginas e secoes pedidas no
// prompt e conta as chamadas.
type stubBatchLLM struct {
	calls *int
}

func (s stubBatchLLM) GenerateText(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	*s.calls++
	if strings.Contains(userPrompt, "Pages:\n") {
		var pages []string
		for _, line := range strings.Split(userPrompt, "\n") {
			if n, ok := strings.CutPrefix(line, "- page_number="); ok {
				num, _, _ := strings.Cut(n, ",")
				pages = append(pages, `{"page_number":`+num+`,"summary":"Resumo da pagina `+num+` com os prazos e nomes citados nela.","topics":["prazo","matricula"]}`)
			}
		}
		return `{"pages":[` + strings.Join(pages, ",") + `]}`, nil
	}
	if strings.Contains(userPrompt, "Sections:\n") {
		var nodes []string
		for _, line := range strings.Split(userPrompt, "\n") {
			if n, ok := strings.CutPrefix(line, "- sequence_order="); ok {
				num, _, _ := strings.Cut(n, ",")
				nodes = append(nodes, `{"sequence_order":`+num+`,"summary":"Resumo da secao `+num+`."}`)
			}
		}
		return `{"nodes":[` + strings.Join(nodes, ",") + `]}`, nil
	}
	return "", errors.New("prompt nao suportado pelo stub")
}

func TestEnriquecimentoResumeTodasAsPaginasESecoesEmLotes(t *testing.T) {
	t.Parallel()

	var pages []models.DocumentPage
	var nodes []models.DocumentOutlineNode
	for i := 1; i <= 20; i++ {
		content := strings.Repeat("Texto da pagina com prazos, nomes e regras suficientes para resumir. ", 4)
		if i == 20 {
			content = "curta" // abaixo do minimo: fica no fallback
		}
		pages = append(pages, models.DocumentPage{DocumentID: 1, PageNumber: i, Content: content})
	}
	for i := 0; i < 15; i++ {
		start := i + 1
		nodes = append(nodes, models.DocumentOutlineNode{DocumentID: 1, SequenceOrder: i, NodeID: "n" + strconv.Itoa(i), Level: 1, Title: "Secao " + strconv.Itoa(i), StartPage: &start, EndPage: &start})
	}
	calls := 0
	result, err := NewLLMEnricher().Enrich(context.Background(), stubBatchLLM{calls: &calls}, &models.Document{ID: 1, Title: "ppc.pdf"}, domain.IndexingResult{DocumentPages: pages, OutlineNodes: nodes})
	if err != nil {
		t.Fatalf("Enrich() erro: %v", err)
	}

	// 19 paginas com texto em lotes de 6 = 4 chamadas; 15 secoes em lotes de 12 = 2.
	if got := lastMetadataValue(result.MetadataPairs, "enrichment_pages_summarized"); got != "19" {
		t.Fatalf("paginas resumidas = %q, esperava 19", got)
	}
	if got := lastMetadataValue(result.MetadataPairs, "enrichment_sections_summarized"); got != "15" {
		t.Fatalf("secoes resumidas = %q, esperava 15", got)
	}
	for _, page := range result.DocumentPages {
		if page.PageNumber == 20 {
			if strings.HasPrefix(page.Summary, "Resumo da pagina") {
				t.Fatalf("pagina curta nao devia ir a LLM: %q", page.Summary)
			}
			continue
		}
		if !strings.HasPrefix(page.Summary, "Resumo da pagina "+strconv.Itoa(page.PageNumber)) {
			t.Fatalf("pagina %d sem resumo da LLM: %q", page.PageNumber, page.Summary)
		}
		if got, _ := page.Metadata["topics"].(string); got != "prazo, matricula" {
			t.Fatalf("pagina %d sem topicos: %#v", page.PageNumber, page.Metadata)
		}
	}
	for _, node := range result.OutlineNodes {
		if node.Summary != "Resumo da secao "+strconv.Itoa(node.SequenceOrder)+"." {
			t.Fatalf("secao %d sem resumo da LLM: %q", node.SequenceOrder, node.Summary)
		}
	}
	// Perguntas de leigo somam um lote por lote de paginas. Numero travado de
	// proposito: orcamento de chamadas que ninguem olha cresce sozinho.
	if calls > 14 {
		t.Fatalf("chamadas = %d, esperava no maximo 14 (sinopse, sinais, arvore, lotes de secao, de pagina e de pergunta)", calls)
	}
}

// stubCache guarda resumos em memoria: prova que a segunda indexacao do
// mesmo texto nao chama a LLM para paginas nem secoes.
type stubCache struct {
	entries map[string]domain.EnrichmentCacheEntry
}

func stubCacheKey(scope domain.EnrichmentCacheScope, kind, hash string) string {
	return strconv.FormatUint(uint64(scope.UserID), 10) + "|" + scope.Provider + "|" + scope.Model + "|" + kind + "|" + hash
}

func (c *stubCache) LookupEnrichment(_ context.Context, scope domain.EnrichmentCacheScope, kind string, hashes []string) (map[string]domain.EnrichmentCacheEntry, error) {
	out := map[string]domain.EnrichmentCacheEntry{}
	for _, h := range hashes {
		if e, ok := c.entries[stubCacheKey(scope, kind, h)]; ok {
			out[h] = e
		}
	}
	return out, nil
}

func (c *stubCache) StoreEnrichment(_ context.Context, scope domain.EnrichmentCacheScope, entries []domain.EnrichmentCacheEntry) error {
	for _, e := range entries {
		c.entries[stubCacheKey(scope, e.Kind, e.ContentHash)] = e
	}
	return nil
}

func lotesDeTeste() ([]models.DocumentPage, []models.DocumentOutlineNode) {
	var pages []models.DocumentPage
	var nodes []models.DocumentOutlineNode
	for i := 1; i <= 12; i++ {
		pages = append(pages, models.DocumentPage{DocumentID: 1, PageNumber: i, Content: strings.Repeat("Texto da pagina "+strconv.Itoa(i)+" com prazos, nomes e regras suficientes para resumir. ", 4)})
	}
	for i := 0; i < 6; i++ {
		start := i + 1
		nodes = append(nodes, models.DocumentOutlineNode{DocumentID: 1, SequenceOrder: i, NodeID: "n" + strconv.Itoa(i), Level: 1, Title: "Secao " + strconv.Itoa(i), StartPage: &start, EndPage: &start})
	}
	return pages, nodes
}

func TestCacheEvitaChamadasNaSegundaIndexacao(t *testing.T) {
	t.Parallel()
	cache := &stubCache{entries: map[string]domain.EnrichmentCacheEntry{}}
	enricher := NewLLMEnricher().WithCache(cache)
	pages, nodes := lotesDeTeste()
	primeira := 0
	r1, err := enricher.Enrich(context.Background(), stubBatchLLM{calls: &primeira}, &models.Document{ID: 1, Title: "ppc.pdf"}, domain.IndexingResult{DocumentPages: pages, OutlineNodes: nodes})
	if err != nil {
		t.Fatal(err)
	}
	if got := lastMetadataValue(r1.MetadataPairs, "enrichment_cache_hits_pages"); got != "0" {
		t.Fatalf("primeira indexacao nao devia ter acerto de cache: %s", got)
	}
	segunda := 0
	pages2, nodes2 := lotesDeTeste()
	r2, err := enricher.Enrich(context.Background(), stubBatchLLM{calls: &segunda}, &models.Document{ID: 1, Title: "ppc.pdf"}, domain.IndexingResult{DocumentPages: pages2, OutlineNodes: nodes2})
	if err != nil {
		t.Fatal(err)
	}
	if got := lastMetadataValue(r2.MetadataPairs, "enrichment_cache_hits_pages"); got != "12" {
		t.Fatalf("paginas do cache = %s, esperava 12", got)
	}
	// A arvore cai no fallback por pagina (stub sem arvore): as secoes sao
	// as que sobraram, todas do cache na segunda vez.
	if got := lastMetadataValue(r2.MetadataPairs, "enrichment_cache_hits_sections"); got != strconv.Itoa(len(r2.OutlineNodes)) {
		t.Fatalf("secoes do cache = %s, esperava %d", got, len(r2.OutlineNodes))
	}
	// 12 paginas = 2 lotes, ate 12 secoes = 1 lote: a segunda vez poupa 3 chamadas.
	if segunda != primeira-3 {
		t.Fatalf("segunda indexacao fez %d chamadas, primeira %d; esperava 3 a menos", segunda, primeira)
	}
	for _, page := range r2.DocumentPages {
		if !strings.HasPrefix(page.Summary, "Resumo da pagina") || page.Metadata["topics"] != "prazo, matricula" {
			t.Fatalf("pagina %d sem resumo/topicos vindos do cache: %q %v", page.PageNumber, page.Summary, page.Metadata)
		}
	}
}

func TestEscopoDesligadoNaoChamaALLM(t *testing.T) {
	t.Parallel()
	pages, nodes := lotesDeTeste()
	calls := 0
	prefs := domain.IndexingPreferences{DocumentSynopsis: false, PageSummaries: false, SectionSummaries: true}
	r, err := NewLLMEnricher().EnrichWith(context.Background(), stubBatchLLM{calls: &calls}, &models.Document{ID: 1, Title: "ppc.pdf"}, domain.IndexingResult{DocumentPages: pages, OutlineNodes: nodes}, prefs)
	if err != nil {
		t.Fatal(err)
	}
	if got := lastMetadataValue(r.MetadataPairs, "enrichment_scope"); got != "sections" {
		t.Fatalf("escopo = %q", got)
	}
	if got := lastMetadataValue(r.MetadataPairs, "synopsis_source"); got != "disabled" {
		t.Fatalf("sinopse devia estar desligada: %q", got)
	}
	if calls != 1 {
		t.Fatalf("so o lote de secoes devia chamar a LLM: %d chamadas", calls)
	}
	if got := lastMetadataValue(r.MetadataPairs, "enrichment_pages_summarized"); got != "0" {
		t.Fatalf("paginas resumidas = %s, esperava 0", got)
	}
	for _, page := range r.DocumentPages {
		if strings.HasPrefix(page.Summary, "Resumo da pagina") {
			t.Fatalf("pagina %d resumida com escopo desligado", page.PageNumber)
		}
	}
}

// Cache escopado por usuario: outro dono com o mesmo texto chama o proprio LLM
// e nao ve acerto de cache.
func TestCacheDeEnriquecimentoNaoCruzaUsuarios(t *testing.T) {
	t.Parallel()
	cache := &stubCache{entries: map[string]domain.EnrichmentCacheEntry{}}
	enricher := NewLLMEnricher().WithCache(cache)

	atacante := 0
	pages, nodes := lotesDeTeste()
	if _, err := enricher.Enrich(context.Background(), stubBatchLLM{calls: &atacante}, &models.Document{ID: 1, UserID: 1, Title: "ppc.pdf"}, domain.IndexingResult{DocumentPages: pages, OutlineNodes: nodes}); err != nil {
		t.Fatal(err)
	}

	vitima := 0
	pages2, nodes2 := lotesDeTeste()
	r2, err := enricher.Enrich(context.Background(), stubBatchLLM{calls: &vitima}, &models.Document{ID: 1, UserID: 2, Title: "ppc.pdf"}, domain.IndexingResult{DocumentPages: pages2, OutlineNodes: nodes2})
	if err != nil {
		t.Fatal(err)
	}
	if got := lastMetadataValue(r2.MetadataPairs, "enrichment_cache_hits_pages"); got != "0" {
		t.Fatalf("usuario 2 acertou o cache do usuario 1: %s paginas", got)
	}
	if vitima != atacante {
		t.Fatalf("usuario 2 devia pagar as mesmas %d chamadas do usuario 1, fez %d", atacante, vitima)
	}
}
