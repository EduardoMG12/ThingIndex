package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"thing-index/internal/domain"
)

// O /api/chat do Ollama devolve prompt_eval_count e eval_count. Antes o cliente
// nativo nao os lia e todo consumo do Ollama (local ou Cloud) virava estimativa.
func TestOllamaNativoReportaConsumoNoGenerateText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"ola"},"done":true,"prompt_eval_count":1234,"eval_count":56}`))
	}))
	defer srv.Close()

	var llm domain.LLMService = &ollamaNativeLLMClient{httpClient: srv.Client(), endpoint: srv.URL + "/api/chat", model: "m"}
	if _, err := llm.GenerateText(context.Background(), "s", "u"); err != nil {
		t.Fatal(err)
	}
	reporter, ok := llm.(domain.TokenUsageReporter)
	if !ok {
		t.Fatal("cliente Ollama nativo nao implementa TokenUsageReporter")
	}
	p, c, total, reported := reporter.LastTokenUsage()
	if !reported || p != 1234 || c != 56 || total != 1290 {
		t.Fatalf("esperava 1234/56/1290 reportado, veio %d/%d/%d %v", p, c, total, reported)
	}
}

func TestOllamaNativoReportaConsumoNoStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(strings.Join([]string{
			`{"message":{"role":"assistant","content":"ol"},"done":false}`,
			`{"message":{"role":"assistant","content":"a"},"done":false}`,
			`{"message":{"role":"assistant","content":""},"done":true,"prompt_eval_count":10,"eval_count":2}`,
		}, "\n")))
	}))
	defer srv.Close()

	var llm domain.LLMService = &ollamaNativeLLMClient{httpClient: srv.Client(), endpoint: srv.URL + "/api/chat", model: "m"}
	var texto strings.Builder
	if err := llm.(domain.LLMStreamingService).StreamText(context.Background(), "s", "u", func(d string) error { texto.WriteString(d); return nil }); err != nil {
		t.Fatal(err)
	}
	p, c, _, reported := llm.(domain.TokenUsageReporter).LastTokenUsage()
	if texto.String() != "ola" || !reported || p != 10 || c != 2 {
		t.Fatalf("texto %q, uso %d/%d reportado=%v", texto.String(), p, c, reported)
	}
}

func TestOllamaNativoSemContadoresNaoFingeConsumo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"ola"},"done":true}`))
	}))
	defer srv.Close()
	var llm domain.LLMService = &ollamaNativeLLMClient{httpClient: srv.Client(), endpoint: srv.URL + "/api/chat", model: "m"}
	if _, err := llm.GenerateText(context.Background(), "s", "u"); err != nil {
		t.Fatal(err)
	}
	if _, _, _, reported := llm.(domain.TokenUsageReporter).LastTokenUsage(); reported {
		t.Fatal("sem contadores o consumo nao pode constar como reportado")
	}
}
