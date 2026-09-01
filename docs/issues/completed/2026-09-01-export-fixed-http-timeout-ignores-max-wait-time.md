# BUG: `export run` fails with "context deadline exceeded" regardless of `--max-wait-time`

---

## Symptoms

`iics export run` fails immediately (before job polling ever starts) with:

```
Error: starting export: Post "https://use4.dm-us.informaticacloud.com/saas/public/core/v3/export?includeTagInformation=true": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
```

even when the user passes a large `--max-wait-time` (e.g. `3600`), because that flag has no
effect on this failure.

---

## Command / Reproduction Steps

```bash
iics export run \
  --profile "dev" \
  --artifacts-file "./conf/export_list_processes_dev_all.txt" \
  --export-file-path "./target/Dev_processes.zip" \
  --name "DEV_Process.zip" \
  --max-wait-time "3600" \
  --polling-interval "10" \
  --include-tags \
  --verbose
```

Reproduces when the server takes longer than 60 seconds to respond to the initial
`POST /export?includeTagInformation=true` call - typically with a large artifact list combined
with `--include-tags`.

---

## Expected Behaviour

The initial "start export" request should honor a configurable timeout, and `--max-wait-time`
should be documented/understood as governing only the job-status polling loop, not the
individual HTTP request timeout.

---

## Actual Behaviour

`internal/client/client.go`'s `NewClient()` hardcoded `httpClient: &http.Client{Timeout: 60 * time.Second}`.
This timeout applies to **every** HTTP request made by the client (login, `StartExport`,
`GetExportStatus` polling, package/log downloads) and was not configurable. `--max-wait-time`
(`cmd/export.go`, `newExportRunCmd`) only bounds the polling loop timer and is applied *after*
`StartExport` already returned - it can never prevent this failure.

---

## Environment

| Field                      | Value                                               |
| -------------------------- | ---------------------------------------------------- |
| Command                    | `iics export run`                                    |
| Region                     | USE4                                                 |

---

## Architecture Layer

- [ ] **`cmd/`** - flag parsing, command wiring, output formatting
- [x] **`internal/client/`** - HTTP logic, API structs, request/response handling
- [x] **`internal/config/`** - config file loading, session cache

---

## Fix

**Root cause:**

`internal/client/client.go` hardcoded a 60s `http.Client.Timeout` for all requests, with no way
to configure it, and unrelated to `--max-wait-time`.

**Files changed:**

```text
internal/config/config.go       - Added Config.HTTPTimeout field and ResolveHTTPTimeoutSeconds()
                                   helper (flag > IICS_HTTP_TIMEOUT env > config > default 120s)
internal/config/config_test.go  - Added TestResolveHTTPTimeoutSeconds covering precedence
cmd/root.go                     - Added --http-timeout persistent flag; getClient() now builds
                                   the HTTP client with the resolved timeout via WithHTTPClient
internal/client/client.go       - Bumped fallback default timeout from 60s to 120s
docs/documentation/export.md    - Documented --http-timeout vs --max-wait-time distinction
docs/documentation/import.md    - Documented --http-timeout vs --max-wait-time distinction
README.md                       - Documented httpTimeout config field, --http-timeout flag,
                                   and IICS_HTTP_TIMEOUT env var
docs/CLAUDE.md                  - Updated Global Flags and Configuration System sections
completions/                    - Regenerated via `make completions`
```

**Test added / updated:**

```text
internal/config/config_test.go - TestResolveHTTPTimeoutSeconds (flag/env/config/default precedence)
```
