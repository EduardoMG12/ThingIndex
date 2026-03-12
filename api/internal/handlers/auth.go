package handlers

import (
	"errors"
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
}

func NewAuthHandler(useCase domain.AuthUseCase) *AuthHandler {
	return &AuthHandler{useCase: useCase}
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
