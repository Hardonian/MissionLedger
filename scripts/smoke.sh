#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PORT="${PORT:-18080}"
LOG_FILE="${TMPDIR:-/tmp}/missionledger-api.log"

cleanup() {
  if [[ -n "${SERVER_PID:-}" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" >/dev/null 2>&1 || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

PORT="$PORT" go run ./cmd/api >"$LOG_FILE" 2>&1 &
SERVER_PID=$!

for _ in $(seq 1 40); do
  if curl -fsS "http://127.0.0.1:${PORT}/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done

curl -fsS "http://127.0.0.1:${PORT}/healthz" >/dev/null

CREATE_RESPONSE=$(curl -fsS -X POST "http://127.0.0.1:${PORT}/v1/missions"   -H 'Content-Type: application/json'   -d '{
    "tenant_id": "demo-tenant",
    "objective": "Investigate safely with bounded authority",
    "requested_tools": ["read_file", "terminal", "secret.read"],
    "budget_usd": 0.50,
    "created_by": "smoke-test"
  }')

MISSION_ID=$(python3 - <<'PY' "$CREATE_RESPONSE"
import json, sys
payload = json.loads(sys.argv[1])
print(payload["id"])
PY
)

SAFE_RESPONSE=$(curl -fsS -X POST "http://127.0.0.1:${PORT}/v1/missions/${MISSION_ID}/tool-calls"   -H 'Content-Type: application/json'   -d '{"tool_name":"read_file","actor_id":"agent-1","cost_usd":0.10,"metadata":{"path":"README.md"}}')
python3 - <<'PY' "$SAFE_RESPONSE"
import json, sys
payload = json.loads(sys.argv[1])
assert payload["result"]["decision"] == "allow", payload
PY

DENIED_RESPONSE=$(curl -fsS -X POST "http://127.0.0.1:${PORT}/v1/missions/${MISSION_ID}/tool-calls"   -H 'Content-Type: application/json'   -d '{"tool_name":"secret.read","actor_id":"agent-1","cost_usd":0.01,"metadata":{"target":"prod-db-password"}}')
python3 - <<'PY' "$DENIED_RESPONSE"
import json, sys
payload = json.loads(sys.argv[1])
assert payload["result"]["decision"] == "deny", payload
PY

ESCALATE_RESPONSE=$(curl -fsS -X POST "http://127.0.0.1:${PORT}/v1/missions/${MISSION_ID}/tool-calls"   -H 'Content-Type: application/json'   -d '{"tool_name":"terminal","actor_id":"agent-1","cost_usd":0.10,"metadata":{"command":"git status"}}')
python3 - <<'PY' "$ESCALATE_RESPONSE"
import json, sys
payload = json.loads(sys.argv[1])
assert payload["result"]["decision"] == "escalate", payload
assert payload["mission"]["state"] == "waiting_approval", payload
PY

APPROVE_RESPONSE=$(curl -fsS -X POST "http://127.0.0.1:${PORT}/v1/missions/${MISSION_ID}/approve"   -H 'Content-Type: application/json'   -d '{"approved_by":"human-approver"}')
python3 - <<'PY' "$APPROVE_RESPONSE"
import json, sys
payload = json.loads(sys.argv[1])
assert payload["state"] == "approved", payload
PY

APPROVED_RESPONSE=$(curl -fsS -X POST "http://127.0.0.1:${PORT}/v1/missions/${MISSION_ID}/tool-calls"   -H 'Content-Type: application/json'   -d '{"tool_name":"terminal","actor_id":"agent-1","cost_usd":0.20,"metadata":{"command":"git status"}}')
python3 - <<'PY' "$APPROVED_RESPONSE"
import json, sys
payload = json.loads(sys.argv[1])
assert payload["result"]["decision"] == "allow", payload
PY

BUDGET_RESPONSE=$(curl -fsS -X POST "http://127.0.0.1:${PORT}/v1/missions/${MISSION_ID}/tool-calls"   -H 'Content-Type: application/json'   -d '{"tool_name":"read_file","actor_id":"agent-1","cost_usd":0.30,"metadata":{"path":"go.mod"}}')
python3 - <<'PY' "$BUDGET_RESPONSE"
import json, sys
payload = json.loads(sys.argv[1])
assert payload["result"]["decision"] == "deny", payload
assert payload["mission"]["state"] == "degraded", payload
PY

PROOFPACK_RESPONSE=$(curl -fsS "http://127.0.0.1:${PORT}/v1/missions/${MISSION_ID}/proofpack")
python3 - <<'PY' "$PROOFPACK_RESPONSE"
import json, sys
payload = json.loads(sys.argv[1])
assert payload["mission"]["id"].startswith("mission-"), payload
assert payload["summary"]["event_count"] >= 6, payload
print(f"smoke ok: {payload['mission']['id']} events={payload['summary']['event_count']}")
PY
