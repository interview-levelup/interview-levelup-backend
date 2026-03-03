package main

import (
	"log"

	"github.com/fan/interview-levelup-backend/internal/config"
	"github.com/fan/interview-levelup-backend/internal/database"
	"github.com/fan/interview-levelup-backend/internal/handlers"
	"github.com/fan/interview-levelup-backend/internal/repository"
	"github.com/fan/interview-levelup-backend/internal/router"
	"github.com/fan/interview-levelup-backend/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()
	log.Println("database connected")

	userRepo := repository.NewUserRepository(db)
	ivRepo := repository.NewInterviewRepository(db)

	authSvc := services.NewAuthService(userRepo, cfg.JWTSecret)
	agentClient := services.NewAgentClient(cfg.AgentBaseURL)
	whisperClient := services.NewWhisperClient(cfg.WhisperAPIKey, cfg.WhisperBaseURL)
	ivSvc := services.NewInterviewService(ivRepo, agentClient, whisperClient)

	authH := handlers.NewAuthHandler(authSvc)
	ivH := handlers.NewInterviewHandler(ivSvc)

	r := router.New(cfg, authSvc, authH, ivH)

	addr := ":" + cfg.Port
	log.Printf("server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
