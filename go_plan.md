# Go Upgrade Plan: 1.24.7 → 1.26.0

## Context

- **Current version:** Go 1.24.7 (sole reference in `go.mod:3`)
- **Target version:** Go 1.26.0 (latest stable, released 2026-02-10)
- **Branch:** `claude/plan-go-upgrade-XTN3P`
- No CI workflows, Dockerfiles, or Makefiles exist — only `go.mod` needs updating.

## Go 1.26 Notable Changes

- Green Tea garbage collector enabled by default (lower GC overhead)
- `new` built-in now accepts expressions as its operand
- Generic types may now refer to themselves in their own type parameter list

## Steps

### 1. Update `go.mod`
Change line 3 from:
```
go 1.24.7
```
to:
```
go 1.26.0
```
Add a `toolchain` line after it:
```
toolchain go1.26.0
```

### 2. Upgrade direct dependencies
Run `go get -u ./...` to pull the latest compatible versions of all direct and indirect dependencies:
- `github.com/charmbracelet/bubbletea`
- `github.com/charmbracelet/bubbles`
- `github.com/charmbracelet/lipgloss`
- `github.com/cloudflare/cloudflare-go/v4`
- `k8s.io/api`, `k8s.io/apimachinery`, `k8s.io/client-go`
- `golang.org/x/*` packages

### 3. Tidy the module
Run `go mod tidy` to:
- Regenerate `go.sum` with updated checksums
- Remove any entries for packages no longer needed

### 4. Build verification
Run `go build ./...` to ensure no compilation errors were introduced by dependency changes or new Go version toolchain.

### 5. Test verification
Run `go test ./...` to ensure all existing tests pass.

### 6. Commit and push
Create a single commit on branch `claude/plan-go-upgrade-XTN3P` with message:
```
chore: upgrade Go from 1.24.7 to 1.26.0 and update dependencies
```
Push with `git push -u origin claude/plan-go-upgrade-XTN3P`.

## Risk Assessment

| Area | Risk | Notes |
|------|------|-------|
| `go.mod` version bump | Low | Single-line change |
| Green Tea GC | Low | Drop-in replacement; improves performance |
| Language changes | Low | Both are additive; no breaking changes |
| Dependency updates | Medium | k8s.io v0.31.0 → newer may have API changes; charmbracelet libs tend to be stable |

## Rollback

If any step fails, revert `go.mod` and `go.sum` to their original state via `git checkout go.mod go.sum`.
