#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "==> go test ./..."
go test ./...

mkdir -p .tmp

echo "==> go build -o .tmp/missionledger-api ./cmd/api"
go build -o .tmp/missionledger-api ./cmd/api

echo "==> ./scripts/smoke.sh"
./scripts/smoke.sh
