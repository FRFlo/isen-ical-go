# HANDLERS KNOWLEDGE BASE

## OVERVIEW
HTTP contract layer: route registration, request validation, compatibility responses, template rendering.

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Route topology | `handlers.go` `(*Handlers).RegisterRoutes` | Canonical endpoint map |
| Home behavior | `handlers.go` `Home`/`serveHomepage` | HTML vs Basic Auth calendar decision |
| Token issuance | `handlers.go` `GenerateToken` | request schema + response URL generation |
| Calendar retrieval | `handlers.go` `Calendar` | token/key validation + iCal response |
| Telemetry ingestion | `handlers.go` `Track` | strict JSON + `frontend_` event prefix |
| Compatibility probes | `handlers.go` `Favicon`/`ChromeDevtoolsProbe` | both return 204 |
| Template content | `homepage.gohtml`, `privacy.gohtml` | embedded via `//go:embed` |

## CONVENTIONS (LOCAL)
- Keep handler methods thin: validation + orchestration; core logic belongs in services.
- Keep API responses stable (status code and field names) to preserve worker parity tests.
- Telemetry endpoint accepts only `Content-Type: application/json` and prefixed event names.
- Health endpoint contract is defined at root (`/api/health` active; `/health` expected 404).

## ANTI-PATTERNS (LOCAL)
- Do not change compatibility routes to 200/404; legacy clients expect 204.
- Do not relax `frontend_` event validation in `Track`.
- Do not bypass decryption-key requirements for `/calendar/:token`.
- Do not rename embedded template files without updating go:embed references.

## QUICK VALIDATION
```bash
go test ./internal/handlers/...
go test ./tests/... -run "Health|Home|GenerateToken|Calendar|Track"
```
