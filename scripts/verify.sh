#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "==> go test ./..."
go test ./...

mkdir -p .tmp

echo "==> go build -o .tmp/missionledger-api ./cmd/api"
go build -o .tmp/missionledger-api ./cmd/api

echo "==> npm --prefix web ci"
npm --prefix web ci

echo "==> npm --prefix web run build"
npm --prefix web run build

echo "==> ./scripts/smoke.sh"
./scripts/smoke.sh
