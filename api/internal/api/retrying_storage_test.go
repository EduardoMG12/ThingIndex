package api

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"thing-index/internal/domain"
)

type relogioFalso struct {
	mu    sync.Mutex
	agora time.Time
}

func (r *relogioFalso) Now() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.agora
}

func (r *relogioFalso) avancar(d time.Duration) {
	r.mu.Lock()
	r.agora = r.agora.Add(d)
	r.mu.Unlock()
}

type storageReal struct{}

func (storageReal) UploadFile(context.Context, io.Reader, int64, string, string) (string, error) {
	return "documents/ok.pdf", nil
}

func (storageReal) OpenFile(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("conteudo")), nil
}

func (storageReal) DeleteFile(context.Context, string) error { return nil }

func (storageReal) FileExists(context.Context, string) (bool, error) { return true, nil }

// fabricaQueFalha falha nas primeiras `falhas` chamadas e depois funciona.
type fabricaQueFalha struct {
	falhas   int
	chamadas int
}

func (f *fabricaQueFalha) build() (domain.StorageService, error) {
	f.chamadas++
	if f.chamadas <= f.falhas {
		return nil, errors.New("dial tcp: lookup minio: no such host")
	}
	return storageReal{}, nil
}

// MinIO fora no boot congelava o erro ate reiniciar a API.
func TestStorageQueFalhouNoBootPassaAFuncionarQuandoOProvedorVolta(t *testing.T) {
	relogio := &relogioFalso{agora: time.Unix(1_000, 0)}
	fabrica := &fabricaQueFalha{falhas: 3}

	_, bootErr := fabrica.build()
	storage := newRetryingStorage(fabrica.build, bootErr, 5*time.Second, relogio.Now)

	if _, err := storage.OpenFile(context.Background(), "a.pdf"); err == nil {
		t.Fatal("dentro do intervalo devia devolver o erro do boot")
	}
	if fabrica.chamadas != 1 {
		t.Fatalf("tentou reconstruir antes do intervalo: %d chamadas", fabrica.chamadas)
	}

	// Duas tentativas que ainda falham, espacadas pelo intervalo.
	for i := 0; i < 2; i++ {
		relogio.avancar(5 * time.Second)
		if _, err := storage.OpenFile(context.Background(), "a.pdf"); err == nil {
			t.Fatalf("tentativa %d devia falhar", i+2)
		}
	}
	if fabrica.chamadas != 3 {
		t.Fatalf("esperava 3 construcoes ate aqui, vieram %d", fabrica.chamadas)
	}

	relogio.avancar(5 * time.Second)
	leitor, err := storage.OpenFile(context.Background(), "a.pdf")
	if err != nil {
		t.Fatalf("o provedor voltou e o storage continuou falhando: %v", err)
	}
	_ = leitor.Close()

	// Depois de construido, nao tenta mais: usa o cliente real direto.
	if _, err := storage.UploadFile(context.Background(), strings.NewReader("x"), 1, "application/pdf", "b.pdf"); err != nil {
		t.Fatalf("upload apos recuperacao falhou: %v", err)
	}
	if fabrica.chamadas != 4 {
		t.Fatalf("reconstruiu depois de recuperado: %d chamadas", fabrica.chamadas)
	}
}

func TestStorageQueFalhouNoBootRespeitaOIntervaloMinimoEntreTentativas(t *testing.T) {
	relogio := &relogioFalso{agora: time.Unix(1_000, 0)}
	fabrica := &fabricaQueFalha{falhas: 100}
	storage := newRetryingStorage(fabrica.build, errors.New("boot"), 5*time.Second, relogio.Now)

	for i := 0; i < 50; i++ {
		relogio.avancar(100 * time.Millisecond)
		_, _ = storage.FileExists(context.Background(), "a.pdf")
	}
	// 5 s de relogio em passos de 100 ms: exatamente uma tentativa.
	if fabrica.chamadas != 1 {
		t.Fatalf("esperava 1 tentativa em 5 s, vieram %d", fabrica.chamadas)
	}
}

func TestResolveDefaultStorageUsaOClienteDiretoQuandoOBootFunciona(t *testing.T) {
	fabrica := &fabricaQueFalha{}
	storage := resolveDefaultStorage(fabrica.build)
	if _, ok := storage.(storageReal); !ok {
		t.Fatalf("boot com sucesso nao devia embrulhar o cliente: %T", storage)
	}
}

func TestResolveDefaultStorageEmbrulhaQuandoOBootFalha(t *testing.T) {
	fabrica := &fabricaQueFalha{falhas: 1}
	storage := resolveDefaultStorage(fabrica.build)
	if _, ok := storage.(*retryingStorage); !ok {
		t.Fatalf("boot com falha devia devolver o storage que tenta de novo: %T", storage)
	}
}
