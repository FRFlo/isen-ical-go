# ISEN-iCal Go: Cloudflare to Docker Migration

## TL;DR

> **Quick Summary**: Port the existing Cloudflare Worker `isen-ical` to a self-hosted Go+Docker solution with Redis backend, maintaining full feature parity.
> 
> **Deliverables**:
> - Go application with Gin HTTP server
> - Docker Compose setup (Go app + Redis)
> - Full feature parity: iCal generation, token auth, HTML pages
> - Unit + Integration tests
> - Structured logging with zerolog
> 
> **Estimated Effort**: Large (Multiple days of development)
> **Parallel Execution**: YES - 3 waves
> **Critical Path**: Project Setup → Aurion Scraping → iCal Generation → HTTP Endpoints → Docker

---

## Context

### Original Request
Port the existing Cloudflare Worker project (`isen-ical`) to a Docker and Golang based successor.

### Interview Summary
**Key Discussions**:
- **Storage**: Redis for sessions (1h TTL), cache (1h TTL), and tokens (persistent)
- **Framework**: Gin for HTTP server
- **Deployment**: Docker Compose (Go app + Redis), HTTP exposed, Cloudflare handles SSL
- **Configuration**: Environment variables (12-factor app)
- **Project Structure**: MVC-like (cmd/, handlers/, services/, models/, config/, middleware/)
- **Logging**: Structured logging with zerolog
- **Feature Parity**: Full parity with original (all endpoints, HTML pages, token limits)
- **Testing**: Unit + Integration tests

**Research Findings (from source code analysis)**:
- **Aurion Scraping**: Complex JSF/PrimeFaces navigation requiring ViewState management
- **Auth Methods**: HTTP Basic Auth and AES-GCM encrypted URL tokens
- **Token Model**: Max 3 tokens per user, stored with encryption IV
- **iCal Format**: RFC 5545 compliant with emoji prefixes for exam/self-study events

### Metis Review
**Identified Gaps (addressed)**:
- **Source Code Access**: RESOLVED - Full source analysis completed, detailed implementation known
- **Aurion Scraping Complexity**: Plan includes detailed JSF form navigation steps
- **Token Compatibility**: New system generates new tokens (no backward compatibility needed)
- **Redis HA**: Out of scope for single VPS deployment (documented as limitation)
- **Health Check**: Added `/health` endpoint to plan

---

## Work Objectives

### Core Objective
Create a production-ready Go application that replicates all functionality of the Cloudflare Worker `isen-ical`, deployable via Docker Compose with Redis backend.

### Concrete Deliverables
1. `cmd/server/main.go` - Application entry point
2. `internal/handlers/` - HTTP route handlers
3. `internal/services/` - Business logic (aurion, ical, token, auth)
4. `internal/models/` - Data structures
5. `internal/config/` - Configuration management
6. `templates/` - HTML templates (homepage, privacy)
7. `Dockerfile` - Multi-stage Go build
8. `docker-compose.yml` - Go app + Redis orchestration
9. `README.md` - Documentation
10. `*_test.go` - Test files

### Definition of Done
- [x] `docker compose up` starts both services successfully
- [x] `curl http://localhost:8080/health` returns `{"status":"healthy"}`
- [x] Basic Auth iCal endpoint works: `curl -u email:password http://localhost:8080/`
- [x] Token generation works: `POST /api/generate-token`
- [x] Token-based calendar works: `GET /calendar/{token}?key={key}`
- [x] All tests pass: `go test ./...`
- [x] Homepage and privacy page render correctly

### Must Have
- All 5 routes from original: `/`, `/api/generate-token`, `/calendar/{token}`, `/privacy`, `/health`
- HTTP Basic Auth support
- Token-based auth with AES-GCM encryption
- Redis storage for sessions, cache, and tokens
- iCal RFC 5545 compliant output
- Emoji prefixes: 🎓 for exams (EXAM_SURV), 🏠 for self-study (AUTO_APPR)
- Max 3 tokens per user
- Request deduplication for concurrent scrapes
- Session reuse and auto-reconnect
- Structured JSON logging
- Docker Compose deployment

### Must NOT Have (Guardrails)
- ❌ No React/Vue/Angular SPA (server-rendered HTML only)
- ❌ No ORM (direct Redis operations)
- ❌ No GraphQL (REST only)
- ❌ No microservices (single monolith)
- ❌ No OAuth/SSO/JWT (keep Basic Auth + tokens only)
- ❌ No admin dashboard or web UI for token management
- ❌ No analytics/metrics beyond logs
- ❌ No database migrations system
- ❌ No horizontal scaling support (single instance assumed)
- ❌ No new endpoints beyond `/health`
- ❌ No changes to URL structure or iCal format

---

## Verification Strategy (MANDATORY)

> **UNIVERSAL RULE: ZERO HUMAN INTERVENTION**
>
> ALL tasks in this plan MUST be verifiable WITHOUT any human action.
> Every criterion is verified by the agent using tools (curl, docker, go test).

### Test Decision
- **Infrastructure exists**: NO (greenfield project)
- **Automated tests**: YES (Tests-after for integration, unit tests alongside)
- **Framework**: Go standard `testing` package + `testify` for assertions

### Agent-Executed QA Scenarios (MANDATORY — ALL tasks)

**Verification Tool by Deliverable Type:**

| Type | Tool | How Agent Verifies |
|------|------|-------------------|
| **HTTP Endpoints** | Bash (curl) | Send requests, parse responses, assert fields |
| **Go Code** | Bash (go test) | Run tests, check exit code |
| **Docker** | Bash (docker compose) | Start containers, check logs, verify health |
| **Configuration** | Bash (env vars) | Set vars, verify app reads them |

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately):
├── Task 1: Project scaffold & config
├── Task 2: Redis client setup
└── Task 3: Aurion service (scraping)

Wave 2 (After Wave 1):
├── Task 4: Token service (depends: 2)
├── Task 5: iCal service (depends: 3)
├── Task 6: Auth service (standalone logic)
└── Task 7: Session service (depends: 2)

Wave 3 (After Wave 2):
├── Task 8: HTTP handlers (depends: 4, 5, 6, 7)
├── Task 9: HTML templates (standalone)
└── Task 10: Middleware (depends: 6)

Wave 4 (After Wave 3):
├── Task 11: Main entry point & routing (depends: 8, 9, 10)
└── Task 12: Docker setup (depends: all)

Wave 5 (Final):
└── Task 13: Integration tests & documentation (depends: all)

Critical Path: Task 1 → Task 2 → Task 4 → Task 8 → Task 11 → Task 12
```

### Dependency Matrix

| Task | Depends On | Blocks | Can Parallelize With |
|------|------------|--------|---------------------|
| 1 | None | 2, 3, 6 | None (first) |
| 2 | 1 | 4, 7 | 3, 6 |
| 3 | 1 | 5, 7 | 2, 6 |
| 4 | 2 | 8 | 5, 6, 7 |
| 5 | 3 | 8 | 4, 6, 7 |
| 6 | 1 | 8, 10 | 2, 3, 4, 5, 7 |
| 7 | 2, 3 | 8 | 4, 5, 6 |
| 8 | 4, 5, 6, 7 | 11 | 9, 10 |
| 9 | None | 11 | 8, 10 |
| 10 | 6 | 11 | 8, 9 |
| 11 | 8, 9, 10 | 12 | None |
| 12 | 11 | 13 | None |
| 13 | 12 | None | None (final) |

### Agent Dispatch Summary

| Wave | Tasks | Recommended Agents |
|------|-------|-------------------|
| 1 | 1, 2, 3 | quick (1), unspecified-low (2, 3) |
| 2 | 4, 5, 6, 7 | unspecified-low (all) |
| 3 | 8, 9, 10 | unspecified-low (8), quick (9, 10) |
| 4 | 11, 12 | unspecified-low |
| 5 | 13 | unspecified-high |

---

## TODOs

### Wave 1: Foundation

- [x] 1. Project Scaffold & Configuration

  **What to do**:
  - Initialize Go module: `go mod init github.com/FRFlo/isen-ical-go`
  - Create directory structure:
    ```
    cmd/server/main.go
    internal/config/config.go
    internal/handlers/
    internal/services/
    internal/models/
    internal/middleware/
    templates/
    ```
  - Implement config loading from environment variables:
    - `AURION_BASE_URL` (default: `https://aurion.junia.com`)
    - `REDIS_URL` (default: `redis://localhost:6379`)
    - `ENCRYPTION_KEY` (32-byte hex for AES-256)
    - `PORT` (default: `8080`)
    - `MAX_TOKENS_PER_USER` (default: `3`)
    - `SESSION_TTL` (default: `3600`)
    - `CACHE_TTL` (default: `3600`)
  - Install dependencies: `gin-gonic/gin`, `redis/go-redis/v9`, `rs/zerolog`

  **Must NOT do**:
  - Don't use viper or complex config management
  - Don't add database migrations
  - Don't create multiple cmd/ entry points

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`
  - Reason: Simple scaffolding task with boilerplate setup

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 1 (first task)
  - **Blocks**: Tasks 2, 3, 6
  - **Blocked By**: None (can start immediately)

  **References**:
  - `C:\Users\Flo\Desktop\isen-ical\wrangler.jsonc` - KV namespace TTL values
  - `C:\Users\Flo\Desktop\isen-ical\src\index.ts` - Environment usage patterns

  **Acceptance Criteria**:
  - [x] `go mod tidy` completes without errors
  - [x] Directory structure matches specification
  - [x] Config struct loads all environment variables with defaults

  **Agent-Executed QA Scenarios**:

  ```
  Scenario: Go module initializes correctly
    Tool: Bash
    Preconditions: Empty project directory
    Steps:
      1. cd C:\Users\Flo\Desktop\isen-ical-go
      2. go mod init github.com/FRFlo/isen-ical-go
      3. go mod tidy
    Expected Result: Exit code 0, go.mod exists
    Evidence: go.mod content captured
  
  Scenario: Config loads environment variables
    Tool: Bash
    Preconditions: Config package implemented
    Steps:
      1. Set PORT=9090
      2. Run test that loads config
      3. Assert config.Port == 9090
    Expected Result: Environment variable overrides default
    Evidence: Test output captured
  
  Scenario: Config uses defaults when env not set
    Tool: Bash
    Preconditions: Config package implemented
    Steps:
      1. Unset all env vars
      2. Run test that loads config
      3. Assert config.Port == 8080
    Expected Result: Default value used
    Evidence: Test output captured
  ```

  **Commit**: YES
  - Message: `feat(core): initialize project structure and configuration`
  - Files: `go.mod`, `go.sum`, `cmd/`, `internal/config/`
  - Pre-commit: `go build ./...`

---

- [x] 2. Redis Client Setup

  **What to do**:
  - Create `internal/storage/redis.go`
  - Implement Redis client wrapper with connection pooling
  - Implement key operations:
    - `Set(key string, value interface{}, ttl time.Duration) error`
    - `Get(key string) (string, error)`
    - `Delete(key string) error`
    - `SetNX(key string, value interface{}, ttl time.Duration) (bool, error)` - for locks
  - Implement health check: `Ping() error`
  - Key patterns (matching original):
    - Sessions: `session:{email}:{hash16}`
    - Cache: `events:user:{email}:{hash16}:{start}:{end}`
    - Locks: `lock:user:{email}:{hash16}:{start}:{end}`
    - Tokens: `token:{uuid}`
    - User tokens list: `user:{email}:tokens`

  **Must NOT do**:
  - Don't implement Redis clustering
  - Don't add Redis Sentinel support
  - Don't create abstraction layers beyond simple wrapper

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
  - **Skills**: `[]`
  - Reason: Standard Redis integration, well-documented

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 3, 6)
  - **Blocks**: Tasks 4, 7
  - **Blocked By**: Task 1

  **References**:
  - `C:\Users\Flo\Desktop\isen-ical\src\services\session.service.ts` - Key patterns, TTL values
  - `C:\Users\Flo\Desktop\isen-ical\src\services\token.service.ts` - Token storage patterns

  **Acceptance Criteria**:
  - [x] Redis client connects successfully
  - [x] Set/Get/Delete operations work
  - [x] SetNX returns true on first call, false on second
  - [x] Ping returns nil when Redis is available

  **Agent-Executed QA Scenarios**:

  ```
  Scenario: Redis connection succeeds
    Tool: Bash
    Preconditions: Redis running on localhost:6379
    Steps:
      1. docker run -d --name test-redis -p 6379:6379 redis:alpine
      2. go test ./internal/storage/... -run TestPing
      3. docker stop test-redis && docker rm test-redis
    Expected Result: Test passes, exit code 0
    Evidence: Test output captured

  Scenario: Set and Get roundtrip
    Tool: Bash
    Preconditions: Redis running, storage package implemented
    Steps:
      1. Run test: Set("test-key", "test-value", 60s)
      2. Run test: Get("test-key")
      3. Assert: returned value equals "test-value"
    Expected Result: Value retrieved matches set value
    Evidence: Test output captured

  Scenario: SetNX lock behavior
    Tool: Bash
    Preconditions: Redis running
    Steps:
      1. Run test: SetNX("lock-key", "1", 60s) → expect true
      2. Run test: SetNX("lock-key", "1", 60s) → expect false
      3. Run test: Delete("lock-key")
      4. Run test: SetNX("lock-key", "1", 60s) → expect true
    Expected Result: SetNX behaves as distributed lock
    Evidence: Test output captured
  ```

  **Commit**: YES
  - Message: `feat(storage): add Redis client with connection pooling`
  - Files: `internal/storage/redis.go`, `internal/storage/redis_test.go`
  - Pre-commit: `go test ./internal/storage/...`

---

- [x] 3. Aurion Service (Web Scraping)

  **What to do**:
  - Create `internal/services/aurion/aurion.go`
  - Implement HTTP client with cookie jar management
  - Implement login flow:
    1. `POST /login` with `username`, `password`, `j_idt28` (form field)
    2. Check for 302 redirect (success) vs error page (failure)
    3. Store session cookies
  - Implement navigation flow:
    1. `GET /` → extract `ViewState`, `idInit`
    2. `GET /faces/MainMenuPage.xhtml` → find sidebar menu ID for 'Mon Planning'
    3. `POST /faces/MainMenuPage.xhtml` → navigate to planning page
    4. `GET /faces/Planning.xhtml` → extract `ViewState`, `formIdPlanning`
    5. `POST /faces/Planning.xhtml` (AJAX) → get events JSON
  - Implement page parsing:
    - `parseViewState(html string) string` - extract `javax.faces.ViewState`
    - `parseIdInit(html string) string` - extract `form:idInit`
    - `parseSidebarMenuId(html string) string` - find menu ID for 'Mon Planning'
    - `parseFormIdPlanning(html string) string` - extract Schedule widget ID
    - `parsePlanningData(response string) []AurionEvent` - extract JSON events
  - Define AurionEvent struct:
    ```go
    type AurionEvent struct {
        ID        string `json:"id"`
        Title     string `json:"title"`
        Start     string `json:"start"`
        End       string `json:"end"`
        AllDay    bool   `json:"allDay"`
        Editable  bool   `json:"editable"`
        ClassName string `json:"className"`
    }
    ```
  - Default date range: 7 days ago to 60 days ahead

  **Must NOT do**:
  - Don't use headless browser (pure HTTP scraping)
  - Don't add retry logic beyond session refresh
  - Don't cache parsed page elements

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
  - **Skills**: `[]`
  - Reason: Complex scraping but well-documented in source

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 6)
  - **Blocks**: Tasks 5, 7
  - **Blocked By**: Task 1

  **References**:
  - `C:\Users\Flo\Desktop\isen-ical\src\services\aurion.service.ts` - Full scraping logic
  - `C:\Users\Flo\Desktop\isen-ical\src\services\page-parser.service.ts` - Regex patterns for parsing
  - `C:\Users\Flo\Desktop\isen-ical\src\services\session.service.ts` - HTTP client patterns

  **Acceptance Criteria**:
  - [x] Login function returns error for invalid credentials
  - [x] ViewState extraction regex works on sample HTML
  - [x] Navigation sequence completes without errors (mocked)
  - [x] Event parsing extracts correct fields from JSON

  **Agent-Executed QA Scenarios**:

  ```
  Scenario: ViewState parsing extracts correct value
    Tool: Bash
    Preconditions: Parser implemented
    Steps:
      1. Create test HTML with: <input name="javax.faces.ViewState" value="abc123" />
      2. Run parseViewState(html)
      3. Assert: result == "abc123"
    Expected Result: ViewState extracted correctly
    Evidence: Test output captured

  Scenario: Menu ID parsing finds Mon Planning
    Tool: Bash
    Preconditions: Parser implemented
    Steps:
      1. Create test HTML with sidebar menu items including 'Mon Planning'
      2. Run parseSidebarMenuId(html)
      3. Assert: result matches expected menu ID pattern
    Expected Result: Correct menu ID extracted
    Evidence: Test output captured

  Scenario: Event JSON parsing extracts all fields
    Tool: Bash
    Preconditions: Parser implemented
    Steps:
      1. Create test JSON: [{"id":"1","title":"Test","start":"2024-01-01","end":"2024-01-02","allDay":false,"editable":false,"className":"exam"}]
      2. Run parsePlanningData(json)
      3. Assert: len(events) == 1, events[0].Title == "Test"
    Expected Result: Events parsed correctly
    Evidence: Test output captured
  ```

  **Commit**: YES
  - Message: `feat(aurion): implement Aurion web scraping service`
  - Files: `internal/services/aurion/`, `internal/models/event.go`
  - Pre-commit: `go test ./internal/services/aurion/...`

---

### Wave 2: Core Services

- [x] 4. Token Service (Encryption & Management)

  **What to do**:
  - Create `internal/services/token/token.go`
  - Implement AES-GCM 256-bit encryption:
    - `Encrypt(plaintext []byte, key []byte) (encrypted []byte, iv []byte, error)`
    - `Decrypt(encrypted []byte, iv []byte, key []byte) (plaintext []byte, error)`
  - Implement token generation:
    - Generate UUID for token ID
    - Generate 32-byte random encryption key
    - Encrypt credentials (JSON: `{"email":"...","password":"..."}`)
    - Store in Redis: `token:{uuid}` → `{encrypted, iv, createdAt, email}`
  - Implement per-user token management:
    - Key: `user:{email}:tokens` → JSON array of `{token, createdAt}`
    - Enforce max tokens (default 3): remove oldest when limit exceeded
  - Implement token retrieval:
    - `GetTokenData(tokenId string) (*StoredToken, error)`
    - `DecryptCredentials(token *StoredToken, key []byte) (*Credentials, error)`
  - Base64URL encoding for keys (URL-safe)

  **Must NOT do**:
  - Don't implement token refresh/rotation
  - Don't add expiration to tokens (they're persistent)
  - Don't implement key derivation (use raw key)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
  - **Skills**: `[]`
  - Reason: Standard crypto operations in Go

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5, 6, 7)
  - **Blocks**: Task 8
  - **Blocked By**: Task 2

  **References**:
  - `C:\Users\Flo\Desktop\isen-ical\src\services\token.service.ts` - Full token logic
  - `C:\Users\Flo\Desktop\isen-ical\src\types\token.types.ts` - Type definitions

  **Acceptance Criteria**:
  - [x] Encrypt/Decrypt roundtrip preserves original data
  - [x] Token generation creates valid UUID
  - [x] 4th token for same user removes oldest token
  - [x] Invalid token ID returns appropriate error

  **Agent-Executed QA Scenarios**:

  ```
  Scenario: Encryption roundtrip preserves data
    Tool: Bash
    Preconditions: Token service implemented
    Steps:
      1. Generate 32-byte key
      2. Encrypt: "test@example.com:password123"
      3. Decrypt with same key
      4. Assert: decrypted == original
    Expected Result: Data preserved through encryption cycle
    Evidence: Test output captured

  Scenario: Max tokens enforcement removes oldest
    Tool: Bash
    Preconditions: Token service and Redis running
    Steps:
      1. Create token 1 for user@test.com
      2. Create token 2 for user@test.com
      3. Create token 3 for user@test.com
      4. Create token 4 for user@test.com
      5. List tokens for user@test.com
      6. Assert: only 3 tokens exist
      7. Assert: token 1 no longer exists
    Expected Result: Oldest token automatically removed
    Evidence: Test output captured

  Scenario: Invalid token returns error
    Tool: Bash
    Preconditions: Token service implemented
    Steps:
      1. Call GetTokenData("nonexistent-uuid")
      2. Assert: error returned (not found)
    Expected Result: Appropriate error for missing token
    Evidence: Test output captured
  ```

  **Commit**: YES
  - Message: `feat(token): add token encryption and management service`
  - Files: `internal/services/token/`, `internal/models/token.go`
  - Pre-commit: `go test ./internal/services/token/...`

---

- [x] 5. iCal Service (RFC 5545 Generation)

  **What to do**:
  - Create `internal/services/ical/ical.go`
  - Implement iCal generation following RFC 5545:
    - VCALENDAR header with PRODID, VERSION, CALSCALE
    - VTIMEZONE for Europe/Paris
    - VEVENT for each event
  - Implement text escaping: `\`, `,`, `;`, newlines
  - Implement line folding at 75 bytes
  - Implement timestamp formatting: `YYYYMMDDTHHMMSSZ` (UTC)
  - Implement title parsing (5-line format from Aurion):
    - Line 1: location
    - Line 2: additionalInfo
    - Line 3: subject
    - Line 4: courseType
    - Line 5: professor
  - Summary generation:
    - Use: subject || courseType || location || additionalInfo || "Événement sans titre"
    - Prefix: 🎓 for EXAM_SURV, 🏠 for AUTO_APPR
  - Description: additionalInfo + professor + courseType
  - Location: first line of title

  **Must NOT do**:
  - Don't use external iCal library (manual generation)
  - Don't add VALARM (reminders)
  - Don't modify event data beyond formatting

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
  - **Skills**: `[]`
  - Reason: Format specification is well-defined

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 4, 6, 7)
  - **Blocks**: Task 8
  - **Blocked By**: Task 3

  **References**:
  - `C:\Users\Flo\Desktop\isen-ical\src\services\ical.service.ts` - Full iCal generation logic
  - RFC 5545: https://tools.ietf.org/html/rfc5545

  **Acceptance Criteria**:
  - [x] Output starts with `BEGIN:VCALENDAR` and ends with `END:VCALENDAR`
  - [x] Each event has DTSTART, DTEND, SUMMARY, UID
  - [x] Lines fold at 75 bytes with space continuation
  - [x] Special characters properly escaped
  - [x] Emoji prefixes applied correctly

  **Agent-Executed QA Scenarios**:

  ```
  Scenario: Valid iCal structure generated
    Tool: Bash
    Preconditions: iCal service implemented
    Steps:
      1. Create test event: {Title: "Test Event", Start: "2024-01-15T10:00:00", End: "2024-01-15T12:00:00"}
      2. Generate iCal
      3. Assert: starts with "BEGIN:VCALENDAR"
      4. Assert: contains "BEGIN:VEVENT"
      5. Assert: contains "END:VCALENDAR"
    Expected Result: Valid iCal structure
    Evidence: iCal output captured

  Scenario: Emoji prefix for exam event
    Tool: Bash
    Preconditions: iCal service implemented
    Steps:
      1. Create test event with ClassName: "EXAM_SURV"
      2. Generate iCal
      3. Assert: SUMMARY starts with "🎓"
    Expected Result: Exam event has graduation cap emoji
    Evidence: iCal output captured

  Scenario: Line folding at 75 bytes
    Tool: Bash
    Preconditions: iCal service implemented
    Steps:
      1. Create event with very long title (100+ chars)
      2. Generate iCal
      3. Assert: no line exceeds 75 bytes
      4. Assert: continuation lines start with space
    Expected Result: Lines properly folded
    Evidence: iCal output captured
  ```

  **Commit**: YES
  - Message: `feat(ical): add RFC 5545 compliant iCal generator`
  - Files: `internal/services/ical/`
  - Pre-commit: `go test ./internal/services/ical/...`

---

- [x] 6. Auth Service (Basic Auth Parsing)

  **What to do**:
  - Create `internal/services/auth/auth.go`
  - Implement Basic Auth header parsing:
    - Check for "Basic " prefix
    - Base64 decode the credentials
    - Split on first ":" for username:password
  - Return structured result:
    ```go
    type AuthResult struct {
        Success     bool
        Credentials *Credentials
        FailReason  string // "missing_header", "invalid_format", "decode_error"
    }
    ```

  **Must NOT do**:
  - Don't validate credentials against Aurion (that's handler's job)
  - Don't add session management
  - Don't implement other auth methods

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`
  - Reason: Simple parsing logic, minimal code

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3) or Wave 2
  - **Blocks**: Tasks 8, 10
  - **Blocked By**: Task 1

  **References**:
  - `C:\Users\Flo\Desktop\isen-ical\src\services\auth.service.ts` - Full auth logic
  - `C:\Users\Flo\Desktop\isen-ical\src\types\auth.types.ts` - Type definitions

  **Acceptance Criteria**:
  - [x] Valid Basic Auth header parsed correctly
  - [x] Missing header returns "missing_header" reason
  - [x] Invalid format returns "invalid_format" reason
  - [x] Password with ":" character handled correctly

  **Agent-Executed QA Scenarios**:

  ```
  Scenario: Valid Basic Auth parsed
    Tool: Bash
    Preconditions: Auth service implemented
    Steps:
      1. Create header: "Basic dGVzdEBleGFtcGxlLmNvbTpwYXNzd29yZDEyMw=="
      2. Parse header
      3. Assert: Success == true
      4. Assert: Credentials.Email == "test@example.com"
      5. Assert: Credentials.Password == "password123"
    Expected Result: Credentials extracted correctly
    Evidence: Test output captured

  Scenario: Missing header handled
    Tool: Bash
    Preconditions: Auth service implemented
    Steps:
      1. Parse empty string
      2. Assert: Success == false
      3. Assert: FailReason == "missing_header"
    Expected Result: Appropriate error returned
    Evidence: Test output captured

  Scenario: Password with colon handled
    Tool: Bash
    Preconditions: Auth service implemented
    Steps:
      1. Create header for "user@test.com:pass:word:123"
      2. Parse header
      3. Assert: Credentials.Password == "pass:word:123"
    Expected Result: Only first colon used as delimiter
    Evidence: Test output captured
  ```

  **Commit**: YES
  - Message: `feat(auth): add HTTP Basic Auth parsing service`
  - Files: `internal/services/auth/`
  - Pre-commit: `go test ./internal/services/auth/...`

---

- [x] 7. Session Service (Aurion Session Management)

  **What to do**:
  - Create `internal/services/session/session.go`
  - Implement session storage in Redis:
    - Key: `session:{email}:{hash16}` (hash16 = first 16 chars of SHA256(password))
    - Value: JSON array of cookies
    - TTL: 1 hour (configurable)
  - Implement session operations:
    - `GetSession(email, passwordHash string) ([]http.Cookie, error)`
    - `SaveSession(email, passwordHash string, cookies []http.Cookie) error`
    - `DeleteSession(email, passwordHash string) error`
  - Implement cache operations:
    - Key: `events:user:{email}:{hash16}:{start}:{end}`
    - `GetCachedEvents(key string) ([]AurionEvent, error)`
    - `CacheEvents(key string, events []AurionEvent) error`
  - Implement distributed locking:
    - Key: `lock:user:{email}:{hash16}:{start}:{end}`
    - TTL: 60 seconds
    - `AcquireLock(key string) (bool, error)`
    - `ReleaseLock(key string) error`
  - Implement SHA256 hashing utility

  **Must NOT do**:
  - Don't implement complex lock retry logic
  - Don't add session encryption
  - Don't implement distributed cache invalidation

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
  - **Skills**: `[]`
  - Reason: Standard Redis operations with domain logic

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 4, 5, 6)
  - **Blocks**: Task 8
  - **Blocked By**: Tasks 2, 3

  **References**:
  - `C:\Users\Flo\Desktop\isen-ical\src\services\session.service.ts` - Session storage patterns
  - `C:\Users\Flo\Desktop\isen-ical\src\services\aurion.service.ts` - Lock/cache key patterns
  - `C:\Users\Flo\Desktop\isen-ical\src\utils\crypto.util.ts` - SHA256 hashing

  **Acceptance Criteria**:
  - [x] Sessions stored and retrieved correctly
  - [x] Session TTL applied (expires after 1 hour)
  - [x] Cached events retrieved correctly
  - [x] Lock acquisition returns true first time, false second time
  - [x] Lock release allows re-acquisition

  **Agent-Executed QA Scenarios**:

  ```
  Scenario: Session roundtrip works
    Tool: Bash
    Preconditions: Session service and Redis running
    Steps:
      1. Save session with test cookies
      2. Get session with same key
      3. Assert: cookies match
    Expected Result: Session data preserved
    Evidence: Test output captured

  Scenario: Distributed lock prevents concurrent access
    Tool: Bash
    Preconditions: Session service and Redis running
    Steps:
      1. AcquireLock("test-lock") → expect true
      2. AcquireLock("test-lock") → expect false
      3. ReleaseLock("test-lock")
      4. AcquireLock("test-lock") → expect true
    Expected Result: Lock provides mutual exclusion
    Evidence: Test output captured
  ```

  **Commit**: YES
  - Message: `feat(session): add session and cache management service`
  - Files: `internal/services/session/`
  - Pre-commit: `go test ./internal/services/session/...`

---

### Wave 3: HTTP Layer

- [x] 8. HTTP Handlers

  **What to do**:
  - Create `internal/handlers/` with:
    - `health.go` - GET /health
    - `calendar.go` - GET / (iCal), GET /calendar/{token}
    - `token.go` - POST /api/generate-token
    - `pages.go` - GET / (HTML), GET /privacy
  - Implement handler logic:
    - `/health`: Return `{"status":"healthy","redis":"ok"}`
    - `/` (Accept: text/html): Render homepage template
    - `/` (other Accept): Parse Basic Auth → fetch events → return iCal
    - `/api/generate-token`: Validate credentials → generate token → return URL
    - `/calendar/{token}`: Validate token → decrypt → fetch events → return iCal
    - `/privacy`: Render privacy template
  - Error responses (match original):
    - 400: Missing required parameters
    - 401: Missing auth header
    - 403: Invalid credentials
    - 404: Token not found
    - 500: Server error

  **Must NOT do**:
  - Don't add new endpoints
  - Don't change response formats
  - Don't add request logging middleware here (separate task)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
  - **Skills**: `[]`
  - Reason: Standard HTTP handlers with business logic integration

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 9, 10)
  - **Blocks**: Task 11
  - **Blocked By**: Tasks 4, 5, 6, 7

  **References**:
  - `C:\Users\Flo\Desktop\isen-ical\src\index.ts` - All route handlers and logic

  **Acceptance Criteria**:
  - [x] /health returns 200 with JSON body
  - [x] / with HTML accept returns HTML
  - [x] / with other accept requires Basic Auth
  - [x] /api/generate-token validates then generates
  - [x] /calendar/{token} decrypts and returns iCal
  - [x] All error codes match specification

  **Agent-Executed QA Scenarios**:

  ```
  Scenario: Health endpoint returns healthy status
    Tool: Bash (curl)
    Preconditions: Server running on localhost:8080
    Steps:
      1. curl -s http://localhost:8080/health
      2. Parse JSON response
      3. Assert: status == "healthy"
    Expected Result: Health check passes
    Evidence: Response body captured

  Scenario: Homepage returns HTML for browser
    Tool: Bash (curl)
    Preconditions: Server running
    Steps:
      1. curl -s -H "Accept: text/html" http://localhost:8080/
      2. Assert: response contains "<html"
      3. Assert: response contains "ISEN Calendar"
    Expected Result: HTML page rendered
    Evidence: Response body captured

  Scenario: Root without auth returns 401
    Tool: Bash (curl)
    Preconditions: Server running
    Steps:
      1. curl -s -w "%{http_code}" http://localhost:8080/ -H "Accept: text/calendar"
      2. Assert: HTTP status == 401
    Expected Result: Authentication required
    Evidence: HTTP status captured

  Scenario: Token generation flow
    Tool: Bash (curl)
    Preconditions: Server running
    Steps:
      1. POST /api/generate-token with valid credentials
      2. Assert: response contains "token"
      3. Assert: response contains "key"
      4. Assert: response contains "url"
    Expected Result: Token generated successfully
    Evidence: Response body captured
  ```

  **Commit**: YES
  - Message: `feat(handlers): add HTTP route handlers`
  - Files: `internal/handlers/`
  - Pre-commit: `go test ./internal/handlers/...`

---

- [x] 9. HTML Templates

  **What to do**:
  - Create `templates/homepage.html`:
    - Title: "ISEN Calendar"
    - Form for token generation (email, password, submit)
    - Instructions for Basic Auth usage
    - Link to privacy policy
  - Create `templates/privacy.html`:
    - Privacy policy content (port from original)
    - Data handling explanation
    - Contact information
  - Use Go's `html/template` package
  - Implement template loading in handler

  **Must NOT do**:
  - Don't add JavaScript frameworks
  - Don't add CSS frameworks (inline styles only)
  - Don't change content significantly from original

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`
  - Reason: Simple HTML port from existing templates

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 8, 10)
  - **Blocks**: Task 11
  - **Blocked By**: None (can start anytime)

  **References**:
  - `C:\Users\Flo\Desktop\isen-ical\src\templates\homepage.template.ts` - Homepage HTML
  - `C:\Users\Flo\Desktop\isen-ical\src\templates\privacy.template.ts` - Privacy HTML

  **Acceptance Criteria**:
  - [x] Homepage template renders without errors
  - [x] Privacy template renders without errors
  - [x] Form posts to correct endpoint
  - [x] All text content matches original

  **Agent-Executed QA Scenarios**:

  ```
  Scenario: Homepage template parses
    Tool: Bash
    Preconditions: Templates created
    Steps:
      1. Run test that loads homepage.html
      2. Execute template with empty data
      3. Assert: no parse errors
    Expected Result: Template valid
    Evidence: Test output captured

  Scenario: Privacy page content matches
    Tool: Bash (curl)
    Preconditions: Server running with templates
    Steps:
      1. curl -s http://localhost:8080/privacy
      2. Assert: contains "Privacy"
      3. Assert: contains expected sections
    Expected Result: Privacy content rendered
    Evidence: Response captured
  ```

  **Commit**: YES
  - Message: `feat(templates): add HTML templates for homepage and privacy`
  - Files: `templates/`
  - Pre-commit: `go build ./...`

---

- [x] 10. Middleware (Logging & Recovery)

  **What to do**:
  - Create `internal/middleware/logging.go`:
    - Log all requests with zerolog
    - Include: method, path, status, duration, request ID
    - JSON format for structured logging
  - Create `internal/middleware/recovery.go`:
    - Recover from panics
    - Log error with stack trace
    - Return 500 response
  - Create `internal/middleware/requestid.go`:
    - Generate UUID for each request
    - Add to context and response header

  **Must NOT do**:
  - Don't add rate limiting
  - Don't add authentication middleware (handlers do this)
  - Don't add CORS (not needed for iCal)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`
  - Reason: Standard middleware patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 8, 9)
  - **Blocks**: Task 11
  - **Blocked By**: Task 6 (uses auth types)

  **References**:
  - Gin middleware documentation
  - zerolog documentation

  **Acceptance Criteria**:
  - [x] All requests logged in JSON format
  - [x] Panics recovered and logged
  - [x] Request ID in response headers

  **Agent-Executed QA Scenarios**:

  ```
  Scenario: Request logging in JSON format
    Tool: Bash
    Preconditions: Server running with middleware
    Steps:
      1. Make request to /health
      2. Check server logs (docker logs)
      3. Parse log line as JSON
      4. Assert: contains "method", "path", "status"
    Expected Result: Structured JSON logs
    Evidence: Log output captured

  Scenario: Panic recovery returns 500
    Tool: Bash
    Preconditions: Test handler that panics
    Steps:
      1. Make request to panic-test endpoint
      2. Assert: HTTP status == 500
      3. Assert: server still running
    Expected Result: Graceful panic recovery
    Evidence: Response and logs captured
  ```

  **Commit**: YES
  - Message: `feat(middleware): add logging, recovery, and request ID`
  - Files: `internal/middleware/`
  - Pre-commit: `go test ./internal/middleware/...`

---

### Wave 4: Integration

- [x] 11. Main Entry Point & Routing

  **What to do**:
  - Create `cmd/server/main.go`:
    - Load configuration from environment
    - Initialize Redis connection
    - Initialize all services
    - Setup Gin router with middleware
    - Register all routes
    - Start HTTP server
  - Route registration:
    - `GET /health` → health handler
    - `GET /` → conditional (HTML/iCal based on Accept)
    - `POST /api/generate-token` → token handler
    - `GET /calendar/:token` → calendar handler
    - `GET /privacy` → privacy handler
  - Graceful shutdown on SIGINT/SIGTERM

  **Must NOT do**:
  - Don't add admin routes
  - Don't add API versioning
  - Don't add WebSocket support

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
  - **Skills**: `[]`
  - Reason: Integration of all components

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4 (sequential)
  - **Blocks**: Task 12
  - **Blocked By**: Tasks 8, 9, 10

  **References**:
  - `C:\Users\Flo\Desktop\isen-ical\src\index.ts` - Route definitions
  - Gin documentation for routing

  **Acceptance Criteria**:
  - [x] Server starts without errors
  - [x] All routes accessible
  - [x] Graceful shutdown works
  - [x] Config loaded from environment

  **Agent-Executed QA Scenarios**:

  ```
  Scenario: Server starts and responds
    Tool: Bash
    Preconditions: Redis running
    Steps:
      1. go run ./cmd/server
      2. Wait 2 seconds
      3. curl http://localhost:8080/health
      4. Assert: response received
    Expected Result: Server operational
    Evidence: Health response captured

  Scenario: All routes registered
    Tool: Bash
    Preconditions: Server running
    Steps:
      1. curl -s http://localhost:8080/health → 200
      2. curl -s http://localhost:8080/ → 200 or 401
      3. curl -s http://localhost:8080/privacy → 200
      4. curl -s http://localhost:8080/calendar/test → 404 (token not found)
    Expected Result: All routes respond
    Evidence: Status codes captured
  ```

  **Commit**: YES
  - Message: `feat(server): add main entry point with routing`
  - Files: `cmd/server/main.go`
  - Pre-commit: `go build ./cmd/server`

---

- [x] 12. Docker Setup

  **What to do**:
  - Create `Dockerfile`:
    - Multi-stage build (builder + runtime)
    - Builder: golang:1.22-alpine, build binary
    - Runtime: alpine:3.19, copy binary, minimal image
    - Expose port 8080
    - Health check command
  - Create `docker-compose.yml`:
    - Service: `app` (Go application)
      - Build from Dockerfile
      - Ports: 8080:8080
      - Environment: from .env file
      - Depends on: redis
      - Restart: unless-stopped
    - Service: `redis`
      - Image: redis:7-alpine
      - Ports: 6379:6379 (optional, for debugging)
      - Volume: redis-data
      - Restart: unless-stopped
    - Network: internal bridge
  - Create `.env.example`:
    - All configuration variables with examples
  - Create `.dockerignore`:
    - Exclude: .git, .env, *.md, tests

  **Must NOT do**:
  - Don't add nginx/traefik (SSL via Cloudflare)
  - Don't add Kubernetes configs
  - Don't add Redis Sentinel/Cluster

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
  - **Skills**: `[]`
  - Reason: Standard Docker patterns

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4 (after Task 11)
  - **Blocks**: Task 13
  - **Blocked By**: Task 11

  **References**:
  - Docker multi-stage build documentation
  - docker-compose v3 documentation

  **Acceptance Criteria**:
  - [x] `docker build .` succeeds
  - [x] `docker compose up` starts both services
  - [x] Health check passes after startup
  - [x] App connects to Redis successfully

  **Agent-Executed QA Scenarios**:

  ```
  Scenario: Docker image builds
    Tool: Bash
    Preconditions: Dockerfile created
    Steps:
      1. docker build -t isen-ical-go .
      2. Assert: exit code 0
      3. docker images | grep isen-ical-go
      4. Assert: image exists
    Expected Result: Image built successfully
    Evidence: Build output captured

  Scenario: Docker Compose starts stack
    Tool: Bash
    Preconditions: docker-compose.yml created
    Steps:
      1. docker compose up -d
      2. Wait 10 seconds
      3. docker compose ps
      4. Assert: both services "running"
      5. curl http://localhost:8080/health
      6. Assert: healthy response
      7. docker compose down
    Expected Result: Full stack operational
    Evidence: Container status and health check captured

  Scenario: Environment variables passed correctly
    Tool: Bash
    Preconditions: Docker Compose running
    Steps:
      1. Create .env with PORT=9090
      2. docker compose up -d
      3. curl http://localhost:9090/health
      4. Assert: response received
    Expected Result: Custom port used
    Evidence: Response captured
  ```

  **Commit**: YES
  - Message: `feat(docker): add Dockerfile and docker-compose setup`
  - Files: `Dockerfile`, `docker-compose.yml`, `.env.example`, `.dockerignore`
  - Pre-commit: `docker build .`

---

### Wave 5: Finalization

- [x] 13. Integration Tests & Documentation

  **What to do**:
  - Create integration tests in `tests/integration/`:
    - Test full token generation flow
    - Test Basic Auth calendar retrieval
    - Test token-based calendar retrieval
    - Test error scenarios (invalid creds, missing token)
  - Create `README.md`:
    - Project overview
    - Quick start with Docker Compose
    - Configuration reference
    - API documentation (all endpoints)
    - Development setup
  - Create `.gitignore`:
    - Exclude: .env, *.exe, vendor/, tmp/

  **Must NOT do**:
  - Don't test against real Aurion (mock it)
  - Don't add complex CI/CD configs
  - Don't add additional documentation files

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: `[]`
  - Reason: Integration testing requires careful setup

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 5 (final)
  - **Blocks**: None (final task)
  - **Blocked By**: Task 12

  **References**:
  - `C:\Users\Flo\Desktop\isen-ical\README.md` - Original README structure

  **Acceptance Criteria**:
  - [x] All integration tests pass
  - [x] README contains all required sections
  - [x] README examples are accurate
  - [x] .gitignore covers common exclusions

  **Agent-Executed QA Scenarios**:

  ```
  Scenario: Integration tests pass
    Tool: Bash
    Preconditions: Full stack running
    Steps:
      1. docker compose up -d
      2. go test ./tests/integration/... -v
      3. Assert: all tests pass
      4. docker compose down
    Expected Result: Integration suite green
    Evidence: Test output captured

  Scenario: README accurate
    Tool: Bash
    Preconditions: README created
    Steps:
      1. Read Quick Start section
      2. Execute commands exactly as written
      3. Verify they work
    Expected Result: Documentation matches reality
    Evidence: Command outputs captured
  ```

  **Commit**: YES
  - Message: `feat(docs): add integration tests and documentation`
  - Files: `tests/`, `README.md`, `.gitignore`
  - Pre-commit: `go test ./...`

---

## Commit Strategy

| After Task | Message | Files | Verification |
|------------|---------|-------|--------------|
| 1 | `feat(core): initialize project structure and configuration` | go.mod, cmd/, internal/config/ | `go build ./...` |
| 2 | `feat(storage): add Redis client with connection pooling` | internal/storage/ | `go test ./internal/storage/...` |
| 3 | `feat(aurion): implement Aurion web scraping service` | internal/services/aurion/, internal/models/event.go | `go test ./internal/services/aurion/...` |
| 4 | `feat(token): add token encryption and management service` | internal/services/token/, internal/models/token.go | `go test ./internal/services/token/...` |
| 5 | `feat(ical): add RFC 5545 compliant iCal generator` | internal/services/ical/ | `go test ./internal/services/ical/...` |
| 6 | `feat(auth): add HTTP Basic Auth parsing service` | internal/services/auth/ | `go test ./internal/services/auth/...` |
| 7 | `feat(session): add session and cache management service` | internal/services/session/ | `go test ./internal/services/session/...` |
| 8 | `feat(handlers): add HTTP route handlers` | internal/handlers/ | `go test ./internal/handlers/...` |
| 9 | `feat(templates): add HTML templates for homepage and privacy` | templates/ | `go build ./...` |
| 10 | `feat(middleware): add logging, recovery, and request ID` | internal/middleware/ | `go test ./internal/middleware/...` |
| 11 | `feat(server): add main entry point with routing` | cmd/server/main.go | `go build ./cmd/server` |
| 12 | `feat(docker): add Dockerfile and docker-compose setup` | Dockerfile, docker-compose.yml, .env.example, .dockerignore | `docker build .` |
| 13 | `feat(docs): add integration tests and documentation` | tests/, README.md, .gitignore | `go test ./...` |

---

## Success Criteria

### Verification Commands
```bash
# Build verification
go build ./...  # Expected: no errors

# Unit tests
go test ./...  # Expected: all pass

# Docker build
docker build -t isen-ical-go .  # Expected: image created

# Docker Compose
docker compose up -d  # Expected: both services running
curl http://localhost:8080/health  # Expected: {"status":"healthy"}

# Full flow test
curl -u "test@example.com:password" http://localhost:8080/ -H "Accept: text/calendar"
# Expected: BEGIN:VCALENDAR... (if valid credentials)

# Token generation
curl -X POST http://localhost:8080/api/generate-token \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}'
# Expected: {"token":"...","key":"...","url":"..."}
```

### Final Checklist
- [x] All "Must Have" present ✅
- [x] All "Must NOT Have" absent ✅
- [x] All tests pass ✅
- [x] Docker Compose deploys successfully ✅
- [x] All 5 routes functional ✅
- [x] iCal output RFC 5545 compliant ✅
- [x] Structured JSON logging active ✅
- [x] README documentation complete ✅
