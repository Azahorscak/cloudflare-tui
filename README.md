# cloudflare-tui

A terminal UI for browsing Cloudflare DNS records, powered by credentials stored in a Kubernetes secret.

## Prerequisites

- Go 1.22+
- Access to a Kubernetes cluster with a secret containing a Cloudflare API token
- The secret must have a key named `api-token`

## Build

```bash
go build -o cloudflare-tui ./cmd/cloudflare-tui
```

For a hardened production build that strips file paths and debug symbols:

```bash
go build -trimpath -ldflags="-s -w" -o cloudflare-tui ./cmd/cloudflare-tui
```

## Usage

```bash
# Run with a Kubernetes secret reference (required)
cloudflare-tui --secret <namespace>/<secret-name>

# Specify a custom kubeconfig
cloudflare-tui --secret my-namespace/cloudflare-creds --kubeconfig ~/.kube/config
```

The `--secret` flag is required and points to a Kubernetes secret in `namespace/secret-name` format. The secret must contain an `api-token` key with a valid Cloudflare API token.

## Navigation

- **Zone list**: use arrow keys to navigate, `/` to filter, `Enter` to select a zone
- **DNS records table**: use arrow keys to scroll, `q` or `Esc` to go back
- `Ctrl+C` quits from any screen

## Architecture

```
cmd/cloudflare-tui/    main entrypoint — parses flags, loads config, starts TUI
internal/
  config/              Kubernetes secret loading (sole credential source)
  api/                 Cloudflare API wrapper (thin structs, no SDK types leak out)
  tui/                 Bubble Tea models — one file per screen
    model.go           Root model, view routing
    zones.go           Zone selection list
    records.go         DNS record table
```

The TUI layer never imports the Cloudflare SDK directly. The API layer never imports Bubble Tea. Dependencies flow one way: `main -> config + api + tui`, `tui -> api`.

## Security

See [SECURITY.md](SECURITY.md) for the full security model, including:

- Cloudflare API token scoping (least privilege)
- Kubernetes RBAC requirements
- Vulnerability reporting

**Key points:**

- The application is **read-only** — it never creates, modifies, or deletes resources.
- Credentials come exclusively from a Kubernetes secret. No env vars, no local files.
- API calls enforce a 30-second timeout to prevent indefinite hangs.
- The API token is held in memory only and is never logged or written to disk.

### Kubernetes Secret Setup

Create the secret containing your scoped Cloudflare API token:

```bash
kubectl create secret generic cloudflare-creds \
  --namespace=my-namespace \
  --from-literal=api-token=<your-cloudflare-api-token>
```

See [SECURITY.md](SECURITY.md) for the minimal RBAC role needed to read this secret.

## Testing

```bash
go test ./...
```

## Linting

```bash
golangci-lint run ./...
```
