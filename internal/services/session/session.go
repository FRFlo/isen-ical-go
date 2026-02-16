// Package session provides session management and caching for Aurion sessions
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

// Service provides session management and caching operations
type Service struct {
	valkey *storage.ValkeyClient
	config *config.Config
}

// NewService creates a new session service
func NewService(valkey *storage.ValkeyClient, cfg *config.Config) *Service {
	return &Service{
		valkey: valkey,
		config: cfg,
	}
}

// HashPassword hashes a password using SHA-256 and returns the first 16 characters
func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])[:16]
}

// GetSession retrieves stored cookies for a user session from Valkey
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

// SaveSession stores cookies for a user session in Valkey with TTL
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

// DeleteSession removes a user session from Valkey
func (s *Service) DeleteSession(email, passwordHash string) error {
	key := storage.SessionKey(email, passwordHash)
	if err := s.valkey.Delete(key); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// GetCachedEvents retrieves cached events from Valkey
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

// CacheEvents stores events in Valkey with TTL
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

// AcquireLock attempts to acquire a distributed lock using SetNX
// Returns true if the lock was acquired, false if it already exists
func (s *Service) AcquireLock(key string) (bool, error) {
	// Lock TTL is 60 seconds as specified
	ttl := 60 * time.Second
	acquired, err := s.valkey.SetNX(key, "1", ttl)
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}
	return acquired, nil
}

// ReleaseLock releases a distributed lock
func (s *Service) ReleaseLock(key string) error {
	if err := s.valkey.Delete(key); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	return nil
}
