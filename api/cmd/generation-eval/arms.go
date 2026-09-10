package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/evalcorpus"
	"thing-index/internal/pkg/provider"
)

// answer e o que cada braco devolve para uma pergunta.
type answer struct {
	Text             string
	Model            string     // modelo reportado pelo braco (T: no evento done)
	Citations        []citation // braco T: fontes estruturadas
	PromptTokens     int
	CompletionTokens int
	UsageReported    bool
	Latency          time.Duration
	Truncated        bool
	Err              error
}

// ---------------------------------------------------------------- braco T

// thingIndexArm fala com a API real: login, conversa nova por pergunta e
// /chat/stream. Mede o produto como o coordenador o usa, nao uma versao
// idealizada do usecase.
type thingIndexArm struct {
	api        string
	token      string
	collection uint
	client     *http.Client
	docByTitle map[string]string // titulo dobrado -> id do gabarito
}

func newThingIndexArm(api, email, password string, collection uint, g *evalcorpus.Gabarito) (*thingIndexArm, error) {
	arm := &thingIndexArm{api: strings.TrimRight(api, "/"), collection: collection, client: &http.Client{}, docByTitle: map[string]string{}}
	for _, d := range g.Documents {
		arm.docByTitle[fold(strings.TrimSpace(d.Title))] = d.ID
	}
	var login struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := arm.post("/auth/login", map[string]any{"email": email, "password": password}, &login); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	if login.Data.Token == "" {
		return nil, errors.New("login sem token")
	}
	arm.token = login.Data.Token
	return arm, nil
}

func (a *thingIndexArm) post(path string, body any, out any) error {
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, a.api+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}
	resp, err := a.doResilient(req, raw)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s: status %d: %s", path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

// doResilient repete a chamada quando a API cai no meio da rodada (o air
// recompila a API a cada edicao em Go do mesmo modulo: "connection reset by
// peer" ou EOF por uns 20 segundos). Sem isto, uma rodada de 72 perguntas
// perdeu 52 por uma edicao no indexador.
func (a *thingIndexArm) doResilient(req *http.Request, body []byte) (*http.Response, error) {
	var resp *http.Response
	var err error
	for tries := 0; tries < 12; tries++ {
		if tries > 0 {
			time.Sleep(5 * time.Second)
			req.Body = io.NopCloser(bytes.NewReader(body))
		}
		resp, err = a.client.Do(req)
		if err == nil || req.Context().Err() != nil {
			return resp, err
		}
	}
	return resp, err
}

func (a *thingIndexArm) Ask(ctx context.Context, q evalcorpus.GabaritoQuestion) answer {
	var created struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	if err := a.post("/chat/conversations", map[string]any{"collection_id": a.collection, "title": "gabarito " + q.ID}, &created); err != nil {
		return answer{Err: err}
	}
	raw, _ := json.Marshal(map[string]any{"conversation_id": created.Data.ID, "collection_id": a.collection, "prompt": q.Text, "history": []any{}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.api+"/chat/stream", bytes.NewReader(raw))
	if err != nil {
		return answer{Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.token)
	// O frontend manda o locale neste header; sem ele a API assume outro
	// idioma e o modelo responde em ingles, o que derruba o casamento.
	req.Header.Set("Accept-Language", "pt-BR")
	start := time.Now()
	resp, err := a.doResilient(req, raw)
	if err != nil {
		return answer{Err: err}
	}
	// A rota /chat/stream tem limitador por usuario (429). O experimento nao
	// e um ataque: espera a janela passar e tenta de novo, sem contar o tempo
	// de espera na latencia.
	for tries := 0; resp.StatusCode == http.StatusTooManyRequests && tries < 10; tries++ {
		resp.Body.Close()
		time.Sleep(30 * time.Second)
		req.Body = io.NopCloser(bytes.NewReader(raw))
		start = time.Now()
		if resp, err = a.client.Do(req); err != nil {
			return answer{Err: err}
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return answer{Err: fmt.Errorf("/chat/stream: status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))}
	}

	var out answer
	var text strings.Builder
	seen := map[citation]bool{}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	eventName := ""
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		var event domain.ChatStreamEvent
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), &event); err != nil {
			continue
		}
		// O handler emite o erro do provedor como evento "error" com {message}
		// sem o campo type; o nome do evento SSE e a fonte de verdade.
		if event.Type == "" {
			event.Type = eventName
		}
		text.WriteString(event.Delta)
		for _, s := range event.Sources {
			docID := a.docByTitle[fold(strings.TrimSpace(s.DocName))]
			if docID == "" {
				docID = s.DocName
			}
			pages := []int{s.Page}
			if s.ContinuesOnPage > 0 {
				pages = append(pages, s.ContinuesOnPage)
			}
			for _, page := range pages {
				c := citation{DocumentID: docID, Page: page}
				if !seen[c] {
					seen[c] = true
					out.Citations = append(out.Citations, c)
				}
			}
		}
		if event.Usage != nil {
			out.PromptTokens, out.CompletionTokens, out.UsageReported = event.Usage.PromptTokens, event.Usage.CompletionTokens, event.Usage.UsageReported
			out.Model = event.Usage.Model
		}
		if event.Type == "error" {
			out.Err = errors.New(event.Message)
		}
	}
	if err := scanner.Err(); err != nil && out.Err == nil {
		out.Err = err
	}
	out.Latency = time.Since(start)
	out.Text = strings.TrimSpace(text.String())
	// A API engole a falha do provedor e fecha o stream sem texto; sem isto a
	// linha entraria como "errada" e contaminaria o acerto.
	if out.Err == nil && out.Text == "" {
		out.Err = errors.New("resposta vazia (provedor falhou no servidor?)")
	}
	// Conversa descartavel: sem isso, 200+ conversas do experimento poluem a
	// tela do usuario demo. Melhor esforco; a linha ja esta medida.
	for tries := 0; tries < 5; tries++ {
		req, err = http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/chat/conversations/%d", a.api, created.Data.ID), nil)
		if err != nil {
			break
		}
		req.Header.Set("Authorization", "Bearer "+a.token)
		resp, err := a.client.Do(req)
		if err != nil {
			break
		}
		resp.Body.Close()
		// O DELETE passa pelo mesmo limitador do chat (429): sem esperar, 182
		// conversas "gabarito" ficaram na tela do usuario demo.
		if resp.StatusCode != http.StatusTooManyRequests {
			break
		}
		time.Sleep(15 * time.Second)
	}
	return out
}

// ---------------------------------------------------------------- braco F

// fullDocumentArm manda o(s) documento(s) inteiro(s) ao mesmo modelo, sem
// recuperacao. E o adversario que o artigo nunca tinha medido.
type fullDocumentArm struct {
	llm      domain.LLMService
	fallback domain.LLMService
	model    string
	pdfDir   string
	docs     []evalcorpus.GabaritoDocument
	scope    string // document | collection
	pages    map[string][]string
	language string
}

func newFullDocumentArm(pdfDir, scope, model string, g *evalcorpus.Gabarito) (*fullDocumentArm, error) {
	providerType := strings.TrimSpace(os.Getenv("THINGINDEX_TEST_LLM_PROVIDER"))
	if providerType == "" {
		return nil, errors.New("THINGINDEX_TEST_LLM_PROVIDER vazio")
	}
	if model == "" {
		model = strings.TrimSpace(os.Getenv("THINGINDEX_TEST_LLM_MODEL"))
	}
	build := func(apiKey string) (domain.LLMService, error) {
		return provider.NewLLMProvider(providerType, map[string]string{
			"api_key":  apiKey,
			"base_url": os.Getenv("THINGINDEX_TEST_LLM_BASE_URL"),
			"model":    model,
		})
	}
	llm, err := build(os.Getenv("THINGINDEX_TEST_LLM_API_KEY"))
	if err != nil {
		return nil, err
	}
	arm := &fullDocumentArm{llm: llm, model: model, pdfDir: pdfDir, docs: g.Documents, scope: scope, pages: map[string][]string{}}
	// Chave reserva (THINGINDEX_TEST_LLM_API_KEY_FALLBACK, outra conta do
	// mesmo provedor): entra quando a principal devolve "usage limit".
	if fallback := strings.TrimSpace(os.Getenv("THINGINDEX_TEST_LLM_API_KEY_FALLBACK")); fallback != "" {
		if alt, err := build(fallback); err == nil {
			arm.fallback = alt
		}
	}
	for _, d := range g.Documents {
		pages, err := extractPages(filepath.Join(pdfDir, d.File))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", d.File, err)
		}
		arm.pages[d.ID] = pages
	}
	return arm, nil
}

// extractPages usa o mesmo pdftotext do indexador, em -layout, uma entrada
// por pagina fisica. O marcador [pagina N] e a unica ancora que o modelo tem.
func extractPages(path string) ([]string, error) {
	out, err := exec.Command("pdftotext", "-layout", path, "-").Output()
	if err != nil {
		return nil, err
	}
	pages := strings.Split(string(out), "\f")
	if n := len(pages); n > 1 && strings.TrimSpace(pages[n-1]) == "" {
		pages = pages[:n-1]
	}
	return pages, nil
}

func (a *fullDocumentArm) documentsFor(q evalcorpus.GabaritoQuestion) []string {
	if a.scope == "collection" || q.IsNegative() {
		ids := make([]string, 0, len(a.docs))
		for _, d := range a.docs {
			ids = append(ids, d.ID)
		}
		return ids
	}
	var ids []string
	for _, t := range q.Targets {
		ids = append(ids, t.DocumentID)
	}
	return ids
}

func (a *fullDocumentArm) Ask(ctx context.Context, q evalcorpus.GabaritoQuestion) answer {
	var b strings.Builder
	for _, id := range a.documentsFor(q) {
		var title string
		for _, d := range a.docs {
			if d.ID == id {
				title = d.Title
			}
		}
		fmt.Fprintf(&b, "=== DOCUMENTO: %s ===\n", title)
		for i, p := range a.pages[id] {
			fmt.Fprintf(&b, "[página %d]\n%s\n", i+1, strings.TrimSpace(p))
		}
	}
	b.WriteString("\nPergunta: " + q.Text + "\nResponda em português do Brasil e cite a página de cada informação no formato \"p. N\".")
	userPrompt := b.String()
	systemPrompt := strings.Join([]string{
		"You are a grounded document-analysis assistant.",
		"Answer only from the provided documents. Each page starts with a marker like [página 12].",
		"If the evidence is insufficient, say that clearly and do not invent facts.",
		"Prefer precise, direct answers and cite the supporting document and page as \"p. N\" for every fact.",
		"Write the answer in Portuguese (Brazil). Responda em português do Brasil.",
	}, "\n")

	estimated := estimateTokens(systemPrompt + userPrompt)
	start := time.Now()
	text, err := a.llm.GenerateText(ctx, systemPrompt, userPrompt)
	if err != nil && a.fallback != nil && strings.Contains(err.Error(), "usage limit") {
		fmt.Fprintln(os.Stderr, "generation-eval: chave principal esgotada; passando para a chave reserva")
		a.llm, a.fallback = a.fallback, nil
		start = time.Now()
		text, err = a.llm.GenerateText(ctx, systemPrompt, userPrompt)
	}
	out := answer{Text: strings.TrimSpace(text), Model: a.model, Latency: time.Since(start), Err: err}
	if reporter, ok := a.llm.(domain.TokenUsageReporter); ok {
		out.PromptTokens, out.CompletionTokens, _, out.UsageReported = reporter.LastTokenUsage()
	}
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "context") {
		out.Truncated = true
	}
	// Provedor que aceita o prompt mas conta bem menos tokens do que o enviado
	// cortou o contexto em silencio. 50% e folga para a diferenca entre a
	// estimativa por runas e o tokenizador real.
	if out.UsageReported && out.PromptTokens < estimated/2 {
		out.Truncated = true
	}
	return out
}
