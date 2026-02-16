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

func main() {
	cmd := &cli.Command{
		Name:  "isen-ical",
		Usage: "ISEN Calendar Generator - Convert Aurion planning to iCal",
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
		Action: func(ctx context.Context, cmd *cli.Command) error {
			logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

			port, err := strconv.Atoi(cmd.String("port"))
			if err != nil {
				return fmt.Errorf("invalid port: %w", err)
			}

			cfg := &config.Config{
				Port:             port,
				ValkeyURL:        cmd.String("valkey-url"),
				AurionBaseURL:    cmd.String("aurion-url"),
				MaxTokensPerUser: cmd.Int("max-tokens"),
				SessionTTL:       cmd.Int("session-ttl"),
				CacheTTL:         cmd.Int("cache-ttl"),
			}

			encKeyHex := cmd.String("encryption-key")
			if encKeyHex != "" {
				encKey, err := hex.DecodeString(encKeyHex)
				if err != nil {
					return fmt.Errorf("encryption-key must be valid hex: %w", err)
				}
				if len(encKey) != 32 {
					return fmt.Errorf("encryption-key must be 32 bytes (64 hex chars), got %d bytes", len(encKey))
				}
				cfg.EncryptionKey = encKey
			}

			if cmd.String("gin-mode") == "release" {
				gin.SetMode(gin.ReleaseMode)
			}

			valkeyClient, err := storage.NewValkeyClient(cfg.ValkeyURL)
			if err != nil {
				logger.Fatal().Err(err).Msg("Failed to connect to Valkey")
			}
			defer valkeyClient.Close()

			router := gin.New()
			middleware.SetupMiddleware(router)

			h := handlers.New(cfg, valkeyClient)
			h.RegisterRoutes(router)

			portStr := strconv.Itoa(cfg.Port)
			logger.Info().Str("port", portStr).Msg("Starting server")
			return router.Run(":" + portStr)
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal().Err(err).Msg("Server failed")
	}
}
