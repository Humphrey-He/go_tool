#!/usr/bin/env bash
set -euo pipefail

go test ./...

go run ./cmd/sqlsafelint scan --config .sqlsafelint.json
