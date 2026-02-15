# cloudflare-tui

A terminal UI for browsing Cloudflare DNS records. Credentials are read from a Kubernetes secret.

## Usage

```bash
# Build
go build -o cloudflare-tui ./cmd/cloudflare-tui

# Run (--secret is required: namespace/secret-name)
./cloudflare-tui --secret my-namespace/cloudflare-creds

# Optional: specify a kubeconfig file
./cloudflare-tui --secret my-namespace/cloudflare-creds --kubeconfig ~/.kube/config
```

The Kubernetes secret must contain a key named `api-token` with a scoped Cloudflare API token.

## Controls

- **Arrow keys / j/k** — navigate lists and tables
- **/** — filter zones
- **Enter** — select a zone
- **q / Esc** — go back to zone list
- **Ctrl+C** — quit
