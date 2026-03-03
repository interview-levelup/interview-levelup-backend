package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port      string
	JWTSecret string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSL      string

	AgentBaseURL string

	WhisperAPIKey  string
	WhisperBaseURL string // optional: override for self-hosted Whisper

	CORSOrigins []string // allowed origins for CORS; defaults to localhost dev ports
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	cfg := &Config{
		Port:         getEnv("PORT", "8080"),
		JWTSecret:    mustGetEnv("JWT_SECRET"),
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       mustGetEnv("DB_USER"),
		DBPassword:   getEnv("DB_PASSWORD", ""),
		DBName:       mustGetEnv("DB_NAME"),
		DBSSL:        getEnv("DB_SSLMODE", "disable"),
		AgentBaseURL: getEnv("AGENT_BASE_URL", "http://localhost:8000"),

		WhisperAPIKey:  getEnv("WHISPER_API_KEY", getEnv("LLM_API_KEY", "")),
		WhisperBaseURL: getEnv("WHISPER_BASE_URL", ""),
	}

	// CORS_ORIGINS — comma-separated list of allowed origins.
	// e.g. "https://example.up.railway.app,http://localhost:5173"
	if raw := getEnv("CORS_ORIGINS", ""); raw != "" {
		for _, o := range strings.Split(raw, ",") {
			if o = strings.TrimSpace(o); o != "" {
				cfg.CORSOrigins = append(cfg.CORSOrigins, o)
			}
		}
	}
	return cfg, nil
}

func (c *Config) DSN() string {
	if c.DBPassword != "" {
		// Include password in DSN only if it's set to avoid issues with some PostgreSQL setups that don't require a password
		return fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSL,
		)
	} else {
		// If password is empty, omit it from the DSN to avoid issues with some PostgreSQL setups
		return fmt.Sprintf(
			"host=%s port=%s user=%s dbname=%s sslmode=%s",
			c.DBHost, c.DBPort, c.DBUser, c.DBName, c.DBSSL,
		)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required env var %q is not set", key))
	}
	return v
}

func maskSecret(s string) string {
	if len(s) == 0 {
		return "(not set)"
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:4] + "****"
}

// LogSummary prints the effective configuration at startup.
// Sensitive values are masked so it is safe to emit to any log sink.
func (c *Config) LogSummary() {
	log.Println("=== config ===")
	log.Printf("  PORT            = %s", c.Port)
	log.Printf("  DB_HOST         = %s", c.DBHost)
	log.Printf("  DB_PORT         = %s", c.DBPort)
	log.Printf("  DB_NAME         = %s", c.DBName)
	log.Printf("  DB_USER         = %s", c.DBUser)
	log.Printf("  DB_PASSWORD     = %s", maskSecret(c.DBPassword))
	log.Printf("  DB_SSLMODE      = %s", c.DBSSL)
	log.Printf("  JWT_SECRET      = %s", maskSecret(c.JWTSecret))
	log.Printf("  AGENT_BASE_URL  = %s", c.AgentBaseURL)
	log.Printf("  WHISPER_API_KEY = %s", maskSecret(c.WhisperAPIKey))
	log.Printf("  WHISPER_BASE_URL= %s", c.WhisperBaseURL)
	log.Printf("  CORS_ORIGINS    = %v", c.CORSOrigins)
	log.Println("===============")
}
