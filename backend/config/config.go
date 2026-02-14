package config

import (
	"os"
	"strings"
)

type Config struct {
	Port                   string
	DatabaseURL            string
	LightsparkClientID     string
	LightsparkClientSecret string
	LightsparkNodeID       string
	LightsparkNodePassword string
	JWTSecret              string
	AdminEmails            []string
	Domain                 string
	UMASigningPrivKeyHex    string
	UMASigningCertChain     string
	UMAEncryptionPrivKeyHex string
	UMAEncryptionCertChain  string
}

func LoadConfig() *Config {
	return &Config{
		Port:                   getEnv("PORT", "8080"),
		DatabaseURL:            getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/tickets_uma?sslmode=disable"),
		LightsparkClientID:     getEnv("LIGHTSPARK_CLIENT_ID", ""),
		LightsparkClientSecret: getEnv("LIGHTSPARK_CLIENT_SECRET", ""),
		LightsparkNodeID:       getEnv("LIGHTSPARK_NODE_ID", ""),
		LightsparkNodePassword: getEnv("LIGHTSPARK_NODE_PASSWORD", ""),
		JWTSecret:              getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		AdminEmails:             strings.Split(getEnv("ADMIN_EMAILS", "admin2@example.com,admin@example.com"), ","),
		Domain:                  getEnv("DOMAIN", "localhost"),
		UMASigningPrivKeyHex:    getEnv("UMA_SIGNING_PRIVKEY", ""),
		UMASigningCertChain:     getEnv("UMA_SIGNING_CERT_CHAIN", ""),
		UMAEncryptionPrivKeyHex: getEnv("UMA_ENCRYPTION_PRIVKEY", ""),
		UMAEncryptionCertChain:  getEnv("UMA_ENCRYPTION_CERT_CHAIN", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
