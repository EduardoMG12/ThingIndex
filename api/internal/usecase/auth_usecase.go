package usecase

import (
	"context"
	"errors"
	"fmt"
	"thing-index/internal/config"
	"thing-index/internal/domain"
	"thing-index/internal/models"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type authUseCase struct {
	repo   domain.AuthRepository
	mailer domain.Mailer
	cfg    *config.Config
}

func NewAuthUseCase(repo domain.AuthRepository, mailer domain.Mailer, cfg *config.Config) domain.AuthUseCase {
	return &authUseCase{
		repo:   repo,
		mailer: mailer,
		cfg:    cfg,
	}
}

func (u *authUseCase) Register(ctx context.Context, name, email, password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	user := &models.User{
		Name:     name,
		Email:    email,
		Password: string(hashedPassword),
	}

	if err := u.repo.CreateUser(ctx, user); err != nil {
		return "", err
	}

	// Generate JWT after registration
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(u.cfg.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (u *authUseCase) Login(ctx context.Context, email, password string) (string, error) {
	user, err := u.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(u.cfg.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (u *authUseCase) ForgotPassword(ctx context.Context, email string) error {
	user, err := u.repo.GetUserByEmail(ctx, email)
	if err != nil {
		// We don't want to reveal if a user exists or not for security
		return nil
	}

	// Simple token generation for demo purposes
	token := fmt.Sprintf("%d", time.Now().UnixNano())
	expiresAt := time.Now().Add(time.Hour)

	user.ResetToken = token
	user.ResetTokenExpiresAt = &expiresAt

	if err := u.repo.UpdateUser(ctx, user); err != nil {
		return err
	}

	return u.mailer.SendResetPasswordEmail(user.Email, token)
}

func (u *authUseCase) ResetPassword(ctx context.Context, token, newPassword string) error {
	user, err := u.repo.GetUserByResetToken(ctx, token)
	if err != nil {
		return errors.New("invalid or expired token")
	}

	if user.ResetTokenExpiresAt == nil || user.ResetTokenExpiresAt.Before(time.Now()) {
		return errors.New("token expired")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	user.ResetToken = ""
	user.ResetTokenExpiresAt = nil

	return u.repo.UpdateUser(ctx, user)
}
