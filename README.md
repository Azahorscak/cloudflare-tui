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

## Testing

```bash
go test ./...
```
