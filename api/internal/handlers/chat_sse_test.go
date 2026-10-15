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

// Servidor real com WriteTimeout curto e eventos espacados alem dele: se
// writeChatEvent nao empurrar o deadline, o cliente recebe menos eventos.
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

	// 6 x 50ms = 300ms, o dobro do WriteTimeout.
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
