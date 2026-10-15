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

// Sem "stream":false explicito o Ollama responde em NDJSON e o decoder le so o
// primeiro pedaco.
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
