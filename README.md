# MissionLedger

MissionLedger is a governed agent execution substrate for AI workflows that need deterministic policy, explicit approvals, budget enforcement, degraded-state truth, and exportable proofpacks.

Current truth:
- Local-first scaffold exists.
- Go API exists and passes tests/build/smoke locally.
- Policy engine is deterministic and intentionally simple: allow / escalate / deny.
- Persistence is in-memory by default and PostgreSQL-backed when `DATABASE_URL` is set.
- A read-only Next.js operator console now exists and renders real API data.

This repo is for the wedge:
- governed coding agents
- governed ops agents
- platform/security teams who need proof-grade execution records instead of vague logs

## Implemented MVP surface

API endpoints:
- `GET /healthz`
- `GET /v1/missions?limit=20`
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
- operator list view backed by live API data
- automatic Postgres schema bootstrapping when database mode is enabled

## Quickstart

Run the API in memory mode:

```bash
go run ./cmd/api
```

Run the API with PostgreSQL:

```bash
export DATABASE_URL=postgres://user:pass@127.0.0.1:5432/missionledger?sslmode=disable
go run ./cmd/api
```

Run the operator console:

```bash
cd web
MISSIONLEDGER_API_BASE_URL=http://127.0.0.1:8080 npm run dev
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
9. Operator console shows the mission, state, budget burn, and proof timeline.

## Repo map

- `cmd/api` — HTTP API entrypoint
- `cmd/seed` — demo payload generator
- `internal/mission` — mission state, repository abstraction, in-memory store, and Postgres store
- `internal/policy` — deterministic tool policy
- `internal/degraded` — verification/degraded truth states
- `docs/` — positioning, architecture, demo, personas
- `migrations/` — SQL schema for Postgres mode
- `scripts/` — verify and smoke workflows
- `web/` — Next.js operator console
- `examples/` — example governed-agent wedges
- `.hermes/` — repo-local operator memory and verification docs

## Known gaps

- No auth yet
- No tenancy enforcement beyond stored fields
- No real tool mediation process yet; current API simulates governed tool calls
- Operator console is read-only and currently focuses on the newest mission detail
- No deployment/runtime target yet

## Next milestone

Keep the API honest while adding tenant enforcement, a real tool-gateway abstraction, approval actions from the operator console, and proofpack export flows that can leave the local dev box without losing deterministic truth.
