# MissionLedger

MissionLedger is a governed agent execution substrate for AI workflows that need deterministic policy, explicit approvals, budget enforcement, degraded-state truth, and exportable proofpacks.

Current truth:
- Local-first scaffold exists.
- Go API exists and passes tests/build/smoke locally.
- Policy engine is deterministic and intentionally simple: allow / escalate / deny.
- Persistence is in-memory only right now.
- Web console is planned but not yet wired into a running app.

This repo is for the wedge:
- governed coding agents
- governed ops agents
- platform/security teams who need proof-grade execution records instead of vague logs

## Implemented MVP surface

API endpoints:
- `GET /healthz`
- `POST /v1/missions`
- `GET /v1/missions/{id}`
- `POST /v1/missions/{id}/approve`
- `POST /v1/missions/{id}/tool-calls`
- `GET /v1/missions/{id}/proofpack`

Implemented behaviors:
- mission creation with requested tool scope and budget
- deterministic tool decisioning
- escalation for risky tools until approved
- hard denial for secret access and messaging tools
- budget breach moves mission into degraded state
- proof events with payload hashes and verification state

## Quickstart

Run the API:

```bash
go run ./cmd/api
```

Run verification:

```bash
./scripts/verify.sh
```

Run smoke demo only:

```bash
./scripts/smoke.sh
```

## Demo story

1. Create a mission with `read_file` and `terminal` in scope.
2. Safe `read_file` tool use passes.
3. `secret.read` is denied.
4. `terminal` escalates until approved.
5. Approval is recorded.
6. Approved `terminal` use passes.
7. Budget overrun is denied and the mission enters degraded state.
8. Proofpack export shows the full chain.

## Repo map

- `cmd/api` — HTTP API entrypoint
- `cmd/seed` — demo payload generator
- `internal/mission` — mission state and in-memory store
- `internal/policy` — deterministic tool policy
- `internal/degraded` — verification/degraded truth states
- `docs/` — positioning, architecture, demo, personas
- `scripts/` — verify and smoke workflows
- `examples/` — example governed-agent wedges
- `.hermes/` — repo-local operator memory and verification docs

## Known gaps

- No database yet
- No auth yet
- No tenancy enforcement beyond data fields
- No real tool mediation process yet; current API simulates governed tool calls
- No running Next.js console yet
- No GitHub remote yet

## Next milestone

Replace in-memory storage with Postgres, add a real tool-gateway abstraction, and build a minimal operator console that visualizes missions, approvals, proof events, and degraded states.
