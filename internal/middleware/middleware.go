package middleware

import (
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// LoggerMiddleware returns a Gin middleware that logs HTTP requests using zerolog.
// It logs method, path, status, latency, client IP, user agent, query params, and errors.
// Log level is determined by status code: Error for 5xx, Warn for 4xx, Info for success.
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

// RecoveryMiddleware returns a Gin middleware that recovers from panics.
// It logs the panic with stack trace and returns a 500 error.
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

// RequestIDMiddleware adds a unique request ID to each request.
// The ID is added to the context and response headers.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set("requestID", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// CORSMiddleware returns a Gin middleware that handles CORS.
// Configured to allow all origins for Cloudflare compatibility.
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

// SetupMiddleware configures all middleware for the router
func SetupMiddleware(router *gin.Engine) {
	router.Use(LoggerMiddleware())
	router.Use(RecoveryMiddleware())
	router.Use(CORSMiddleware())
}
