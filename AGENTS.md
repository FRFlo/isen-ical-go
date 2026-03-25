# PROJECT KNOWLEDGE BASE

**Generated:** 2026-03-25 15:12:48 GMT
**Commit:** 721986d
**Branch:** develop

## OVERVIEW
Go HTTP service converting Aurion planning data to iCal; Gin + Valkey; tokenized calendar access; Worker-compatibility endpoints preserved.

## STRUCTURE
```text
./
├── cmd/server/              # Single runtime entrypoint + CLI/env wiring
├── internal/config/         # Runtime config struct consumed at startup
├── internal/handlers/       # HTTP routes + embedded gohtml templates
├── internal/middleware/     # Request ID/trace headers, logger, recovery, CORS
├── internal/models/         # Aurion/token DTOs
├── internal/services/       # Domain services (aurion, token, session, ical, auth, template)
├── internal/storage/        # Valkey client + key naming + lock primitives
├── tests/                   # Integration + worker parity tests + fixtures
└── .github/workflows/       # CI and Docker image publish chain
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Runtime boot flow | `cmd/server/main.go` | `urfave/cli/v3` flags + env vars -> `config.Config` |
| Route surface | `internal/handlers/handlers.go` | `RegisterRoutes` defines public API and compatibility routes |
| Trace/log middleware | `internal/middleware/middleware.go` | Worker-style tracing headers + status-aware logging |
| Valkey integration | `internal/storage/valkey.go` | URL parsing, pooling, key schema, lock support |
| Aurion HTTP/session flow | `internal/services/aurion/aurion.go` | Login/init/planning navigation and parsing |
| Session/cache behavior | `internal/services/session/session.go` | Password hash16, cache TTL, lock acquire/release |
| Calendar generation rules | `internal/services/ical/ical.go` | RFC5545 escaping + line folding |
| End-to-end behavior | `tests/integration_test.go` | mock Aurion + miniredis + Gin router wiring |

## CODE MAP
| Symbol | Type | Location | Refs | Role |
|--------|------|----------|------|------|
| `main` | function | `cmd/server/main.go` | 1 | Application entrypoint |
| `handlers.New` | function | `internal/handlers/handlers.go` | 2 | Handler graph assembly |
| `(*Handlers).RegisterRoutes` | method | `internal/handlers/handlers.go` | 1 | Route registration hub |
| `storage.NewValkeyClient` | function | `internal/storage/valkey.go` | 5 | Valkey bootstrap used by app/tests |
| `middleware.SetupMiddleware` | function | `internal/middleware/middleware.go` | 2 | Standard middleware stack |
| `aurion.NewClient` | function | `internal/services/aurion/aurion.go` | 1 | Aurion service client constructor |

## CONVENTIONS
- Config source-of-truth is CLI flags (`urfave/cli/v3`) backed by env var sources in `main.go`; no separate config loader.
- `internal/*` package boundaries are meaningful: orchestration in handlers, business logic in services, infra in storage.
- Integration tests include miniredis fallback details in `tests/AGENTS.md`.
- CI is intentionally minimal: `.github/workflows/ci.yml` runs `go test ./...` only.
- Docker publish is chained from CI success via `workflow_run` in `.github/workflows/docker.yml`.
- Comments are mixed FR/EN in core files; keep naming and API fields English.

## ANTI-PATTERNS (THIS PROJECT)
- Never compare admin token with non-constant-time operators (`internal/middleware/admin_auth.go`).
- Never emit unescaped or unfolded iCal text; follow RFC5545 escaping/folding logic (`internal/services/ical/ical.go`).
- Telemetry validation contract is scoped in `internal/handlers/AGENTS.md`.
- Never treat `/health` as canonical endpoint; `/api/health` is active route, old `/health` expected 404 in tests.
- Never store encryption keys server-side (documented security model in `README.md`).
- Treat `/calendar/:token?key=...` as secret material: never log raw query values without redaction (`internal/middleware/middleware.go`).

## UNIQUE STYLES
- Compatibility endpoints return `204 No Content` for legacy probe paths.
- Trace headers middleware normalizes request correlation (`x-request-id`, `x-trace-id`, `traceparent`).
- Valkey key schema is explicit and namespaced (`session:`, `events:user:`, `lock:user:`, `token:`).

## COMMANDS
```bash
go run cmd/server/main.go
go test ./...
go test ./tests/...
go test -cover ./...
go build -o isen-ical cmd/server/main.go
docker-compose up -d
docker build -t isen-ical .
```

## NOTES
- README endpoint docs track current routes (`/api/health` active; `/health` expected 404 in tests).
- `internal/handlers` embeds templates via `//go:embed`; keep template names stable.
- One root AGENTS plus scoped child AGENTS are authoritative; avoid duplicating full project context in child files.
