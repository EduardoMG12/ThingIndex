package domain

import (
	"context"
	"thing-index/internal/models"
)

type AuthRepository interface {
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
	UpdateUser(ctx context.Context, user *models.User) error
	GetUserByResetTokenHash(ctx context.Context, tokenHash string) (*models.User, error)
}

type Mailer interface {
	SendResetPasswordEmail(to string, token string) error
}

type AuthUseCase interface {
	Register(ctx context.Context, name, email, password string) (string, error)
	Login(ctx context.Context, email, password string) (string, error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
}
