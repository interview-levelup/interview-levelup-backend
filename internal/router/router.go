package router

import (
	"net/http"

	"github.com/fan/interview-levelup-backend/internal/config"
	"github.com/fan/interview-levelup-backend/internal/handlers"
	"github.com/fan/interview-levelup-backend/internal/middleware"
	"github.com/fan/interview-levelup-backend/internal/services"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func New(cfg *config.Config, authSvc *services.AuthService, authH *handlers.AuthHandler, ivH *handlers.InterviewHandler) *gin.Engine {
	r := gin.Default()

	if len(cfg.CORSOrigins) > 0 {
		corsConfig := cors.DefaultConfig()
		corsConfig.AllowOrigins = cfg.CORSOrigins
		corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
		corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
		corsConfig.ExposeHeaders = []string{"Content-Length", "Content-Type", "X-Accel-Buffering"}
		r.Use(cors.New(corsConfig))
	}

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
			interviews.POST("/stream", ivH.StartStream)
			interviews.GET("", ivH.List)
			interviews.GET("/:id", ivH.Get)
			interviews.POST("/:id/answer", ivH.SubmitAnswer)
			interviews.POST("/:id/answer/stream", ivH.SubmitAnswerStream)
			interviews.POST("/:id/end", ivH.End)
		}

		transcribe := v1.Group("/transcribe")
		transcribe.Use(middleware.JWT(authSvc))
		{
			transcribe.POST("", ivH.Transcribe)
		}
	}

	return r
}
