package repository

import (
	"context"
	"strings"
	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
)

type gormAuthRepository struct {
	db *gorm.DB
}

func NewGormAuthRepository(db *gorm.DB) domain.AuthRepository {
	return &gormAuthRepository{db: db}
}

func (r *gormAuthRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	if err := r.db.WithContext(ctx).Where("email = ?", normalizedEmail).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *gormAuthRepository) CreateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *gormAuthRepository) UpdateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *gormAuthRepository) GetUserByResetTokenHash(ctx context.Context, tokenHash string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("reset_token = ?", tokenHash).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
