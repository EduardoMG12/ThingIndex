//go:build integration

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"thing-index/internal/domain"
)

// O encerramento gracioso e comportamento observavel, nao configuracao: ou a
// requisicao em andamento termina, ou o cliente recebe conexao cortada. Estes
// testes exercitam um servidor real com uma rota lenta, porque asserção sobre
// codigo ("existe uma chamada a Shutdown") nao prova que ele funciona.

// TestShutdownEsperaRequisicaoEmAndamento prova a razao de existir do recurso:
// um deploy no meio de um upload nao pode cortar o cliente que ja enviou o
// arquivo e aguarda resposta.
func TestShutdownEsperaRequisicaoEmAndamento(t *testing.T) {
	gin.SetMode(gin.TestMode)

	comecouAResponder := make(chan struct{})
	router := gin.New()
	router.GET("/lento", func(c *gin.Context) {
		close(comecouAResponder)
		// Simula trabalho em andamento quando o sinal chega.
		time.Sleep(400 * time.Millisecond)
		c.String(http.StatusOK, "terminei")
	})

	servidor := httptest.NewUnstartedServer(router)
	servidor.Start()
	defer servidor.Close()

	respostas := make(chan *http.Response, 1)
	erros := make(chan error, 1)

	go func() {
		resp, err := http.Get(servidor.URL + "/lento")
		if err != nil {
			erros <- err
			return
		}
		respostas <- resp
	}()

	// So dispara o encerramento depois que a requisicao ja esta sendo servida.
	<-comecouAResponder

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	inicio := time.Now()
	if err := servidor.Config.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown falhou: %v", err)
	}
	duracao := time.Since(inicio)

	select {
	case err := <-erros:
		t.Fatalf("a requisicao em andamento foi cortada pelo shutdown: %v", err)
	case resp := <-respostas:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("esperava 200 na requisicao em andamento, veio %d", resp.StatusCode)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a requisicao nunca terminou")
	}

	// O Shutdown precisa ter ESPERADO. Retorno imediato significaria que ele
	// nao acompanhou a requisicao -- o teste passaria por acidente de
	// temporizacao.
	if duracao < 200*time.Millisecond {
		t.Fatalf("shutdown retornou em %v: nao esperou a requisicao em andamento", duracao)
	}
}

// TestShutdownRecusaConexaoNova garante a outra metade do contrato: durante o
// dreno o servidor para de aceitar trabalho novo. Sem isso o encerramento nunca
// convergiria sob trafego.
func TestShutdownRecusaConexaoNova(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	servidor := httptest.NewServer(router)
	url := servidor.URL

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := servidor.Config.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown falhou: %v", err)
	}

	cliente := &http.Client{Timeout: time.Second}
	if resp, err := cliente.Get(url + "/ok"); err == nil {
		resp.Body.Close()
		t.Fatal("servidor continuou aceitando conexao nova apos o shutdown")
	}
}

// TestSetupRoutesWithContextEncerraTarefaDeFundo prova que a goroutine de sync
// do catalogo de modelos morre com o contexto, em vez de depender do processo
// terminar. Era o defeito registrado no handover: ela recebia
// context.Background() e ficava fora do ciclo de vida da aplicacao.
func TestSetupRoutesWithContextEncerraTarefaDeFundo(t *testing.T) {
	chamadas := make(chan struct{}, 8)
	useCase := &syncSpyProviderUseCase{chamadas: chamadas}

	ctx, cancel := context.WithCancel(context.Background())
	startProviderModelCatalogSync(ctx, useCase, 40*time.Millisecond)

	// Primeira execucao acontece imediatamente.
	select {
	case <-chamadas:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("a tarefa de fundo nunca rodou")
	}

	cancel()

	// Drena o que ja estava em voo e confirma que o laco parou.
	time.Sleep(150 * time.Millisecond)
	for len(chamadas) > 0 {
		<-chamadas
	}

	time.Sleep(150 * time.Millisecond)
	if len(chamadas) > 0 {
		t.Fatalf("a tarefa continuou rodando %d vez(es) apos o cancelamento", len(chamadas))
	}
}

// syncSpyProviderUseCase conta as execucoes da tarefa de fundo.
//
// Embute a interface em vez de implementar os catorze metodos: so
// SyncModelCatalogs importa aqui, e stub que implementa tudo com corpo vazio
// esconde qual metodo o teste realmente exercita.
type syncSpyProviderUseCase struct {
	domain.ProviderUseCase
	chamadas chan struct{}
}

func (s *syncSpyProviderUseCase) SyncModelCatalogs(context.Context) error {
	select {
	case s.chamadas <- struct{}{}:
	default:
	}
	return nil
}
