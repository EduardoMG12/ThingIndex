package db

import (
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func dropLegacyIndexIfExists(db *gorm.DB, indexName string) error {
	exists, err := indexExists(db, indexName)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	return db.Exec(fmt.Sprintf("DROP INDEX %s", quoteIdentifier(indexName))).Error
}

func indexExists(db *gorm.DB, indexName string) (bool, error) {
	indexDefinition, err := indexDefinition(db, indexName)
	if err != nil {
		return false, err
	}

	return indexDefinition != "", nil
}

func indexDefinition(db *gorm.DB, indexName string) (string, error) {
	var definition sql.NullString
	if err := db.Raw(
		`SELECT indexdef
		FROM pg_indexes
		WHERE schemaname = current_schema()
		  AND indexname = ?
		LIMIT 1`,
		indexName,
	).Scan(&definition).Error; err != nil {
		return "", err
	}

	if !definition.Valid {
		return "", nil
	}

	return definition.String, nil
}

func indexDefinitionContains(indexDefinition string, fragments ...string) bool {
	normalized := strings.ToLower(strings.Join(strings.Fields(indexDefinition), " "))
	for _, fragment := range fragments {
		if !strings.Contains(normalized, strings.ToLower(fragment)) {
			return false
		}
	}

	return true
}

func columnIsNullable(db *gorm.DB, tableName, columnName string) (bool, error) {
	var isNullable sql.NullString
	if err := db.Raw(
		`SELECT is_nullable
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = ?
		  AND column_name = ?
		LIMIT 1`,
		tableName,
		columnName,
	).Scan(&isNullable).Error; err != nil {
		return false, err
	}
	if !isNullable.Valid {
		return false, fmt.Errorf("column %s.%s not found", tableName, columnName)
	}

	return strings.EqualFold(isNullable.String, "YES"), nil
}

func setColumnNotNull(db *gorm.DB, tableName, columnName string) error {
	return db.Exec(
		fmt.Sprintf(
			"ALTER TABLE %s ALTER COLUMN %s SET NOT NULL",
			quoteIdentifier(tableName),
			quoteIdentifier(columnName),
		),
	).Error
}

func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
