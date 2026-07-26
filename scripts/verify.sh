#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

NPM_BIN="${NPM_BIN:-}"
if [[ -z "$NPM_BIN" ]]; then
  if npm --version >/dev/null 2>&1; then
    NPM_BIN="npm"
  elif [[ -x /usr/bin/npm ]] && /usr/bin/npm --version >/dev/null 2>&1; then
    echo "warning: active npm is unusable; using /usr/bin/npm" >&2
    NPM_BIN="/usr/bin/npm"
  else
    echo "error: no usable npm executable found" >&2
    exit 1
  fi
fi

echo "==> go test ./..."
go test ./...

mkdir -p .tmp

echo "==> go build -o .tmp/missionledger-api ./cmd/api"
go build -o .tmp/missionledger-api ./cmd/api

echo "==> npm --prefix web ci"
"$NPM_BIN" --prefix web ci

echo "==> npm --prefix web run build"
"$NPM_BIN" --prefix web run build

echo "==> ./scripts/smoke.sh"
./scripts/smoke.sh
