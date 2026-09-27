package usecase

import (
	"context"
	"errors"
	"net"
	"testing"

	"thing-index/internal/domain"
)

// Parecer B7: fmt.Errorf("%w: %v", sentinela, err) preservava a sentinela mas
// convertia a causa em texto; errors.Is/As sobre a causa deixavam de
// funcionar acima do caso de uso. Com "%w: %w" os dois lados continuam na
// cadeia, e a mensagem e a mesma.
func TestErroEmbrulhadoPreservaSentinelaECausa(t *testing.T) {
	causa := &net.OpError{Op: "dial", Net: "tcp", Err: context.DeadlineExceeded}

	err := classifyProviderConnectionError(domain.FamilyLLM, "OPENAI", causa)

	if !errors.Is(err, domain.ErrProviderConnectionTimeout) {
		t.Fatalf("sentinela perdida: %v", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("causa perdida: errors.Is(err, context.DeadlineExceeded) = false para %v", err)
	}
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		t.Fatalf("errors.As nao alcanca a causa *net.OpError em %v", err)
	}
	if want := domain.ErrProviderConnectionTimeout.Error() + ": " + causa.Error(); err.Error() != want {
		t.Fatalf("mensagem mudou: %q, esperava %q", err.Error(), want)
	}
}
