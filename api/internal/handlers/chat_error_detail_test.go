package handlers

import (
	"errors"
	"strings"
	"testing"
)

func TestChatErrorDetailUmaLinhaCurta(t *testing.T) {
	err := errors.New("you have reached your\n monthly usage limit " + strings.Repeat("x", 300))
	got := chatErrorDetail(err)
	if strings.Contains(got, "\n") || len([]rune(got)) > 201 || !strings.HasPrefix(got, "you have reached your monthly usage limit") {
		t.Fatalf("detalhe mal formatado: %q", got)
	}
	if chatErrorDetail(nil) != "" {
		t.Fatal("sem erro nao ha detalhe")
	}
}
