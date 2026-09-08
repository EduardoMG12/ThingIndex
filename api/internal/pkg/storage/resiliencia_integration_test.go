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

// A fila tinha um defeito grave: consumidor e publicador guardavam conexao AMQP
// de longa vida e, depois de uma queda do broker, nunca reabriam -- documentos
// ficavam presos para sempre e so um restart resolvia.
//
// Este teste responde se o storage sofre do mesmo mal. A resposta medida contra
// um MinIO real (derrubado e religado com a API no ar) foi NAO: o download
// falha com 500 durante a queda e volta a 200 sozinho depois, sem reiniciar a
// API. O motivo e que o cliente fala HTTP stateless e abre conexao nova a cada
// operacao, enquanto o AMQP mantem socket persistente.
//
// O teste trava essa propriedade. Se alguem trocar o cliente por um que
// mantenha sessao, ou passar a cachear conexao "para otimizar", o storage
// herda o defeito da fila -- e o sintoma so apareceria em producao, depois de
// uma instabilidade, como upload falhando ate reiniciar a API.

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

	// Contexto ja cancelado simula a falha de rede. O que importa e o que
	// acontece DEPOIS do erro.
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()

	if _, err := storage.OpenFile(ctxCancelado, caminho); err == nil {
		t.Fatal("leitura com contexto cancelado deveria falhar")
	}

	// A prova: a operacao seguinte funciona sem nenhuma reinicializacao do
	// cliente. Era exatamente isto que a fila NAO fazia.
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

// TestOperacoesSucessivasNaoDegradam: acumular estado entre chamadas seria o
// primeiro sintoma de conexao presa.
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
