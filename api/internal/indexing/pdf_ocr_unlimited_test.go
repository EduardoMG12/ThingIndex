package indexing

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"thing-index/internal/models"
)

func newTestUnlimitedExtractor() *unlimitedOCRExtractor {
	return &unlimitedOCRExtractor{
		model:    defaultUnlimitedOCRModel,
		prompt:   defaultUnlimitedOCRPrompt,
		dpi:      150,
		boxScale: 1000,
		client:   http.DefaultClient,
	}
}

// A resposta do modelo intercala marcadores de deteccao com o conteudo. Cada
// bloco vira um textBlock ancorado.
func TestUnlimitedOCRParsesDetectionMarkers(t *testing.T) {
	t.Parallel()

	// Pagina carta em 150 DPI: 8.5x11pol -> 1275x1650px -> 612x792pt.
	const width, height = 1275, 1650

	raw := "<|det|>title [100, 50, 900, 120]<|/det|>JUIZO DE DIREITO DA VARA CIVEL\n" +
		"<|det|>text [100, 200, 900, 260]<|/det|>A parte autora requer a procedencia do pedido."

	blocks := newTestUnlimitedExtractor().blocksFromResponse(raw, 3, width, height)

	if len(blocks) != 2 {
		t.Fatalf("blocos = %d, quero 2: %+v", len(blocks), blocks)
	}

	if blocks[0].Content != "JUIZO DE DIREITO DA VARA CIVEL" {
		t.Fatalf("conteudo do primeiro bloco = %q", blocks[0].Content)
	}
	if blocks[1].Content != "A parte autora requer a procedencia do pedido." {
		t.Fatalf("conteudo do segundo bloco = %q", blocks[1].Content)
	}

	for index, block := range blocks {
		if block.PageNumber == nil || *block.PageNumber != 3 {
			t.Fatalf("bloco %d com pagina errada: %+v", index, block.PageNumber)
		}
		if block.Source != "ocr_unlimited" {
			t.Fatalf("bloco %d com origem %q", index, block.Source)
		}
		if block.Box == nil {
			t.Fatalf("bloco %d veio sem caixa", index)
		}
	}
}

// A caixa sai no mesmo espaco do `pdftotext -bbox-layout`: pontos PDF, origem
// no canto superior esquerdo.
func TestUnlimitedOCRConvertsBoxToPDFPoints(t *testing.T) {
	t.Parallel()

	const width, height = 1275, 1650

	// Coordenada normalizada no meio da pagina em ambos os eixos.
	raw := "<|det|>text [0, 0, 500, 500]<|/det|>metade da pagina"
	blocks := newTestUnlimitedExtractor().blocksFromResponse(raw, 1, width, height)

	if len(blocks) != 1 || blocks[0].Box == nil {
		t.Fatalf("esperava um bloco ancorado, veio %+v", blocks)
	}

	box := blocks[0].Box
	if box.PageWidth != 612 || box.PageHeight != 792 {
		t.Fatalf("pagina = %vx%v pt, quero 612x792", box.PageWidth, box.PageHeight)
	}
	if box.XMin != 0 || box.YMin != 0 {
		t.Fatalf("canto superior esquerdo = (%v, %v), quero (0, 0)", box.XMin, box.YMin)
	}
	if box.XMax != 306 || box.YMax != 396 {
		t.Fatalf("canto inferior direito = (%v, %v), quero (306, 396)", box.XMax, box.YMax)
	}
}

// Com boxScale 0 as coordenadas sao pixel bruto da imagem renderizada.
func TestUnlimitedOCRSupportsRawPixelCoordinates(t *testing.T) {
	t.Parallel()

	extractor := newTestUnlimitedExtractor()
	extractor.boxScale = 0

	raw := "<|det|>text [0, 0, 1275, 1650]<|/det|>pagina inteira"
	blocks := extractor.blocksFromResponse(raw, 1, 1275, 1650)

	if len(blocks) != 1 || blocks[0].Box == nil {
		t.Fatalf("esperava um bloco ancorado, veio %+v", blocks)
	}
	if box := blocks[0].Box; box.XMax != 612 || box.YMax != 792 {
		t.Fatalf("caixa = %+v, quero cobrir 612x792", *box)
	}
}

// Texto sem caixa nao e descartado.
func TestUnlimitedOCRKeepsBlocksWithoutBox(t *testing.T) {
	t.Parallel()

	raw := "<|det|>text<|/det|>conteudo sem coordenada nenhuma nesta linha."
	blocks := newTestUnlimitedExtractor().blocksFromResponse(raw, 1, 1275, 1650)

	if len(blocks) != 1 {
		t.Fatalf("blocos = %d, quero 1", len(blocks))
	}
	if blocks[0].Box != nil {
		t.Fatalf("esperava bloco sem caixa, veio %+v", *blocks[0].Box)
	}
	if !strings.Contains(blocks[0].Content, "conteudo sem coordenada") {
		t.Fatalf("conteudo perdido: %q", blocks[0].Content)
	}
}

// Resposta em markdown puro, sem marcadores, continua util.
func TestUnlimitedOCRFallsBackToPlainText(t *testing.T) {
	t.Parallel()

	raw := "# Peticao inicial\n\nA parte autora requer a procedencia do pedido formulado."
	blocks := newTestUnlimitedExtractor().blocksFromResponse(raw, 2, 1275, 1650)

	if len(blocks) == 0 {
		t.Fatal("resposta sem marcadores foi descartada")
	}
	joined := ""
	for _, block := range blocks {
		joined += block.Content
	}
	if !strings.Contains(joined, "procedencia do pedido") {
		t.Fatalf("texto perdido: %q", joined)
	}
}

// Caixa degenerada (area zero ou invertida) e descartada em vez de virar um
// retangulo invisivel sobre o documento.
func TestUnlimitedOCRRejectsDegenerateBox(t *testing.T) {
	t.Parallel()

	extractor := newTestUnlimitedExtractor()
	for _, raw := range []string{
		"<|det|>text [500, 500, 500, 500]<|/det|>area zero",
		"<|det|>text [100, 100, 50]<|/det|>coordenadas insuficientes",
		"<|det|>text [a, b, c, d]<|/det|>coordenadas nao numericas",
	} {
		blocks := extractor.blocksFromResponse(raw, 1, 1275, 1650)
		if len(blocks) != 1 {
			t.Fatalf("blocos = %d para %q", len(blocks), raw)
		}
		if blocks[0].Box != nil {
			t.Fatalf("caixa invalida aceita em %q: %+v", raw, *blocks[0].Box)
		}
	}
}

// Contrato HTTP: o provider precisa falar a API compativel com OpenAI que o
// vLLM e o SGLang expoem.
func TestUnlimitedOCRSendsOpenAICompatibleRequest(t *testing.T) {
	t.Parallel()

	var captured ocrChatRequest
	var authorization string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("caminho = %q, quero /v1/chat/completions", r.URL.Path)
		}
		authorization = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Errorf("decodificar requisicao: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"<|det|>text [0, 0, 100, 100]<|/det|>ok"}}]}`))
	}))
	defer server.Close()

	extractor := newTestUnlimitedExtractor()
	extractor.endpoint = server.URL
	extractor.apiKey = "chave-de-teste"

	content, err := extractor.parseDocument(t.Context(), []byte("imagem-falsa"))
	if err != nil {
		t.Fatalf("parseDocument: %v", err)
	}
	if !strings.Contains(content, "ok") {
		t.Fatalf("conteudo = %q", content)
	}

	if authorization != "Bearer chave-de-teste" {
		t.Fatalf("Authorization = %q", authorization)
	}
	if captured.Model != defaultUnlimitedOCRModel {
		t.Fatalf("model = %q", captured.Model)
	}
	if len(captured.Messages) != 1 || len(captured.Messages[0].Content) != 2 {
		t.Fatalf("mensagem malformada: %+v", captured.Messages)
	}
	image := captured.Messages[0].Content[0]
	if image.Type != "image_url" || image.ImageURL == nil {
		t.Fatalf("primeira parte deveria ser a imagem: %+v", image)
	}
	if !strings.HasPrefix(image.ImageURL.URL, "data:image/png;base64,") {
		t.Fatalf("imagem nao veio como data URI: %q", image.ImageURL.URL)
	}
	if prompt := captured.Messages[0].Content[1]; prompt.Text != defaultUnlimitedOCRPrompt {
		t.Fatalf("prompt = %q", prompt.Text)
	}
}

// Erro do servidor sobe com a mensagem, nao vira pagina vazia.
func TestUnlimitedOCRSurfacesServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"message":"modelo ainda carregando"}}`))
	}))
	defer server.Close()

	extractor := newTestUnlimitedExtractor()
	extractor.endpoint = server.URL

	_, err := extractor.parseDocument(t.Context(), []byte("imagem-falsa"))
	if err == nil {
		t.Fatal("erro do servidor foi engolido")
	}
	if !strings.Contains(err.Error(), "modelo ainda carregando") {
		t.Fatalf("mensagem do servidor perdida: %v", err)
	}
}

// PDF digitalizado de ponta a ponta (pdftoppm + servidor fake no formato do
// modelo): a ancora precisa chegar ao ParagraphNode.
func TestUnlimitedOCRAnchorsScannedPDFEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		t.Skip("pdftoppm indisponivel no PATH")
	}

	var pagesSeen int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pagesSeen++
		w.Header().Set("Content-Type", "application/json")
		body := map[string]any{"choices": []any{map[string]any{"message": map[string]any{
			"content": "<|det|>title [100, 60, 900, 130]<|/det|>JUIZO DE DIREITO DA VARA CIVEL\n" +
				"<|det|>text [100, 200, 900, 280]<|/det|>A parte autora requer a procedencia do pedido formulado na inicial.",
		}}}}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()

	extractor := newTestUnlimitedExtractor()
	extractor.endpoint = server.URL
	extractor.rendererPath = mustLookPath(t, "pdftoppm")

	textExtractor := newCommandPDFTextExtractor()
	if textExtractor == nil {
		t.Skip("pdftotext indisponivel no PATH")
	}

	indexer := newDeterministicIndexerWithExtractors(textExtractor, extractor)
	doc := &models.Document{ID: 1, MimeType: "application/pdf"}

	result, err := indexer.Index(t.Context(), doc, bytes.NewReader(scannedPDFBytes(t)))
	if err != nil {
		t.Fatalf("Index: %v", err)
	}

	if pagesSeen != 3 {
		t.Fatalf("o modelo recebeu %d paginas, quero 3", pagesSeen)
	}

	anchored := 0
	for _, node := range result.ParagraphNodes {
		box, ok := node.Metadata["box"].(*textBox)
		if !ok || box == nil {
			continue
		}
		anchored++
		if box.PageWidth <= 0 || box.XMax <= box.XMin {
			t.Fatalf("caixa invalida persistida: %+v", *box)
		}
	}

	if anchored == 0 {
		t.Fatal("nenhum paragrafo de pagina digitalizada chegou ancorado")
	}
	t.Logf("paragrafos ancorados vindos de OCR: %d de %d", anchored, len(result.ParagraphNodes))
}

func mustLookPath(t *testing.T, binary string) string {
	t.Helper()
	path, err := exec.LookPath(binary)
	if err != nil {
		t.Skipf("%s indisponivel", binary)
	}
	return path
}
