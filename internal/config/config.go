package config

import (
	"fmt"
	"os"

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
