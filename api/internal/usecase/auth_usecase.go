package usecase

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"thing-index/internal/config"
	"thing-index/internal/domain"
	"thing-index/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type authClaims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

type authUseCase struct {
	repo    domain.AuthRepository
	mailer  domain.Mailer
	storage domain.StorageService
	cfg     *config.Config
}

const maxAvatarSizeBytes int64 = 5 * 1024 * 1024

var allowedAvatarMimeTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
}

func NewAuthUseCase(repo domain.AuthRepository, mailer domain.Mailer, storage domain.StorageService, cfg *config.Config) domain.AuthUseCase {
	return &authUseCase{
		repo:    repo,
		mailer:  mailer,
		storage: storage,
		cfg:     cfg,
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

func (u *authUseCase) GetCurrentUser(ctx context.Context, userID uint) (*models.User, error) {
	return u.repo.GetUserByID(ctx, userID)
}

func (u *authUseCase) UpdateCurrentUserProfile(ctx context.Context, userID uint, input domain.UpdateCurrentUserProfileInput) (*models.User, error) {
	user, err := u.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	currentAvatarURL := strings.TrimSpace(user.AvatarURL)
	newAvatarURL := currentAvatarURL
	uploadedAvatarURL := ""

	if input.Name != nil {
		trimmedName := strings.TrimSpace(*input.Name)
		if trimmedName == "" {
			return nil, domain.ErrUserNameRequired
		}
		user.Name = trimmedName
	}

	if input.Avatar != nil {
		if input.Avatar.Size <= 0 {
			return nil, domain.ErrFileRequired
		}
		if input.Avatar.Size > maxAvatarSizeBytes {
			return nil, domain.ErrFileTooLarge
		}

		validatedReader, resolvedContentType, err := validateAvatarUpload(input.Avatar.File, input.Avatar.ContentType)
		if err != nil {
			return nil, err
		}

		objectName := buildAvatarObjectName(userID, resolvedContentType)
		uploadedAvatarURL, err = u.storage.UploadFile(ctx, validatedReader, input.Avatar.Size, resolvedContentType, objectName)
		if err != nil {
			return nil, err
		}
		newAvatarURL = uploadedAvatarURL
		user.AvatarURL = newAvatarURL
	}

	if err := u.repo.UpdateUser(ctx, user); err != nil {
		if uploadedAvatarURL != "" {
			_ = u.storage.DeleteFile(ctx, uploadedAvatarURL)
		}
		return nil, err
	}

	if uploadedAvatarURL != "" && currentAvatarURL != "" && currentAvatarURL != uploadedAvatarURL {
		_ = u.storage.DeleteFile(ctx, currentAvatarURL)
	}

	return user, nil
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

func validateAvatarUpload(file io.Reader, declaredContentType string) (io.Reader, string, error) {
	head := make([]byte, 512)
	readBytes, readErr := io.ReadFull(file, head)
	if readErr != nil && !errors.Is(readErr, io.ErrUnexpectedEOF) && !errors.Is(readErr, io.EOF) {
		return nil, "", fmt.Errorf("%w: %v", domain.ErrUploadFailed, readErr)
	}
	if readBytes == 0 {
		return nil, "", domain.ErrFileRequired
	}

	sniffedContentType := http.DetectContentType(head[:readBytes])
	resolvedContentType := normalizeMediaType(declaredContentType)
	if resolvedContentType == "" {
		resolvedContentType = sniffedContentType
	}
	if _, ok := allowedAvatarMimeTypes[resolvedContentType]; !ok {
		if _, sniffedOk := allowedAvatarMimeTypes[sniffedContentType]; !sniffedOk {
			return nil, "", domain.ErrUserAvatarInvalidMime
		}
		resolvedContentType = sniffedContentType
	}

	return io.MultiReader(bytes.NewReader(head[:readBytes]), file), resolvedContentType, nil
}

func buildAvatarObjectName(userID uint, contentType string) string {
	return fmt.Sprintf("avatars/%d/%d_%s%s", userID, time.Now().Unix(), uuid.NewString()[:8], avatarExtension(contentType))
}

func avatarExtension(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}
