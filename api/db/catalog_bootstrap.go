package db

import (
	"fmt"
	"strings"

	"thing-index/internal/models"

	"gorm.io/gorm"
)

func ensureDocumentCatalog(db *gorm.DB) error {
	if err := db.Exec("DROP INDEX IF EXISTS idx_user_category").Error; err != nil {
		return err
	}

	if err := backfillCategoryNormalizedNames(db); err != nil {
		return err
	}

	if err := backfillCollections(db); err != nil {
		return err
	}

	if err := db.Exec("ALTER TABLE documents ALTER COLUMN collection_id SET NOT NULL").Error; err != nil {
		return err
	}

	return nil
}

func backfillCategoryNormalizedNames(db *gorm.DB) error {
	var categories []models.DocumentCategory
	if err := db.Unscoped().Find(&categories).Error; err != nil {
		return err
	}

	for _, category := range categories {
		normalizedName := normalizeCatalogName(category.Name)
		if normalizedName == "" {
			normalizedName = fmt.Sprintf("legacy-category-%d", category.ID)
		}

		if category.NormalizedName == normalizedName {
			continue
		}

		if err := db.Unscoped().Model(&models.DocumentCategory{}).
			Where("id = ?", category.ID).
			Update("normalized_name", normalizedName).Error; err != nil {
			return err
		}
	}

	return nil
}

func backfillCollections(db *gorm.DB) error {
	var collections []models.Collection
	if err := db.Unscoped().Find(&collections).Error; err != nil {
		return err
	}

	cache := make(map[string]*models.Collection, len(collections))
	for i := range collections {
		normalizedName := normalizeCatalogName(collections[i].Name)
		if normalizedName == "" {
			normalizedName = fmt.Sprintf("legacy-collection-%d", collections[i].ID)
		}

		if collections[i].NormalizedName != normalizedName {
			if err := db.Unscoped().Model(&models.Collection{}).
				Where("id = ?", collections[i].ID).
				Updates(map[string]any{
					"normalized_name": normalizedName,
					"name":            collections[i].Name,
				}).Error; err != nil {
				return err
			}
			collections[i].NormalizedName = normalizedName
		}

		cache[collectionCacheKey(collections[i].UserID, normalizedName)] = &collections[i]
	}

	var documents []models.Document
	if err := db.Unscoped().Select("id", "user_id", "title", "collection", "category_id", "collection_id").
		Where("collection_id IS NULL").
		Order("id ASC").
		Find(&documents).Error; err != nil {
		return err
	}

	for _, doc := range documents {
		displayName := strings.TrimSpace(doc.Collection)
		if displayName == "" {
			baseTitle := strings.TrimSpace(doc.Title)
			if baseTitle == "" {
				baseTitle = "Legacy document"
			}
			displayName = fmt.Sprintf("%s #%d", baseTitle, doc.ID)
		}

		normalizedName := normalizeCatalogName(displayName)
		key := collectionCacheKey(doc.UserID, normalizedName)

		collection, ok := cache[key]
		if !ok {
			collection = &models.Collection{
				UserID:         doc.UserID,
				Name:           displayName,
				NormalizedName: normalizedName,
				Metadata:       map[string]any{},
			}
			if doc.CategoryID != nil {
				collection.CategoryID = doc.CategoryID
			}

			if err := db.Unscoped().Create(collection).Error; err != nil {
				return err
			}

			cache[key] = collection
		} else if collection.CategoryID == nil && doc.CategoryID != nil {
			if err := db.Unscoped().Model(&models.Collection{}).
				Where("id = ?", collection.ID).
				Update("category_id", doc.CategoryID).Error; err != nil {
				return err
			}
			collection.CategoryID = doc.CategoryID
		}

		if err := db.Unscoped().Model(&models.Document{}).
			Where("id = ?", doc.ID).
			Updates(map[string]any{
				"collection_id": collection.ID,
				"collection":    collection.Name,
			}).Error; err != nil {
			return err
		}
	}

	return nil
}

func collectionCacheKey(userID uint, normalizedName string) string {
	return fmt.Sprintf("%d:%s", userID, normalizedName)
}

func normalizeCatalogName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
