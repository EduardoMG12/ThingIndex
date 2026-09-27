package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/pkg/netsec"
)

func streamDeltasDoProvider(t *testing.T, llm domain.LLMService) (string, error) {
	t.Helper()
	streamer, ok := llm.(domain.LLMStreamingService)
	if !ok {
		t.Fatalf("cliente %T nao faz streaming", llm)
	}
	var recebido strings.Builder
	err := streamer.StreamText(context.Background(), "system", "user", func(delta string) error {
		recebido.WriteString(delta)
		return nil
	})
	return recebido.String(), err
}

// Regressao P1-2: http.Client{Timeout: 45s} inclui a leitura do corpo, entao
// todo stream de chat mais longo que o prazo era cortado no meio. O prazo do
// cliente agora cobre so a espera pelo cabecalho; aqui o stream dura varias
// vezes esse prazo e precisa chegar inteiro.
func TestStreamDoLLMMaisLongoQueOPrazoDeCabecalhoNaoECortado(t *testing.T) {
	t.Parallel()

	const prazo = 80 * time.Millisecond
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		for i := 0; i < 5; i++ {
			time.Sleep(prazo)
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"p%d \"}}]}\n\n", i)
			w.(http.Flusher).Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer servidor.Close()

	llm, err := NewLLMProvider(domain.ProviderOpenAI, map[string]string{
		"api_key":  "chave",
		"base_url": servidor.URL,
	}, LLMClientOptions{responseHeaderTimeout: prazo})
	if err != nil {
		t.Fatalf("NewLLMProvider: %v", err)
	}

	texto, err := streamDeltasDoProvider(t, llm)
	if err != nil {
		t.Fatalf("stream mais longo que o prazo de cabecalho foi cortado: %v", err)
	}
	if texto != "p0 p1 p2 p3 p4 " {
		t.Fatalf("stream chegou incompleto: %q", texto)
	}
}

func TestClienteHTTPDoLLMNaoTemTimeoutGlobal(t *testing.T) {
	t.Parallel()

	cliente, err := buildLLMHTTPClient("https://api.openai.com/v1/chat/completions", false, LLMClientOptions{})
	if err != nil {
		t.Fatalf("buildLLMHTTPClient: %v", err)
	}
	if cliente.Timeout != 0 {
		t.Fatalf("Timeout global corta stream longo; esperava 0, veio %s", cliente.Timeout)
	}
	transporte, ok := cliente.Transport.(*http.Transport)
	if !ok || transporte.ResponseHeaderTimeout != llmResponseHeaderTimeout {
		t.Fatalf("esperava ResponseHeaderTimeout=%s no transporte, veio %#v", llmResponseHeaderTimeout, cliente.Transport)
	}
}

// O prazo de cabecalho continua valendo: provedor que nao responde nada e
// cortado mesmo sem prazo no ctx.
func TestClienteHTTPDoLLMCortaProvedorQueNaoMandaCabecalho(t *testing.T) {
	t.Parallel()

	liberar := make(chan struct{})
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-liberar
	}))
	defer servidor.Close()
	defer close(liberar)

	llm, err := NewLLMProvider(domain.ProviderOpenAI, map[string]string{
		"api_key":  "chave",
		"base_url": servidor.URL,
	}, LLMClientOptions{responseHeaderTimeout: 50 * time.Millisecond})
	if err != nil {
		t.Fatalf("NewLLMProvider: %v", err)
	}

	if _, err := streamDeltasDoProvider(t, llm); err == nil {
		t.Fatal("provedor mudo deveria estourar o prazo de cabecalho")
	}
}

// Regressao P1-3: o cliente seguia redirect, entao um base_url publico que
// responde 302 para a rede interna levava a API a requisitar o destino.
func TestClienteHTTPDoLLMRecusaRedirect(t *testing.T) {
	t.Parallel()

	var destinoAcessado atomic.Bool
	destino := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		destinoAcessado.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer destino.Close()

	origem := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destino.URL+"/models", http.StatusFound)
	}))
	defer origem.Close()

	_, err := ListLLMModels(context.Background(), domain.ProviderOpenAI, map[string]string{
		"api_key":  "chave",
		"base_url": origem.URL,
	})
	if !errors.Is(err, errLLMRedirectRecusado) {
		t.Fatalf("esperava redirect recusado, veio %v", err)
	}
	if destinoAcessado.Load() {
		t.Fatal("o destino do redirect foi requisitado")
	}
}

// Regressao P1-3: o base_url so era validado no save. Com o validador no dial,
// um host que resolve para IP privado e recusado na conexao; com a flag de
// endpoints privados (desenvolvimento, Ollama local) continua funcionando.
func TestClienteHTTPDoLLMValidaDestinoNoDialRespeitandoAFlagDePrivados(t *testing.T) {
	t.Parallel()

	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"modelo-local"}]}`))
	}))
	defer servidor.Close()

	credenciais := map[string]string{"api_key": "chave", "base_url": servidor.URL}

	_, err := ListLLMModels(context.Background(), domain.ProviderOpenAI, credenciais,
		LLMClientOptions{EndpointValidator: netsec.NewStorageEndpointValidator(false)})
	if err == nil {
		t.Fatal("base_url em loopback deveria ser recusado no dial sem a flag de privados")
	}

	modelos, err := ListLLMModels(context.Background(), domain.ProviderOpenAI, credenciais,
		LLMClientOptions{EndpointValidator: netsec.NewStorageEndpointValidator(true)})
	if err != nil {
		t.Fatalf("com a flag de privados o endpoint local deveria funcionar: %v", err)
	}
	if len(modelos) != 1 || modelos[0] != "modelo-local" {
		t.Fatalf("modelos inesperados: %#v", modelos)
	}
}
