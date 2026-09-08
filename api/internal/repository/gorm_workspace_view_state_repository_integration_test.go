//go:build integration

package repository

import (
	"context"
	"testing"

	"thing-index/internal/models"
)

func TestWorkspaceViewStateUpsertEIsolamento(t *testing.T) {
	db := openIntegrationDB(t)
	repo := NewGormWorkspaceViewStateRepository(db)
	ctx := context.Background()

	userA := seedIntegrationUser(t, db, "estado-a")
	catA := seedIntegrationCategory(t, db, userA.ID, "Editais")
	colA := seedIntegrationCollection(t, db, userA.ID, catA.ID, "PS 2027")
	userB := seedIntegrationUser(t, db, "estado-b")

	if err := repo.UpsertViewState(ctx, userA.ID, colA.ID, map[string]any{"viewport": map[string]any{"escala": 0.5}}); err != nil {
		t.Fatalf("primeiro upsert: %v", err)
	}
	if err := repo.UpsertViewState(ctx, userA.ID, colA.ID, map[string]any{"viewport": map[string]any{"escala": 1.25}}); err != nil {
		t.Fatalf("segundo upsert: %v", err)
	}
	var total int64
	db.Model(&models.WorkspaceViewState{}).Where("user_id = ?", userA.ID).Count(&total)
	if total != 1 {
		t.Fatalf("linhas = %d, quero 1 (upsert nao pode duplicar)", total)
	}
	lido, err := repo.GetViewState(ctx, userA.ID, colA.ID)
	if err != nil {
		t.Fatalf("GetViewState: %v", err)
	}
	vp, _ := lido["viewport"].(map[string]any)
	if vp["escala"] != 1.25 {
		t.Fatalf("estado apos upsert = %v, quero escala 1.25", lido)
	}
	deB, err := repo.GetViewState(ctx, userB.ID, colA.ID)
	if err != nil || len(deB) != 0 {
		t.Fatalf("B lendo estado de A: %v, %v", err, deB)
	}
}
