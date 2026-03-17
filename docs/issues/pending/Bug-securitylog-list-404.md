# BUG: securitylog list fails with HTTP 404

---

## Symptoms

`iics securitylog list` exits with HTTP 404. The API reports the endpoint path
does not exist.

---

## Command / Reproduction Steps

```bash
iics securitylog list --profile dev
```

---

## Expected Behaviour

A table of security log entries is printed.

---

## Actual Behaviour

```text
Error: IICS API error (HTTP 404):
HTTP 404 Not Found

Response Body:
  {
    "type": "about:blank",
    "title": "Not Found",
    "status": 404,
    "detail": "No static resource core/v3/securityLogs.",
    "instance": "/saas/public/core/v3/securityLogs",
    "properties": null
  }
```

---

## Environment

| Field                      | Value         |
| -------------------------- | ------------- |
| OS                         | macOS 25.3.0  |
| `iics --version`           | dev build     |
| Go version                 | 1.25.0        |
| IICS region                | US            |
| Output format (`--output`) | table         |

---

## Architecture Layer

- [x] **`internal/client/`** - HTTP logic, API structs, request/response handling

---

## Likely Affected Files

```text
internal/client/securitylogs.go
```

---

## API Details

| Field          | Value                                 |
| -------------- | ------------------------------------- |
| API version    | V3 (`public/core/v3`)                 |
| HTTP method    | GET                                   |
| Endpoint path  | `public/core/v3/securityLogs` (wrong) |
| Session header | `INFA-SESSION-ID`                     |

The client currently constructs the URL as `BaseAPIPathV3 + "/securityLogs"`
(`public/core/v3/securityLogs`). The API returns 404 with the detail
`"No static resource core/v3/securityLogs"`, indicating the path is incorrect.

**Docs reference:** `https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/security-logs.html`

The correct endpoint path must be verified against the API documentation before
applying the fix.

---

## Error Message

```text
Error: IICS API error (HTTP 404):
HTTP 404 Not Found
detail: "No static resource core/v3/securityLogs."
instance: "/saas/public/core/v3/securityLogs"
```

---

## Fix Instructions

1. Read `docs/CLAUDE.md` and `internal/client/securitylogs.go` before writing any code.
2. Fetch the Informatica security logs API docs at the URL above and identify the correct
   endpoint path (e.g., `securityLog`, `activityLog`, or other).
3. In `internal/client/securitylogs.go:50`, update the path in `doJSONWithQuery` to use
   the correct endpoint.
4. Also verify whether the list response is a plain array or wrapped in an object (same
   class of bug as the schedules fix in this repo - see `scheduleListResponse`).
5. Add or update tests in `internal/client/securitylogs_test.go` to cover the fixed case.
6. Run `/opt/local/bin/go test ./internal/client/...` and verify it passes.
7. Run `/opt/local/bin/go build ./...` to confirm no compilation errors.

---

## Fix (filled in after resolution)

**Root cause:**

Three issues in `internal/client/securitylogs.go`:

1. Wrong endpoint path: `/securityLogs` should be `/securityLog` (singular, lowercase L).
2. List response decoded into `[]SecurityLog` but API wraps the array in `{"entries": [...]}`.
3. Struct fields did not match API field names: `userName`/`action`/`objectType` →
   `actor`/`actionEvent`/`actionCategory`. Fields `status`, `sourceIp`, `additionalInfo`
   are not returned by the API. Added `objectId`. Filter params changed from `startTime`/
   `endTime` query params to a `q` filter string (`entryTime>="...";entryTime<="..."`).

**Files changed:**

```text
internal/client/securitylogs.go - corrected endpoint, wrapper struct, struct fields, query building
cmd/securitylog.go              - updated output columns to match new field names
```

---

## Acceptance Criteria

- [x] `iics securitylog list --profile dev` returns a table of entries without error
- [x] All existing tests still pass (`/opt/local/bin/go test ./...`)
- [x] No unrelated code is refactored
- [x] `go vet ./...` and `golangci-lint run ./...` report no new issues

---

## Do NOT

- Refactor, reformat, or add comments to code outside the fix scope
- Guess the correct endpoint path - verify against the API docs
- Change function signatures or struct names not directly involved in the bug
- Switch tablewriter to v0.x API patterns
- Call `os.Exit()` - always return errors from `RunE`
