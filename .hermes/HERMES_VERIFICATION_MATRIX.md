# HERMES_VERIFICATION_MATRIX

| Surface | Command | Status | Notes |
|---|---|---|---|
| Unit tests | `go test ./...` | verified | passes locally at scaffold creation |
| Build | `go build ./cmd/api` | verified | API compiles locally |
| End-to-end demo | `./scripts/smoke.sh` | verified | exercises allow / deny / escalate / approve / budget degrade / proofpack |
| Web console | n/a | degraded | not implemented yet |
| Persistence | n/a | degraded | in-memory only |
