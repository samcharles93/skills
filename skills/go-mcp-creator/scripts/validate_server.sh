#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <project-dir>" >&2
  exit 1
fi

project_dir=$1

if [[ ! -d "$project_dir" ]]; then
  echo "[ERROR] directory not found: $project_dir" >&2
  exit 1
fi

cd "$project_dir"

echo "[INFO] formatting"
gofmt -w .

echo "[INFO] testing"
go test ./...

echo "[INFO] vet"
go vet ./...

if command -v govulncheck >/dev/null 2>&1; then
  echo "[INFO] govulncheck"
  govulncheck ./...
else
  echo "[WARN] govulncheck not installed; skipping"
fi

echo "[OK] validation finished"
