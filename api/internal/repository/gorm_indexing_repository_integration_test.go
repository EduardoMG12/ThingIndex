//go:build integration

package repository

import (
	"context"
	"testing"

	"thing-index/internal/domain"
)

// Regressao P0-1 contra Postgres real: o indice unico precisa aceitar o mesmo
// (tipo, modelo, hash) para usuarios diferentes, e a leitura precisa filtrar
// por usuario e provedor.
func TestCacheDeEnriquecimentoIsolaUsuarioEProvedor(t *testing.T) {
	db := openIntegrationDB(t)
	repo := NewGormIndexingRepository(db)
	ctx := context.Background()

	// user_id tem FK para users: os dois donos precisam existir.
	idAtacante := seedIntegrationUser(t, db, "atacante").ID
	idVitima := seedIntegrationUser(t, db, "vitima").ID

	atacante := domain.EnrichmentCacheScope{UserID: idAtacante, Provider: "OPENAI|evil.example.com", Model: "gpt-4o-mini"}
	vitima := domain.EnrichmentCacheScope{UserID: idVitima, Provider: "OPENAI|api.openai.com", Model: "gpt-4o-mini"}
	vitimaOutroHost := domain.EnrichmentCacheScope{UserID: idVitima, Provider: "OPENAI|gateway.example.com", Model: "gpt-4o-mini"}

	if err := repo.StoreEnrichment(ctx, atacante, []domain.EnrichmentCacheEntry{{Kind: "page", ContentHash: "h1", Summary: "envenenado"}}); err != nil {
		t.Fatalf("gravar do atacante: %v", err)
	}
	encontrados, err := repo.LookupEnrichment(ctx, vitima, "page", []string{"h1"})
	if err != nil {
		t.Fatalf("lookup da vitima: %v", err)
	}
	if len(encontrados) != 0 {
		t.Fatalf("a vitima leu a entrada do atacante: %+v", encontrados)
	}

	// Mesmo (tipo, modelo, hash) para outro usuario precisa gravar, nao colidir.
	if err := repo.StoreEnrichment(ctx, vitima, []domain.EnrichmentCacheEntry{{Kind: "page", ContentHash: "h1", Summary: "legitimo"}}); err != nil {
		t.Fatalf("gravar da vitima: %v", err)
	}
	encontrados, err = repo.LookupEnrichment(ctx, vitima, "page", []string{"h1"})
	if err != nil {
		t.Fatalf("lookup da vitima: %v", err)
	}
	if encontrados["h1"].Summary != "legitimo" {
		t.Fatalf("a vitima devia ler o proprio resumo, veio %+v", encontrados)
	}

	// Mesmo usuario, outro host com o mesmo nome de modelo: nao reaproveita.
	encontrados, err = repo.LookupEnrichment(ctx, vitimaOutroHost, "page", []string{"h1"})
	if err != nil {
		t.Fatalf("lookup em outro host: %v", err)
	}
	if len(encontrados) != 0 {
		t.Fatalf("resumo de um host foi servido para outro: %+v", encontrados)
	}
}

func TestCacheDeEnriquecimentoSemUsuarioNaoLeNemGrava(t *testing.T) {
	db := openIntegrationDB(t)
	repo := NewGormIndexingRepository(db)
	ctx := context.Background()

	semUsuario := domain.EnrichmentCacheScope{Model: "m"}
	if err := repo.StoreEnrichment(ctx, semUsuario, []domain.EnrichmentCacheEntry{{Kind: "page", ContentHash: "h", Summary: "s"}}); err != nil {
		t.Fatalf("gravar sem usuario: %v", err)
	}
	var total int64
	if err := db.Table("enrichment_cache_entries").Count(&total).Error; err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Fatalf("entrada sem usuario foi gravada: %d linhas", total)
	}
}
