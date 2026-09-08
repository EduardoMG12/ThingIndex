//go:build integration

package repository

import (
	"context"
	"sync"
	"testing"

	"thing-index/internal/models"
)

// Vinte uploads simultaneos com o MESMO nome de colecao devolveram um HTTP 500:
// "failed to create collection: duplicated key not allowed".
//
// A causa e um check-then-act classico no usecase: busca a colecao, nao
// encontra, cria. Entre a busca e a criacao, outra requisicao criou a mesma --
// e a constraint de unicidade (correta) rejeita a segunda.
//
// O usuario que perde a corrida ve um erro de servidor por ter feito algo
// legitimo: enviar varios documentos de uma vez para a mesma colecao. E o caso
// mais comum de uso real do sistema.
//
// A correcao certa e no repositorio, nao no usecase: so o banco pode decidir
// quem ganha a corrida. Reexecutar a busca no usecase apenas moveria a janela.

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

	// Todas as goroutines largam juntas: sem a barreira, elas serializariam
	// naturalmente e a corrida nunca aconteceria.
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

	// Todas precisam ter recebido o MESMO id: ids diferentes significariam
	// colecoes duplicadas, e o usuario veria o acervo dividido em duas.
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

// TestEnsureColecaoDevolveAExistenteSemDuplicar cobre o caminho simples: chamar
// duas vezes em sequencia nao pode criar duas.
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

// TestColecoesDeUsuariosDiferentesNaoColidem: a unicidade e por usuario, entao
// dois usuarios podem ter "Contratos" sem um enxergar o outro.
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
