# TESTS KNOWLEDGE BASE

## OVERVIEW
Integration and parity coverage for end-to-end behavior (Gin routes, Valkey cache, Aurion mock flows, worker compatibility).

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Full API lifecycle | `integration_test.go` | token generation → calendar fetch → status assertions |
| Legacy contract parity | `worker_parity_test.go` | behavioral alignment with previous Worker implementation |
| Sample planning payload | `fixtures/planning.sample.json` | parity fixture for deterministic calendar checks |

## CONVENTIONS (LOCAL)
- Use `httptest.NewServer` mock Aurion endpoints for deterministic integration tests.
- If `VALKEY_URL` missing, spin `miniredis` fallback; skip only when infrastructure bootstrap fails.
- Keep assertions explicit on status codes and content types for contract stability.
- Prefer subtests (`t.Run`) for payload variants and edge-case matrices.

## ANTI-PATTERNS (LOCAL)
- Do not couple tests to wall-clock-sensitive timestamps without normalization.
- Do not remove worker parity tests when adjusting handlers/services.

## EXECUTION
```bash
go test ./tests/...
go test ./... -run "EndToEnd|Concurrent|Parity"
```
