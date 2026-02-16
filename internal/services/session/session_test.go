package session

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"testing"
	"time"

	"github.com/FRFlo/isen-ical-go/internal/config"
	"github.com/FRFlo/isen-ical-go/internal/models"
	"github.com/FRFlo/isen-ical-go/internal/storage"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func isDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	return cmd.Run() == nil
}

func setupValkeyContainer(t *testing.T) (string, func()) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping test")
	}

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "valkey/valkey:8-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	valkeyC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Skipf("Failed to start Valkey container: %v", err)
	}

	host, err := valkeyC.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get valkey host: %v", err)
	}

	port, err := valkeyC.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("failed to get valkey port: %v", err)
	}

	valkeyURL := fmt.Sprintf("valkey://%s:%s", host, port.Port())

	cleanup := func() {
		if err := valkeyC.Terminate(ctx); err != nil {
			t.Logf("failed to terminate valkey container: %v", err)
		}
	}

	return valkeyURL, cleanup
}

func setupTestService(t *testing.T) (*Service, func()) {
	valkeyURL, cleanup := setupValkeyContainer(t)

	valkeyClient, err := storage.NewValkeyClient(valkeyURL)
	if err != nil {
		cleanup()
		t.Fatalf("failed to create valkey client: %v", err)
	}

	cfg := &config.Config{
		SessionTTL: 3600,
		CacheTTL:   3600,
	}

	service := NewService(valkeyClient, cfg)

	return service, func() {
		valkeyClient.Close()
		cleanup()
	}
}

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantLen  int
	}{
		{
			name:     "simple password",
			password: "password123",
			wantLen:  16,
		},
		{
			name:     "empty password",
			password: "",
			wantLen:  16,
		},
		{
			name:     "long password",
			password: "this is a very long password with many characters",
			wantLen:  16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := HashPassword(tt.password)
			if len(hash) != tt.wantLen {
				t.Errorf("HashPassword() length = %d, want %d", len(hash), tt.wantLen)
			}
		})
	}
}

func TestHashPassword_Deterministic(t *testing.T) {
	password := "testpassword"
	hash1 := HashPassword(password)
	hash2 := HashPassword(password)

	if hash1 != hash2 {
		t.Errorf("HashPassword() not deterministic: %s != %s", hash1, hash2)
	}
}

func TestService_SessionRoundtrip(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	email := "test@example.com"
	passwordHash := HashPassword("password123")

	cookies := []http.Cookie{
		{
			Name:  "session_id",
			Value: "abc123",
			Path:  "/",
		},
		{
			Name:  "auth_token",
			Value: "xyz789",
			Path:  "/",
		},
	}

	err := service.SaveSession(email, passwordHash, cookies)
	if err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	retrievedCookies, err := service.GetSession(email, passwordHash)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if len(retrievedCookies) != len(cookies) {
		t.Errorf("got %d cookies, want %d", len(retrievedCookies), len(cookies))
	}

	for i, cookie := range retrievedCookies {
		if cookie.Name != cookies[i].Name {
			t.Errorf("cookie[%d].Name = %s, want %s", i, cookie.Name, cookies[i].Name)
		}
		if cookie.Value != cookies[i].Value {
			t.Errorf("cookie[%d].Value = %s, want %s", i, cookie.Value, cookies[i].Value)
		}
	}
}

func TestService_GetSession_NotFound(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	email := "nonexistent@example.com"
	passwordHash := HashPassword("password123")

	_, err := service.GetSession(email, passwordHash)
	if err == nil {
		t.Error("expected error for nonexistent session, got nil")
	}
}

func TestService_DeleteSession(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	email := "test@example.com"
	passwordHash := HashPassword("password123")
	cookies := []http.Cookie{{Name: "session", Value: "test"}}

	if err := service.SaveSession(email, passwordHash, cookies); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	if err := service.DeleteSession(email, passwordHash); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	_, err := service.GetSession(email, passwordHash)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestService_CacheEventsRoundtrip(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	key := "test:cache:key"
	events := []models.AurionEvent{
		{
			ID:        "1",
			Title:     "Test Event 1",
			Start:     "2024-01-01T10:00:00",
			End:       "2024-01-01T11:00:00",
			AllDay:    false,
			Editable:  true,
			ClassName: "event-class",
		},
		{
			ID:        "2",
			Title:     "Test Event 2",
			Start:     "2024-01-02T14:00:00",
			End:       "2024-01-02T15:00:00",
			AllDay:    false,
			Editable:  false,
			ClassName: "event-class-2",
		},
	}

	err := service.CacheEvents(key, events)
	if err != nil {
		t.Fatalf("CacheEvents failed: %v", err)
	}

	retrievedEvents, err := service.GetCachedEvents(key)
	if err != nil {
		t.Fatalf("GetCachedEvents failed: %v", err)
	}

	if len(retrievedEvents) != len(events) {
		t.Errorf("got %d events, want %d", len(retrievedEvents), len(events))
	}

	for i, event := range retrievedEvents {
		if event.ID != events[i].ID {
			t.Errorf("event[%d].ID = %s, want %s", i, event.ID, events[i].ID)
		}
		if event.Title != events[i].Title {
			t.Errorf("event[%d].Title = %s, want %s", i, event.Title, events[i].Title)
		}
	}
}

func TestService_GetCachedEvents_NotFound(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	key := "nonexistent:cache:key"

	_, err := service.GetCachedEvents(key)
	if err == nil {
		t.Error("expected error for nonexistent cache, got nil")
	}
}

func TestService_LockBehavior(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	key := "test:lock:key"

	acquired, err := service.AcquireLock(key)
	if err != nil {
		t.Fatalf("first AcquireLock failed: %v", err)
	}
	if !acquired {
		t.Error("first AcquireLock should return true")
	}

	acquired, err = service.AcquireLock(key)
	if err != nil {
		t.Fatalf("second AcquireLock failed: %v", err)
	}
	if acquired {
		t.Error("second AcquireLock should return false")
	}

	if err := service.ReleaseLock(key); err != nil {
		t.Fatalf("ReleaseLock failed: %v", err)
	}

	acquired, err = service.AcquireLock(key)
	if err != nil {
		t.Fatalf("third AcquireLock failed: %v", err)
	}
	if !acquired {
		t.Error("third AcquireLock should return true after release")
	}
}

func TestService_LockTTL(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	key := "test:lock:ttl"

	acquired, err := service.AcquireLock(key)
	if err != nil {
		t.Fatalf("AcquireLock failed: %v", err)
	}
	if !acquired {
		t.Error("AcquireLock should return true")
	}

	if err := service.ReleaseLock(key); err != nil {
		t.Fatalf("ReleaseLock failed: %v", err)
	}

	acquired, err = service.AcquireLock(key)
	if err != nil {
		t.Fatalf("second AcquireLock failed: %v", err)
	}
	if !acquired {
		t.Error("should be able to acquire lock after release")
	}
}

func TestService_SessionWithCacheKey(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	email := "user@example.com"
	passwordHash := HashPassword("secretpassword")
	start := "2024-01-01"
	end := "2024-12-31"

	cacheKey := storage.CacheKey(email, passwordHash, start, end)
	events := []models.AurionEvent{
		{ID: "1", Title: "Event 1", Start: "2024-06-01T10:00:00", End: "2024-06-01T11:00:00"},
	}

	err := service.CacheEvents(cacheKey, events)
	if err != nil {
		t.Fatalf("CacheEvents failed: %v", err)
	}

	retrievedEvents, err := service.GetCachedEvents(cacheKey)
	if err != nil {
		t.Fatalf("GetCachedEvents failed: %v", err)
	}

	if len(retrievedEvents) != 1 {
		t.Errorf("got %d events, want 1", len(retrievedEvents))
	}
}

func TestService_LockWithLockKey(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	email := "user@example.com"
	passwordHash := HashPassword("secretpassword")
	start := "2024-01-01"
	end := "2024-12-31"

	lockKey := storage.LockKey(email, passwordHash, start, end)

	acquired, err := service.AcquireLock(lockKey)
	if err != nil {
		t.Fatalf("AcquireLock failed: %v", err)
	}
	if !acquired {
		t.Error("AcquireLock should return true")
	}

	if err := service.ReleaseLock(lockKey); err != nil {
		t.Fatalf("ReleaseLock failed: %v", err)
	}
}

func TestService_EmptyEvents(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	key := "test:empty:events"
	events := []models.AurionEvent{}

	err := service.CacheEvents(key, events)
	if err != nil {
		t.Fatalf("CacheEvents failed: %v", err)
	}

	retrievedEvents, err := service.GetCachedEvents(key)
	if err != nil {
		t.Fatalf("GetCachedEvents failed: %v", err)
	}

	if len(retrievedEvents) != 0 {
		t.Errorf("got %d events, want 0", len(retrievedEvents))
	}
}

func TestService_EmptyCookies(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	email := "test@example.com"
	passwordHash := HashPassword("password123")
	cookies := []http.Cookie{}

	err := service.SaveSession(email, passwordHash, cookies)
	if err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	retrievedCookies, err := service.GetSession(email, passwordHash)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if len(retrievedCookies) != 0 {
		t.Errorf("got %d cookies, want 0", len(retrievedCookies))
	}
}

func TestService_CacheTTLExpiration(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	service.config.CacheTTL = 1

	key := "test:ttl:expiration"
	events := []models.AurionEvent{
		{ID: "1", Title: "Event 1", Start: "2024-01-01T10:00:00", End: "2024-01-01T11:00:00"},
	}

	err := service.CacheEvents(key, events)
	if err != nil {
		t.Fatalf("CacheEvents failed: %v", err)
	}

	_, err = service.GetCachedEvents(key)
	if err != nil {
		t.Fatalf("GetCachedEvents should succeed immediately: %v", err)
	}

	time.Sleep(2 * time.Second)

	_, err = service.GetCachedEvents(key)
	if err == nil {
		t.Error("expected error after TTL expiration, got nil")
	}
}
