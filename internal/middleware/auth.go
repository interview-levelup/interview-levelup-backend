package middleware

import (
	"net/http"
	"strings"

	"github.com/fan/interview-levelup-backend/internal/services"
	"github.com/gin-gonic/gin"
)

const UserIDKey = "userID"

func JWT(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or malformed token"})
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		userID, err := authSvc.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(UserIDKey, userID)
		c.Next()
	}
}
