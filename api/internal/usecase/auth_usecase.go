package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"thing-index/internal/config"
	"thing-index/internal/domain"
	"thing-index/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type authClaims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

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
		Name:     strings.TrimSpace(name),
		Email:    normalizeEmail(email),
		Password: string(hashedPassword),
	}

	if err := u.repo.CreateUser(ctx, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return "", domain.ErrUserExists
		}
		return "", err
	}

	return u.issueToken(user.ID)
}

func (u *authUseCase) Login(ctx context.Context, email, password string) (string, error) {
	user, err := u.repo.GetUserByEmail(ctx, normalizeEmail(email))
	if err != nil {
		return "", domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", domain.ErrInvalidCredentials
	}

	return u.issueToken(user.ID)
}

func (u *authUseCase) ForgotPassword(ctx context.Context, email string) error {
	user, err := u.repo.GetUserByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	rawToken, tokenHash, err := generateResetToken()
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(time.Hour)
	user.ResetToken = tokenHash
	user.ResetTokenExpiresAt = &expiresAt

	if err := u.repo.UpdateUser(ctx, user); err != nil {
		return err
	}

	return u.mailer.SendResetPasswordEmail(user.Email, rawToken)
}

func (u *authUseCase) ResetPassword(ctx context.Context, token, newPassword string) error {
	user, err := u.repo.GetUserByResetTokenHash(ctx, hashResetToken(token))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrResetTokenInvalid
		}
		return err
	}

	if user.ResetTokenExpiresAt == nil || user.ResetTokenExpiresAt.Before(time.Now()) {
		return domain.ErrResetTokenExpired
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	user.ResetToken = ""
	user.ResetTokenExpiresAt = nil
	invalidAfter := time.Now().UTC()
	user.SessionInvalidAfter = &invalidAfter

	return u.repo.UpdateUser(ctx, user)
}

func (u *authUseCase) issueToken(userID uint) (string, error) {
	now := time.Now()
	claims := authClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(u.cfg.JWTSecret))
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func generateResetToken() (string, string, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", err
	}

	rawToken := base64.RawURLEncoding.EncodeToString(randomBytes)
	return rawToken, hashResetToken(rawToken), nil
}

func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}
