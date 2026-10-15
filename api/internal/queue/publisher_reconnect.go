package queue

import (
	"context"
	"fmt"
)

// publicarComReconexao reabre a conexao se a publicacao falhar por conexao
// morta e tenta mais uma vez so; o outbox ja tem backoff proprio.
func publicarComReconexao(
	ctx context.Context,
	publicar func(context.Context) error,
	reconectar func() error,
) error {
	err := publicar(ctx)
	if err == nil {
		return nil
	}

	if !erroDeConexaoTransitoria(err) {
		return err
	}

	if reconErr := reconectar(); reconErr != nil {
		// Mantem a causa original visivel.
		return fmt.Errorf("%w (reconexao tambem falhou: %v)", err, reconErr)
	}

	return publicar(ctx)
}
