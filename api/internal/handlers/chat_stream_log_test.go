package handlers

import (
	"fmt"
	"strings"
	"testing"

	"thing-index/internal/domain"
)

// Regressao P2-9: o handler logava cada delta do SSE, centenas de linhas por
// resposta. Deltas agora viram uma unica linha de resumo.
func TestChatStreamEventLogNaoLogaCadaDelta(t *testing.T) {
	var linhas []string
	registro := newChatStreamEventLog(func(format string, args ...any) {
		linhas = append(linhas, fmt.Sprintf(format, args...))
	})

	registro.observe(domain.ChatStreamEvent{Type: "stage", Stage: "generating_answer"})
	for i := 0; i < 300; i++ {
		registro.observe(domain.ChatStreamEvent{Type: "delta", Delta: "ab"})
	}
	registro.observe(domain.ChatStreamEvent{Type: "done"})
	registro.finish(7)

	if len(linhas) != 3 {
		t.Fatalf("esperava 3 linhas (stage, done, resumo), vieram %d", len(linhas))
	}
	resumo := linhas[len(linhas)-1]
	if !strings.Contains(resumo, "deltas=300") || !strings.Contains(resumo, "delta_bytes=600") {
		t.Fatalf("resumo sem a contagem de deltas: %q", resumo)
	}
}
