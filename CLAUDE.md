# CLAUDE.md — cloudflare-tui

## Project Overview

A Go-based terminal UI for interacting with the Cloudflare API. The initial goal is listing DNS records for a specified zone. Credentials are read from a Kubernetes secret specified at launch time.

## Tech Stack

- **Language:** Go (1.22+)
- **TUI framework:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) with [Lip Gloss](https://github.com/charmbracelet/lipgloss) for styling and [Bubbles](https://github.com/charmbracelet/bubbles) for common components (tables, spinners, text input)
- **Cloudflare SDK:** [cloudflare-go](https://github.com/cloudflare/cloudflare-go) (official Go library)
- **Kubernetes:** [client-go](https://github.com/kubernetes/client-go) for reading secrets from a cluster
- **Build:** standard `go build` / `go run`

## Commands

```bash
# Build
go build -o cloudflare-tui ./cmd/cloudflare-tui

# Run (--secret is required: namespace/secret-name)
go run ./cmd/cloudflare-tui --secret my-namespace/cloudflare-creds

# Test
go test ./...

# Lint (if golangci-lint is available)
golangci-lint run ./...
```

## Project Structure

```
cmd/cloudflare-tui/    # main entrypoint
internal/
  api/                 # Cloudflare API client wrapper
  tui/                 # Bubble Tea models, views, updates
  config/              # Configuration / credential loading from Kubernetes
```

## Code Conventions

- Keep packages small and focused. `internal/` for all non-main code.
- Bubble Tea models live in `internal/tui/`. One file per major view/screen.
- API interactions are wrapped in `internal/api/` — never call cloudflare-go directly from TUI code.
- Use `context.Context` for all API calls.
- Errors from the API should surface as user-visible messages inside the TUI, not panics.
- No global state. Pass dependencies through struct fields.
- Tests go in `_test.go` files next to the code they test.

## Authentication

The app reads a Cloudflare API token from a Kubernetes secret. The secret is specified at launch via a required `--secret` flag in `namespace/secret-name` format.

- The secret must contain a key named `api-token` whose value is a scoped Cloudflare API token.
- The app uses the current kubeconfig context (respects `KUBECONFIG` env var and `--kubeconfig` flag).
- No environment-variable or local-file credential paths exist. Kubernetes is the sole credential source.
- The app fails fast with a clear error if the secret is missing, inaccessible, or lacks the `api-token` key.

## Guiding Principles

- **Minimum viable first.** Ship the smallest thing that works, then iterate.
- **Separation of concerns.** TUI rendering knows nothing about HTTP; API layer knows nothing about terminal rendering.
- **No over-abstraction.** Don't build interfaces or generics until there are at least two concrete consumers.
