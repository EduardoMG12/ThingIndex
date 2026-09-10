package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

// GormIndexingRepository: preferencias de enriquecimento por usuario e cache
// de resumos por hash. Usado pelo indexador (le) e pela API (le e grava).
type GormIndexingRepository struct {
	db *gorm.DB
}

func NewGormIndexingRepository(db *gorm.DB) *GormIndexingRepository {
	return &GormIndexingRepository{db: db}
}

func (r *GormIndexingRepository) GetIndexingPreferences(ctx context.Context, userID uint) (domain.IndexingPreferences, error) {
	var row models.UserIndexingPreferences
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.DefaultIndexingPreferences(), nil
	}
	if err != nil {
		return domain.DefaultIndexingPreferences(), err
	}
	return domain.IndexingPreferences{DocumentSynopsis: row.DocumentSynopsis, PageSummaries: row.PageSummaries, SectionSummaries: row.SectionSummaries}, nil
}

func (r *GormIndexingRepository) SaveIndexingPreferences(ctx context.Context, userID uint, prefs domain.IndexingPreferences) error {
	row := models.UserIndexingPreferences{UserID: userID, DocumentSynopsis: prefs.DocumentSynopsis, PageSummaries: prefs.PageSummaries, SectionSummaries: prefs.SectionSummaries}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"document_synopsis", "page_summaries", "section_summaries", "updated_at"}),
	}).Create(&row).Error
}

func (r *GormIndexingRepository) LookupEnrichment(ctx context.Context, kind, model string, hashes []string) (map[string]domain.EnrichmentCacheEntry, error) {
	out := make(map[string]domain.EnrichmentCacheEntry, len(hashes))
	if len(hashes) == 0 {
		return out, nil
	}
	var rows []models.EnrichmentCacheEntry
	if err := r.db.WithContext(ctx).Where("kind = ? AND model = ? AND content_hash IN ?", kind, model, hashes).Find(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ContentHash] = domain.EnrichmentCacheEntry{Kind: row.Kind, Model: row.Model, ContentHash: row.ContentHash, Summary: row.Summary, Topics: row.Topics}
	}
	return out, nil
}

func (r *GormIndexingRepository) StoreEnrichment(ctx context.Context, entries []domain.EnrichmentCacheEntry) error {
	if len(entries) == 0 {
		return nil
	}
	rows := make([]models.EnrichmentCacheEntry, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, models.EnrichmentCacheEntry{Kind: e.Kind, Model: e.Model, ContentHash: e.ContentHash, Summary: e.Summary, Topics: e.Topics})
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(rows, 100).Error
}
