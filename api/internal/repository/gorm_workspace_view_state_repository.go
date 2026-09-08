package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

type GormWorkspaceViewStateRepository struct {
	db *gorm.DB
}

func NewGormWorkspaceViewStateRepository(db *gorm.DB) domain.WorkspaceViewStateRepository {
	return &GormWorkspaceViewStateRepository{db: db}
}

// GetViewState devolve mapa vazio quando nao ha estado salvo: ausencia nao e erro.
func (r *GormWorkspaceViewStateRepository) GetViewState(ctx context.Context, userID, collectionID uint) (map[string]any, error) {
	var row models.WorkspaceViewState
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND collection_id = ?", userID, collectionID).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	if row.State == nil {
		return map[string]any{}, nil
	}
	return row.State, nil
}

func (r *GormWorkspaceViewStateRepository) UpsertViewState(ctx context.Context, userID, collectionID uint, state map[string]any) error {
	row := models.WorkspaceViewState{UserID: userID, CollectionID: collectionID, State: state}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "collection_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"state", "updated_at"}),
		}).
		Create(&row).Error
}
