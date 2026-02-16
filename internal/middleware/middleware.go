package middleware

import (
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// LoggerMiddleware retourne un middleware Gin qui journalise les requêtes HTTP avec zerolog.
// Il enregistre la méthode, le chemin, le statut, la latence, l'IP client, le user agent,
// les paramètres de requête et les erreurs.
// Le niveau de log est déterminé par le code de statut : Error pour 5xx, Warn pour 4xx, Info pour le succès.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get client IP
		clientIP := c.ClientIP()

		// Get status code
		statusCode := c.Writer.Status()

		// Build log event with common fields
		event := log.Info().
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", statusCode).
			Dur("latency", latency).
			Str("ip", clientIP).
			Str("user-agent", c.Request.UserAgent())

		// Add query params if present
		if raw != "" {
			event = event.Str("query", raw)
		}

		// Add route params if present
		if len(c.Params) > 0 {
			params := make(map[string]string)
			for _, param := range c.Params {
				params[param.Key] = param.Value
			}
			event = event.Interface("params", params)
		}

		// Add errors if any
		if len(c.Errors) > 0 {
			event = event.Interface("errors", c.Errors.Errors())
		}

		// Determine log level based on status code
		msg := fmt.Sprintf("%s %s %d", c.Request.Method, path, statusCode)

		switch {
		case statusCode >= 500:
			event.Str("level", "error").Msg(msg)
		case statusCode >= 400:
			event.Str("level", "warn").Msg(msg)
		default:
			event.Msg(msg)
		}
	}
}

// RecoveryMiddleware retourne un middleware Gin qui récupère les panics.
// Il journalise le panic avec la trace d'appels et retourne une erreur 500.
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err any) {
		log.Error().
			Interface("error", err).
			Str("path", c.Request.URL.Path).
			Str("method", c.Request.Method).
			Msg("Panic recovered")

		c.AbortWithStatusJSON(500, gin.H{
			"error": "Internal Server Error",
		})
	})
}

// CORSMiddleware retourne un middleware Gin qui gère le CORS.
// Configuré pour autoriser toutes les origines pour la compatibilité Cloudflare.
func CORSMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
}

// SetupMiddleware configure tous les middlewares pour le routeur.
func SetupMiddleware(router *gin.Engine) {
	router.Use(LoggerMiddleware())
	router.Use(RecoveryMiddleware())
	router.Use(CORSMiddleware())
}
