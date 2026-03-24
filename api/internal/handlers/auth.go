package handlers

import (
	"errors"
	"io"
	"net/http"
	"thing-index/internal/domain"
	"thing-index/internal/dto"
	"thing-index/internal/pkg/i18n"
	"thing-index/internal/pkg/response"
	"thing-index/internal/pkg/validation"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	useCase domain.AuthUseCase
	storage domain.StorageService
}

func NewAuthHandler(useCase domain.AuthUseCase, storage domain.StorageService) *AuthHandler {
	return &AuthHandler{useCase: useCase, storage: storage}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account and return a JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.RegisterRequest  true  "Registration Info"
// @Success      201      {object}  response.Response{data=map[string]string} "Success token"
// @Failure      400      {object}  response.Response "Invalid input"
// @Failure      500      {object}  response.Response "Internal error"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	lang := c.GetString("language")

	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	token, err := h.useCase.Register(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserExists):
			response.Error(c, http.StatusConflict, "error", i18n.T(lang, "auth.user_exists"), nil)
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		}
		return
	}

	response.Success(c, http.StatusCreated, "success", i18n.T(lang, "auth.success_register"), gin.H{"token": token})
}

// Login godoc
// @Summary      Login user
// @Description  Authenticate user with email and password and return a JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.LoginRequest  true  "Login Credentials"
// @Success      200      {object}  response.Response{data=map[string]string} "Success token"
// @Failure      400      {object}  response.Response "Invalid input"
// @Failure      401      {object}  response.Response "Invalid credentials"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	lang := c.GetString("language")

	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	token, err := h.useCase.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			response.Error(c, http.StatusUnauthorized, "error", i18n.T(lang, "auth.invalid_creds"), nil)
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "auth.success_login"), gin.H{"token": token})
}

// Me godoc
// @Summary      Get authenticated user
// @Description  Return the authenticated user profile for the current token
// @Tags         auth
// @Produce      json
// @Success      200 {object} response.Response{data=dto.UserResponse}
// @Failure      401 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /user/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	user, err := h.useCase.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "auth.success_me"), dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}

// UpdateMe godoc
// @Summary      Update authenticated user
// @Description  Update the authenticated user name and optional avatar image
// @Tags         auth
// @Accept       multipart/form-data
// @Produce      json
// @Param        name formData string false "Updated user name"
// @Param        avatar formData file false "Avatar image (jpeg, png, webp)"
// @Success      200 {object} response.Response{data=dto.UserResponse}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /user/me [put]
func (h *AuthHandler) UpdateMe(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	var input domain.UpdateCurrentUserProfileInput

	if name := c.PostForm("name"); name != "" {
		input.Name = &name
	}

	fileHeader, err := c.FormFile("avatar")
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), nil)
		return
	}

	if fileHeader != nil {
		openedFile, openErr := fileHeader.Open()
		if openErr != nil {
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), nil)
			return
		}
		defer openedFile.Close()

		input.Avatar = &domain.UserProfileAvatarInput{
			File:        openedFile,
			Size:        fileHeader.Size,
			ContentType: fileHeader.Header.Get("Content-Type"),
			Filename:    fileHeader.Filename,
		}
	}

	if input.Name == nil && input.Avatar == nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "user.error_update_empty"), nil)
		return
	}

	user, updateErr := h.useCase.UpdateCurrentUserProfile(c.Request.Context(), userID, input)
	if updateErr != nil {
		switch {
		case errors.Is(updateErr, domain.ErrUserNameRequired):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "user.error_name_required"), nil)
		case errors.Is(updateErr, domain.ErrUserAvatarInvalidMime):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "user.error_avatar_invalid_mime"), nil)
		case errors.Is(updateErr, domain.ErrFileTooLarge):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "user.error_avatar_too_large"), nil)
		case errors.Is(updateErr, domain.ErrFileRequired):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "user.error_avatar_missing"), nil)
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "user.error_update_profile"), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "user.success_profile_update"), dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}

// GetMyAvatar godoc
// @Summary      Get authenticated user avatar
// @Description  Stream the authenticated user avatar image if configured
// @Tags         auth
// @Produce      octet-stream
// @Success      200 {file} binary
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /user/me/avatar [get]
func (h *AuthHandler) GetMyAvatar(c *gin.Context) {
	lang := c.GetString("language")
	userID := c.GetUint("user_id")

	user, err := h.useCase.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		return
	}

	if user.AvatarURL == "" {
		response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "user.error_avatar_not_found"), nil)
		return
	}

	reader, err := h.storage.OpenFile(c.Request.Context(), user.AvatarURL)
	if err != nil {
		response.Error(c, http.StatusNotFound, "error", i18n.T(lang, "user.error_avatar_not_found"), nil)
		return
	}
	defer reader.Close()

	payload, err := io.ReadAll(reader)
	if err != nil || len(payload) == 0 {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, http.DetectContentType(payload), payload)
}

// ForgotPassword godoc
// @Summary      Forgot password
// @Description  Send a reset password link to the user's email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.ForgotPasswordRequest  true  "User Email"
// @Success      200      {object}  response.Response "Success message"
// @Failure      400      {object}  response.Response "Invalid input"
// @Failure      500      {object}  response.Response "Internal error"
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	lang := c.GetString("language")

	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	if err := h.useCase.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		return
	}

	response.Success(c, http.StatusOK, "info", i18n.T(lang, "auth.success_forgot_password"), nil)
}

// ResetPassword godoc
// @Summary      Reset password
// @Description  Reset the user's password using a reset token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.ResetPasswordRequest  true  "Reset Info"
// @Success      200      {object}  response.Response "Success message"
// @Failure      400      {object}  response.Response "Invalid/Expired token"
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	lang := c.GetString("language")

	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "error.validation"), validation.FormatValidationErrors(c, err))
		return
	}

	if err := h.useCase.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, domain.ErrResetTokenInvalid):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "auth.invalid_reset_token"), nil)
		case errors.Is(err, domain.ErrResetTokenExpired):
			response.Error(c, http.StatusBadRequest, "error", i18n.T(lang, "auth.expired_reset_token"), nil)
		default:
			response.Error(c, http.StatusInternalServerError, "error", i18n.T(lang, "error.internal"), nil)
		}
		return
	}

	response.Success(c, http.StatusOK, "success", i18n.T(lang, "auth.success_reset_password"), nil)
}
