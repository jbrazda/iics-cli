# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> For the full authoritative reference (patterns, struct rules, API versioning, examples), read **`docs/CLAUDE.md`** first.

---

## Project Structure

This is a Go CLI project using Cobra. Key paths:

- `cmd/` - command definitions (thin wiring only)
- `internal/` - core logic (HTTP client, config, output)
- `docs/` - documentation and Change Requests (CRs)

Change Requests live in `docs/ChangeRequests/` as markdown files and must be referenced when implementing features.

---

## Git Workflow

- Always work on the `dev` branch. Never commit directly to `main`.
- Before making any commits, verify the current branch with `git branch --show-current`.

---

## Build & Validation

After implementing changes, run the following before committing:

```bash
/opt/local/bin/go build ./...
/opt/local/bin/go vet ./...
```

Also verify that the Go version in `go.mod` matches the version used in `.github/workflows/` CI files to avoid version mismatch failures.

---

## API Integration

When modifying API request/response structs:

- Always verify field names and types against actual API responses - never guess.
- Check for type mismatches (e.g., `string` vs `int`) by referencing existing working examples in `internal/client/`.
- Test with a real API call or review existing client code before committing struct changes.

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

- **`cmd/`** - thin Cobra command files; only flag parsing, `getClient()`, one client call, `getFormatter()`, format output. No API logic, no URL construction.
- **`internal/client/`** - all HTTP logic, API structs, session management, request/response handling. No Cobra or output dependencies.
- **`internal/config/`** - config file (`~/.iics/config.yaml`) and session cache (`~/.iics/sessions.yaml`) with 30-minute TTL.
- **`internal/output/`** - `Formatter` interface with `table`, `json`, `csv`, `yaml` backends. Columns are specified by JSON tag name, supporting dot notation for nested fields.

**HTTP client** (`internal/client/client.go`): single `*Client` struct with `do()` → auto-login on first request, 401 → re-login + retry once. Auto-detects V2 vs V3 API from URL path and sets the correct session header (`icSessionId` for `/v2/`, `INFA-SESSION-ID` for all others).

**Config resolution order** (highest to lowest): env vars (`IICS_*`) → `--profile` flag → `defaultProfile` in config file.

---

## Commit Style

- Do **not** include `Co-Authored-By: Claude ...` trailers in commit messages.
- **Before every commit**, run the following and fix all reported issues:

  ```bash
  gofmt -s -w .
  /opt/local/bin/go vet ./...
  golangci-lint run ./...
  ```

---

## Critical Constraints

- **Table output**: use `output.New(format, w, output.TableStyle{})` from `internal/output/`. Do not add new `tablewriter` imports - the dependency was removed.
- **JSON tags must exactly match API field names** - never guess; verify against API docs.
- **Nested API objects must use typed structs** - never `[]string` for arrays of objects.
- **Never call `os.Exit()`** - always return errors from `RunE`.
- **Tests in `internal/` only** - use `newTestClient(handler)` with `httptest.Server`. No `cmd/` tests.
- **Do not refactor unrelated code** when fixing a bug or implementing a change request.

---

## Adding a New Resource

1. `internal/client/<resource>s.go` - struct + CRUD methods using `doJSON` / `doJSONWithQuery`
2. `internal/client/<resource>s_test.go` - test each method with `newTestClient`
3. `cmd/<resource>.go` - thin command wiring, `getClient` → client method → `getFormatter` → `f.Format`
4. `cmd/root.go` `init()` - `rootCmd.AddCommand(newResourceCmd())`
5. `docs/documentation/<resource>.md` - command reference page (synopsis, flags, output columns, examples)
6. `README.md` Commands table - add a row linking to the new doc page

**Documentation is mandatory.** Every time a command is added or updated, steps 5 and 6 must be completed before the work is considered done.

After updating documentation, regenerate shell completion scripts:

```bash
make completions
```

Commit the updated files in `completions/` together with the code change.

## Change Request and Bug Lifecycle

### Change Requests (`docs/ChangeRequests/`)

- **New CRs** are created in `docs/ChangeRequests/new/`.
- **After implementing** a CR, move the file to `docs/ChangeRequests/pending/` and include that move in the same commit as the implementation.
- **After the developer confirms** the CR is complete and correct, move the file to `docs/ChangeRequests/completed/` and include that move in the commit.
- If the developer has **not yet confirmed**, keep the file in `docs/ChangeRequests/pending/`.

### Bugs (`docs/issues/`)

- **New bugs** are created in `docs/issues/new/`.
- **After fixing** a bug, move the file to `docs/issues/pending/` and include that move in the same commit as the fix.
- **After the developer confirms** the fix is correct, move the file to `docs/issues/completed/` and include that move in the commit.
- If the developer has **not yet confirmed**, keep the file in `docs/issues/pending/`.

---

## Markdown Rules

- Generate Markdown that follows the [Markdown Lint Rules](https://github.com/markdownlint/markdownlint/blob/main/docs/RULES.md)
- Do **not** use em dashes (`-`) in generated Markdown; use a regular hyphen (`-`) instead

See [docs/CLAUDE.md](docs/CLAUDE.md) for complete code templates for all four files.
