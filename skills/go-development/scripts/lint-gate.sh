#!/usr/bin/env bash
# Go quality gate: formatting, static analysis, lint, vulnerabilities, tests.
#
# Usage:
#   scripts/lint-gate.sh            # full gate
#   scripts/lint-gate.sh --quick    # skip -race and govulncheck (development loop)
#   scripts/lint-gate.sh --fix      # apply formatters and safe fixes first
#
# Exits non-zero if any stage fails. Every stage runs even when an earlier one
# fails, so one invocation reports the whole picture rather than the first
# problem only.

set -uo pipefail

QUICK=0
FIX=0
for arg in "$@"; do
    case "$arg" in
        --quick) QUICK=1 ;;
        --fix) FIX=1 ;;
        -h|--help) sed -n '2,12p' "$0"; exit 0 ;;
        *) echo "unknown flag: $arg" >&2; exit 2 ;;
    esac
done

FAILURES=()
SKIPPED=()

have() { command -v "$1" >/dev/null 2>&1; }

stage() {
    local name="$1"; shift
    printf '\n▸ %s\n' "$name"
    if "$@"; then
        return 0
    fi
    FAILURES+=("$name")
    return 1
}

# Formatting is reported as a list of offending files; an empty list is a pass.
check_gofmt() {
    local offenders
    offenders=$(gofmt -l . 2>/dev/null | grep -v '^$' || true)
    if [[ -n "$offenders" ]]; then
        echo "not gofmt-clean:"
        echo "$offenders"
        return 1
    fi
    echo "clean"
}

if [[ $FIX -eq 1 ]]; then
    if have golangci-lint; then
        stage "format (golangci-lint fmt)" golangci-lint fmt ./... || true
        stage "autofix (golangci-lint run --fix)" golangci-lint run --fix ./... || true
    else
        stage "format (gofmt -w)" gofmt -w . || true
    fi
fi

stage "gofmt" check_gofmt || true
stage "go build" go build ./... || true
stage "go vet" go vet ./... || true

if have golangci-lint; then
    stage "golangci-lint" golangci-lint run ./... || true
else
    SKIPPED+=("golangci-lint (not installed: https://golangci-lint.run/docs/welcome/install/)")
fi

if have staticcheck; then
    stage "staticcheck" staticcheck ./... || true
else
    SKIPPED+=("staticcheck (go install honnef.co/go/tools/cmd/staticcheck@latest)")
fi

if [[ $QUICK -eq 0 ]]; then
    if have govulncheck; then
        stage "govulncheck" govulncheck ./... || true
    else
        SKIPPED+=("govulncheck (go install golang.org/x/vuln/cmd/govulncheck@latest)")
    fi
fi

stage "go test" go test ./... || true

if [[ $QUICK -eq 0 ]]; then
    stage "go test -race" go test -race ./... || true
fi

printf '\n─────────────────────────────────\n'
if [[ ${#SKIPPED[@]} -gt 0 ]]; then
    printf 'skipped:\n'
    printf '  - %s\n' "${SKIPPED[@]}"
fi

if [[ ${#FAILURES[@]} -eq 0 ]]; then
    printf 'gate passed\n'
    exit 0
fi

printf 'gate FAILED:\n'
printf '  - %s\n' "${FAILURES[@]}"
exit 1
