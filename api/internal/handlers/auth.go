package handlers

import (
	"net/http"
	"thing-index/internal/domain"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	useCase domain.AuthUseCase
}

func NewAuthHandler(useCase domain.AuthUseCase) *AuthHandler {
	return &AuthHandler{useCase: useCase}
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account and return a JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      RegisterRequest  true  "Registration Info"
// @Success      201      {object}  map[string]string "Success token"
// @Failure      400      {object}  map[string]string "Invalid input"
// @Failure      500      {object}  map[string]string "Internal error"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.useCase.Register(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": token})
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login godoc
// @Summary      Login user
// @Description  Authenticate user with email and password and return a JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "Login Credentials"
// @Success      200      {object}  map[string]string "Success token"
// @Failure      400      {object}  map[string]string "Invalid input"
// @Failure      401      {object}  map[string]string "Invalid credentials"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.useCase.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPassword godoc
// @Summary      Forgot password
// @Description  Send a reset password link to the user's email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      ForgotPasswordRequest  true  "User Email"
// @Success      200      {object}  map[string]string "Success message"
// @Failure      400      {object}  map[string]string "Invalid input"
// @Failure      500      {object}  map[string]string "Internal error"
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.useCase.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process forgot password request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "if the email exists, a reset link will be sent"})
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ResetPassword godoc
// @Summary      Reset password
// @Description  Reset the user's password using a reset token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      ResetPasswordRequest  true  "Reset Info"
// @Success      200      {object}  map[string]string "Success message"
// @Failure      400      {object}  map[string]string "Invalid/Expired token"
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.useCase.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password reset successfully"})
}
