package storage

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func getValkeyURL() string {
	url := os.Getenv("VALKEY_URL")
	if url == "" {
		url = "valkey://localhost:6379"
	}
	return url
}

func skipIfNoValkey(t *testing.T) *ValkeyClient {
	client, err := NewValkeyClient(getValkeyURL())
	if err != nil {
		t.Skipf("Valkey not available: %v", err)
	}
	return client
}

func TestNewValkeyClient(t *testing.T) {
	client := skipIfNoValkey(t)
	defer client.Close()

	if err := client.Ping(); err != nil {
		t.Errorf("ping failed: %v", err)
	}
}

func TestNewValkeyClient_InvalidURL(t *testing.T) {
	_, err := NewValkeyClient("invalid://url")
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestValkeyClient_SetAndGet(t *testing.T) {
	client := skipIfNoValkey(t)
	defer client.Close()

	tests := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{
			name:  "simple string value",
			key:   "test:key:1",
			value: "hello world",
			want:  "hello world",
		},
		{
			name:  "json-like value",
			key:   "test:key:2",
			value: `{"name":"test","value":123}`,
			want:  `{"name":"test","value":123}`,
		},
		{
			name:  "empty string",
			key:   "test:key:3",
			value: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.Set(tt.key, tt.value, time.Minute)
			if err != nil {
				t.Fatalf("Set failed: %v", err)
			}

			got, err := client.Get(tt.key)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}

			if got != tt.want {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValkeyClient_Get_NotFound(t *testing.T) {
	client := skipIfNoValkey(t)
	defer client.Close()

	_, err := client.Get("nonexistent:key")
	if err == nil {
		t.Error("expected error for nonexistent key, got nil")
	}
}

func TestValkeyClient_Delete(t *testing.T) {
	client := skipIfNoValkey(t)
	defer client.Close()

	key := "test:delete:key"
	value := "to be deleted"

	if err := client.Set(key, value, time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	if err := client.Delete(key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := client.Get(key)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestValkeyClient_SetNX(t *testing.T) {
	client := skipIfNoValkey(t)
	defer client.Close()

	key := "test:setnx:key"

	set, err := client.SetNX(key, "first", time.Minute)
	if err != nil {
		t.Fatalf("SetNX failed: %v", err)
	}
	if !set {
		t.Error("first SetNX should return true")
	}

	set, err = client.SetNX(key, "second", time.Minute)
	if err != nil {
		t.Fatalf("SetNX failed: %v", err)
	}
	if set {
		t.Error("second SetNX should return false")
	}

	val, err := client.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "first" {
		t.Errorf("value = %v, want first", val)
	}

	_ = client.Delete(key)
}

func TestValkeyClient_Set_WithTTL(t *testing.T) {
	client := skipIfNoValkey(t)
	defer client.Close()

	key := "test:ttl:key"
	value := "expires quickly"

	if err := client.Set(key, value, 100*time.Millisecond); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	_, err := client.Get(key)
	if err != nil {
		t.Fatalf("key should exist: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	_, err = client.Get(key)
	if err == nil {
		t.Error("expected error after TTL expiration, got nil")
	}
}

func TestKeyPatterns(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() string
		expected string
	}{
		{
			name:     "SessionKey",
			fn:       func() string { return SessionKey("user@example.com", "abc123") },
			expected: "session:user@example.com:abc123",
		},
		{
			name:     "CacheKey",
			fn:       func() string { return CacheKey("user@example.com", "abc123", "2024-01-01", "2024-12-31") },
			expected: "events:user:user@example.com:abc123:2024-01-01:2024-12-31",
		},
		{
			name:     "LockKey",
			fn:       func() string { return LockKey("user@example.com", "abc123", "2024-01-01", "2024-12-31") },
			expected: "lock:user:user@example.com:abc123:2024-01-01:2024-12-31",
		},
		{
			name:     "TokenKey",
			fn:       func() string { return TokenKey("550e8400-e29b-41d4-a716-446655440000") },
			expected: "token:550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "UserTokensKey",
			fn:       func() string { return UserTokensKey("user@example.com") },
			expected: "user:user@example.com:tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseValkeyURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		wantHost     string
		wantPassword string
		wantDB       int
		wantErr      bool
	}{
		{
			name:     "simple valkey URL",
			url:      "valkey://localhost:6379",
			wantHost: "localhost:6379",
			wantDB:   0,
		},
		{
			name:         "valkey URL with password",
			url:          "valkey://:secret@localhost:6379",
			wantHost:     "localhost:6379",
			wantPassword: "secret",
			wantDB:       0,
		},
		{
			name:     "valkey URL with DB",
			url:      "valkey://localhost:6379/3",
			wantHost: "localhost:6379",
			wantDB:   3,
		},
		{
			name:         "valkey URL with password and DB",
			url:          "valkey://user:secret@localhost:6379/5",
			wantHost:     "localhost:6379",
			wantPassword: "secret",
			wantDB:       5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, password, db, err := ParseValkeyURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseValkeyURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if host != tt.wantHost {
				t.Errorf("host = %v, want %v", host, tt.wantHost)
			}
			if password != tt.wantPassword {
				t.Errorf("password = %v, want %v", password, tt.wantPassword)
			}
			if db != tt.wantDB {
				t.Errorf("db = %v, want %v", db, tt.wantDB)
			}
		})
	}
}

func TestValkeyClient_ConcurrentAccess(t *testing.T) {
	client := skipIfNoValkey(t)
	defer client.Close()

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := fmt.Sprintf("test:concurrent:%d", id)
			value := fmt.Sprintf("value-%d", id)

			if err := client.Set(key, value, time.Minute); err != nil {
				t.Errorf("Set failed: %v", err)
			}

			got, err := client.Get(key)
			if err != nil {
				t.Errorf("Get failed: %v", err)
			}
			if got != value {
				t.Errorf("got %v, want %v", got, value)
			}

			if err := client.Delete(key); err != nil {
				t.Errorf("Delete failed: %v", err)
			}

			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
