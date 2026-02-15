# Plan: MVP — List DNS Records for a Zone

## Goal

The absolute minimum useful action: given a Cloudflare zone, display its DNS records in an interactive terminal table.

---

## Step 1: Initialize the Go module and install dependencies

Create `go.mod` and pull in the three core dependencies:

- `github.com/cloudflare/cloudflare-go/v4` — Cloudflare API client
- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/charmbracelet/bubbles` — table component
- `github.com/charmbracelet/lipgloss` — styling
- `k8s.io/client-go` — Kubernetes API client (for reading secrets)

Deliverable: `go.mod` and `go.sum` exist; `go build ./...` succeeds (even if main is a stub).

## Step 2: Scaffold the directory structure

```
cmd/cloudflare-tui/main.go   # entrypoint — parse flags, build deps, start TUI
internal/
  config/config.go            # read Cloudflare API token from a Kubernetes secret
  api/client.go               # thin wrapper: NewClient, ListZones, ListDNSRecords
  tui/model.go                # root Bubble Tea model
  tui/zones.go                # zone-selection list view
  tui/records.go              # DNS record table view
```

Deliverable: all files created with minimal placeholder code; project compiles.

## Step 3: Implement credential loading (`internal/config`)

- Accept the `--secret` flag value (`namespace/secret-name`) and an optional `--kubeconfig` path.
- Use `client-go` to build a Kubernetes client from the current kubeconfig context.
- Fetch the specified Secret and extract the `api-token` key.
- Return a `Config` struct containing the API token string.
- Fail fast with a clear error if:
  - `--secret` is missing or malformed.
  - The secret does not exist or is inaccessible.
  - The `api-token` key is absent or empty.

Deliverable: `config.Load(ctx, secretRef, kubeconfig)` returns a populated `Config` or a descriptive error.

## Step 4: Implement the API layer (`internal/api`)

- `NewClient(cfg config.Config)` — creates an authenticated `cloudflare-go` client.
- `ListZones(ctx) ([]Zone, error)` — returns zone ID + name for all zones the token can see.
- `ListDNSRecords(ctx, zoneID) ([]DNSRecord, error)` — returns records for a zone (type, name, content, TTL, proxied).

Keep return types as thin structs (not raw cloudflare-go types) so the TUI never imports the SDK.

Deliverable: API functions work when called from a throwaway `main()` with real credentials.

## Step 5: Build the zone-selection view (`internal/tui/zones.go`)

- On startup, fire a Bubble Tea `Cmd` that calls `api.ListZones`.
- Show a spinner while loading.
- Render zones in a `bubbles/list` (filterable list).
- When the user selects a zone, transition to the records view.

Deliverable: running the app shows a list of zones; pressing Enter on one transitions forward.

## Step 6: Build the DNS records table view (`internal/tui/records.go`)

- On entry, fire a `Cmd` that calls `api.ListDNSRecords` for the selected zone.
- Show a spinner while loading.
- Render records in a `bubbles/table` with columns: Type, Name, Content, TTL, Proxied.
- Support `q` / `Esc` to go back to zone selection, `Ctrl+C` to quit.

Deliverable: selecting a zone shows its DNS records in a navigable table.

## Step 7: Wire everything together in `cmd/cloudflare-tui/main.go`

- Parse flags: `--secret` (required, `namespace/secret-name`), `--kubeconfig` (optional).
- Load config from the Kubernetes secret.
- Create API client with the retrieved token.
- Initialize root TUI model (starts on zone-selection view).
- Run `tea.NewProgram(model).Run()`.
- Exit cleanly on error with a human-readable message.

Deliverable: `go run ./cmd/cloudflare-tui --secret ns/name` works end-to-end.

## Step 8: Polish and basic tests

- Add a test for `config.Load` (use a fake Kubernetes clientset to verify secret reading).
- Add a test for API struct mapping (unit test with mocked responses if practical).
- Verify graceful error handling: missing secret, missing `api-token` key, invalid token, network failure.
- Add a one-line usage note to `README.md`.

Deliverable: `go test ./...` passes; README has basic run instructions.

---

## Out of Scope (future milestones)

- Creating / editing / deleting DNS records
- Multiple account support
- Alternative credential sources (env vars, local files)
- Caching or pagination beyond what cloudflare-go handles
- CI/CD pipeline
