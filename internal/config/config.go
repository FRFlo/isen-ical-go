// Package config gère la configuration de l'application.
// Il charge les paramètres depuis les variables d'environnement avec des valeurs par défaut.
package config

// Config contient tous les paramètres de configuration de l'application.
// Les valeurs sont chargées depuis les variables d'environnement.
type Config struct {
	AurionBaseURL    string `envconfig:"AURION_BASE_URL"`
	ValkeyURL        string `envconfig:"VALKEY_URL"`
	EncryptionKey    []byte `envconfig:"ENCRYPTION_KEY"`
	Port             int    `envconfig:"PORT"`
	MaxTokensPerUser int    `envconfig:"MAX_TOKENS_PER_USER"`
	SessionTTL       int    `envconfig:"SESSION_TTL"`
	CacheTTL         int    `envconfig:"CACHE_TTL"`
	AdminAPIToken    string `envconfig:"ADMIN_API_TOKEN"`
}
