# golangci-lint Installation & Setup Guide

Proper installation of `golangci-lint` is critical for reproducible lint results across local development environments and CI pipelines.

---

## ⚠️ The `go install` Trap: Why Binary Installation is Required

The `golangci-lint` maintainers **strongly discourage and officially warn against** installing via `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`.

### Why `go install` Causes Bugs & Inconsistencies

1. **Go Toolchain & Compiler Mismatch**: `go install` compiles `golangci-lint` locally using your installed Go version. Bugs or subtle behavior differences occur if your Go toolchain differs from the version used by `golangci-lint` maintainers to build and test releases.
2. **Untested Dependency Assemblies**: `go install` builds from `go.mod` dependencies. If project dependencies or flags alter transitive dependencies, `golangci-lint` runs with an untested combination of linter packages.
3. **Ignored `replace` Directives**: `go.mod` `replace` directives are NOT applied transitively during `go install`. Any upstream patches or dependency fixes applied by `golangci-lint` maintainers are silently skipped.
4. **Build Performance**: `go install` compiles dozens of heavy AST linter packages from source, making installation significantly slower than downloading pre-compiled binaries.

> **Rule**: Always install `golangci-lint` via official pre-built binaries or package managers, NEVER via `go install`.

---

## Local Development Installation

### 1. Official Binary Install Script (Recommended)

Install the binary into `$(go env GOPATH)/bin` or `./bin/`:

```bash
# Install to $(go env GOPATH)/bin
curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.12.2

# Install to local ./bin/ directory
curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b ./bin v2.12.2
```

### 2. Package Managers

- **macOS / Linux (Homebrew)**:
  ```bash
  brew install golangci-lint
  ```
- **Windows (Scoop / Chocolatey)**:
  ```powershell
  scoop install golangci-lint
  # or
  choco install golangci-lint
  ```
- **Alpine Linux**:
  ```sh
  apk add golangci-lint
  ```

---

## CI & Automated Pipeline Installation

To maintain reproducible CI builds, **always pin the exact version** (e.g. `v2.12.2`). Unpinned versions or `@latest` tags can cause builds to fail randomly when new linters or rules are released upstream.

### 1. GitHub Actions (Official Action)

Use the official `golangci/golangci-lint-action`. It manages binary downloading, version pinning, and automatic caching:

```yaml
name: golangci-lint
on:
  push:
    branches: [ main ]
  pull_request:

jobs:
  golangci:
    name: lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          cache: false
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v2.12.2
          # Optional: args: --timeout=5m
```

### 2. Taskfile / Makefile / Generic CI Script

In non-GitHub CI environments (GitLab CI, Jenkins, Buildkite), download the pinned binary explicitly:

```yaml
# Taskfile.yml snippet
tasks:
  lint:
    desc: Run golangci-lint
    cmds:
      - golangci-lint run ./...

  lint-install:
    desc: Install pinned golangci-lint binary
    cmds:
      - curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.12.2
```

### 3. Docker Official Image

For containerized CI pipelines, use the official Docker image:

```dockerfile
FROM golangci/golangci-lint:v2.12.2-alpine AS linter
WORKDIR /app
COPY . .
RUN golangci-lint run ./...
```
