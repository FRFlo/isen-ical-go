package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	AurionBaseURL    string `envconfig:"AURION_BASE_URL"`
	ValkeyURL        string `envconfig:"VALKEY_URL"`
	EncryptionKey    []byte `envconfig:"ENCRYPTION_KEY"`
	Port             int    `envconfig:"PORT"`
	MaxTokensPerUser int    `envconfig:"MAX_TOKENS_PER_USER"`
	SessionTTL       int    `envconfig:"SESSION_TTL"`
	CacheTTL         int    `envconfig:"CACHE_TTL"`
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	cfg := &Config{
		AurionBaseURL:    getEnv("AURION_BASE_URL", "https://aurion.junia.com"),
		ValkeyURL:        getEnv("VALKEY_URL", "valkey://localhost:6379"),
		Port:             getEnvInt("PORT", 8080),
		MaxTokensPerUser: getEnvInt("MAX_TOKENS_PER_USER", 3),
		SessionTTL:       getEnvInt("SESSION_TTL", 3600),
		CacheTTL:         getEnvInt("CACHE_TTL", 3600),
	}

	// Load encryption key (32-byte hex for AES-256)
	encKeyHex := getEnv("ENCRYPTION_KEY", "")
	if encKeyHex != "" {
		encKey, err := hex.DecodeString(encKeyHex)
		if err != nil {
			return nil, fmt.Errorf("ENCRYPTION_KEY must be valid hex: %w", err)
		}
		if len(encKey) != 32 {
			return nil, fmt.Errorf("ENCRYPTION_KEY must be 32 bytes (64 hex chars), got %d bytes", len(encKey))
		}
		cfg.EncryptionKey = encKey
	}

	return cfg, nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt retrieves an environment variable as int or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
