#!/usr/bin/env bash
set -euo pipefail

base_url="${MISSIONLEDGER_API_BASE_URL:-http://127.0.0.1:${PORT:-8080}}"
if ! response="$(curl -fsS --max-time 3 "${base_url}/healthz")"; then
  printf '{"schema_version":"1.0","product":"MissionLedger","status":"unreachable","base_url":"%s"}\n' "$base_url"
  exit 1
fi
python3 -c 'import json,sys; payload=json.load(sys.stdin); payload.update(schema_version="1.0", product="MissionLedger"); print(json.dumps(payload, sort_keys=True))' <<<"$response"
