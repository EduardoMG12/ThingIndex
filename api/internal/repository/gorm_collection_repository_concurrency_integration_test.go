//go:build integration

package repository

import (
	"context"
	"sync"
	"testing"

	"thing-index/internal/models"
)

// Uploads simultaneos com o mesmo nome de colecao davam 500 por duplicated key
// (check-then-act no usecase).

func TestCriarColecaoConcorrenteNaoDuplicaNemFalha(t *testing.T) {
	db := openIntegrationDB(t)
	repo := NewGormCollectionRepository(db)
	ctx := context.Background()

	usuario := seedIntegrationUser(t, db, "concorrencia")

	const concorrentes = 12
	const nome = "Colecao Concorrente"
	const normalizado = "colecao concorrente"

	var espera sync.WaitGroup
	resultados := make([]error, concorrentes)
	ids := make([]uint, concorrentes)

	// Barreira para todas largarem juntas e a corrida acontecer de fato.
	largada := make(chan struct{})

	for i := 0; i < concorrentes; i++ {
		espera.Add(1)
		go func(indice int) {
			defer espera.Done()
			<-largada

			colecao := &models.Collection{
				UserID:         usuario.ID,
				Name:           nome,
				NormalizedName: normalizado,
				Metadata:       map[string]any{},
			}

			err := repo.EnsureCollection(ctx, colecao)
			resultados[indice] = err
			if err == nil {
				ids[indice] = colecao.ID
			}
		}(i)
	}

	close(largada)
	espera.Wait()

	for i, err := range resultados {
		if err != nil {
			t.Fatalf("goroutine %d falhou ao garantir a colecao: %v", i, err)
		}
	}

	primeiro := ids[0]
	for i, id := range ids {
		if id == 0 {
			t.Fatalf("goroutine %d nao recebeu id de colecao", i)
		}
		if id != primeiro {
			t.Fatalf("colecao duplicada: goroutine %d recebeu id %d, esperado %d", i, id, primeiro)
		}
	}

	var total int64
	if err := db.Model(&models.Collection{}).
		Where("user_id = ? AND normalized_name = ?", usuario.ID, normalizado).
		Count(&total).Error; err != nil {
		t.Fatalf("contagem falhou: %v", err)
	}
	if total != 1 {
		t.Fatalf("esperava exatamente 1 colecao no banco, encontrou %d", total)
	}
}

func TestEnsureColecaoDevolveAExistenteSemDuplicar(t *testing.T) {
	db := openIntegrationDB(t)
	repo := NewGormCollectionRepository(db)
	ctx := context.Background()

	usuario := seedIntegrationUser(t, db, "concorrencia")

	primeira := &models.Collection{
		UserID:         usuario.ID,
		Name:           "Contratos",
		NormalizedName: "contratos",
		Metadata:       map[string]any{},
	}
	if err := repo.EnsureCollection(ctx, primeira); err != nil {
		t.Fatalf("primeira chamada falhou: %v", err)
	}

	segunda := &models.Collection{
		UserID:         usuario.ID,
		Name:           "Contratos",
		NormalizedName: "contratos",
		Metadata:       map[string]any{},
	}
	if err := repo.EnsureCollection(ctx, segunda); err != nil {
		t.Fatalf("segunda chamada falhou: %v", err)
	}

	if segunda.ID != primeira.ID {
		t.Fatalf("segunda chamada criou colecao nova (%d) em vez de reusar (%d)",
			segunda.ID, primeira.ID)
	}
}

func TestColecoesDeUsuariosDiferentesNaoColidem(t *testing.T) {
	db := openIntegrationDB(t)
	repo := NewGormCollectionRepository(db)
	ctx := context.Background()

	usuarioA := seedIntegrationUser(t, db, "colisaoA")
	usuarioB := seedIntegrationUser(t, db, "colisaoB")

	colecaoA := &models.Collection{
		UserID: usuarioA.ID, Name: "Contratos", NormalizedName: "contratos",
		Metadata: map[string]any{},
	}
	colecaoB := &models.Collection{
		UserID: usuarioB.ID, Name: "Contratos", NormalizedName: "contratos",
		Metadata: map[string]any{},
	}

	if err := repo.EnsureCollection(ctx, colecaoA); err != nil {
		t.Fatalf("colecao do usuario A falhou: %v", err)
	}
	if err := repo.EnsureCollection(ctx, colecaoB); err != nil {
		t.Fatalf("colecao do usuario B falhou: %v", err)
	}

	if colecaoA.ID == colecaoB.ID {
		t.Fatal("usuarios diferentes compartilharam a mesma colecao: vazamento entre contas")
	}
}
