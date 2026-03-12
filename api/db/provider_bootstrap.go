package db

import "gorm.io/gorm"

func ensureProviderSchema(db *gorm.DB) error {
	if err := db.Exec("DROP INDEX IF EXISTS idx_user_provider").Error; err != nil {
		return err
	}

	return db.Exec(
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_user_provider ON user_provider_configs (user_id, provider_type)",
	).Error
}
