package processing

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestReaperCalculaOCutoffAPartirDoLimiteConfigurado(t *testing.T) {
	t.Parallel()

	repo := &processingRepoStub{}
	reaper := NewStuckDocumentReaper(repo, 30*time.Minute, time.Minute)

	antes := time.Now().UTC()
	if err := reaper.reapOnce(context.Background()); err != nil {
		t.Fatalf("reapOnce retornou erro: %v", err)
	}
	depois := time.Now().UTC()

	if len(repo.reapCutoffs) != 1 {
		t.Fatalf("esperava 1 chamada ao repositorio, veio %d", len(repo.reapCutoffs))
	}

	// O cutoff precisa apontar para o PASSADO pelo tamanho da janela. Um erro
	// de sinal aqui marcaria FAILED todo documento em processamento no
	// instante seguinte ao upload -- exatamente o oposto do objetivo.
	cutoff := repo.reapCutoffs[0]
	if !cutoff.Before(antes) {
		t.Fatalf("o cutoff deveria estar no passado, veio %s (agora ~%s)", cutoff, antes)
	}

	esperadoMin := antes.Add(-30 * time.Minute).Add(-time.Second)
	esperadoMax := depois.Add(-30 * time.Minute).Add(time.Second)
	if cutoff.Before(esperadoMin) || cutoff.After(esperadoMax) {
		t.Fatalf("cutoff %s fora da janela esperada [%s, %s]", cutoff, esperadoMin, esperadoMax)
	}
}

func TestReaperPropagaErroDoRepositorio(t *testing.T) {
	t.Parallel()

	falha := errors.New("conexao recusada")
	repo := &processingRepoStub{reapErr: falha}
	reaper := NewStuckDocumentReaper(repo, 30*time.Minute, time.Minute)

	if err := reaper.reapOnce(context.Background()); !errors.Is(err, falha) {
		t.Fatalf("esperava o erro do repositorio, veio %v", err)
	}
}

// O Run nao pode morrer por falha transitoria de banco: o worker continua
// consumindo a fila normalmente, e o reaper tenta de novo no proximo tick.
// Este teste falha (por timeout do -race/CI) se Run retornar no primeiro erro.
func TestReaperSegueRodandoDepoisDeUmErro(t *testing.T) {
	t.Parallel()

	repo := &processingRepoStub{reapErr: errors.New("indisponivel")}
	reaper := NewStuckDocumentReaper(repo, 30*time.Minute, 10*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()

	err := reaper.Run(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run deveria sair apenas pelo contexto, veio %v", err)
	}

	if len(repo.reapCutoffs) < 2 {
		t.Fatalf("esperava varias tentativas apesar do erro, veio %d", len(repo.reapCutoffs))
	}
}

func TestReaperParaQuandoOContextoEncerra(t *testing.T) {
	t.Parallel()

	repo := &processingRepoStub{}
	reaper := NewStuckDocumentReaper(repo, 30*time.Minute, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := reaper.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("esperava context.Canceled, veio %v", err)
	}
}
