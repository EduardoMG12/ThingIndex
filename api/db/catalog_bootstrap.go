package db

import (
	"fmt"
	"strings"

	"thing-index/internal/models"

	"gorm.io/gorm"
)

func ensureDocumentCatalog(db *gorm.DB) error {
	if err := dropLegacyIndexIfExists(db, "idx_user_category"); err != nil {
		return err
	}

	if err := backfillCategoryNormalizedNames(db); err != nil {
		return err
	}

	if err := backfillCollectionNormalizedNames(db); err != nil {
		return err
	}

	if err := backfillDocumentCollections(db); err != nil {
		return err
	}

	if err := ensureDocumentCollectionNotNull(db); err != nil {
		return err
	}

	return nil
}

func backfillCategoryNormalizedNames(db *gorm.DB) error {
	var categories []models.DocumentCategory
	if err := db.Unscoped().
		Where("normalized_name IS NULL OR normalized_name = ''").
		Find(&categories).Error; err != nil {
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

func backfillCollectionNormalizedNames(db *gorm.DB) error {
	var collections []models.Collection
	if err := db.Unscoped().
		Where("normalized_name IS NULL OR normalized_name = ''").
		Find(&collections).Error; err != nil {
		return err
	}

	for _, collection := range collections {
		normalizedName := normalizedCollectionName(collection.Name, collection.ID)
		if err := db.Unscoped().Model(&models.Collection{}).
			Where("id = ?", collection.ID).
			Update("normalized_name", normalizedName).Error; err != nil {
			return err
		}
	}

	return nil
}

func backfillDocumentCollections(db *gorm.DB) error {
	var documents []models.Document
	if err := db.Unscoped().Select("id", "user_id", "title", "collection", "category_id", "collection_id").
		Where("collection_id IS NULL").
		Order("id ASC").
		Find(&documents).Error; err != nil {
		return err
	}
	if len(documents) == 0 {
		return nil
	}

	var collections []models.Collection
	if err := db.Unscoped().
		Select("id", "user_id", "name", "normalized_name", "category_id").
		Find(&collections).Error; err != nil {
		return err
	}

	cache := make(map[string]*models.Collection, len(collections))
	for i := range collections {
		normalizedName := collections[i].NormalizedName
		if normalizedName == "" {
			normalizedName = normalizedCollectionName(collections[i].Name, collections[i].ID)
			if err := db.Unscoped().Model(&models.Collection{}).
				Where("id = ?", collections[i].ID).
				Update("normalized_name", normalizedName).Error; err != nil {
				return err
			}
			collections[i].NormalizedName = normalizedName
		}

		cache[collectionCacheKey(collections[i].UserID, normalizedName)] = &collections[i]
	}

	for _, doc := range documents {
		displayName := documentCollectionDisplayName(doc)
		normalizedName := normalizedCollectionName(displayName, doc.ID)
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

func ensureDocumentCollectionNotNull(db *gorm.DB) error {
	isNullable, err := columnIsNullable(db, "documents", "collection_id")
	if err != nil {
		return err
	}
	if !isNullable {
		return nil
	}

	var pendingNulls int64
	if err := db.Unscoped().
		Model(&models.Document{}).
		Where("collection_id IS NULL").
		Count(&pendingNulls).Error; err != nil {
		return err
	}
	if pendingNulls > 0 {
		return fmt.Errorf("cannot enforce documents.collection_id NOT NULL: %d rows still missing collection_id", pendingNulls)
	}

	return setColumnNotNull(db, "documents", "collection_id")
}

func collectionCacheKey(userID uint, normalizedName string) string {
	return fmt.Sprintf("%d:%s", userID, normalizedName)
}

func normalizeCatalogName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizedCollectionName(name string, fallbackID uint) string {
	normalizedName := normalizeCatalogName(name)
	if normalizedName == "" {
		return fmt.Sprintf("legacy-collection-%d", fallbackID)
	}

	return normalizedName
}

func documentCollectionDisplayName(doc models.Document) string {
	displayName := strings.TrimSpace(doc.Collection)
	if displayName != "" {
		return displayName
	}

	baseTitle := strings.TrimSpace(doc.Title)
	if baseTitle == "" {
		baseTitle = "Legacy document"
	}

	return fmt.Sprintf("%s #%d", baseTitle, doc.ID)
}
