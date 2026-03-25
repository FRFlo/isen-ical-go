# STORAGE KNOWLEDGE BASE

## OVERVIEW
Valkey integration boundary: connection bootstrap, timeout/pool tuning, namespaced key schema, lock primitive.

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Connection bootstrap | `valkey.go` `NewValkeyClient` | Parse URL + ping + pool/timeouts |
| Key naming | `valkey.go` key helpers | `session:`, `events:user:`, `lock:user:`, `token:`, `user:{email}:tokens` |
| Lock primitive | `valkey.go` `SetNX` | used by session service lock flow |
| URL decomposition | `valkey.go` `ParseValkeyURL` | host/password/db extraction helper |

## CONVENTIONS (LOCAL)
- Keep all cache/session/lock key formats centralized in key helper functions.
- Keep explicit context timeouts on every network call.
- Validate connectivity at startup with `Ping` before returning client.

## ANTI-PATTERNS (LOCAL)
- Do not construct Valkey keys ad hoc in callers; use helper constructors.
- Do not remove timeout guards on Get/Set/Delete/SetNX.
- Do not collapse lock and data key namespaces.

## LOCAL CHECKS
```bash
go test ./internal/storage/...
go test ./internal/services/session/...
```
