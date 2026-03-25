# SERVICES KNOWLEDGE BASE

## OVERVIEW
Business logic lives here: Aurion fetch/parsing, session/cache orchestration, token crypto, iCal generation.

## STRUCTURE
```text
internal/services/
├── aurion/    # Stateful HTTP + JSF parsing pipeline
├── auth/      # Basic auth parsing helpers
├── ical/      # RFC5545 event serialization
├── session/   # Session/caching + distributed lock operations
├── template/  # HTML template helper wrapper
└── token/     # Token encryption/decryption + lifecycle limits
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Aurion authentication flow | `aurion/aurion.go` | `Login` + `InitializeSession` + navigation sequence |
| Aurion planning extraction | `aurion/aurion.go` | `FetchPlanningData` + parser helpers |
| Session persistence | `session/session.go` | `GetSession` / `SaveSession` using Valkey keys |
| Locking strategy | `session/session.go` | `AcquireLock` uses SetNX + 60s TTL |
| Token lifecycle | `token/token.go` | max tokens per user + encryption/decryption path |
| Calendar rendering | `ical/ical.go` | escaping + line folding before output |

## CONVENTIONS (LOCAL)
- `aurion.Client` is stateful (`ViewState`/form IDs + cookie jar); avoid shared mutable use across concurrent flows.
- Keep orchestration stateful only where required (`aurion.Client` keeps view/form IDs).
- Error wrapping is contextual (`fmt.Errorf("...: %w", err)`) across services.
- Service tests are colocated (`*_test.go`) and table-driven where parsing is involved.
- `session.HashPassword` intentionally truncates SHA-256 to 16 hex chars for key partitioning.
- Token persistence uses TTL=0 for `token:*` and `user:{email}:tokens`; lifecycle is controlled by max-token pruning.

## ANTI-PATTERNS (LOCAL)
- Do not bypass `session` locking for concurrent planning fetches.
- Do not alter RFC5545 escaping/folding order in `ical` (backslash first; fold >75 octets).
- Do not change iCal SUMMARY emoji prefixes (`EXAM_SURV` -> 🎓, `AUTO_APPR` -> 🏠) without updating tests.
- Do not move Aurion parser regexes without updating parity tests (`aurion_test.go`, `worker_parity_test.go`).
- Do not introduce storage primitives in services; go through `internal/storage` client API.

## TEST HOTSPOTS
- `aurion/aurion_test.go`: JSF/HTML parser edge cases.
- `ical/ical_test.go`: fixture parity for date and output stability.
- `session/session_test.go`: lock/cache/session behavior with Valkey-backed tests.
