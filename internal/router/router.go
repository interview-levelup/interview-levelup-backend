package router

import (
	"net/http"

	"github.com/fan/interview-levelup-backend/internal/handlers"
	"github.com/fan/interview-levelup-backend/internal/middleware"
	"github.com/fan/interview-levelup-backend/internal/services"
	"github.com/gin-gonic/gin"
)

func New(authSvc *services.AuthService, authH *handlers.AuthHandler, ivH *handlers.InterviewHandler) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authH.Register)
			auth.POST("/login", authH.Login)
		}

		interviews := v1.Group("/interviews")
		interviews.Use(middleware.JWT(authSvc))
		{
			interviews.POST("", ivH.Start)
			interviews.GET("", ivH.List)
			interviews.GET("/:id", ivH.Get)
			interviews.POST("/:id/answer", ivH.SubmitAnswer)
		}
	}

	return r
}
