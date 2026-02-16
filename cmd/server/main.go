// Package main est le point d'entrée de l'application ISEN iCal.
// Il configure et démarre le serveur HTTP avec toutes ses dépendances.
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

	"github.com/FRFlo/isen-ical-go/internal/config"
	"github.com/FRFlo/isen-ical-go/internal/handlers"
	"github.com/FRFlo/isen-ical-go/internal/middleware"
	"github.com/FRFlo/isen-ical-go/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

// main est le point d'entrée principal de l'application.
// Il configure l'interface en ligne de commande avec toutes les options disponibles
// et démarre le serveur HTTP avec la configuration appropriée.
func main() {
	cmd := &cli.Command{
		Name:  "isen-ical",
		Usage: "ISEN Calendar Generator - Convert Aurion planning to iCal",
		// Configuration des flags CLI - chaque flag peut être défini via
		// les arguments de ligne de commande ou les variables d'environnement
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "Server port",
				Value:   "8080",
				Sources: cli.EnvVars("PORT"),
			},
			&cli.StringFlag{
				Name:    "valkey-url",
				Aliases: []string{"v"},
				Usage:   "Valkey connection URL",
				Value:   "valkey://localhost:6379",
				Sources: cli.EnvVars("VALKEY_URL"),
			},
			&cli.StringFlag{
				Name:    "aurion-url",
				Usage:   "Aurion base URL",
				Value:   "https://aurion.junia.com",
				Sources: cli.EnvVars("AURION_BASE_URL"),
			},
			&cli.StringFlag{
				Name:    "encryption-key",
				Usage:   "AES-256 encryption key (hex)",
				Sources: cli.EnvVars("ENCRYPTION_KEY"),
			},
			&cli.IntFlag{
				Name:    "max-tokens",
				Usage:   "Maximum tokens per user",
				Value:   3,
				Sources: cli.EnvVars("MAX_TOKENS_PER_USER"),
			},
			&cli.IntFlag{
				Name:    "session-ttl",
				Usage:   "Session cache TTL in seconds",
				Value:   3600,
				Sources: cli.EnvVars("SESSION_TTL"),
			},
			&cli.IntFlag{
				Name:    "cache-ttl",
				Usage:   "Events cache TTL in seconds",
				Value:   3600,
				Sources: cli.EnvVars("CACHE_TTL"),
			},
			&cli.StringFlag{
				Name:    "gin-mode",
				Usage:   "Gin framework mode (release or debug)",
				Value:   "debug",
				Sources: cli.EnvVars("GIN_MODE"),
			},
		},
		// Action principale : initialise et démarre le serveur
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// Initialisation du logger structuré avec zerolog
			logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

			// Récupération et validation du port
			port, err := strconv.Atoi(cmd.String("port"))
			if err != nil {
				return fmt.Errorf("port invalide: %w", err)
			}

			// Construction de la configuration à partir des flags
			cfg := &config.Config{
				Port:             port,
				ValkeyURL:        cmd.String("valkey-url"),
				AurionBaseURL:    cmd.String("aurion-url"),
				MaxTokensPerUser: cmd.Int("max-tokens"),
				SessionTTL:       cmd.Int("session-ttl"),
				CacheTTL:         cmd.Int("cache-ttl"),
			}

			// Traitement de la clé de chiffrement (optionnelle mais recommandée)
			encKeyHex := cmd.String("encryption-key")
			if encKeyHex != "" {
				encKey, err := hex.DecodeString(encKeyHex)
				if err != nil {
					return fmt.Errorf("encryption-key doit être en hexadécimal valide: %w", err)
				}
				if len(encKey) != 32 {
					return fmt.Errorf("encryption-key doit faire 32 octets (64 caractères hex), reçu %d octets", len(encKey))
				}
				cfg.EncryptionKey = encKey
			}

			// Configuration du mode Gin (debug ou release)
			if cmd.String("gin-mode") == "release" {
				gin.SetMode(gin.ReleaseMode)
			}

			// Connexion à Valkey (Redis) pour le stockage des sessions et du cache
			valkeyClient, err := storage.NewValkeyClient(cfg.ValkeyURL)
			if err != nil {
				logger.Fatal().Err(err).Msg("Échec de connexion à Valkey")
			}
			defer valkeyClient.Close()

			// Configuration du routeur Gin avec les middlewares
			router := gin.New()
			middleware.SetupMiddleware(router)

			// Initialisation des handlers avec leurs dépendances
			h := handlers.New(cfg, valkeyClient)
			h.RegisterRoutes(router)

			// Démarrage du serveur HTTP
			portStr := strconv.Itoa(cfg.Port)
			logger.Info().Str("port", portStr).Msg("Démarrage du serveur")
			return router.Run(":" + portStr)
		},
	}

	// Exécution de la commande CLI
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal().Err(err).Msg("Échec du serveur")
	}
}
