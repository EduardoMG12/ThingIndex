package queue

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

type publicadorFalso struct {
	tentativas  int32
	reconexoes  int32
	falharAte   int32
	erroDeCanal error
}

func (p *publicadorFalso) publicar(context.Context) error {
	n := atomic.AddInt32(&p.tentativas, 1)
	if n <= p.falharAte {
		return p.erroDeCanal
	}
	return nil
}

func (p *publicadorFalso) reconectar() error {
	atomic.AddInt32(&p.reconexoes, 1)
	return nil
}

func TestPublicadorReconectaQuandoOCanalMorreu(t *testing.T) {
	falso := &publicadorFalso{
		falharAte:   1,
		erroDeCanal: errors.New(`Exception (504) Reason: "channel/connection is not open"`),
	}

	err := publicarComReconexao(context.Background(), falso.publicar, falso.reconectar)
	if err != nil {
		t.Fatalf("publicacao deveria ter sucesso apos reconectar, veio: %v", err)
	}

	if falso.reconexoes != 1 {
		t.Fatalf("esperava 1 reconexao, houve %d", falso.reconexoes)
	}
	if falso.tentativas != 2 {
		t.Fatalf("esperava 2 tentativas (falha + sucesso), houve %d", falso.tentativas)
	}
}

func TestPublicadorNaoReconectaEmErroDeNegocio(t *testing.T) {
	erroDeNegocio := errors.New("evento sem document_id")
	falso := &publicadorFalso{falharAte: 5, erroDeCanal: erroDeNegocio}

	err := publicarComReconexao(context.Background(), falso.publicar, falso.reconectar)

	if !errors.Is(err, erroDeNegocio) {
		t.Fatalf("esperava o erro original, veio: %v", err)
	}
	if falso.reconexoes != 0 {
		t.Fatalf("reconectou em erro que conexao nova nao resolve: %d", falso.reconexoes)
	}
	if falso.tentativas != 1 {
		t.Fatalf("tentou de novo sem motivo: %d", falso.tentativas)
	}
}

// Se falhar de novo apos reconectar, o erro sobe para o backoff do outbox.
func TestPublicadorTentaUmaVezSo(t *testing.T) {
	erroPersistente := errors.New("connection refused")
	falso := &publicadorFalso{falharAte: 99, erroDeCanal: erroPersistente}

	err := publicarComReconexao(context.Background(), falso.publicar, falso.reconectar)

	if err == nil {
		t.Fatal("esperava erro quando a falha persiste apos a reconexao")
	}
	if falso.tentativas != 2 {
		t.Fatalf("esperava exatamente 2 tentativas, houve %d -- laco de retry aqui competiria "+
			"com o backoff do outbox", falso.tentativas)
	}
}

func TestFalhaAoReconectarDevolveOErroOriginal(t *testing.T) {
	erroDeCanal := errors.New("channel/connection is not open")
	falso := &publicadorFalso{falharAte: 1, erroDeCanal: erroDeCanal}

	erroDeDial := errors.New("dial rabbitmq: connection refused")
	err := publicarComReconexao(context.Background(), falso.publicar, func() error {
		return erroDeDial
	})

	if err == nil {
		t.Fatal("esperava erro quando a reconexao falha")
	}
	if !errors.Is(err, erroDeCanal) {
		t.Fatalf("o erro devolvido precisa preservar a causa original (%v), veio: %v",
			erroDeCanal, err)
	}
}
