package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gormChatConversationRepository struct {
	db *gorm.DB
}

func NewGormChatConversationRepository(db *gorm.DB) domain.ChatConversationRepository {
	return &gormChatConversationRepository{db: db}
}

func (r *gormChatConversationRepository) CreateConversation(ctx context.Context, conversation *models.ChatConversation) error {
	return r.db.WithContext(ctx).Create(conversation).Error
}

func (r *gormChatConversationRepository) GetConversationByID(ctx context.Context, userID uint, conversationID uint) (*models.ChatConversation, error) {
	var conversation models.ChatConversation
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", conversationID, userID).
		First(&conversation).Error; err != nil {
		return nil, err
	}

	return &conversation, nil
}

func (r *gormChatConversationRepository) ListConversationsByCollection(ctx context.Context, userID uint, collectionID uint) ([]models.ChatConversation, error) {
	var conversations []models.ChatConversation
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND collection_id = ?", userID, collectionID).
		Order("COALESCE(last_message_at, created_at) DESC").
		Order("id DESC").
		Find(&conversations).Error; err != nil {
		return nil, err
	}

	return conversations, nil
}

func (r *gormChatConversationRepository) ListMessagesByConversation(ctx context.Context, userID uint, conversationID uint) ([]models.ChatConversationEntry, error) {
	if _, err := r.GetConversationByID(ctx, userID, conversationID); err != nil {
		return nil, err
	}

	var messages []models.ChatConversationEntry
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Order("id ASC").
		Find(&messages).Error; err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *gormChatConversationRepository) UpdateConversationTitle(ctx context.Context, userID uint, conversationID uint, title string) (*models.ChatConversation, error) {
	normalizedTitle := normalizeWhitespace(title)

	var conversation models.ChatConversation
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", conversationID, userID).
		First(&conversation).Error; err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).
		Model(&conversation).
		Updates(map[string]any{"title": normalizedTitle}).
		Error; err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).
		Where("id = ?", conversation.ID).
		First(&conversation).Error; err != nil {
		return nil, err
	}

	return &conversation, nil
}

func (r *gormChatConversationRepository) DeleteConversation(ctx context.Context, userID uint, conversationID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id = ? AND user_id = ?", conversationID, userID).
			Delete(&models.ChatConversation{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		if err := tx.Where("conversation_id = ?", conversationID).
			Delete(&models.ChatConversationEntry{}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *gormChatConversationRepository) AddMessage(ctx context.Context, userID uint, conversationID uint, message *models.ChatConversationEntry) error {
	now := time.Now().UTC()

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var conversation models.ChatConversation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", conversationID, userID).
			First(&conversation).Error; err != nil {
			return err
		}

		message.ConversationID = conversationID
		if err := tx.Create(message).Error; err != nil {
			return err
		}

		title := conversation.Title
		if title == "" && message.Role == "user" {
			title = deriveConversationTitle(message.Content)
		}

		updates := map[string]any{
			"last_message_at": now,
		}
		if title != "" {
			updates["title"] = title
		}

		if err := tx.Model(&models.ChatConversation{}).
			Where("id = ?", conversationID).
			Updates(updates).Error; err != nil {
			return err
		}

		return nil
	})
}

func deriveConversationTitle(content string) string {
	trimmed := normalizeWhitespace(content)
	// Corta por rune para nao quebrar UTF-8 no meio.
	if runes := []rune(trimmed); len(runes) > 80 {
		return strings.TrimSpace(string(runes[:80])) + "..."
	}
	if trimmed == "" {
		return fmt.Sprintf("Chat %s", time.Now().UTC().Format("2006-01-02 15:04"))
	}
	return trimmed
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
