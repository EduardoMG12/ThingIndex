package db

import "gorm.io/gorm"

func ensureProviderSchema(db *gorm.DB) error {
	indexDefinition, err := indexDefinition(db, "idx_user_provider")
	if err != nil {
		return err
	}
	if indexDefinitionContains(indexDefinition, "create unique index", "user_provider_configs", "(user_id, provider_type)") {
		return nil
	}

	if err := dropLegacyIndexIfExists(db, "idx_user_provider"); err != nil {
		return err
	}

	return db.Exec(
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_user_provider ON user_provider_configs (user_id, provider_type)",
	).Error
}
