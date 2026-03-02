package router

import (
	"net/http"
	"time"

	"github.com/fan/interview-levelup-backend/internal/handlers"
	"github.com/fan/interview-levelup-backend/internal/middleware"
	"github.com/fan/interview-levelup-backend/internal/services"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func New(authSvc *services.AuthService, authH *handlers.AuthHandler, ivH *handlers.InterviewHandler) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

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

		// Protected auth routes (require JWT)
		authProtected := v1.Group("/auth")
		authProtected.Use(middleware.JWT(authSvc))
		{
			authProtected.PUT("/password", authH.ChangePassword)
		}

		interviews := v1.Group("/interviews")
		interviews.Use(middleware.JWT(authSvc))
		{
			interviews.POST("", ivH.Start)
			interviews.GET("", ivH.List)
			interviews.GET("/:id", ivH.Get)
			interviews.POST("/:id/answer", ivH.SubmitAnswer)
			interviews.POST("/:id/end", ivH.End)
		}
	}

	return r
}
