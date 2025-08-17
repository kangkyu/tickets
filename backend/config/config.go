package config

import (
	"os"
	"strings"
)

type Config struct {
	Port               string
	DatabaseURL        string
	LightsparkAPIToken string
	LightsparkEndpoint string
	LightsparkNodeID   string
	JWTSecret          string
	AdminEmails        []string
}

func LoadConfig() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/tickets_uma?sslmode=disable"),
		LightsparkAPIToken: getEnv("LIGHTSPARK_API_TOKEN", ""),
		LightsparkEndpoint: getEnv("LIGHTSPARK_ENDPOINT", "api.lightspark.com"),
		LightsparkNodeID:   getEnv("LIGHTSPARK_NODE_ID", ""),
		JWTSecret:          getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		AdminEmails:        strings.Split(getEnv("ADMIN_EMAILS", "admin@example.com"), ","),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
