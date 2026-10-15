//go:build integration

package storage

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"

	"github.com/joho/godotenv"
)

// O cliente do storage fala HTTP sem estado e abre conexao a cada operacao, entao
// volta sozinho depois de uma queda (ao contrario da conexao AMQP de longa vida da
// fila). Este teste trava isso contra um MinIO real.

var loadResilienciaEnvOnce sync.Once

func storageDeTeste(t *testing.T) *MinioStorage {
	t.Helper()

	loadResilienciaEnvOnce.Do(func() {
		_ = godotenv.Load("../../../../.env")
	})

	cfg := minioIntegrationConfigFromEnv(t)
	storage, err := NewMinioStorage(cfg.endpoint, cfg.accessKey, cfg.secretKey, cfg.bucketName, cfg.useSSL)
	if err != nil {
		t.Fatalf("NewMinioStorage devolveu erro: %v", err)
	}
	return storage
}

func TestClienteDeStorageSobreviveAErroDeRede(t *testing.T) {
	storage := storageDeTeste(t)
	ctx := context.Background()

	conteudo := []byte("conteudo de teste para resiliencia")
	caminho, err := storage.UploadFile(ctx, bytes.NewReader(conteudo), int64(len(conteudo)), "text/plain", "resiliencia/documento.txt")
	if err != nil {
		t.Fatalf("upload inicial falhou: %v", err)
	}
	t.Cleanup(func() { _ = storage.DeleteFile(context.Background(), caminho) })

	// Contexto ja cancelado simula a falha de rede.
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()

	if _, err := storage.OpenFile(ctxCancelado, caminho); err == nil {
		t.Fatal("leitura com contexto cancelado deveria falhar")
	}

	// A operacao seguinte tem de funcionar sem reinicializar o cliente.
	leitor, err := storage.OpenFile(ctx, caminho)
	if err != nil {
		t.Fatalf("cliente nao se recuperou apos erro de rede: %v", err)
	}
	defer leitor.Close()

	recebido, err := io.ReadAll(leitor)
	if err != nil {
		t.Fatalf("leitura apos recuperacao falhou: %v", err)
	}
	if !bytes.Equal(recebido, conteudo) {
		t.Fatalf("conteudo divergente apos recuperacao: %q", recebido)
	}
}

// Acumular estado entre chamadas seria o primeiro sintoma de conexao presa.
func TestOperacoesSucessivasNaoDegradam(t *testing.T) {
	storage := storageDeTeste(t)
	ctx := context.Background()

	conteudo := []byte("payload")
	var caminho string
	t.Cleanup(func() {
		if caminho != "" {
			_ = storage.DeleteFile(context.Background(), caminho)
		}
	})

	for i := 0; i < 12; i++ {
		var err error
		caminho, err = storage.UploadFile(ctx, bytes.NewReader(conteudo), int64(len(conteudo)), "text/plain", "resiliencia/lote.txt")
		if err != nil {
			t.Fatalf("upload %d falhou: %v", i, err)
		}

		leitor, err := storage.OpenFile(ctx, caminho)
		if err != nil {
			t.Fatalf("leitura %d falhou: %v", i, err)
		}
		_ = leitor.Close()
	}
}
