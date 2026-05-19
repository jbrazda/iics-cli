---
id: BUG-0004
title: user get --id returns HTTP 405 Method Not Allowed
status: pending
priority: medium
affects: internal/client/users.go, cmd/user.go
---

# BUG-0004: user get --id returns HTTP 405 Method Not Allowed

## Summary

`iics user get --id <id>` fails with HTTP 405 Method Not Allowed. The response header
`Allow: DELETE` indicates that `GET /public/core/v3/users/{id}` is not a valid endpoint -
only `DELETE` is supported at that path. The `GetUser()` implementation is using the wrong
HTTP method or URL for retrieving a single user by ID.

## Observed Behavior

```
$ ./iics user get --id '9IB4JZitmQDhj2cLTgWS64'
Error: IICS API error (HTTP 405):
HTTP 405 Method Not Allowed

Response Headers:
  Allow: DELETE
  ...

Response Body:
  {
    "error": {
      "message": "Method not supported.",
      ...
    }
  }
```

## Expected Behavior

The command returns the user details for the specified ID, formatted as a table (or JSON/CSV
depending on `--output`).

## Root Cause

`GetUser()` in `internal/client/users.go:76-82` issues:

```
GET /public/core/v3/users/{id}
```

The IICS v3 API does not support `GET` on this path. The `Allow: DELETE` header confirms
that only `DELETE /public/core/v3/users/{id}` is valid. The correct way to fetch a single
user is not yet confirmed - see fix guidance below.

## Fix Guidance

1. Consult the Informatica REST API v3 docs for the Users resource:
   `https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/`

2. Determine the correct approach for fetching a single user by ID. Likely candidates:
   - Filtered list: `GET /public/core/v3/users?q=id=='<id>'`
   - Filtered list: `GET /public/core/v3/users?q=userName=='<userName>'`
   - A different path such as `GET /public/core/v3/users/<userName>`

3. Update `GetUser()` in `internal/client/users.go` to use the verified endpoint and method.

4. If the API only supports lookup by `userName` (not by ID), update `cmd/user.go` to
   expose a `--username` flag instead of or in addition to `--id`, and update
   `GetUser()` accordingly.

5. Update `internal/client/users_test.go` to reflect the corrected endpoint.

## Files Affected

- `internal/client/users.go` - `GetUser()` method (lines 75-82)
- `cmd/user.go` - `newUserGetCmd()` (lines 69-108)
- `internal/client/users_test.go` - test for `GetUser`
