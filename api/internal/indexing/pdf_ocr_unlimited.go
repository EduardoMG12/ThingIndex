package indexing

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Provider de OCR baseado no baidu/Unlimited-OCR, servido por vLLM ou SGLang
// atras de uma API compativel com OpenAI.
//
// Diferenca central para o tesseract: alem do texto, o modelo devolve a regiao
// de cada bloco no formato `<|det|>tipo [x1, y1, x2, y2]<|/det|>conteudo`, o que
// permite ancorar posicionalmente tambem as paginas digitalizadas — que sao
// justamente as que o pdftotext nao consegue ancorar.
//
// O modelo exige GPU. Quando o endpoint nao esta configurado ou nao responde, a
// indexacao continua com o tesseract, que roda em qualquer lugar. Ver
// newPDFOCRExtractor.

const (
	defaultUnlimitedOCRModel   = "Unlimited-OCR"
	defaultUnlimitedOCRPrompt  = "<image>document parsing."
	defaultUnlimitedOCRTimeout = 120 * time.Second
	defaultUnlimitedOCRDPI     = 150
)

type unlimitedOCRExtractor struct {
	endpoint     string
	model        string
	apiKey       string
	prompt       string
	dpi          int
	rendererPath string
	client       *http.Client
	// boxScale e o denominador das coordenadas devolvidas pelo modelo. Modelos
	// desta familia costumam normalizar em 0..1000; alguns emitem pixel bruto da
	// imagem. Configuravel porque errar a escala desalinha o grifo em silencio,
	// e a conferencia so pode ser feita contra uma execucao real.
	boxScale float64
}

func newUnlimitedOCRExtractor() pdfOCRExtractor {
	endpoint := strings.TrimSpace(os.Getenv("THINGINDEX_OCR_ENDPOINT"))
	if endpoint == "" {
		return nil
	}

	rendererPath, err := exec.LookPath("pdftoppm")
	if err != nil {
		return nil
	}

	model := strings.TrimSpace(os.Getenv("THINGINDEX_OCR_MODEL"))
	if model == "" {
		model = defaultUnlimitedOCRModel
	}

	prompt := strings.TrimSpace(os.Getenv("THINGINDEX_OCR_PROMPT"))
	if prompt == "" {
		prompt = defaultUnlimitedOCRPrompt
	}

	timeout := defaultUnlimitedOCRTimeout
	if raw := strings.TrimSpace(os.Getenv("THINGINDEX_OCR_TIMEOUT_SECONDS")); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	dpi := defaultUnlimitedOCRDPI
	if raw := strings.TrimSpace(os.Getenv("THINGINDEX_OCR_DPI")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			dpi = parsed
		}
	}

	// 0 desliga a normalizacao e trata as coordenadas como pixel da imagem.
	boxScale := 1000.0
	if raw := strings.TrimSpace(os.Getenv("THINGINDEX_OCR_BOX_SCALE")); raw != "" {
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil && parsed >= 0 {
			boxScale = parsed
		}
	}

	return &unlimitedOCRExtractor{
		endpoint:     strings.TrimSuffix(endpoint, "/"),
		model:        model,
		apiKey:       strings.TrimSpace(os.Getenv("THINGINDEX_OCR_API_KEY")),
		prompt:       prompt,
		dpi:          dpi,
		rendererPath: rendererPath,
		client:       &http.Client{Timeout: timeout},
		boxScale:     boxScale,
	}
}

func (e *unlimitedOCRExtractor) Name() string {
	return "unlimited_ocr"
}

func (e *unlimitedOCRExtractor) Extract(
	ctx context.Context,
	data []byte,
	pageNumbers []int,
) (map[int][]textBlock, error) {
	if e == nil || len(pageNumbers) == 0 {
		return nil, nil
	}

	tempDir, err := os.MkdirTemp("", "thingindex-unlimited-ocr-*")
	if err != nil {
		return nil, fmt.Errorf("create ocr temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	pdfPath := filepath.Join(tempDir, "document.pdf")
	if err := os.WriteFile(pdfPath, data, 0o600); err != nil {
		return nil, fmt.Errorf("write temporary pdf: %w", err)
	}

	results := make(map[int][]textBlock, len(pageNumbers))
	for _, pageNumber := range pageNumbers {
		if pageNumber <= 0 {
			continue
		}

		imagePath, err := e.renderPage(ctx, pdfPath, tempDir, pageNumber)
		if err != nil {
			return nil, err
		}

		imageBytes, err := os.ReadFile(imagePath)
		if err != nil {
			return nil, fmt.Errorf("read rendered page %d: %w", pageNumber, err)
		}

		width, height := imageDimensions(imageBytes)

		raw, err := e.parseDocument(ctx, imageBytes)
		if err != nil {
			return nil, fmt.Errorf("ocr page %d: %w", pageNumber, err)
		}

		blocks := e.blocksFromResponse(raw, pageNumber, width, height)
		if len(blocks) > 0 {
			results[pageNumber] = blocks
		}
	}

	return results, nil
}

func (e *unlimitedOCRExtractor) renderPage(ctx context.Context, pdfPath, tempDir string, pageNumber int) (string, error) {
	imageBase := filepath.Join(tempDir, fmt.Sprintf("page-%d", pageNumber))
	cmd := exec.CommandContext(
		ctx,
		e.rendererPath,
		"-f", strconv.Itoa(pageNumber),
		"-l", strconv.Itoa(pageNumber),
		"-r", strconv.Itoa(e.dpi),
		"-singlefile",
		"-png",
		pdfPath,
		imageBase,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("render pdf page %d for ocr: %w: %s", pageNumber, err, strings.TrimSpace(string(output)))
	}

	return imageBase + ".png", nil
}

type ocrChatRequest struct {
	Model     string           `json:"model"`
	Messages  []ocrChatMessage `json:"messages"`
	MaxTokens int              `json:"max_tokens"`
}

type ocrChatMessage struct {
	Role    string           `json:"role"`
	Content []ocrChatContent `json:"content"`
}

type ocrChatContent struct {
	Type     string       `json:"type"`
	Text     string       `json:"text,omitempty"`
	ImageURL *ocrImageURL `json:"image_url,omitempty"`
}

type ocrImageURL struct {
	URL string `json:"url"`
}

type ocrChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (e *unlimitedOCRExtractor) parseDocument(ctx context.Context, imageBytes []byte) (string, error) {
	payload := ocrChatRequest{
		Model:     e.model,
		MaxTokens: 32768,
		Messages: []ocrChatMessage{{
			Role: "user",
			Content: []ocrChatContent{
				{
					Type:     "image_url",
					ImageURL: &ocrImageURL{URL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageBytes)},
				},
				{Type: "text", Text: e.prompt},
			},
		}},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode ocr request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, e.endpoint+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build ocr request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	response, err := e.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("call ocr endpoint: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	var decoded ocrChatResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("decode ocr response (status %d): %w", response.StatusCode, err)
	}

	if response.StatusCode < 200 || response.StatusCode > 299 {
		if decoded.Error != nil {
			return "", fmt.Errorf("ocr endpoint returned %d: %s", response.StatusCode, decoded.Error.Message)
		}
		return "", fmt.Errorf("ocr endpoint returned %d", response.StatusCode)
	}
	if len(decoded.Choices) == 0 {
		return "", fmt.Errorf("ocr endpoint returned no choices")
	}

	return decoded.Choices[0].Message.Content, nil
}

// O modelo intercala marcadores de deteccao com o conteudo:
//
//	<|det|>text [12, 34, 560, 78]<|/det|>Juizo de Direito da Vara Civel
//
// A caixa e opcional: quando ausente, o bloco entra sem ancora em vez de ser
// descartado — texto sem posicao ainda serve para busca e para o contexto.
var unlimitedOCRDetectionPattern = regexp.MustCompile(`<\|det\|>([^\[<]*?)(?:\s*\[([^\]]*)\])?\s*<\|/det\|>`)

func (e *unlimitedOCRExtractor) blocksFromResponse(raw string, pageNumber, imageWidth, imageHeight int) []textBlock {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	matches := unlimitedOCRDetectionPattern.FindAllStringSubmatchIndex(raw, -1)
	if len(matches) == 0 {
		// Sem marcadores de deteccao a resposta e markdown puro. Ainda e util:
		// vira blocos sem ancora, como o tesseract.
		return blocksFromOCRPageText(normalizeText(raw), pageNumber)
	}

	blocks := make([]textBlock, 0, len(matches))
	for index, match := range matches {
		contentStart := match[1]
		contentEnd := len(raw)
		if index+1 < len(matches) {
			contentEnd = matches[index+1][0]
		}

		content := normalizeText(raw[contentStart:contentEnd])
		if content == "" {
			continue
		}

		block := textBlock{
			Content:    content,
			PageNumber: intPtr(pageNumber),
			Source:     "ocr_unlimited",
			LineCount:  1,
		}
		if match[4] >= 0 {
			block.Box = e.boxFromCoordinates(raw[match[4]:match[5]], imageWidth, imageHeight)
		}

		blocks = append(blocks, block)
	}

	return blocks
}

// boxFromCoordinates converte a caixa do modelo para pontos PDF com origem no
// canto superior esquerdo, que e o mesmo espaco usado pelas caixas do
// `pdftotext -bbox-layout`. Sem essa conversao, blocos nativos e blocos de OCR
// ficariam em sistemas diferentes e o grifo sairia deslocado.
func (e *unlimitedOCRExtractor) boxFromCoordinates(raw string, imageWidth, imageHeight int) *textBox {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
	if len(fields) != 4 || imageWidth <= 0 || imageHeight <= 0 {
		return nil
	}

	values := make([]float64, 0, 4)
	for _, field := range fields {
		value, err := strconv.ParseFloat(strings.TrimSpace(field), 64)
		if err != nil {
			return nil
		}
		values = append(values, value)
	}

	// Pixels da imagem renderizada -> pontos PDF (72 pontos por polegada).
	pageWidth := float64(imageWidth) * 72.0 / float64(e.dpi)
	pageHeight := float64(imageHeight) * 72.0 / float64(e.dpi)

	// boxScale > 0: coordenada normalizada (ex.: 0..1000) reprojetada na pagina.
	// boxScale == 0: pixel bruto da imagem, convertido por DPI.
	scaleX, scaleY := 72.0/float64(e.dpi), 72.0/float64(e.dpi)
	if e.boxScale > 0 {
		scaleX = pageWidth / e.boxScale
		scaleY = pageHeight / e.boxScale
	}

	for index := range values {
		if index%2 == 0 {
			values[index] *= scaleX
		} else {
			values[index] *= scaleY
		}
	}

	box := &textBox{
		XMin:       math.Min(values[0], values[2]),
		YMin:       math.Min(values[1], values[3]),
		XMax:       math.Max(values[0], values[2]),
		YMax:       math.Max(values[1], values[3]),
		PageWidth:  pageWidth,
		PageHeight: pageHeight,
	}
	if box.XMax <= box.XMin || box.YMax <= box.YMin {
		return nil
	}

	return box
}

func imageDimensions(data []byte) (int, int) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0
	}
	return config.Width, config.Height
}
