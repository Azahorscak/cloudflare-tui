# Plan: Flox Development Environment for cloudflare-tui

## Goal

Set up a complete, reproducible Flox environment so that any developer (or CI runner) can clone the repo, run `flox activate`, and immediately build, test, and lint cloudflare-tui — with zero system prerequisites beyond Flox itself.

---

## Step 1: Initialize the Flox environment

Run `flox init` in the project root. Flox will detect `go.mod` and suggest a Go environment. Accept the scaffold, then customize.

**Deliverable:** `.flox/env/manifest.toml` exists; `flox activate` launches a shell.

---

## Step 2: Define packages in `[install]`

Install all development and runtime dependencies declaratively:

```toml
[install]
# Go toolchain — pin to the major version used by the project
go.pkg-path = "go"
go.version = "^1.26"

# Linting
golangci-lint.pkg-path = "golangci-lint"

# Kubernetes tooling (needed at runtime for credential loading, useful for dev/testing)
kubectl.pkg-path = "kubectl"

# General dev utilities
git.pkg-path = "git"
gnumake.pkg-path = "gnumake"
```

**Rationale:**
- `go` — the compiler and toolchain; SemVer caret (`^1.26`) accepts any 1.x >= 1.26.
- `golangci-lint` — the project's stated lint tool (see CLAUDE.md).
- `kubectl` — useful for verifying Kubernetes secret access during development.
- `gnumake` — for the Makefile we'll add in Step 4.
- `git` — ensures git is available even on bare machines.

**Deliverable:** `flox activate` drops you into a shell with `go`, `golangci-lint`, `kubectl`, `make`, and `git` all on PATH.

---

## Step 3: Configure hooks and environment variables

```toml
[vars]
CGO_ENABLED = "0"

[hook]
on-activate = '''
  # Set Go environment cache inside Flox env cache (keeps $HOME clean)
  export GOENV="$FLOX_ENV_CACHE/goenv"
  export GOPATH="$FLOX_ENV_CACHE/gopath"
  export GOBIN="$FLOX_ENV_CACHE/gopath/bin"
  export PATH="$GOBIN:$PATH"

  # Download module dependencies on first activation
  echo "Syncing Go module dependencies..."
  go mod download
'''

[profile]
common = '''
  echo ""
  echo "cloudflare-tui dev environment ready"
  echo "  go version: $(go version | awk '{print $3}')"
  echo "  Run 'make help' for available commands."
  echo ""
'''
```

**Rationale:**
- `CGO_ENABLED=0` — the project uses no C bindings; static builds are simpler and more portable.
- `GOPATH` / `GOBIN` inside `$FLOX_ENV_CACHE` — keeps Go caches per-environment and avoids polluting the user's home directory.
- `go mod download` in `on-activate` — ensures dependencies are available immediately after activation.
- The `[profile]` section prints a welcome banner with the Go version and a hint to run `make help`.

**Deliverable:** `flox activate` downloads dependencies and displays environment info.

---

## Step 4: Add a Makefile with standard targets

Create a `Makefile` that wraps the common development commands. This provides a consistent interface whether invoked manually or by automation.

```makefile
.PHONY: build test lint clean help

BIN       := cloudflare-tui
BUILD_DIR := ./cmd/cloudflare-tui

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	go build -trimpath -ldflags="-s -w" -o $(BIN) $(BUILD_DIR)

test: ## Run all tests
	go test ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

clean: ## Remove build artifacts
	rm -f $(BIN)

all: lint test build ## Lint, test, and build (CI target)
```

**Deliverable:** `make build`, `make test`, `make lint`, and `make all` work inside the Flox environment.

---

## Step 5: Add a Flox manifest build

Define a reproducible build in the `[build]` section of `manifest.toml` so the binary can be built in an isolated sandbox:

```toml
[build.cloudflare-tui]
description = "Cloudflare DNS management TUI"
sandbox = "pure"
command = '''
  export CGO_ENABLED=0
  mkdir -p "$out/bin"
  go build -trimpath -ldflags="-s -w" -o "$out/bin/cloudflare-tui" ./cmd/cloudflare-tui
'''
```

This enables `flox build` to produce a reproducible binary in `result-cloudflare-tui/bin/cloudflare-tui`, built in a clean sandbox that only sees git-tracked files.

**Deliverable:** `flox build` succeeds and produces a working binary.

---

## Step 6: Add `.flox` to version control

Ensure `.flox/env/manifest.toml` is committed to git so any contributor gets the same environment:

```
# In .gitignore — do NOT ignore .flox/env/manifest.toml
# Flox caches and lock files can be ignored:
.flox/env.lock
.flox/cache/
```

Verify `.gitignore` includes the build artifact (`cloudflare-tui`) and any Flox-generated output (`result-*`).

**Deliverable:** `git status` shows `.flox/env/manifest.toml` tracked; lock/cache files ignored.

---

## Step 7: Test the environment on a clean machine (validation)

Verify the zero-prerequisite claim:

1. On a fresh machine (or container) with only Flox installed:
   ```bash
   git clone <repo-url>
   cd cloudflare-tui
   flox activate
   make all       # lint + test + build
   ```
2. Confirm `go`, `golangci-lint`, `kubectl`, and `make` are all provided by Flox and not the host system.
3. Confirm `go test ./...` passes.
4. Confirm `go build` produces a working binary.

**Deliverable:** The full development and build workflow succeeds with zero host prerequisites beyond Flox.

---

## Step 8: Document the Flox workflow

Update `README.md` with a "Development with Flox" section:

```markdown
## Development

This project uses [Flox](https://flox.dev) for reproducible development environments.

### Prerequisites

Install Flox: https://flox.dev/docs/install/

### Quick start

    git clone <repo-url>
    cd cloudflare-tui
    flox activate          # enters the dev environment
    make help              # see available commands
    make all               # lint, test, build

### Available make targets

    build    Build the binary
    test     Run all tests
    lint     Run golangci-lint
    all      Lint, test, and build
    clean    Remove build artifacts

### Reproducible build

    flox build             # build in isolated sandbox
    ./result-cloudflare-tui/bin/cloudflare-tui --help
```

**Deliverable:** README documents the Flox workflow for new contributors.

---

## Complete `manifest.toml` (reference)

```toml
version = 1

[install]
go.pkg-path = "go"
go.version = "^1.26"
golangci-lint.pkg-path = "golangci-lint"
kubectl.pkg-path = "kubectl"
git.pkg-path = "git"
gnumake.pkg-path = "gnumake"

[vars]
CGO_ENABLED = "0"

[hook]
on-activate = '''
  export GOENV="$FLOX_ENV_CACHE/goenv"
  export GOPATH="$FLOX_ENV_CACHE/gopath"
  export GOBIN="$FLOX_ENV_CACHE/gopath/bin"
  export PATH="$GOBIN:$PATH"
  echo "Syncing Go module dependencies..."
  go mod download
'''

[profile]
common = '''
  echo ""
  echo "cloudflare-tui dev environment ready"
  echo "  go version: $(go version | awk '{print $3}')"
  echo "  Run 'make help' for available commands."
  echo ""
'''

[build.cloudflare-tui]
description = "Cloudflare DNS management TUI"
sandbox = "pure"
command = '''
  export CGO_ENABLED=0
  mkdir -p "$out/bin"
  go build -trimpath -ldflags="-s -w" -o "$out/bin/cloudflare-tui" ./cmd/cloudflare-tui
'''
```

---

## Summary

| Step | What | Why |
|------|------|-----|
| 1 | `flox init` | Bootstrap the environment scaffold |
| 2 | `[install]` packages | Go, linter, kubectl, make, git — all declarative |
| 3 | `[hook]` + `[vars]` | GOPATH isolation, auto-dependency sync, welcome banner |
| 4 | Makefile | Consistent build/test/lint interface |
| 5 | `[build]` section | Reproducible sandboxed binary build via `flox build` |
| 6 | Version control `.flox` | Share the environment with all contributors |
| 7 | Clean-machine test | Validate zero-prerequisite developer experience |
| 8 | README update | Document the workflow for onboarding |

**Key Flox best practices applied:**
- Declarative `manifest.toml` checked into version control
- SemVer version constraints (not exact pins) for flexibility
- `on-activate` hook for environment setup (bash, portable across shells)
- `[profile]` for shell-specific niceties (welcome banner)
- `$FLOX_ENV_CACHE` for Go caches (keeps environments isolated)
- Manifest builds with `sandbox = "pure"` for reproducible output
- No host system assumptions — everything provided by the Flox environment
