# Plan: DNS Record Viewer and Editor

## Goal

Given a Cloudflare zone, display its DNS records in an interactive terminal table and allow selecting individual records to edit them in-place.

### Milestone 1 (Steps 1–8): List DNS records for a zone -- COMPLETE
### Milestone 2 (Steps 9–14): Select and edit a DNS record

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
- Fetch the specified Secret and extract the `cloudflare_api_token` key.
- Return a `Config` struct containing the API token string.
- Fail fast with a clear error if:
  - `--secret` is missing or malformed.
  - The secret does not exist or is inaccessible.
  - The `cloudflare_api_token` key is absent or empty.

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
- Verify graceful error handling: missing secret, missing `cloudflare_api_token` key, invalid token, network failure.
- Add a one-line usage note to `README.md`.

Deliverable: `go test ./...` passes; README has basic run instructions.

---

## Step 9: Extend the API layer with update and record-ID support

- Add an `ID` field to the `api.DNSRecord` struct so individual records can be targeted.
- Populate `ID` from the Cloudflare SDK response in `ListDNSRecords`.
- Add `GetDNSRecord(ctx, zoneID, recordID) (DNSRecord, error)` — fetches a single record (used to refresh after edit).
- Add `UpdateDNSRecord(ctx, zoneID, recordID, params UpdateDNSRecordParams) (DNSRecord, error)`:
  - `UpdateDNSRecordParams` contains the editable fields: `Name`, `Type`, `Content`, `TTL`, `Proxied`.
  - Maps to the cloudflare-go `DNS.Records.Update` call.
  - Returns the updated record or a descriptive error.
- Add unit tests with HTTP mocks for both new functions.

Deliverable: `api.Client` can read and update individual DNS records; `go test ./internal/api/...` passes.

## Step 10: Build the record-edit view (`internal/tui/edit.go`)

- Create an `EditModel` struct that represents a form for editing a single DNS record.
- Display the current record values as pre-filled text inputs using `bubbles/textinput` for editable string fields (`Name`, `Content`, `TTL`).
- Use a toggle or key binding for the `Proxied` boolean (e.g., `Tab` to toggle Yes/No).
- Display `Type` as read-only (changing a record's type typically requires delete + recreate).
- Show the record type and current zone name in a header for context.
- Layout the form fields vertically with labels, using lipgloss for alignment and styling.

Deliverable: `EditModel` renders a populated edit form; compiles and can be instantiated from test code.

## Step 11: Add form navigation and input handling to the edit view

- Support `Tab` / `Shift+Tab` to move focus between form fields.
- Support `Enter` on the submit button to trigger the save action.
- Support `Esc` to cancel and return to the records table without saving.
- Add client-side validation before submitting:
  - `Name` must be non-empty.
  - `Content` must be non-empty.
  - `TTL` must be a positive integer or "Auto" (mapped to `1`).
- Display inline validation errors below the relevant field in a warning style.

Deliverable: form fields are navigable and editable; validation prevents obviously bad submissions.

## Step 12: Wire up the save action and success/error feedback

- When the user submits the form, fire a `Cmd` that calls `api.UpdateDNSRecord` with the edited values.
- Show a spinner or "Saving…" indicator while the API call is in flight.
- On success:
  - Emit a message that transitions back to the records table.
  - Refresh the records list so the table reflects the updated values.
  - Show a brief success status message (e.g., in the help bar area).
- On error:
  - Stay on the edit form.
  - Display the API error message prominently so the user can correct and retry.

Deliverable: editing a record and pressing Enter persists the change to Cloudflare; errors are visible and recoverable.

## Step 13: Integrate the edit view into the root model and records table

- In `RecordsModel`, handle `Enter` on a selected table row to emit an `editRecordMsg` containing the selected `DNSRecord` (requires storing the full record list alongside the table rows so the ID is accessible).
- In the root `Model`, add `ViewEdit` to the `View` enum.
- Handle `editRecordMsg`: create a new `EditModel` for the selected record, set `currentView = ViewEdit`.
- Handle a new `backToRecordsMsg` (from cancel or successful save): set `currentView = ViewRecords`.
- On successful save, also trigger a records refresh so the table is up to date.
- Update the records table help bar to indicate `Enter: edit record`.

Deliverable: full navigation loop works — zones → records → edit → records; the table updates after a save.

## Step 14: Tests and polish for the edit flow

- Add unit tests for `EditModel`: verify initial field population, tab navigation order, validation error messages, and message emission on submit/cancel.
- Add an integration-style test that simulates the full flow: select zone → select record → edit content → save → verify table refresh (using mocked API).
- Verify edge cases: editing a record with a very long content value, TTL boundary values (1 = Auto, minimum non-auto value), toggling proxied on record types that don't support it.
- Update the help bar / key hints on all views to reflect the new navigation options.
- Update `README.md` with a note about the editing capability.

Deliverable: `go test ./...` passes; edit flow is stable and discoverable.

---

## Out of Scope (future milestones)

- Creating / deleting DNS records
- Bulk editing multiple records at once
- Multiple account support
- Alternative credential sources (env vars, local files)
- Caching or pagination beyond what cloudflare-go handles
- CI/CD pipeline
