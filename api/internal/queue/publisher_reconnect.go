package queue

import (
	"context"
	"fmt"
)

// publicarComReconexao executa a publicacao e, se ela falhar por conexao morta,
// reabre a conexao e tenta MAIS UMA VEZ.
//
// MOTIVO: o publicador guardava conexao e canal criados na subida do processo.
// Quando o RabbitMQ caia e voltava, todo Publish seguinte falhava com
// "channel/connection is not open" para sempre -- nada reabria a conexao.
//
// Observado numa queda real: os eventos do outbox esgotaram as tentativas e
// foram para FAILED DEPOIS que o broker ja tinha voltado. O documento ficou
// QUEUED indefinidamente e so um restart do worker resolvia.
//
// Uma tentativa extra, e nao um laco: o outbox ja tem backoff proprio com
// contagem de tentativas persistida. Um laco aqui competiria com ele e
// esconderia a falha da tabela, que e onde o operador consegue ver.
func publicarComReconexao(
	ctx context.Context,
	publicar func(context.Context) error,
	reconectar func() error,
) error {
	err := publicar(ctx)
	if err == nil {
		return nil
	}

	// Erro de negocio (payload invalido, por exemplo) nao melhora com conexao
	// nova; reconectar mascararia o defeito e dobraria o custo da falha.
	if !erroDeConexaoTransitoria(err) {
		return err
	}

	if reconErr := reconectar(); reconErr != nil {
		// Preserva a causa ORIGINAL: um erro de dial secundario esconderia do
		// operador o motivo pelo qual a publicacao falhou.
		return fmt.Errorf("%w (reconexao tambem falhou: %v)", err, reconErr)
	}

	return publicar(ctx)
}
