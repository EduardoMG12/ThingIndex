package repository

import (
	"strings"

	"thing-index/internal/domain"

	"gorm.io/gorm"
)

type auditBucketRow struct {
	Key   string
	Count int64
}

func clampAuditLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func applyCommonAuditFilters(query *gorm.DB, filter domain.AuditPageFilter, includeCursor bool) (*gorm.DB, error) {
	if query == nil {
		return nil, nil
	}

	if status := strings.ToUpper(strings.TrimSpace(filter.Status)); status != "" {
		query = query.Where("status = ?", status)
	}
	if actorType := strings.ToUpper(strings.TrimSpace(filter.ActorType)); actorType != "" {
		query = query.Where("actor_type = ?", actorType)
	}
	if filter.ActorID != nil {
		query = query.Where("actor_id = ?", *filter.ActorID)
	}
	if correlationID := strings.TrimSpace(filter.CorrelationID); correlationID != "" {
		query = query.Where("correlation_id = ?", correlationID)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", filter.CreatedFrom.UTC())
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at <= ?", filter.CreatedTo.UTC())
	}
	if includeCursor && strings.TrimSpace(filter.Cursor) != "" {
		cursor, err := domain.DecodeAuditCursor(filter.Cursor)
		if err != nil {
			return nil, err
		}
		query = query.Where("(created_at < ?) OR (created_at = ? AND id < ?)", cursor.CreatedAt.UTC(), cursor.CreatedAt.UTC(), cursor.ID)
	}

	return query, nil
}

func escapeLikePattern(value string) string {
	replacer := strings.NewReplacer(
		`\\`, `\\\\`,
		`%`, `\%`,
		`_`, `\_`,
	)

	return replacer.Replace(strings.ToLower(strings.TrimSpace(value)))
}

func loadAuditBuckets(query *gorm.DB, column string) ([]domain.AuditCountBucket, error) {
	if query == nil {
		return nil, nil
	}

	var rows []auditBucketRow
	if err := query.
		Session(&gorm.Session{}).
		Select(column + " AS key, COUNT(*) AS count").
		Where(column + " <> ''").
		Group(column).
		Order("count DESC, key ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]domain.AuditCountBucket, 0, len(rows))
	for i := range rows {
		items = append(items, domain.AuditCountBucket{
			Key:   rows[i].Key,
			Count: rows[i].Count,
		})
	}

	return items, nil
}
