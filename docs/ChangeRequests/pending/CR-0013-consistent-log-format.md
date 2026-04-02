# CR-0013: Consistent Log Format Across All Commands

| Field     | Value                                                         |
| --------- | ------------------------------------------------------------- |
| ID        | CR-0013                                                       |
| Status    | New                                                           |
| Priority  | Medium                                                        |
| Component | `cmd/root.go`, `internal/client/client.go`, `cmd/publish.go`, `cmd/unpublish.go`, `cmd/export.go`, `cmd/import_.go` |

---

## Summary

Three independent logging mechanisms produce inconsistent output format:

1. `slog` (structured logging in `cmd/package.go` and others):
   `time="[2026-04-01 23:02:03.971]" level=INFO msg="..." key=value`

2. Debug HTTP trace (`internal/client/client.go` `debugPrintHTTP`):
   `DEBUG < 200 OK`

3. Verbose progress output (`cmd/publish.go`, `cmd/export.go`, `cmd/import_.go`):
   `[2026-04-01 23:02:04.979] Publishing 6 assets in 1 batch(es)...`

Desired unified format: `[timestamp][LEVEL] msg key=value`

Additionally:
- Publish verbose output is overly chatty (5+ separate lines per run) with no key=value structure
- Long-running commands lack elapsed-time measurements in verbose mode
- Verbose progress in publish/export/import writes to stdout; in a piped scenario
  (`package dependencies ... | publish run`) this pollutes the data stream

---

## Issue 1 - slog TextHandler produces `time="..." level=... msg="..."` format

### Observed behaviour

```
time="[2026-04-01 23:02:03.971]" level=INFO msg="resolving dependencies: phase 1 - scanning package objects" total=25 publishMode=true
time="[2026-04-01 23:02:03.971]" level=INFO msg="resolving dependencies: phase 1 complete" packageObjects=6 externalGUIDs=0
time="[2026-04-01 23:02:03.971]" level=INFO msg="validating dependencies against target org" profile=dev count=6
```

### Root cause

`initLogger` in `cmd/root.go` uses `slog.NewTextHandler` with a custom `ReplaceAttr` for the
time value. `slog.TextHandler` always formats output as `key="value" key=value ...` pairs in
fixed order (time, level, msg, then user attrs). There is no way to produce `[ts][LEVEL] msg`
using `ReplaceAttr` alone.

### Proposed fix

Replace `slog.NewTextHandler` with a minimal custom handler `cliHandler` in `cmd/root.go`
that produces:

```
[2026-04-01 23:02:03.971][INFO] resolving dependencies: phase 1 - scanning package objects total=25 publishMode=true
[2026-04-01 23:02:03.971][INFO] resolving dependencies: phase 1 complete packageObjects=6 externalGUIDs=0
[2026-04-01 23:02:03.971][INFO] validating dependencies against target org profile=dev count=6
```

Implementation in `cmd/root.go` (alongside existing `initLogger`):

```go
type cliHandler struct {
    w     io.Writer
    level slog.Level
    mu    sync.Mutex
}

func (h *cliHandler) Enabled(_ context.Context, level slog.Level) bool {
    return level >= h.level
}

func (h *cliHandler) Handle(_ context.Context, r slog.Record) error {
    var sb strings.Builder
    sb.WriteString(r.Time.Format("[2006-01-02 15:04:05.000]"))
    sb.WriteString("[")
    sb.WriteString(r.Level.String())
    sb.WriteString("] ")
    sb.WriteString(r.Message)
    r.Attrs(func(a slog.Attr) bool {
        sb.WriteString(" ")
        sb.WriteString(a.Key)
        sb.WriteString("=")
        sb.WriteString(fmt.Sprint(a.Value.Any()))
        return true
    })
    sb.WriteString("\n")
    h.mu.Lock()
    defer h.mu.Unlock()
    _, _ = io.WriteString(h.w, sb.String())
    return nil
}

func (h *cliHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *cliHandler) WithGroup(_ string) slog.Handler      { return h }
```

Update `initLogger`:

```go
func initLogger() {
    h := &cliHandler{w: os.Stderr, level: logLevel()}
    logger = slog.New(h)
    slog.SetDefault(logger)
}
```

Add imports `"io"`, `"strings"`, `"sync"` to `cmd/root.go` if not already present.
Remove the now-unused `ReplaceAttr` closure and `time.Now().Format` call from `initLogger`.

---

## Issue 2 - Debug HTTP trace has no timestamp and uses plain `DEBUG` prefix

### Observed behaviour

```
DEBUG < 200 OK
Response Headers:
  Content-Type: application/json;charset=UTF-8
  ...
Response Body:
  { ... }
```

### Root cause

`debugPrintHTTP` in `internal/client/client.go` uses bare
`fmt.Fprintf(os.Stderr, "DEBUG < %s\n", resp.Status)`. No timestamp, no bracket-delimited
level label. `internal/client/` cannot import `cmd/` so the `ts()` helper in `cmd/root.go`
is not accessible.

### Proposed fix

In `debugPrintHTTP`, compute a local timestamp and use `[DEBUG]` prefix:

```go
func debugPrintHTTP(req *http.Request, reqData []byte, resp *http.Response, respData []byte) {
    if !slog.Default().Enabled(req.Context(), slog.LevelDebug) {
        return
    }
    now := time.Now().Format("[2006-01-02 15:04:05.000]")
    w := os.Stderr
    _, _ = fmt.Fprintf(w, "%s[DEBUG] > %s %s\n", now, req.Method, req.URL)
    _, _ = fmt.Fprintf(w, "%s[DEBUG] Request Headers:\n", now)
    // ... headers ...
    _, _ = fmt.Fprintf(w, "%s[DEBUG] < HTTP %s\n", now, resp.Status)
    _, _ = fmt.Fprintf(w, "%s[DEBUG] Response Headers:\n", now)
    // ... headers + body ...
}
```

`time` is already imported in `internal/client/client.go`. No new imports required.

The `now` variable is captured once at the start of the function call and reused for all
lines belonging to the same request/response pair. Every section header ("Request Headers:",
"Response Headers:", "Response Body:") also gets the `now+"[DEBUG] "` prefix.

---

## Issue 3 - Publish verbose output is chatty and writes to stdout

### Observed behaviour

```
[2026-04-01 23:02:04.979] Publishing 6 assets in 1 batch(es)...
[2026-04-01 23:02:04.979] Submitting batch 1/1 (6 assets)...
[2026-04-01 23:02:05.681] Batch 1/1 job ID: 1224192035368853504
[2026-04-01 23:02:15.718] Published 4 out of 6 asset(s). State: IN_PROGRESS elapsed: 11s
[2026-04-01 23:02:25.748] Published 6 out of 6 asset(s). State: SUCCESS elapsed: 21s
```

These 5 lines use `fmt.Fprintf(out, ...)` where `out` is `cmd.OutOrStdout()`. In a piped
scenario (`package dependencies ... | publish run --verbose`), verbose messages mixed into
stdout contaminate the data stream.

### Proposed fix

Replace `fmt.Fprintf(out, "%s...\n", ts(), ...)` calls in `cmd/publish.go` with `slog.Info`
calls. Slog writes to stderr and uses the new `[timestamp][LEVEL]` format automatically.

Desired output after fix:

```
[2026-04-01 23:02:04.979][INFO] publishing assets count=6 batches=1
[2026-04-01 23:02:05.681][INFO] batch submitted batch=1/1 job=1224192035368853504 assets=6
[2026-04-01 23:02:15.718][INFO] publish progress processed=4 total=6 state=IN_PROGRESS elapsed=11s
[2026-04-01 23:02:25.748][INFO] publish complete processed=6 total=6 state=SUCCESS elapsed=21s
```

Key changes:
- Combine the "Submitting batch N/M (K assets)..." and "Batch N/M job ID: ..." lines into
  one `slog.Info("batch submitted", "batch", batchLabel, "job", jobID, "assets", len(batch))` call
- Use key=value pairs instead of positional text
- `slog.Info` is gated by log level - outputs only when `--verbose` or `--debug` is set
- Writes to stderr (via slog) instead of stdout - safe in piped scenarios

Apply the same pattern to `cmd/unpublish.go`.

---

## Issue 4 - Long-running commands lack timing information in verbose mode

### Observed behaviour

`cmd/export.go`, `cmd/import_.go`, and `cmd/package.go` verbose messages do not report
elapsed time for operations that may take many seconds.

### Proposed fix

Add `start := time.Now()` at the beginning of long-running operations and include
`"elapsed", time.Since(start).Round(time.Millisecond).String()` in the terminal `slog.Info` call.

For `cmd/export.go` and `cmd/import_.go`: switch the existing `fmt.Fprintf + ts()` verbose
lines to `slog.Info` (same as Issue 3), and add timing to completion messages.

For `cmd/package.go` `resolveDependencies` and `validateTargetDependencies`: add
`start := time.Now()` and a completion `slog.Info` with `"elapsed"` at the end of each
function.

`validateTargetDependencies` should also count outcomes after the loop and include them in
the completion message:

```go
var found, missing, unknown int
for _, d := range deps {
    switch d.TargetStatus {
    case "found":
        found++
    case "missing":
        missing++
    default:
        unknown++
    }
}
slog.Info("dependencies validated",
    "profile", targetProfileName,
    "found", found,
    "missing", missing,
    "unknown", unknown,
    "elapsed", time.Since(start).Round(time.Millisecond).String(),
)
```

Expected package.go verbose output after fix:

```
[2026-04-01 23:02:03.971][INFO] resolving dependencies phase=1 total=25 publishMode=true
[2026-04-01 23:02:03.971][INFO] dependencies resolved packageObjects=6 externalGUIDs=0 elapsed=0ms
[2026-04-01 23:02:03.971][INFO] validating dependencies against target org profile=dev count=6
[2026-04-01 23:02:04.250][INFO] dependencies validated profile=dev found=5 missing=1 unknown=0 elapsed=279ms
```

---

## Acceptance Criteria

1. `initLogger` in `cmd/root.go` uses `cliHandler` producing `[timestamp][LEVEL] msg key=value`
2. All existing `slog.Info/Debug/Warn` call sites produce the new format with no changes at call sites
3. `debugPrintHTTP` prefixes every output line with `[timestamp][DEBUG]`; response status line reads `< HTTP STATUS`
4. `cmd/publish.go` and `cmd/unpublish.go` verbose messages use `slog.Info` key=value calls writing to stderr
5. "Submitting batch" and "Batch job ID" lines combined into a single `slog.Info("batch submitted", ...)` call
6. `cmd/export.go` and `cmd/import_.go` verbose progress lines switch to `slog.Info` with elapsed timing
7. `validateTargetDependencies` in `cmd/package.go` emits a completion `slog.Info` with `found`, `missing`, `unknown`, `elapsed`
8. `--verbose` shows INFO-and-above; `--debug` shows DEBUG-and-above (unchanged behaviour)
9. Non-verbose/non-debug runs produce no extra stderr output (no regression)
10. `go build ./...` and `go vet ./...` pass with no issues
11. `golangci-lint run ./...` reports no new issues

---

## Files to Modify

| File | Change |
| ---- | ------ |
| `cmd/root.go` | Add `cliHandler` struct; update `initLogger` to use it; add `io`, `strings`, `sync` imports; remove old `ReplaceAttr` closure |
| `internal/client/client.go` | Add `now` timestamp in `debugPrintHTTP`; prefix lines with `[timestamp][DEBUG]`; change `< STATUS` to `< HTTP STATUS` |
| `cmd/publish.go` | Replace `fmt.Fprintf(out, ts()+...)` with `slog.Info(...)` calls; combine batch/jobID lines |
| `cmd/unpublish.go` | Same as `cmd/publish.go` |
| `cmd/export.go` | Switch verbose `fmt.Fprintf + ts()` calls to `slog.Info`; add `start` and `elapsed` timing |
| `cmd/import_.go` | Switch verbose `fmt.Fprintf + ts()` calls to `slog.Info`; add `start` and `elapsed` timing |
| `cmd/package.go` | Add completion `slog.Info` with found/missing/unknown counts and elapsed to `validateTargetDependencies`; add elapsed to `resolveDependencies` completion message |

## Do NOT

- Add new Go module dependencies
- Change the `ts()` function (still used by `cmd/state.go`)
- Modify `internal/output/`
- Change behaviour of non-verbose, non-debug runs
- Remove the `--debug` HTTP trace feature - only change its format
- Add `Co-Authored-By` trailers to commit messages
