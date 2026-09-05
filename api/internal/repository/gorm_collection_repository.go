package repository

import (
	"context"
	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormCollectionRepository struct {
	db *gorm.DB
}

func NewGormCollectionRepository(db *gorm.DB) domain.CollectionRepository {
	return &GormCollectionRepository{db: db}
}

func (r *GormCollectionRepository) CreateCollection(ctx context.Context, collection *models.Collection) error {
	return r.db.WithContext(ctx).Create(collection).Error
}

// EnsureCollection cria a colecao ou devolve a existente, numa unica ida ao
// banco.
//
// MOTIVO: o usecase fazia buscar-e-criar, um check-then-act. Com 20 uploads
// simultaneos para a mesma colecao, um deles caia em
// "duplicated key not allowed" e o usuario recebia HTTP 500 por ter feito algo
// legitimo -- enviar varios documentos de uma vez.
//
// ON CONFLICT DO NOTHING + busca no mesmo fluxo: so o banco pode arbitrar quem
// ganha a corrida. Reexecutar a busca no usecase apenas moveria a janela.
//
// O DoNothing e deliberado, em vez de DoUpdates: quem perde a corrida quer a
// colecao existente como ela esta, nao sobrescrever metadados de quem chegou
// primeiro.
func (r *GormCollectionRepository) EnsureCollection(ctx context.Context, collection *models.Collection) error {
	db := r.db.WithContext(ctx)

	resultado := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "normalized_name"}},
		DoNothing: true,
	}).Create(collection)

	if resultado.Error != nil {
		return resultado.Error
	}

	// Insercao efetiva: o id ja veio preenchido.
	if resultado.RowsAffected > 0 && collection.ID != 0 {
		return nil
	}

	// Conflito: outra requisicao criou primeiro. Busca a vencedora para
	// devolver o mesmo id a todos os chamadores -- ids diferentes dividiriam o
	// acervo do usuario em duas colecoes com o mesmo nome.
	var existente models.Collection
	if err := db.
		Where("user_id = ? AND normalized_name = ?", collection.UserID, collection.NormalizedName).
		First(&existente).Error; err != nil {
		return err
	}

	*collection = existente
	return nil
}

func (r *GormCollectionRepository) UpdateCollection(ctx context.Context, collection *models.Collection) error {
	return r.db.WithContext(ctx).Save(collection).Error
}

func (r *GormCollectionRepository) GetCollectionByID(ctx context.Context, id uint, userID uint) (*models.Collection, error) {
	var collection models.Collection
	if err := r.db.WithContext(ctx).
		Preload("Category").
		Where("id = ? AND user_id = ?", id, userID).
		First(&collection).Error; err != nil {
		return nil, err
	}

	return &collection, nil
}

func (r *GormCollectionRepository) GetCollectionByNormalizedName(ctx context.Context, normalizedName string, userID uint) (*models.Collection, error) {
	var collection models.Collection
	if err := r.db.WithContext(ctx).
		Preload("Category").
		Where("normalized_name = ? AND user_id = ?", normalizedName, userID).
		First(&collection).Error; err != nil {
		return nil, err
	}

	return &collection, nil
}

func (r *GormCollectionRepository) ListCollectionsByUserID(ctx context.Context, userID uint) ([]models.Collection, error) {
	var collections []models.Collection
	if err := r.db.WithContext(ctx).
		Model(&models.Collection{}).
		Preload("Category").
		Select("collections.*, COUNT(documents.id) AS document_count").
		Joins("LEFT JOIN documents ON documents.collection_id = collections.id AND documents.deleted_at IS NULL").
		Where("collections.user_id = ?", userID).
		Group("collections.id").
		Order("collections.name ASC").
		Find(&collections).Error; err != nil {
		return nil, err
	}

	return collections, nil
}

func (r *GormCollectionRepository) CountDocuments(ctx context.Context, userID uint, collectionID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Document{}).
		Where("user_id = ? AND collection_id = ?", userID, collectionID).
		Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
