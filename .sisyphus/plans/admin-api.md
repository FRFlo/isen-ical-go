# Admin API with Bearer Token Authentication

## TL;DR

> **Quick Summary**: Add Bearer token authentication infrastructure for future admin routes. Move `/health` endpoint to `/api/health` (public). Create `AdminAuthMiddleware` ready for future protected routes.
> 
> **Deliverables**:
> - `ADMIN_API_TOKEN` env var and CLI flag in config
> - `AdminAuthMiddleware` function in middleware package
> - `/api/health` endpoint (public, moved from `/health`)
> - Unit tests for middleware and health endpoint
> 
> **Estimated Effort**: Quick
> **Parallel Execution**: YES - 3 waves
> **Critical Path**: Task 1 → Task 4 → Task 7

---

## Context

### Original Request
Create an API for the ical app with admin Bearer token authentication stored in env variables. Move the `/health` endpoint to `/api/health` to populate the new API structure.

### Interview Summary
**Key Discussions**:
- **Scope**: Minimal - move health endpoint and add auth middleware placeholder for future use
- **Auth Strategy**: Single static token in `ADMIN_API_TOKEN` env var, simple string comparison
- **Health endpoint**: Relocated to `/api/health` but remains PUBLIC (no auth required)
- **Protected routes**: Future `/api/*` routes will use Bearer middleware (not created yet)
- **Existing routes**: `/api/generate-token` and `/api/track` remain unchanged and public

### Metis Review
**Identified Gaps** (addressed):
- Token comparison security → Use `crypto/subtle.ConstantTimeCompare`
- RFC 6750 compliance → Add `WWW-Authenticate: Bearer` header on 401
- Bearer prefix case sensitivity → Make case-insensitive
- Empty token handling → Server startup behavior (default: allow, warn if empty)
- Edge cases (double-space, no token after Bearer) → Reject as malformed

---

## Work Objectives

### Core Objective
Add Bearer token authentication infrastructure and reorganize health endpoint under `/api/` prefix.

### Concrete Deliverables
- `internal/config/config.go`: Add `AdminAPIToken string` field
- `cmd/server/main.go`: Add `--admin-api-token` CLI flag with `ADMIN_API_TOKEN` env source
- `internal/middleware/admin_auth.go`: New file with `AdminAuthMiddleware` function
- `internal/middleware/admin_auth_test.go`: Unit tests for middleware
- `internal/handlers/handlers.go`: Move `/health` registration to `/api/health`
- `.env.example`: Document `ADMIN_API_TOKEN` variable

### Definition of Done
- [ ] `curl http://localhost:8080/health` returns 404
- [ ] `curl http://localhost:8080/api/health` returns `{"status":"healthy"}` (200)
- [ ] `go test ./internal/middleware/...` passes with admin auth tests
- [ ] `go test ./...` all tests pass

### Must Have
- Bearer token extracted from `Authorization: Bearer <token>` header
- Constant-time string comparison for security (`subtle.ConstantTimeCompare`)
- 401 response with JSON body `{"error": "..."}` for auth failures
- `WWW-Authenticate: Bearer` header on 401 responses
- Case-insensitive Bearer prefix matching
- Middleware exported and ready for future route groups

### Must NOT Have (Guardrails)
- ❌ Do NOT add any `/api/admin/*` routes (middleware only for now)
- ❌ Do NOT add rate limiting, audit logging, or token rotation
- ❌ Do NOT enforce token minimum length (document recommendation only)
- ❌ Do NOT add multiple token support
- ❌ Do NOT modify existing `/api/generate-token` or `/api/track` routes
- ❌ Do NOT add authentication to `/api/health` (must remain public)
- ❌ Do NOT fail server startup if token is empty (log warning instead)

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES (Go tests with testify)
- **Automated tests**: YES (tests-after)
- **Framework**: go test

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Middleware**: Use Bash (go test) — Run unit tests, assert pass
- **API endpoints**: Use Bash (curl) — Send requests, assert status + response

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately — independent config/structure):
├── Task 1: Add ADMIN_API_TOKEN to config [quick]
├── Task 2: Create AdminAuthMiddleware file structure [quick]
└── Task 3: Update .env.example [quick]

Wave 2 (After Wave 1 — implementation):
├── Task 4: Implement AdminAuthMiddleware (depends: 1, 2) [quick]
├── Task 5: Move /health to /api/health (depends: none) [quick]
└── Task 6: Write middleware unit tests (depends: 4) [quick]

Wave 3 (After Wave 2 — verification):
└── Task 7: Integration verification (depends: 4, 5, 6) [quick]

Critical Path: Task 1 → Task 4 → Task 7
Parallel Speedup: ~40% faster than sequential
Max Concurrent: 3 (Wave 1)
```

### Dependency Matrix

| Task | Depends On | Blocks |
|------|------------|--------|
| 1 | — | 4 |
| 2 | — | 4 |
| 3 | — | — |
| 4 | 1, 2 | 6, 7 |
| 5 | — | 7 |
| 6 | 4 | 7 |
| 7 | 4, 5, 6 | — |

### Agent Dispatch Summary

- **Wave 1**: **3** — T1 → `quick`, T2 → `quick`, T3 → `quick`
- **Wave 2**: **3** — T4 → `quick`, T5 → `quick`, T6 → `quick`
- **Wave 3**: **1** — T7 → `quick`
- **FINAL**: **4** — F1-F4 → verification agents

---

## TODOs

- [x] 1. Add ADMIN_API_TOKEN to config

  **What to do**:
  - Add `AdminAPIToken string` field to `internal/config/config.go` Config struct
  - Add `--admin-api-token` CLI flag in `cmd/server/main.go` with env source `ADMIN_API_TOKEN`
  - Log warning at startup if token is empty (but don't fail)

  **Must NOT do**:
  - Do NOT enforce minimum token length
  - Do NOT fail server startup if token is empty

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3)
  - **Blocks**: Task 4
  - **Blocked By**: None

  **References**:
  - `internal/config/config.go:7-15` — Existing Config struct pattern
  - `cmd/server/main.go:31-81` — Existing CLI flag definitions with env sources

  **Acceptance Criteria**:
  - [ ] `AdminAPIToken` field exists in Config struct
  - [ ] `--admin-api-token` flag works: `go run cmd/server/main.go --admin-api-token=test123`
  - [ ] `ADMIN_API_TOKEN=test123 go run cmd/server/main.go` loads token from env

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Config struct has AdminAPIToken field
    Tool: Bash (go build)
    Preconditions: Code changes applied
    Steps:
      1. Run `go build ./...` to verify compilation
      2. Run `go vet ./internal/config/...` to verify no issues
    Expected Result: Build succeeds, vet passes
    Evidence: .sisyphus/evidence/task-1-config-build.txt

  Scenario: CLI flag loads token
    Tool: Bash (grep + build verification)
    Preconditions: Code changes applied
    Steps:
      1. grep for "admin-api-token" in main.go
      2. grep for "ADMIN_API_TOKEN" in main.go
    Expected Result: Both patterns found
    Evidence: .sisyphus/evidence/task-1-cli-flag.txt
  ```

  **Commit**: YES
  - Message: `feat(config): add ADMIN_API_TOKEN configuration`
  - Files: `internal/config/config.go`, `cmd/server/main.go`

- [x] 2. Create AdminAuthMiddleware file structure

  **What to do**:
  - Create new file `internal/middleware/admin_auth.go`
  - Add package declaration and imports (gin, crypto/subtle, net/http, strings)
  - Add placeholder function signature: `func AdminAuthMiddleware(token string) gin.HandlerFunc`

  **Must NOT do**:
  - Do NOT implement full logic yet (Task 4 does that)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3)
  - **Blocks**: Task 4
  - **Blocked By**: None

  **References**:
  - `internal/middleware/middleware.go:1-24` — Existing middleware file structure and imports

  **Acceptance Criteria**:
  - [ ] File `internal/middleware/admin_auth.go` exists
  - [ ] Package declaration is `package middleware`
  - [ ] Function signature `AdminAuthMiddleware(token string) gin.HandlerFunc` exists

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: File compiles successfully
    Tool: Bash (go build)
    Preconditions: File created
    Steps:
      1. Run `go build ./internal/middleware/...`
    Expected Result: Build succeeds with no errors
    Evidence: .sisyphus/evidence/task-2-middleware-build.txt
  ```

  **Commit**: NO (groups with Task 4)

- [x] 3. Update .env.example with ADMIN_API_TOKEN

  **What to do**:
  - Add `ADMIN_API_TOKEN` variable to `.env.example`
  - Add documentation comment explaining purpose and format
  - Recommend 32+ character token for security

  **Must NOT do**:
  - Do NOT add actual token value (use placeholder)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2)
  - **Blocks**: None
  - **Blocked By**: None

  **References**:
  - `.env.example:1-34` — Existing env variable documentation style

  **Acceptance Criteria**:
  - [ ] `ADMIN_API_TOKEN` variable present in `.env.example`
  - [ ] Comment explains purpose (admin API authentication)
  - [ ] Comment mentions recommended length (32+ chars)

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Env example contains ADMIN_API_TOKEN
    Tool: Bash (grep)
    Preconditions: File edited
    Steps:
      1. Run `grep "ADMIN_API_TOKEN" .env.example`
    Expected Result: Line found with variable name
    Evidence: .sisyphus/evidence/task-3-env-example.txt
  ```

  **Commit**: YES
  - Message: `docs(env): document ADMIN_API_TOKEN in .env.example`
  - Files: `.env.example`

- [x] 4. Implement AdminAuthMiddleware

  **What to do**:
  - Implement full `AdminAuthMiddleware` function in `internal/middleware/admin_auth.go`
  - Extract token from `Authorization: Bearer <token>` header
  - Use case-insensitive comparison for "Bearer" prefix
  - Use `subtle.ConstantTimeCompare` for token comparison
  - Return 401 with JSON `{"error": "..."}` for auth failures
  - Add `WWW-Authenticate: Bearer` header on 401 responses
  - Call `c.Next()` on success

  **Must NOT do**:
  - Do NOT add rate limiting
  - Do NOT add audit logging beyond basic structure
  - Do NOT enforce token minimum length

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 2 (sequential after Wave 1)
  - **Blocks**: Task 6, Task 7
  - **Blocked By**: Task 1, Task 2

  **References**:
  - `internal/middleware/middleware.go` — Existing middleware patterns
  - `internal/handlers/handlers.go:164,178,217-224` — Existing JSON error response patterns (`gin.H{"error": "..."}`)

  **Acceptance Criteria**:
  - [ ] Missing Authorization header → 401 with `{"error":"Authorization header required"}`
  - [ ] Malformed header (not Bearer) → 401 with `{"error":"Invalid authorization format"}`
  - [ ] Wrong token → 401 with `{"error":"Invalid token"}`
  - [ ] Correct token → `c.Next()` called, no error
  - [ ] All 401 responses include `WWW-Authenticate: Bearer` header
  - [ ] Uses `subtle.ConstantTimeCompare` (grep confirms)

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Middleware compiles and uses constant-time comparison
    Tool: Bash (grep + build)
    Preconditions: Implementation complete
    Steps:
      1. Run `go build ./internal/middleware/...`
      2. Run `grep "subtle.ConstantTimeCompare" internal/middleware/admin_auth.go`
    Expected Result: Build succeeds, ConstantTimeCompare found
    Evidence: .sisyphus/evidence/task-4-middleware-impl.txt

  Scenario: Middleware exports WWW-Authenticate header
    Tool: Bash (grep)
    Preconditions: Implementation complete
    Steps:
      1. Run `grep "WWW-Authenticate" internal/middleware/admin_auth.go`
    Expected Result: Header string found
    Evidence: .sisyphus/evidence/task-4-www-auth-header.txt
  ```

  **Commit**: YES
  - Message: `feat(middleware): implement AdminAuthMiddleware`
  - Files: `internal/middleware/admin_auth.go`

- [x] 5. Move /health endpoint to /api/health

  **What to do**:
  - In `internal/handlers/handlers.go`, change `r.GET("/health", h.Health)` to `r.GET("/api/health", h.Health)`
  - Ensure no other changes to Health handler implementation

  **Must NOT do**:
  - Do NOT add authentication to /api/health
  - Do NOT change Health handler logic

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (parallel with Task 4, 6)
  - **Blocks**: Task 7
  - **Blocked By**: None

  **References**:
  - `internal/handlers/handlers.go:60` — Current `/health` route registration
  - `internal/handlers/handlers.go:360-378` — Health handler implementation (don't change)

  **Acceptance Criteria**:
  - [ ] `r.GET("/api/health", h.Health)` in RegisterRoutes
  - [ ] `r.GET("/health", ...)` no longer exists
  - [ ] Health handler unchanged

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Route registration changed
    Tool: Bash (grep)
    Preconditions: Edit applied
    Steps:
      1. Run `grep '"/api/health"' internal/handlers/handlers.go`
      2. Run `grep -c '"/health"' internal/handlers/handlers.go` (should be 0 or only in comments)
    Expected Result: /api/health found, /health route not found
    Evidence: .sisyphus/evidence/task-5-route-change.txt
  ```

  **Commit**: YES
  - Message: `refactor(routes): move /health to /api/health`
  - Files: `internal/handlers/handlers.go`

- [x] 6. Write AdminAuthMiddleware unit tests

  **What to do**:
  - Create `internal/middleware/admin_auth_test.go`
  - Test: missing Authorization header → 401
  - Test: malformed header (wrong scheme) → 401
  - Test: empty token after Bearer → 401
  - Test: wrong token → 401
  - Test: correct token → next handler called
  - Test: Bearer prefix is case-insensitive ("bearer", "BEARER", "Bearer")

  **Must NOT do**:
  - Do NOT add integration tests with real server

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 2 (after Task 4)
  - **Blocks**: Task 7
  - **Blocked By**: Task 4

  **References**:
  - `internal/middleware/middleware_test.go` — Existing middleware test patterns
  - `internal/handlers/handlers.go` — Test patterns for Gin handlers (if any)

  **Acceptance Criteria**:
  - [ ] Test file exists at `internal/middleware/admin_auth_test.go`
  - [ ] `go test ./internal/middleware/...` passes
  - [ ] At least 5 test cases covering auth scenarios

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: All middleware tests pass
    Tool: Bash (go test)
    Preconditions: Test file created
    Steps:
      1. Run `go test ./internal/middleware/... -v`
    Expected Result: All tests PASS, including admin_auth tests
    Evidence: .sisyphus/evidence/task-6-middleware-tests.txt

  Scenario: Test coverage includes auth scenarios
    Tool: Bash (grep)
    Preconditions: Test file created
    Steps:
      1. Run `grep -c "func Test" internal/middleware/admin_auth_test.go`
    Expected Result: At least 5 test functions
    Evidence: .sisyphus/evidence/task-6-test-count.txt
  ```

  **Commit**: YES
  - Message: `test(middleware): add AdminAuthMiddleware unit tests`
  - Files: `internal/middleware/admin_auth_test.go`

- [x] 7. Integration verification

  **What to do**:
  - Verify full project builds: `go build ./...`
  - Verify all tests pass: `go test ./...`
  - Verify vet passes: `go vet ./...`
  - Start server and test `/api/health` endpoint returns 200
  - Verify `/health` returns 404

  **Must NOT do**:
  - Do NOT skip any verification step

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 3 (after all implementation)
  - **Blocks**: None
  - **Blocked By**: Task 4, Task 5, Task 6

  **References**:
  - All previous task deliverables

  **Acceptance Criteria**:
  - [ ] `go build ./...` succeeds
  - [ ] `go test ./...` all pass
  - [ ] `go vet ./...` no issues
  - [ ] `/api/health` returns 200 with healthy status
  - [ ] `/health` returns 404

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Full project builds and tests pass
    Tool: Bash
    Preconditions: All previous tasks complete
    Steps:
      1. Run `go build ./...`
      2. Run `go test ./...`
      3. Run `go vet ./...`
    Expected Result: All commands succeed with exit code 0
    Evidence: .sisyphus/evidence/task-7-full-build.txt

  Scenario: Health endpoint moved correctly
    Tool: Bash (curl via background server)
    Preconditions: Server started
    Steps:
      1. Start server in background: `go run cmd/server/main.go &`
      2. Wait 2 seconds for startup
      3. Run `curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/health`
      4. Run `curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health`
      5. Kill background server
    Expected Result: /api/health returns 200, /health returns 404
    Evidence: .sisyphus/evidence/task-7-health-endpoints.txt
  ```

  **Commit**: NO (verification only)

---

## Final Verification Wave

- [x] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. Verify implementation of Must Have items, absence of Must NOT Have items. Check evidence files exist.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [x] F2. **Code Quality Review** — `unspecified-high`
  Run `go build ./...` + `go vet ./...` + `go test ./...`. Review changed files for: `any` casts, empty error handlers, console prints in prod.
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | VERDICT`

- [x] F3. **Real Manual QA** — `unspecified-high`
  Start server, test endpoints manually via curl. Test middleware with mock routes if possible.
  Output: `Scenarios [N/N pass] | VERDICT`

- [x] F4. **Scope Fidelity Check** — `deep`
  Verify each task did exactly what spec said. Flag scope creep or missing items.
  Output: `Tasks [N/N compliant] | VERDICT`

---

## Commit Strategy

| Task | Commit Message | Files |
|------|----------------|-------|
| 1 | `feat(config): add ADMIN_API_TOKEN configuration` | `internal/config/config.go`, `cmd/server/main.go` |
| 3 | `docs(env): document ADMIN_API_TOKEN in .env.example` | `.env.example` |
| 4 | `feat(middleware): implement AdminAuthMiddleware` | `internal/middleware/admin_auth.go` |
| 5 | `refactor(routes): move /health to /api/health` | `internal/handlers/handlers.go` |
| 6 | `test(middleware): add AdminAuthMiddleware unit tests` | `internal/middleware/admin_auth_test.go` |

---

## Success Criteria

### Verification Commands
```bash
# Health endpoint moved
curl -s http://localhost:8080/health          # Expected: 404
curl -s http://localhost:8080/api/health      # Expected: {"status":"healthy"}

# All tests pass
go test ./...                                  # Expected: PASS

# Middleware tests pass
go test ./internal/middleware/... -v          # Expected: PASS with admin_auth tests
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass
- [ ] Server starts without errors
- [ ] `/api/health` accessible without auth
