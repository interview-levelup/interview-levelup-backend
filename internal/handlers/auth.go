package handlers

import (
	"net/http"

	"github.com/fan/interview-levelup-backend/internal/apierror"
	"github.com/fan/interview-levelup-backend/internal/services"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authSvc *services.AuthService
}

func NewAuthHandler(svc *services.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: svc}
}

type registerRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password"     binding:"required,min=6"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := h.authSvc.Register(req.Email, req.Password)
	if err != nil {
		apierror.Respond(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": user})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, user, err := h.authSvc.Login(req.Email, req.Password)
	if err != nil {
		apierror.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := c.GetString("userID")
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.authSvc.ChangePassword(userID, req.CurrentPassword, req.NewPassword); err != nil {
		apierror.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}
