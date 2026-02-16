// Package handlers provides HTTP request handlers for the API endpoints
package handlers

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	htmpl "html/template"
	"net/http"
	"strings"
	"time"

	"github.com/FRFlo/isen-ical-go/internal/config"
	"github.com/FRFlo/isen-ical-go/internal/services/aurion"
	"github.com/FRFlo/isen-ical-go/internal/services/auth"
	"github.com/FRFlo/isen-ical-go/internal/services/ical"
	"github.com/FRFlo/isen-ical-go/internal/services/session"
	"github.com/FRFlo/isen-ical-go/internal/services/template"
	"github.com/FRFlo/isen-ical-go/internal/services/token"
	"github.com/FRFlo/isen-ical-go/internal/storage"
	"github.com/gin-gonic/gin"
)

//go:embed homepage.gohtml
var homepageTemplate string

//go:embed privacy.gohtml
var privacyTemplate string

// Handlers holds all HTTP handlers with their dependencies
type Handlers struct {
	config     *config.Config
	valkey     *storage.ValkeyClient
	tokenSvc   *token.Service
	sessionSvc *session.Service
	aurionSvc  *aurion.Client
}

// New creates a new Handlers instance with all dependencies
func New(cfg *config.Config, valkey *storage.ValkeyClient) *Handlers {
	return &Handlers{
		config:     cfg,
		valkey:     valkey,
		tokenSvc:   token.NewService(valkey),
		sessionSvc: session.NewService(valkey, cfg),
		aurionSvc:  aurion.NewClient(cfg.AurionBaseURL),
	}
}

func (h *Handlers) RegisterRoutes(r *gin.Engine) {
	r.GET("/", h.Home)
	r.POST("/api/generate-token", h.GenerateToken)
	r.GET("/calendar/:token", h.Calendar)
	r.GET("/privacy", h.Privacy)
	r.GET("/health", h.Health)
}

func (h *Handlers) Home(c *gin.Context) {
	acceptHeader := c.GetHeader("Accept")

	if strings.Contains(acceptHeader, "text/html") {
		h.serveHomepage(c)
		return
	}

	h.handleBasicAuthCalendar(c)
}

// HomepageData holds data for the homepage template
type HomepageData struct {
	WebcalURL htmpl.URL
}

func (h *Handlers) serveHomepage(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")

	host := c.Request.Host
	if forwardedHost := c.Request.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
		host = forwardedHost
	}

	// webcal:// protocol for calendar apps
	webcalURL := fmt.Sprintf("webcal://%s/", host)

	data := HomepageData{
		WebcalURL: htmpl.URL(webcalURL),
	}

	rendered, err := template.Render(homepageTemplate, data)
	if err != nil {
		c.String(http.StatusInternalServerError, "Template error: "+err.Error())
		return
	}
	c.String(http.StatusOK, rendered)
}

func (h *Handlers) handleBasicAuthCalendar(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	authResult := auth.ParseBasicAuth(authHeader)

	if !authResult.Success {
		switch authResult.FailReason {
		case "missing_header":
			c.Header("WWW-Authenticate", `Basic realm="ISEN iCal"`)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
		case "invalid_format":
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid authorization format"})
		case "decode_error":
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to decode credentials"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Authentication failed"})
		}
		return
	}

	creds := authResult.Credentials

	if err := h.aurionSvc.Login(creds.Email, creds.Password); err != nil {
		c.Header("WWW-Authenticate", `Basic realm="ISEN iCal"`)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate iCal
	icalData, err := h.generateICal(creds.Email, creds.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate calendar"})
		return
	}

	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Content-Disposition", "inline; filename=calendar.ics")
	c.String(http.StatusOK, icalData)
}

type GenerateTokenRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type GenerateTokenResponse struct {
	Token         string `json:"token"`
	EncryptionKey string `json:"encryptionKey"`
	URL           string `json:"url"`
}

func (h *Handlers) GenerateToken(c *gin.Context) {
	var req GenerateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if err := h.aurionSvc.Login(req.Email, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	tokenID, key, _, err := h.tokenSvc.GenerateToken(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	scheme := "https"
	if c.Request.TLS == nil && c.Request.Header.Get("X-Forwarded-Proto") == "" {
		if c.Request.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
	}

	host := c.Request.Host
	if forwardedHost := c.Request.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
		host = forwardedHost
	}

	calendarURL := fmt.Sprintf("%s://%s/calendar/%s?key=%s", scheme, host, tokenID, key)

	resp := GenerateTokenResponse{
		Token:         tokenID,
		EncryptionKey: key,
		URL:           calendarURL,
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handlers) Calendar(c *gin.Context) {
	tokenID := c.Param("token")
	key := c.Query("key")

	if tokenID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
		return
	}

	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Encryption key is required"})
		return
	}

	keyBytes, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid encryption key format"})
		return
	}

	storedToken, err := h.tokenSvc.GetTokenData(tokenID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	creds, err := token.DecryptCredentials(storedToken, keyBytes)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Failed to decrypt token"})
		return
	}

	// Generate iCal
	icalData, err := h.generateICal(creds.Email, creds.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate calendar"})
		return
	}

	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Content-Disposition", "inline; filename=calendar.ics")
	c.String(http.StatusOK, icalData)
}

func (h *Handlers) generateICal(email, password string) (string, error) {
	passwordHash := session.HashPassword(password)

	now := time.Now()
	start := now.Add(-7*24*time.Hour).Unix() * 1000
	end := now.Add(60*24*time.Hour).Unix() * 1000

	cacheKey := storage.CacheKey(email, passwordHash, fmt.Sprintf("%d", start), fmt.Sprintf("%d", end))

	cachedEvents, err := h.sessionSvc.GetCachedEvents(cacheKey)
	if err == nil && len(cachedEvents) > 0 {
		return ical.GenerateICal(cachedEvents), nil
	}

	lockKey := storage.LockKey(email, passwordHash, fmt.Sprintf("%d", start), fmt.Sprintf("%d", end))
	acquired, err := h.sessionSvc.AcquireLock(lockKey)
	if err != nil {
		return "", fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !acquired {
		time.Sleep(2 * time.Second)
		cachedEvents, err := h.sessionSvc.GetCachedEvents(cacheKey)
		if err == nil && len(cachedEvents) > 0 {
			return ical.GenerateICal(cachedEvents), nil
		}
		return "", fmt.Errorf("calendar data temporarily unavailable, please retry")
	}

	defer func(sessionSvc *session.Service, key string) {
		err := sessionSvc.ReleaseLock(key)
		if err != nil {
			fmt.Printf("Failed to release lock: %v\n", err)
		}
	}(h.sessionSvc, lockKey)

	events, err := h.aurionSvc.GetPlanning(email, password, &start, &end)
	if err != nil {
		return "", fmt.Errorf("failed to fetch planning: %w", err)
	}

	_ = h.sessionSvc.CacheEvents(cacheKey, events)

	return ical.GenerateICal(events), nil
}

func (h *Handlers) Privacy(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")

	rendered, err := template.Render(privacyTemplate, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Template error: "+err.Error())
		return
	}
	c.String(http.StatusOK, rendered)
}

func (h *Handlers) Health(c *gin.Context) {
	valkeyHealthy := true
	if err := h.valkey.Ping(); err != nil {
		valkeyHealthy = false
	}

	status := gin.H{
		"status": "healthy",
	}

	if !valkeyHealthy {
		status["status"] = "unhealthy"
		status["valkey"] = "disconnected"
		c.JSON(http.StatusServiceUnavailable, status)
		return
	}

	c.JSON(http.StatusOK, status)
}
