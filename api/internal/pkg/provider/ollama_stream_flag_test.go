package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// O /api/chat do Ollama tem stream=true como padrao. Se o pedido nao disser
// "stream":false por escrito, a resposta vem em NDJSON, um pedaco por token,
// e o decoder de resposta unica le so o primeiro. Foi assim que o
// enriquecimento por LLM devolveu um token por chamada e caiu em fallback
// sem erro nenhum.
func TestOllamaNativeGenerateTextEnviaStreamFalsePorEscrito(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var pedido map[string]any
		if err := json.Unmarshal(body, &pedido); err != nil {
			t.Fatalf("pedido invalido: %v", err)
		}
		if stream, ok := pedido["stream"]; !ok || stream != false {
			// Comportamento real do Ollama sem o campo: streaming em NDJSON.
			w.Header().Set("Content-Type", "application/x-ndjson")
			_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"{"},"done":false}` + "\n"))
			_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"\"ok\":true}"},"done":true}` + "\n"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"{\"ok\":true}"},"done":true}`))
	}))
	defer server.Close()

	client := &ollamaNativeLLMClient{httpClient: server.Client(), endpoint: server.URL + "/api/chat", model: "gemma4:31b"}
	saida, err := client.GenerateText(context.Background(), "system", "user")
	if err != nil {
		t.Fatalf("GenerateText falhou: %v", err)
	}
	if strings.TrimSpace(saida) != `{"ok":true}` {
		t.Fatalf("resposta lida pela metade (stream sem false explicito): %q", saida)
	}
}
