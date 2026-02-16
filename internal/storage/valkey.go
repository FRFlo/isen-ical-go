// Package storage provides Valkey client wrapper with connection pooling
package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"
)

// ValkeyClient wraps go-redis client with connection pooling
type ValkeyClient struct {
	client *redis.Client
}

// NewValkeyClient creates a new Valkey client from a Valkey URL
// Supports valkey:// and valkeys:// URL formats
func NewValkeyClient(valkeyURL string) (*ValkeyClient, error) {
	opt, err := redis.ParseURL(valkeyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Valkey URL: %w", err)
	}

	// Configure connection pooling
	opt.PoolSize = 10
	opt.MinIdleConns = 2
	opt.MaxRetries = 3
	opt.DialTimeout = 5 * time.Second
	opt.ReadTimeout = 3 * time.Second
	opt.WriteTimeout = 3 * time.Second
	opt.PoolTimeout = 4 * time.Second
	opt.ConnMaxIdleTime = 5 * time.Minute
	opt.ConnMaxLifetime = 30 * time.Minute

	client := redis.NewClient(opt)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Valkey: %w", err)
	}

	return &ValkeyClient{client: client}, nil
}

// Close closes the Valkey client connection
func (v *ValkeyClient) Close() error {
	return v.client.Close()
}

// Ping checks if Valkey is reachable
func (v *ValkeyClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return v.client.Ping(ctx).Err()
}

// Set stores a value with the given key and TTL
func (v *ValkeyClient) Set(key string, value interface{}, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return v.client.Set(ctx, key, value, ttl).Err()
}

// Get retrieves a value by key
func (v *ValkeyClient) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	val, err := v.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key not found: %s", key)
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// Delete removes a key from Valkey
func (v *ValkeyClient) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return v.client.Del(ctx, key).Err()
}

// SetNX sets a value only if the key does not exist (for distributed locks)
// Returns true if the key was set, false if it already existed
func (v *ValkeyClient) SetNX(key string, value interface{}, ttl time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return v.client.SetNX(ctx, key, value, ttl).Result()
}

// Key pattern functions

// SessionKey returns the Valkey key for a user session
// Format: session:{email}:{hash16}
func SessionKey(email, hash16 string) string {
	return fmt.Sprintf("session:%s:%s", email, hash16)
}

// CacheKey returns the Valkey key for cached events
// Format: events:user:{email}:{hash16}:{start}:{end}
func CacheKey(email, hash16, start, end string) string {
	return fmt.Sprintf("events:user:%s:%s:%s:%s", email, hash16, start, end)
}

// LockKey returns the Valkey key for distributed locks
// Format: lock:user:{email}:{hash16}:{start}:{end}
func LockKey(email, hash16, start, end string) string {
	return fmt.Sprintf("lock:user:%s:%s:%s:%s", email, hash16, start, end)
}

// TokenKey returns the Valkey key for a token
// Format: token:{uuid}
func TokenKey(uuid string) string {
	return fmt.Sprintf("token:%s", uuid)
}

// UserTokensKey returns the Valkey key for a user's token list
// Format: user:{email}:tokens
func UserTokensKey(email string) string {
	return fmt.Sprintf("user:%s:tokens", email)
}

// ParseValkeyURL extracts components from a Valkey URL
// Useful for debugging or custom connection logic
func ParseValkeyURL(valkeyURL string) (host, password string, db int, err error) {
	u, err := url.Parse(valkeyURL)
	if err != nil {
		return "", "", 0, err
	}

	host = u.Host
	if u.User != nil {
		password, _ = u.User.Password()
	}

	// Extract DB from path
	db = 0
	if u.Path != "" && u.Path != "/" {
		fmt.Sscanf(u.Path, "/%d", &db)
	}

	return host, password, db, nil
}
