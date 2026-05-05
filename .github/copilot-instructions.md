# GitHub Copilot Instructions

## Build, Test, and Lint

```bash
# Build
/opt/local/bin/go build ./...
make build                        # with version/commit ldflags injected

# Test (full suite with race detector)
/opt/local/bin/go test -v -race ./...

# Single test
/opt/local/bin/go test -v -run TestFunctionName ./internal/client/...

# Single package
/opt/local/bin/go test -v -race ./internal/client/...

# Vet / Format / Lint
/opt/local/bin/go vet ./...
gofmt -s -w .
golangci-lint run ./...

# Regenerate shell completions (required after adding/changing commands)
make completions
```

## Architecture

This project uses a strict **two-layer design**:

- **`cmd/`** - Thin Cobra command wiring only. Each file defines flags, calls `getClient()`, calls exactly one client method, calls `getFormatter()`, and formats output. No URL construction, no HTTP logic, no direct printing.
- **`internal/client/`** - All HTTP logic, API structs, request building, response parsing, session management. No Cobra imports, no output dependencies.
- **`internal/config/`** - Loads `~/.iics/config.yaml` (Viper-backed) and `~/.iics/sessions.yaml` (30-min TTL session cache). Profile resolution precedence: env vars (`IICS_*`) > `--profile` flag > `defaultProfile` in config file.
- **`internal/output/`** - `Formatter` interface with `table`, `json`, `csv`, `yaml` backends. Table uses `charmbracelet/lipgloss` (no `tablewriter`).

### Adding a New Resource

1. `internal/client/<resource>s.go` - struct + CRUD methods using `doJSON` / `doJSONWithQuery`
2. `internal/client/<resource>s_test.go` - test each method with `newTestClient(handler)`
3. `cmd/<resource>.go` - thin command wiring only
4. Register in `cmd/root.go` `init()` via `rootCmd.AddCommand(newResourceCmd())`
5. `docs/documentation/<resource>.md` - command reference (synopsis, flags, output columns, examples)
6. Add a row to the Commands table in `README.md`
7. Run `make completions` and commit updated `completions/` files

Documentation (steps 5-6) and updated completions (step 7) are mandatory parts of every new command.

## Key Conventions

### HTTP Client

- `do()` auto-logins on first request and retries once on 401 with a fresh session.
- API version is auto-detected from URL path: `/v2/` uses `icSessionId` header; all others use `INFA-SESSION-ID`.
- `doJSON()` handles JSON request/response; `doRaw()` returns `io.ReadCloser` for file downloads.
- Session state (`sessionID`, `baseAPIURL`) is protected by `sync.RWMutex`.
- Register a session persistence callback with `c.SetOnLoginSuccess()` in `cmd/root.go`.

### Output Formatting

```go
// Always construct the formatter this way:
f := output.New(format, w, output.TableStyle{})
f.Format(rows, []output.Column{
    {Header: "NAME", Field: "name"},
    {Header: "STATUS", Field: "status.value"},  // dot-notation for nested fields
})
```

`Field` values must match JSON tag names. Dot notation addresses nested struct fields.

### Struct / API Rules

- JSON tags must exactly match API field names - verify against real API responses, never guess.
- Nested API arrays of objects must use typed structs, never `[]string`.
- All API response structs live in `internal/client/`; no struct definitions in `cmd/`.

### Config and Session

- Password field `"@keyring"` is a sentinel that triggers OS keychain lookup.
- `DeriveCaiURL()` converts a base API URL to a CAI URL by inserting `-cai` into the first DNS label.
- Region constants and their login URL mappings are in `internal/config/pods.go`.

### Tests

- All tests live in `internal/` only - no tests in `cmd/`.
- Use `newTestClient(handler)` with `httptest.NewServer` for HTTP mocking.
- Use `t.TempDir()` for temp files and `t.Cleanup()` for teardown.
- For interactive prompt tests, redirect `os.Stdin` using an `os.Pipe()`.

### Error Handling

- Never call `os.Exit()` - always return errors from `RunE`.
- Session expiry returns `SessionExpiredError` with a message directing the user to run `iics login`.

### Git Workflow

- Always work on the `dev` branch. Never commit directly to `main`.
- Verify branch with `git branch --show-current` before committing.
- Before every commit: `gofmt -s -w .`, `go vet ./...`, `golangci-lint run ./...`.
- Do **not** include `Co-Authored-By: ...` trailers in commit messages.


### Change Requests and Bug Fixes

Change Requests live in `docs/ChangeRequests/` as markdown files:

- `new/` - Pending implementation
- `pending/` - Implemented, awaiting confirmation; move from `new/` in the same commit as the implementation
- `completed/` - Confirmed complete; move from `pending/` after confirmation

Bugs follow the same lifecycle under `docs/issues/`.
