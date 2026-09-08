//go:build integration

package repository

import (
	"context"
	"errors"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

// Cenario: dois usuarios, cada um com sua colecao e seu documento com um
// paragrafo. O que se prova aqui e o escopo transitivo -- a constituicao exige
// teste para ele -- mais o upsert e o cascade.
func TestParagraphHighlightIsolamentoEntreUsuarios(t *testing.T) {
	db := openIntegrationDB(t)
	repo := NewGormParagraphHighlightRepository(db)
	ctx := context.Background()

	userA := seedIntegrationUser(t, db, "marca-a")
	catA := seedIntegrationCategory(t, db, userA.ID, "Editais")
	colA := seedIntegrationCollection(t, db, userA.ID, catA.ID, "PS 2027")
	docA := seedIntegrationDocument(t, db, userA.ID, catA.ID, colA.ID, "edital.pdf")
	paraA := models.ParagraphNode{DocumentID: docA.ID, SequenceOrder: 0, Content: "prazo de inscricao"}
	mustCreateRecord(t, db, &paraA)

	userB := seedIntegrationUser(t, db, "marca-b")
	catB := seedIntegrationCategory(t, db, userB.ID, "Normas")
	colB := seedIntegrationCollection(t, db, userB.ID, catB.ID, "Regimento")

	// A marca o proprio paragrafo.
	marca, err := repo.UpsertHighlight(ctx, userA.ID, colA.ID, paraA.ID, "amarelo")
	if err != nil {
		t.Fatalf("UpsertHighlight: %v", err)
	}
	if marca.DocumentID != docA.ID || marca.Color != "amarelo" {
		t.Fatalf("marca = %+v", marca)
	}

	// B nao consegue marcar o paragrafo de A, nem via a propria colecao, nem
	// via a colecao de A.
	if _, err := repo.UpsertHighlight(ctx, userB.ID, colB.ID, paraA.ID, "verde"); !errors.Is(err, domain.ErrParagraphNotInCollection) {
		t.Fatalf("B marcando paragrafo de A pela propria colecao: quero ErrParagraphNotInCollection, veio %v", err)
	}
	if _, err := repo.UpsertHighlight(ctx, userB.ID, colA.ID, paraA.ID, "verde"); !errors.Is(err, domain.ErrParagraphNotInCollection) {
		t.Fatalf("B marcando paragrafo de A pela colecao de A: quero ErrParagraphNotInCollection, veio %v", err)
	}

	// B nao lista nem apaga a marca de A.
	lista, err := repo.ListHighlights(ctx, userB.ID, colA.ID)
	if err != nil || len(lista) != 0 {
		t.Fatalf("B listando a colecao de A: %v, %d itens", err, len(lista))
	}
	if err := repo.DeleteHighlight(ctx, userB.ID, colA.ID, paraA.ID); err != nil {
		t.Fatalf("delete de B nao devia falhar (idempotente): %v", err)
	}
	lista, _ = repo.ListHighlights(ctx, userA.ID, colA.ID)
	if len(lista) != 1 {
		t.Fatalf("a marca de A sumiu depois do delete de B: %d itens", len(lista))
	}
}

func TestParagraphHighlightUpsertTrocaACorSemDuplicar(t *testing.T) {
	db := openIntegrationDB(t)
	repo := NewGormParagraphHighlightRepository(db)
	ctx := context.Background()

	user := seedIntegrationUser(t, db, "marca-upsert")
	cat := seedIntegrationCategory(t, db, user.ID, "Editais")
	col := seedIntegrationCollection(t, db, user.ID, cat.ID, "PS 2027")
	doc := seedIntegrationDocument(t, db, user.ID, cat.ID, col.ID, "edital.pdf")
	para := models.ParagraphNode{DocumentID: doc.ID, SequenceOrder: 0, Content: "x"}
	mustCreateRecord(t, db, &para)

	if _, err := repo.UpsertHighlight(ctx, user.ID, col.ID, para.ID, "amarelo"); err != nil {
		t.Fatalf("primeiro upsert: %v", err)
	}
	segunda, err := repo.UpsertHighlight(ctx, user.ID, col.ID, para.ID, "roxo")
	if err != nil {
		t.Fatalf("segundo upsert: %v", err)
	}
	if segunda.Color != "roxo" {
		t.Fatalf("cor apos upsert = %q", segunda.Color)
	}
	var total int64
	db.Model(&models.ParagraphHighlight{}).Where("user_id = ?", user.ID).Count(&total)
	if total != 1 {
		t.Fatalf("linhas = %d, quero 1", total)
	}

	// Apagar e marcar de novo nao pode esbarrar no indice unico.
	if err := repo.DeleteHighlight(ctx, user.ID, col.ID, para.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.UpsertHighlight(ctx, user.ID, col.ID, para.ID, "azul"); err != nil {
		t.Fatalf("upsert apos delete: %v", err)
	}
}

func TestParagraphHighlightCascadeAoApagarParagrafo(t *testing.T) {
	db := openIntegrationDB(t)
	repo := NewGormParagraphHighlightRepository(db)
	ctx := context.Background()

	user := seedIntegrationUser(t, db, "marca-cascade")
	cat := seedIntegrationCategory(t, db, user.ID, "Editais")
	col := seedIntegrationCollection(t, db, user.ID, cat.ID, "PS 2027")
	doc := seedIntegrationDocument(t, db, user.ID, cat.ID, col.ID, "edital.pdf")
	para := models.ParagraphNode{DocumentID: doc.ID, SequenceOrder: 0, Content: "x"}
	mustCreateRecord(t, db, &para)
	if _, err := repo.UpsertHighlight(ctx, user.ID, col.ID, para.ID, "verde"); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Reindexar substitui os paragrafos (delete fisico): a marca vai junto,
	// pela FK com ON DELETE CASCADE, em vez de virar linha orfa.
	if err := db.Unscoped().Delete(&models.ParagraphNode{}, para.ID).Error; err != nil {
		t.Fatalf("apagar paragrafo: %v", err)
	}
	var total int64
	db.Unscoped().Model(&models.ParagraphHighlight{}).Where("paragraph_node_id = ?", para.ID).Count(&total)
	if total != 0 {
		t.Fatalf("marca orfa sobreviveu ao delete do paragrafo: %d", total)
	}
}

func TestUpdateCollectionMetadataKeyPreservaAsOutrasChaves(t *testing.T) {
	db := openIntegrationDB(t)
	ctx := context.Background()
	user := seedIntegrationUser(t, db, "legenda")
	cat := seedIntegrationCategory(t, db, user.ID, "Editais")
	col := seedIntegrationCollection(t, db, user.ID, cat.ID, "PS 2027")
	if err := db.Model(&models.Collection{}).Where("id = ?", col.ID).Update("metadata", map[string]any{"origem": "seed"}).Error; err != nil {
		t.Fatalf("metadata inicial: %v", err)
	}

	repo := &GormCollectionRepository{db: db}
	if err := repo.UpdateCollectionMetadataKey(ctx, user.ID, col.ID, domain.LegendKey, []byte(`{"amarelo":"prazo"}`)); err != nil {
		t.Fatalf("UpdateCollectionMetadataKey: %v", err)
	}
	var lida models.Collection
	if err := db.First(&lida, col.ID).Error; err != nil {
		t.Fatalf("reler: %v", err)
	}
	if lida.Metadata["origem"] != "seed" {
		t.Fatalf("a outra chave sumiu: %v", lida.Metadata)
	}
	legenda, _ := lida.Metadata[domain.LegendKey].(map[string]any)
	if legenda["amarelo"] != "prazo" {
		t.Fatalf("legenda = %v", lida.Metadata[domain.LegendKey])
	}

	outro := seedIntegrationUser(t, db, "legenda-outro")
	if err := repo.UpdateCollectionMetadataKey(ctx, outro.ID, col.ID, domain.LegendKey, []byte(`{}`)); !errors.Is(err, domain.ErrCollectionNotFound) {
		t.Fatalf("outro usuario escrevendo a legenda: quero ErrCollectionNotFound, veio %v", err)
	}
}
