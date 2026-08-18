package handlers

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// O ServerWriteTimeout do http.Server e um deadline ABSOLUTO contado do inicio
// da requisicao, nao um limite de ociosidade. Com 30s configurados em producao,
// qualquer resposta de chat mais longa era cortada no meio: o texto parava de
// aparecer, sem erro e sem explicacao.
//
// Este teste sobe um servidor HTTP de verdade com WriteTimeout deliberadamente
// curto e emite eventos espacados que ultrapassam esse limite. Se writeChatEvent
// nao empurrar o deadline, a conexao morre e o cliente recebe menos eventos do
// que os emitidos.
func TestWriteChatEventEstendeOPrazoEEvitaCorteDoStream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		writeTimeout   = 150 * time.Millisecond
		eventos        = 6
		intervaloEvent = 50 * time.Millisecond
	)

	router := gin.New()
	router.GET("/stream", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Status(http.StatusOK)

		for i := 0; i < eventos; i++ {
			if err := writeChatEvent(c, "delta", gin.H{"i": i}); err != nil {
				t.Errorf("writeChatEvent falhou no evento %d: %v", i, err)
				return
			}
			time.Sleep(intervaloEvent)
		}
	})

	// A duracao total (6 x 50ms = 300ms) e o DOBRO do WriteTimeout. Sem a
	// renovacao do deadline o stream e cortado antes do fim.
	server := httptest.NewUnstartedServer(router)
	server.Config.WriteTimeout = writeTimeout
	server.Start()
	defer server.Close()

	resp, err := http.Get(server.URL + "/stream")
	if err != nil {
		t.Fatalf("GET no stream: %v", err)
	}
	defer resp.Body.Close()

	var recebidos int
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "data: ") {
			recebidos++
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("o stream foi interrompido antes do fim: %v", err)
	}

	if recebidos != eventos {
		t.Fatalf("recebi %d de %d eventos; o WriteTimeout absoluto cortou o stream no meio", recebidos, eventos)
	}
}
