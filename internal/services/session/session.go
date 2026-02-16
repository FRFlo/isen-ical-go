// Package session fournit la gestion des sessions et le caching pour les sessions Aurion
package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/FRFlo/isen-ical-go/internal/config"
	"github.com/FRFlo/isen-ical-go/internal/models"
	"github.com/FRFlo/isen-ical-go/internal/storage"
)

// Service fournit les opérations de gestion des sessions et de caching
type Service struct {
	valkey *storage.ValkeyClient
	config *config.Config
}

// NewService crée un nouveau service de session
func NewService(valkey *storage.ValkeyClient, cfg *config.Config) *Service {
	return &Service{
		valkey: valkey,
		config: cfg,
	}
}

// HashPassword hache un mot de passe avec SHA-256 et retourne les 16 premiers caractères
func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])[:16]
}

// GetSession récupère les cookies stockés pour une session utilisateur depuis Valkey
func (s *Service) GetSession(email, passwordHash string) ([]http.Cookie, error) {
	key := storage.SessionKey(email, passwordHash)
	data, err := s.valkey.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var cookies []http.Cookie
	if err := json.Unmarshal([]byte(data), &cookies); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cookies: %w", err)
	}

	return cookies, nil
}

// SaveSession stocke les cookies pour une session utilisateur dans Valkey avec TTL
func (s *Service) SaveSession(email, passwordHash string, cookies []http.Cookie) error {
	key := storage.SessionKey(email, passwordHash)
	data, err := json.Marshal(cookies)
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %w", err)
	}

	ttl := time.Duration(s.config.SessionTTL) * time.Second
	if err := s.valkey.Set(key, data, ttl); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// DeleteSession supprime une session utilisateur de Valkey
func (s *Service) DeleteSession(email, passwordHash string) error {
	key := storage.SessionKey(email, passwordHash)
	if err := s.valkey.Delete(key); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// GetCachedEvents récupère les événements en cache depuis Valkey
func (s *Service) GetCachedEvents(key string) ([]models.AurionEvent, error) {
	data, err := s.valkey.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get cached events: %w", err)
	}

	var events []models.AurionEvent
	if err := json.Unmarshal([]byte(data), &events); err != nil {
		return nil, fmt.Errorf("failed to unmarshal events: %w", err)
	}

	return events, nil
}

// CacheEvents stocke les événements dans Valkey avec TTL
func (s *Service) CacheEvents(key string, events []models.AurionEvent) error {
	data, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	ttl := time.Duration(s.config.CacheTTL) * time.Second
	if err := s.valkey.Set(key, data, ttl); err != nil {
		return fmt.Errorf("failed to cache events: %w", err)
	}

	return nil
}

// AcquireLock tente d'acquérir un verrou distribué avec SetNX
// Retourne true si le verrou a été acquis, false s'il existe déjà
func (s *Service) AcquireLock(key string) (bool, error) {
	// Le TTL du verrou est de 60 secondes comme spécifié
	ttl := 60 * time.Second
	acquired, err := s.valkey.SetNX(key, "1", ttl)
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}
	return acquired, nil
}

// ReleaseLock libère un verrou distribué
func (s *Service) ReleaseLock(key string) error {
	if err := s.valkey.Delete(key); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	return nil
}
