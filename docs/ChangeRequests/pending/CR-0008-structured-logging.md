# CR-0008: Structured Logging and Centralized Timestamp for All Commands

## CR Type

- [x] **Enhancement** - improve observability and developer experience across all commands

---

## Affected Commands

All commands that make HTTP calls are affected. They fall into three tiers by current logging
state:

**Tier 1 - Async/polling commands with `ts()` progress output** (timestamp format needs update):

| Command file | ts() calls | Notes |
| ------------ | ---------- | ----- |
| `cmd/import_.go` | 8 | `ts()` is defined here - must be removed and centralized |
| `cmd/export.go` | 13 | uses `ts()` from same package |
| `cmd/publish.go` | 7 | uses `ts()` from same package |
| `cmd/unpublish.go` | 2 | uses `ts()` from same package |

**Tier 2 - Commands with verbose progress but inconsistent or missing timestamps**
(adopt `ts()` and new format):

| Command file | Issue |
| ------------ | ----- |
| `cmd/objects.go` | uses inline `time.Now().Format(time.RFC3339)` at lines 137, 192, 195 - different format from `ts()` |

**Tier 3 - Single-call commands with no progress output**
(no `ts()` changes needed; HTTP debug via `--debug` already covers them):

| Command file | HTTP calls | Current logging |
| ------------ | ---------- | --------------- |
| `cmd/activitylog.go` | 2 | none |
| `cmd/agent.go` | 4 | none |
| `cmd/connection.go` | 5 | none |
| `cmd/folder.go` | 3 | none |
| `cmd/login.go` | 1 | passes `verbose` to client constructor only |
| `cmd/logout.go` | 1 | none |
| `cmd/metering.go` | 1 | none |
| `cmd/permission.go` | 2 | none |
| `cmd/privilege.go` | 1 | none |
| `cmd/project.go` | 3 | none |
| `cmd/role.go` | 5 | none |
| `cmd/runtime.go` | 4 | none |
| `cmd/schedule.go` | 5 | none |
| `cmd/securitylog.go` | 1 | none |
| `cmd/user.go` | 5 | none |
| `cmd/usergroup.go` | 5 | none |

Tier 3 commands gain full HTTP request/response tracing automatically via `--debug` once
`debugPrintHTTP()` is migrated to `slog.Default()`.

---

## Problem

1. **`ts()` is scattered and inconsistent.** The timestamp helper is defined once in
   `cmd/import_.go` but used by `cmd/export.go`, `cmd/publish.go`, and `cmd/unpublish.go` -
   all in the same package. There is no central location to change the format. The current
   format `15:04:05` (HH:MM:SS only) omits the date and has no sub-second precision, making
   it hard to correlate logs across day boundaries or diagnose timing issues in long runs.

2. **`cmd/objects.go` uses a different timestamp format.** Inline `time.Now().Format(time.RFC3339)`
   (e.g. `2026-03-22T14:52:41Z`) is used instead of `ts()`, creating a third inconsistent
   format.

3. **HTTP debug output has no structure or levels.** The existing `debugPrintHTTP()` in
   `internal/client/client.go` writes free-form text to `os.Stderr` with no log level, no
   timestamp, and no structured fields. There is no way to filter or redirect output.

4. **No consistent logging framework.** Each command invents its own output approach
   (`fmt.Fprintf` to stdout vs stderr, direct writes, etc.). Adding a new async command
   requires re-implementing the same verbose/debug pattern from scratch.

---

## Recommendation: `log/slog` (Go stdlib)

Given iics-cli is a Cobra CLI with no existing logging framework:

| Choice | Reason |
| ------ | ------ |
| `log/slog` | Zero deps, Go stdlib (1.21+), modern structured logging, custom handlers supported |
| `zerolog` | Extra dependency; colorized CLI UX; better if rich console formatting is a priority |

**Recommendation: `log/slog`.**

Rationale:

- Project already requires Go 1.24; `slog` has been stable since Go 1.21.
- Zero additional dependencies - important for a security-sensitive enterprise CLI.
- The project's current logging needs (timestamped debug lines, HTTP traces) are well within
  what a simple custom `slog.Handler` can deliver.
- The common CLI pattern of "quiet by default, INFO with `--verbose`, DEBUG with `--debug`"
  maps cleanly to slog levels.
- If richer output is needed later (colors, JSON toggle), `slog` supports custom handlers
  without changing call sites.

---

## Desired Change

### 1. Move and update `ts()` - centralize in `cmd/root.go`

Remove the `ts()` definition from `cmd/import_.go` (line 494) and place it in `cmd/root.go`.
Change the format to include date and milliseconds using the host local timezone:

```go
// ts returns the current local time formatted for progress output.
func ts() string {
    return time.Now().Format("[2006-01-02 15:04:05.000]")
}
```

The `[...]` brackets are included in the return value so call sites become:

```go
_, _ = fmt.Fprintf(out, "%s Publishing %d assets...\n", ts(), n)
```

All existing call sites in `cmd/import_.go`, `cmd/export.go`, `cmd/publish.go`, and
`cmd/unpublish.go` must be updated to remove the surrounding `[%s]` format wrapper they
currently use (e.g. change `"[%s] foo"` + `ts()` to `"%s foo"` + `ts()`).

### 2. Introduce `log/slog` logger initialized in `cmd/root.go`

Add a package-level `var logger *slog.Logger` initialized early in a `PersistentPreRunE`
hook (or inside `getClient()`) based on the active flags:

```go
// initLogger configures the package-level slog logger and sets it as the default.
func initLogger() {
    h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
        Level: logLevel(),
        ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
            if a.Key == slog.TimeKey {
                a.Value = slog.StringValue(time.Now().Format("[2006-01-02 15:04:05.000]"))
            }
            return a
        },
    })
    logger = slog.New(h)
    slog.SetDefault(logger)
}

func logLevel() slog.Level {
    if debug {
        return slog.LevelDebug
    }
    if verbose {
        return slog.LevelInfo
    }
    return slog.LevelWarn
}
```

### 3. Replace `debugPrintHTTP()` with structured slog output

Replace the free-form `fmt.Fprintf(os.Stderr, ...)` calls in `debugPrintHTTP()` with
`slog.Default()`. Because `initLogger()` calls `slog.SetDefault(logger)`, no changes to
the `Client` struct or its functional options are needed (Option B - minimal change surface).

`debugPrintHTTP` becomes:

```go
func debugPrintHTTP(req *http.Request, reqData []byte, resp *http.Response, respData []byte) {
    log := slog.Default()
    attrs := []any{
        slog.String("method", req.Method),
        slog.String("url", req.URL.String()),
        slog.String("status", resp.Status),
    }
    // Log request headers, masking session tokens
    for k, vs := range req.Header {
        if strings.EqualFold(k, sessionHeaderV3) || strings.EqualFold(k, sessionHeaderV2) {
            attrs = append(attrs, slog.String("req."+k, "***"))
        } else {
            attrs = append(attrs, slog.String("req."+k, strings.Join(vs, ",")))
        }
    }
    if len(reqData) > 0 {
        attrs = append(attrs, slog.String("reqBody", prettyJSONString(reqData)))
    }
    if len(respData) > 0 {
        attrs = append(attrs, slog.String("respBody", prettyJSONString(respData)))
    }
    log.Debug("http", attrs...)
}

// prettyJSONString pretty-prints JSON bytes; returns raw string on failure.
func prettyJSONString(data []byte) string {
    var buf bytes.Buffer
    if json.Indent(&buf, data, "", "  ") == nil {
        return buf.String()
    }
    return string(data)
}
```

### 4. Log levels in use

| Level | Controlling flag | Usage |
| ----- | ---------------- | ----- |
| DEBUG | `--debug` | Full HTTP request/response traces (replaces `debugPrintHTTP` direct writes) |
| INFO | `--verbose` | Batch/job progress messages (commands may continue using `fmt.Fprintf` to stdout for structured progress; slog INFO for stderr side-channel messages) |
| WARN | (always active) | Timeout warnings, partial job failures - already printed via `cmd.ErrOrStderr()` |
| ERROR | (always active) | Not used directly; errors surface via Cobra `RunE` return values |

---

## Scope

### Files to MODIFY

```text
cmd/root.go                # add ts(), initLogger(), logLevel(), var logger *slog.Logger; hook initLogger() in PersistentPreRunE
cmd/import_.go             # remove ts() definition; update ~8 ts() call sites to new format
cmd/export.go              # update ~11 ts() call sites to new format
cmd/publish.go             # update ~7 ts() call sites to new format
cmd/unpublish.go           # update ~3 ts() call sites to new format
internal/client/client.go  # replace debugPrintHTTP fmt.Fprintf calls with slog.Default(); add prettyJSONString helper
```

### Files to READ (context only - do NOT modify)

```text
cmd/root.go                # PersistentPreRunE, getClient, verbose/debug package-level vars
cmd/import_.go             # ts() at line 494; all ts() call sites use "[%s]" + ts() pattern
cmd/export.go              # ts() call sites
cmd/publish.go             # ts() call sites; note [%s] bracket pattern
cmd/unpublish.go           # ts() call sites
internal/client/client.go  # debugPrintHTTP at line 214; sessionHeaderV2/V3 constants
```

---

## Call-site Migration Pattern

Current pattern (4 files, ~29 occurrences):

```go
_, _ = fmt.Fprintf(out, "[%s] Some message: %s\n", ts(), value)
```

New pattern after `ts()` returns brackets in its value:

```go
_, _ = fmt.Fprintf(out, "%s Some message: %s\n", ts(), value)
```

The only change per line is removing the `[` and `]` from the format string.

---

## Verification

```bash
/opt/local/bin/go build ./...
/opt/local/bin/go vet ./...
/opt/local/bin/go test ./...

# Confirm timestamp format on verbose progress
iics publish run --from-file assets.csv --verbose 2>&1 | head -5
# Expected: [2026-03-22 14:52:41.123] Publishing 10 assets in 1 batch(es)...

# Confirm HTTP debug output via slog
iics --debug objects list 2>&1 | head -20
# Expected: time=[2026-03-22 14:52:41.123] level=DEBUG msg=http method=GET url=...
```
