
# BUG: Invalid request in export run

## Symptoms

Getting error when executing the following

```shell
/iics objects list -q "location==ZZ_TEST_CLI" -o csv | ./iics export run --export-file-path ~/tmp/myExport.zip --verbose
[23:57:26] Read 14 artifact entries
[23:57:26] Export request: name="iics-cli(version:f7bee6c-dirty) 2026-03-09 23-57-26", objects=14, includeDeps=true
  - 1vRc0Rb7ua4kT8ccjL6sKf (includeDeps: true)
  - 8aJAqowEF2phOFeUtU8TuS (includeDeps: true)
  - aLX7qnviqxJdmqpVsd17SG (includeDeps: true)
  - guoHBxV3L8GdMprXqqSdKH (includeDeps: true)
  - 3OrYJxTOzUJdwgHskwZCcj (includeDeps: true)
  - 9k4OaSJhnYyd4g0NUDlE6s (includeDeps: true)
  - 2evb0eclhUEfvPKOyj794R (includeDeps: true)
  - 6diNoBL2NEqlCYHhYgc68O (includeDeps: true)
  - 191ZGSrZLeTgtPaXNRQpWV (includeDeps: true)
  - 1yeVVpAOUcHioT3sSTii2G (includeDeps: true)
  - 3N4yMZtC2GBhTfdAhT3yG0 (includeDeps: true)
  - 2mGokcuTPCjdbdeyCmzFKM (includeDeps: true)
  - 7fDEP5uXWTWdA9vmHsb7Ex (includeDeps: true)
  - 8XK79DgSovVblOcwrOWF62 (includeDeps: true)
[23:57:26] Starting export job...
Error: IICS API error (HTTP 400): 
HTTP 400 Bad Request

Response Headers:
  Access-Control-Allow-Credentials: true
  Connection: keep-alive
  Content-Security-Policy: default-src 'self'; font-src https: data: 'self'; img-src https: data: 'self'; script-src https: 'unsafe-eval' 'unsafe-inline' 'self'; style-src https: 'unsafe-inline' 'self'; frame-ancestors 'self'; frame-src https: 'self'; object-src; connect-src https:
  Content-Type: application/json
  Date: Tue, 10 Mar 2026 03:57:26 GMT
  Referrer-Policy: strict-origin-when-cross-origin
  Server: istio-envoy
  Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
  X-Content-Type-Options: nosniff
  X-Envoy-Upstream-Service-Time: 3
  X-Frame-Options: SAMEORIGIN
  X-Xss-Protection: 1; mode=block

Response Body:
  {
    "error": {
      "code": "V3API_007",
      "message": "Bad Request. Please check your request.",
      "requestId": "a6C8FkgTv8LbCGyXu6Tj0G",
      "details": null
    }
  }
```

## Likely affected files

- internal/client/export.go

## Error Message

```text
Bad Request. Please check your request.
```

## Fix Instructions

- Review the [API documentation ] for start export job
- Update the Command implementation to match the payload of the API
- Update Documentation
- Commit Changes

## Fix

**Applied 2026-03-10** — removed top-level `includeDependencies` from `ExportRequest` in
`internal/client/export.go`. The IICS v3 export API does not accept `includeDependencies` at the
request root; it is valid only per-object inside the `objects` array.

Also added global `--debug` flag (`cmd/root.go` + `internal/client/client.go`) which prints the
full JSON request body to stderr on any API error — making future payload mismatches visible without
manual logging. Use it as:

```shell
./iics export run --artifacts-file ... --export-file-path ... --debug
```

**Needs live re-test** against ZZ_TEST_CLI to confirm the 400 is resolved. If it persists, the
`--debug` output will show the exact payload being sent.
