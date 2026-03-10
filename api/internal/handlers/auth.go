package handlers

import (
	"net/http"
	"thing-index/internal/domain"
	"thing-index/internal/dto"
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
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Erro de validação", validation.FormatValidationErrors(err))
		return
	}

	token, err := h.useCase.Register(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Erro interno", []string{err.Error()})
		return
	}

	response.Success(c, http.StatusCreated, "Usuário registrado com sucesso", gin.H{"token": token})
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
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Erro de validação", validation.FormatValidationErrors(err))
		return
	}

	token, err := h.useCase.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Falha na autenticação", []string{err.Error()})
		return
	}

	response.Success(c, http.StatusOK, "Login realizado com sucesso", gin.H{"token": token})
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
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Erro de validação", validation.FormatValidationErrors(err))
		return
	}

	if err := h.useCase.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		response.Error(c, http.StatusInternalServerError, "Erro ao processar recuperação de senha", []string{err.Error()})
		return
	}

	response.Success(c, http.StatusOK, "Se o e-mail existir, um link de recuperação será enviado", nil)
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
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Erro de validação", validation.FormatValidationErrors(err))
		return
	}

	if err := h.useCase.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		response.Error(c, http.StatusBadRequest, "Erro ao redefinir senha", []string{err.Error()})
		return
	}

	response.Success(c, http.StatusOK, "Senha redefinida com sucesso", nil)
}
