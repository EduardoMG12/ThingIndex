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
	reaper := NewStuckDocumentReaper(repo, 30*time.Minute, 0, time.Minute)

	antes := time.Now().UTC()
	if err := reaper.reapOnce(context.Background()); err != nil {
		t.Fatalf("reapOnce retornou erro: %v", err)
	}
	depois := time.Now().UTC()

	if len(repo.reapCutoffs) != 1 {
		t.Fatalf("esperava 1 chamada ao repositorio, veio %d", len(repo.reapCutoffs))
	}

	// O cutoff fica no passado; sinal trocado marcaria FAILED tudo em processamento.
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
	reaper := NewStuckDocumentReaper(repo, 30*time.Minute, 0, time.Minute)

	if err := reaper.reapOnce(context.Background()); !errors.Is(err, falha) {
		t.Fatalf("esperava o erro do repositorio, veio %v", err)
	}
}

// Run nao pode retornar em falha transitoria de banco; tenta de novo no proximo tick.
func TestReaperSegueRodandoDepoisDeUmErro(t *testing.T) {
	t.Parallel()

	repo := &processingRepoStub{reapErr: errors.New("indisponivel")}
	reaper := NewStuckDocumentReaper(repo, 30*time.Minute, 0, 10*time.Millisecond)

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
	reaper := NewStuckDocumentReaper(repo, 30*time.Minute, 0, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := reaper.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("esperava context.Canceled, veio %v", err)
	}
}

func TestReaperReenfileiraQueuedComCutoffConfigurado(t *testing.T) {
	t.Parallel()

	repo := &processingRepoStub{}
	reaper := NewStuckDocumentReaper(repo, 30*time.Minute, 45*time.Minute, time.Minute)

	antes := time.Now().UTC()
	if err := reaper.reapOnce(context.Background()); err != nil {
		t.Fatalf("reapOnce retornou erro: %v", err)
	}

	if len(repo.requeueCutoffs) != 1 {
		t.Fatalf("esperava 1 chamada de reenfileiramento, veio %d", len(repo.requeueCutoffs))
	}
	esperado := antes.Add(-45 * time.Minute)
	if diff := repo.requeueCutoffs[0].Sub(esperado); diff < -time.Second || diff > time.Second {
		t.Fatalf("cutoff de QUEUED %s longe do esperado %s", repo.requeueCutoffs[0], esperado)
	}
}

func TestReaperComLimiteZeroNaoReenfileira(t *testing.T) {
	t.Parallel()

	repo := &processingRepoStub{}
	reaper := NewStuckDocumentReaper(repo, 30*time.Minute, 0, time.Minute)

	if err := reaper.reapOnce(context.Background()); err != nil {
		t.Fatalf("reapOnce retornou erro: %v", err)
	}
	if len(repo.requeueCutoffs) != 0 {
		t.Fatalf("limite zero deveria desligar a reconciliacao de QUEUED, veio %d chamadas", len(repo.requeueCutoffs))
	}
}

// Uma falha ao liberar PROCESSING nao pode impedir a reconciliacao de QUEUED
// no mesmo ciclo, e o erro precisa subir para o log.
func TestReaperReconciliaQueuedMesmoComErroEmProcessing(t *testing.T) {
	t.Parallel()

	falha := errors.New("indisponivel")
	repo := &processingRepoStub{reapErr: falha}
	reaper := NewStuckDocumentReaper(repo, 30*time.Minute, 30*time.Minute, time.Minute)

	if err := reaper.reapOnce(context.Background()); !errors.Is(err, falha) {
		t.Fatalf("esperava o erro de PROCESSING, veio %v", err)
	}
	if len(repo.requeueCutoffs) != 1 {
		t.Fatalf("a reconciliacao de QUEUED deveria rodar mesmo com erro em PROCESSING")
	}
}
