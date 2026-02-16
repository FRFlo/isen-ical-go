# Aurion Web Scraping Service - Learnings

## Implementation Summary

Created a complete Aurion web scraping service in Go that replicates the TypeScript reference implementation.

## Files Created

1. `internal/models/event.go` - AurionEvent struct with JSON tags
2. `internal/services/aurion/aurion.go` - Main scraping service with:
   - HTTP client with cookie jar for session management
   - Login flow with redirect handling
   - Navigation flow through JSF/PrimeFaces pages
   - All regex parsers for ViewState, idInit, menu IDs, and events
3. `internal/services/aurion/aurion_test.go` - Comprehensive tests for all parsing functions

## Key Technical Decisions

### HTTP Client Configuration
- Used `net/http/cookiejar` for automatic cookie management
- Set `CheckRedirect` to return `http.ErrUseLastResponse` to handle redirects manually
- This allows checking for 302 status on login success

### Regex Patterns (from TypeScript reference)
- ViewState: `<input type="hidden" name="javax\.faces\.ViewState" id="j_id1:javax\.faces\.ViewState:0" value="([^"]+)"`
- idInit: String search for `name="form:idInit" value="`
- Sidebar Menu: Complex regex for PrimeFaces.addSubmitParam with Mon Planning
- Form ID Planning: `PrimeFaces\.cw\("Schedule","schedule",\{id:"([^"]+)"`
- Planning Data: `\[\{"id".*?\}\]` - matches JSON array of events

### Date Handling
- Default range: 7 days ago to 60 days ahead (in milliseconds)
- Week calculation uses ISO week format via `time.Time.ISOWeek()`
- Date formatting for Aurion: French format DD/MM/YYYY

### Form Data Encoding
- Used `net/url.Values` for proper URL encoding
- All form fields from reference implementation included
- AJAX headers for planning data request

## Testing Strategy

- Unit tests for each parser function with edge cases
- Tests cover: valid inputs, missing data, empty HTML, special characters
- JSON marshaling/unmarshaling tests for AurionEvent struct
- All 29 test cases passing

## Notes for Future Development

- The service doesn't include caching or session persistence (to be handled by caller)
- No retry logic - caller should handle session expiration and re-login
- HTTP-only implementation (no headless browser) as specified
- All regex patterns match the TypeScript reference exactly

## Redis Client Implementation - 2026-02-13

### Implementation Summary
Created `internal/storage/redis.go` with Redis client wrapper using go-redis/v9:

**Key Features:**
- Connection pooling configured (PoolSize: 10, MinIdleConns: 2)
- Timeout configurations for production resilience
- Support for redis:// and rediss:// URL formats
- All required operations: Set, Get, Delete, SetNX, Ping

**Key Pattern Functions:**
- `SessionKey(email, hash16)` → `session:{email}:{hash16}`
- `CacheKey(email, hash16, start, end)` → `events:user:{email}:{hash16}:{start}:{end}`
- `LockKey(email, hash16, start, end)` → `lock:user:{email}:{hash16}:{start}:{end}`
- `TokenKey(uuid)` → `token:{uuid}`
- `UserTokensKey(email)` → `user:{email}:tokens`

**Testing Approach:**
- Tests skip if Redis unavailable (using `skipIfNoRedis` helper)
- Unit tests for key patterns and URL parsing (no Redis required)
- Integration tests for actual Redis operations
- Concurrent access test validates connection pooling

**Connection Pool Settings:**
```go
PoolSize:     10
MinIdleConns: 2
MaxRetries:   3
DialTimeout:  5s
ReadTimeout:  3s
WriteTimeout: 3s
PoolTimeout:  4s
IdleTimeout:  5m
MaxConnAge:   30m
```

**Error Handling:**
- Returns descriptive errors for connection failures
- Distinguishes between "key not found" and other errors
- Graceful handling of Redis URL parsing errors

## Docker Configuration - 2026-02-13
- Created multi-stage Dockerfile (golang:1.21-alpine → alpine:latest)
- Created docker-compose.yml with app and Redis services
- Created .dockerignore excluding dev files
- Configuration uses environment variables for all settings
- Redis persistence via named volume
