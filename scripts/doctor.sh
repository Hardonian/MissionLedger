#!/usr/bin/env bash
set -euo pipefail

command -v go >/dev/null || { echo 'doctor: go is required' >&2; exit 1; }
if [[ -n "${DATABASE_URL:-}" ]]; then
  echo 'doctor: DATABASE_URL configured (connectivity is checked by verify/smoke)'
else
  echo 'doctor: memory mode (DATABASE_URL not set)'
fi
go version
printf 'doctor: repository prerequisites PASS\n'
