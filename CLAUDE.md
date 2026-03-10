# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> For the full authoritative reference (patterns, struct rules, API versioning, examples), read **`docs/CLAUDE.md`** first.

---

## Common Commands

```bash
# Build
/opt/local/bin/go build ./...
make build         # includes ldflags with version injection

# Test
/opt/local/bin/go test ./...
/opt/local/bin/go test -v -race ./internal/client/...   # single package, with race detector
/opt/local/bin/go test -run TestListWidgets ./internal/client/...  # single test

# Vet / Format / Lint
/opt/local/bin/go vet ./...
gofmt -s -w .
golangci-lint run ./...
```

---

## Architecture Overview

**Two-layer design:**

- **`cmd/`** — thin Cobra command files; only flag parsing, `getClient()`, one client call, `getFormatter()`, format output. No API logic, no URL construction.
- **`internal/client/`** — all HTTP logic, API structs, session management, request/response handling. No Cobra or output dependencies.
- **`internal/config/`** — config file (`~/.iics/config.yaml`) and session cache (`~/.iics/sessions.yaml`) with 30-minute TTL.
- **`internal/output/`** — `Formatter` interface with `table`, `json`, `csv`, `yaml` backends. Columns are specified by JSON tag name, supporting dot notation for nested fields.

**HTTP client** (`internal/client/client.go`): single `*Client` struct with `do()` → auto-login on first request, 401 → re-login + retry once. Auto-detects V2 vs V3 API from URL path and sets the correct session header (`icSessionId` for `/v2/`, `INFA-SESSION-ID` for all others).

**Config resolution order** (highest to lowest): env vars (`IICS_*`) → `--profile` flag → `defaultProfile` in config file.

---

## Commit Style

- Do **not** include `Co-Authored-By: Claude ...` trailers in commit messages.

---

## Critical Constraints

- **tablewriter v1.x API only**: `NewTable(w)`, `Header(...)`, `Append([]interface{}{...})`, `err := Render()`. Never v0.x patterns.
- **JSON tags must exactly match API field names** — never guess; verify against API docs.
- **Nested API objects must use typed structs** — never `[]string` for arrays of objects.
- **Never call `os.Exit()`** — always return errors from `RunE`.
- **Tests in `internal/` only** — use `newTestClient(handler)` with `httptest.Server`. No `cmd/` tests.
- **Do not refactor unrelated code** when fixing a bug or implementing a change request.

---

## Adding a New Resource

1. `internal/client/<resource>s.go` — struct + CRUD methods using `doJSON` / `doJSONWithQuery`
2. `internal/client/<resource>s_test.go` — test each method with `newTestClient`
3. `cmd/<resource>.go` — thin command wiring, `getClient` → client method → `getFormatter` → `f.Format`
4. `cmd/root.go` `init()` — `rootCmd.AddCommand(newResourceCmd())`

See `docs/CLAUDE.md` for complete code templates for all four files.
