# Build Pipeline Plan — cloudflare-tui

## Current State

- No CI/CD, no Makefile, no Dockerfile, no release automation
- Go 1.24.7 project with tests across three packages (`api`, `config`, `tui`)
- Build: `go build -o cloudflare-tui ./cmd/cloudflare-tui`
- Lint: `golangci-lint run ./...`
- Test: `go test ./...`

---

## Phase 1: Local Build Tooling (Makefile)

Add a `Makefile` to standardize local development commands.

**Targets:**

| Target | Command | Purpose |
|---|---|---|
| `build` | `go build -trimpath -ldflags="-s -w" -o cloudflare-tui ./cmd/cloudflare-tui` | Produce a hardened binary |
| `test` | `go test -race -count=1 ./...` | Run all tests with race detector |
| `lint` | `golangci-lint run ./...` | Static analysis |
| `fmt` | `gofmt -l -w .` | Format all Go files |
| `vet` | `go vet ./...` | Built-in Go static checks |
| `clean` | `rm -f cloudflare-tui` | Remove build artifacts |
| `all` | `fmt vet lint test build` | Full local pipeline |

**Version injection:** Use `-ldflags` to inject a `version` variable from git tags at build time:

```makefile
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)
```

---

## Phase 2: GitHub Actions CI

Create `.github/workflows/ci.yml` triggered on push and pull request to `main`.

### Job 1: `lint`

- **Runs on:** `ubuntu-latest`
- **Steps:**
  1. Checkout code
  2. Set up Go 1.24
  3. Run `golangci-lint` via the official `golangci/golangci-lint-action`

### Job 2: `test`

- **Runs on:** `ubuntu-latest`
- **Steps:**
  1. Checkout code
  2. Set up Go 1.24
  3. `go test -race -count=1 -coverprofile=coverage.out ./...`
  4. Upload coverage artifact (optional: report to Codecov)

### Job 3: `build`

- **Runs on:** `ubuntu-latest`
- **Depends on:** `lint`, `test` (runs only if both pass)
- **Steps:**
  1. Checkout code
  2. Set up Go 1.24
  3. `make build`
  4. Upload binary as workflow artifact

### Caching

Use `actions/setup-go` with its built-in module caching (`cache: true`) to speed up dependency downloads across runs.

---

## Phase 3: Container Image

Add a multi-stage `Dockerfile` to produce a minimal container image.

```
# Stage 1: Build
FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /cloudflare-tui ./cmd/cloudflare-tui

# Stage 2: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /cloudflare-tui /usr/local/bin/cloudflare-tui
ENTRYPOINT ["cloudflare-tui"]
```

**Key decisions:**
- Alpine base keeps the image small (~15 MB)
- `ca-certificates` is required for HTTPS calls to Cloudflare and Kubernetes APIs
- No shell, package manager, or other tools in the runtime stage beyond what Alpine provides
- The container is intended to run inside a Kubernetes cluster where it can use in-cluster config to read secrets

Add a `docker-build` target to the Makefile:

```makefile
docker-build:
	docker build -t cloudflare-tui:$(VERSION) .
```

---

## Phase 4: Release Automation

Use [GoReleaser](https://goreleaser.com/) to automate binary and container releases on git tags.

### `.goreleaser.yaml`

- **Builds:** Cross-compile for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`
- **Archives:** tar.gz with LICENSE and README
- **Docker:** Build and push multi-arch image to GitHub Container Registry (`ghcr.io`)
- **Changelog:** Auto-generated from conventional commits

### GitHub Actions workflow: `.github/workflows/release.yml`

- **Trigger:** Push of a `v*` tag (e.g., `v0.1.0`)
- **Steps:**
  1. Checkout code with full history (`fetch-depth: 0`)
  2. Set up Go 1.24
  3. Log in to GHCR (`docker/login-action`)
  4. Run GoReleaser (`goreleaser/goreleaser-action`)

This creates GitHub Releases with pre-built binaries and pushes a container image in a single step.

---

## Phase 5: Additional Quality Gates (Optional / Future)

These can be added incrementally as the project matures:

| Gate | Tool | Purpose |
|---|---|---|
| Vulnerability scanning | `govulncheck` | Check dependencies for known CVEs |
| License compliance | `go-licenses` | Ensure all deps have compatible licenses |
| Binary SBOM | `syft` / GoReleaser built-in | Software bill of materials for each release |
| Image scanning | `trivy` (in CI) | Scan container image for vulnerabilities |
| Branch protection | GitHub settings | Require CI pass before merge to `main` |

---

## Implementation Order

- [ ] **Step 1:** Add `Makefile` with `build`, `test`, `lint`, `fmt`, `vet`, `clean` targets
- [ ] **Step 2:** Add `.github/workflows/ci.yml` with lint, test, and build jobs
- [ ] **Step 3:** Add `Dockerfile` (multi-stage) and `docker-build` Makefile target
- [ ] **Step 4:** Add `.goreleaser.yaml` and `.github/workflows/release.yml`
- [ ] **Step 5:** Enable branch protection rules on `main` requiring CI to pass
- [ ] **Step 6:** (Future) Add `govulncheck`, `trivy`, and SBOM generation to CI

Each step is independently mergeable and provides value on its own.
