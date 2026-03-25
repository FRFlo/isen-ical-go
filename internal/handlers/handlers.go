// Package handlers fournit les gestionnaires de requêtes HTTP pour les endpoints de l'API.
package handlers

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
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

// Handlers contient tous les gestionnaires HTTP avec leurs dépendances.
type Handlers struct {
	config     *config.Config
	valkey     *storage.ValkeyClient
	tokenSvc   *token.Service
	sessionSvc *session.Service
	aurionSvc  *aurion.Client
}

// New crée une nouvelle instance de Handlers avec toutes ses dépendances.
func New(cfg *config.Config, valkey *storage.ValkeyClient) *Handlers {
	return &Handlers{
		config:     cfg,
		valkey:     valkey,
		tokenSvc:   token.NewService(valkey),
		sessionSvc: session.NewService(valkey, cfg),
		aurionSvc:  aurion.NewClient(cfg.AurionBaseURL),
	}
}

// RegisterRoutes enregistre toutes les routes HTTP sur le moteur Gin.
func (h *Handlers) RegisterRoutes(r *gin.Engine) {
	r.GET("/", h.Home)
	r.POST("/api/track", h.Track)
	r.POST("/api/generate-token", h.GenerateToken)
	r.GET("/calendar/:token", h.Calendar)
	r.GET("/privacy", h.Privacy)
	r.GET("/favicon.ico", h.Favicon)
	r.GET("/.well-known/appspecific/com.chrome.devtools.json", h.ChromeDevtoolsProbe)
	r.GET("/api/health", h.Health)
}

// Home gère la route racine et affiche soit la page d'accueil, soit le calendrier en fonction du header Accept.
func (h *Handlers) Home(c *gin.Context) {
	acceptHeader := c.GetHeader("Accept")

	if strings.Contains(acceptHeader, "text/html") {
		h.serveHomepage(c)
		return
	}

	h.handleBasicAuthCalendar(c)
}

// HomepageData contient les données pour le template de la page d'accueil.
type HomepageData struct {
	WebcalURL htmpl.URL
}

// serveHomepage sert la page d'accueil HTML avec le lien webcal://.
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

// handleBasicAuthCalendar gère l'authentification basique et retourne le calendrier au format iCal.
func (h *Handlers) handleBasicAuthCalendar(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	authResult := auth.ParseBasicAuth(authHeader)

	if !authResult.Success {
		switch authResult.FailReason {
		case "missing_header":
			c.Header("WWW-Authenticate", `Basic realm="Identifiants Aurion"`)
			c.String(http.StatusUnauthorized, "Authorization required")
		case "invalid_format":
			c.Header("WWW-Authenticate", `Basic realm="Identifiants Aurion"`)
			c.String(http.StatusUnauthorized, "Invalid authorization format")
		case "decode_error":
			c.Header("WWW-Authenticate", `Basic realm="Identifiants Aurion"`)
			c.String(http.StatusUnauthorized, "Invalid authorization format")
		default:
			c.Header("WWW-Authenticate", `Basic realm="Identifiants Aurion"`)
			c.String(http.StatusUnauthorized, "Invalid authorization format")
		}
		return
	}

	creds := authResult.Credentials

	if err := h.aurionSvc.Login(creds.Email, creds.Password); err != nil {
		c.String(http.StatusForbidden, "Invalid credentials")
		return
	}

	// Generate iCal
	icalData, err := h.generateICal(creds.Email, creds.Password)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error fetching schedule: "+err.Error())
		return
	}

	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="isen-ical.ics"`)
	c.String(http.StatusOK, icalData)
}

// GenerateTokenRequest représente la requête de génération de token.
type GenerateTokenRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
}

// GenerateTokenResponse représente la réponse de génération de token.
type GenerateTokenResponse struct {
	Token         string `json:"token"`
	EncryptionKey string `json:"encryptionKey"`
	URL           string `json:"url"`
}

// GenerateToken génère un nouveau token de calendrier pour l'utilisateur authentifié.
func (h *Handlers) GenerateToken(c *gin.Context) {
	var req GenerateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	username := strings.TrimSpace(req.Username)
	if username == "" {
		username = strings.TrimSpace(req.Email)
	}
	if username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username and password are required"})
		return
	}

	if err := h.aurionSvc.Login(username, req.Password); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid credentials"})
		return
	}

	tokenID, key, _, err := h.tokenSvc.GenerateToken(username, req.Password)
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

// Calendar retourne le calendrier iCal associé au token fourni.
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
		if strings.Contains(err.Error(), "Login failed") || strings.Contains(err.Error(), "No session cookie") {
			c.String(http.StatusForbidden, "Invalid credentials")
			return
		}
		c.String(http.StatusInternalServerError, "Error fetching schedule: "+err.Error())
		return
	}

	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="isen-ical.ics"`)
	c.String(http.StatusOK, icalData)
}

type TrackRequest struct {
	Event      string                 `json:"event"`
	DistinctID string                 `json:"distinctId"`
	Properties map[string]interface{} `json:"properties"`
}

// Track gère les événements de télémétrie frontend.
func (h *Handlers) Track(c *gin.Context) {
	if !strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "Content-Type must be application/json"})
		return
	}

	var req TrackRequest
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tracking payload"})
		return
	}

	if !strings.HasPrefix(req.Event, "frontend_") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Event name is invalid"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

// Favicon retourne une réponse vide pour compatibilité Worker.
func (h *Handlers) Favicon(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=3600")
	c.Status(http.StatusNoContent)
}

// ChromeDevtoolsProbe retourne une réponse vide pour la sonde Chrome devtools.
func (h *Handlers) ChromeDevtoolsProbe(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

// generateICal génère les données du calendrier iCal pour l'utilisateur donné avec mise en cache.
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

// Privacy sert la page de politique de confidentialité.
func (h *Handlers) Privacy(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")

	rendered, err := template.Render(privacyTemplate, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Template error: "+err.Error())
		return
	}
	c.String(http.StatusOK, rendered)
}

// Health vérifie l'état de santé de l'application et retourne le statut.
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
