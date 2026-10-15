package usecase

import (
	"context"
	"errors"
	"net"
	"testing"

	"thing-index/internal/domain"
)

// Com "%w: %w" sentinela e causa ficam na cadeia; com "%v" errors.Is/As sobre
// a causa deixava de funcionar.
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
