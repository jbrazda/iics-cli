# Bug: Publish/Unpublish API fails with HTTP 500 - missing Accept header

## Summary

`publish run` and `unpublish run` fail with HTTP 500 `javax.ws.rs.NotAcceptableException: No match for accept header`
because `doCAIJSON` sets `Content-Type: application/vnd.api+json` but does not set the required
`Accept: application/vnd.api+json` header.

## Error

```text
Error: IICS API error (HTTP 500):
HTTP 500 Internal Server Error
...
  {
    "errors": [
      {
        "id": "req_uid=d11d8016-f6e7-40cc-99df-971561321a22",
        "status": "500",
        "code": "INTERNAL_ERROR",
        "title": "Internal Error",
        "detail": " javax.ws.rs.NotAcceptableException:RESTEASY003635: No match for accept header"
      }
    ]
  }
```

## Root Cause

`internal/client/client.go` - `doCAIJSON` method (line 338) only sets:

```
Content-Type: application/vnd.api+json
```

The CAI REST API requires both:

```
Content-Type: application/vnd.api+json
Accept: application/vnd.api+json
```

Without the `Accept` header the HTTP client defaults to `*/*`, which the JAX-RS server rejects
because no media type negotiation match is found.

## Affected Commands

- `publish run`
- `publish start`
- `publish status`
- `unpublish run`
- `unpublish start`
- `unpublish status`

## Fix

In `doCAIJSON` (`internal/client/client.go`), add:

```go
req.Header.Set("Accept", "application/vnd.api+json")
```

immediately after (or alongside) the existing `Content-Type` line.

## Steps to Reproduce

```bash
iics publish run --from-file testdata/publish/publish_list.csv
```

## Expected Behaviour

Publish job starts successfully and returns a job ID.

## Actual Behaviour

HTTP 500 with `NotAcceptableException` from the CAI server.
