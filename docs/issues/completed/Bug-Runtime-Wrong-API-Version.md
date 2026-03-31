# Bug: Runtime command uses wrong API version (v3 instead of v2)

## Status

Fixed - pending confirmation

## Summary

The `iics runtime list` (and all `iics runtime` subcommands) failed with HTTP 404 because
the client called `/public/core/v3/runtimeEnvironments`, but the Runtime Environments
resource is only defined in the V2 API. Additionally, the response struct did not match the
actual V2 API response shape, and the display did not show the environment-agents hierarchy.

## Steps to Reproduce

1. Configure a valid IICS profile.
2. Run: `iics runtime list`
3. Observe the error:

```text
Error: IICS API error (HTTP 404):HTTP 404 Not Found
Response Body:
  {
    "type": "about:blank",
    "title": "Not Found",
    "status": 404,
    "detail": "No static resource core/v3/runtimeEnvironments.",
    "instance": "/saas/public/core/v3/runtimeEnvironments",
    "properties": null
  }
```

## Root Cause

In `internal/client/runtimes.go`, all client methods built paths using `BaseAPIPathV3`
(`public/core/v3`). The Runtime Environments resource does not exist in V3 - it is a
V2-only resource. The correct base path is `api/v2` (singular resource name:
`runtimeEnvironment`) and the session header is `icSessionId` (auto-selected by the
client when the URL contains `/v2/`).

Additionally, the `RuntimeEnvironment` struct had stale fields (`status`, `type`,
`description`) that do not appear in the actual API response, and was missing the real
fields: `agents`, `isShared`, `federatedId`, `serverlessConfig`, `createTimeUTC`,
`updateTimeUTC`.

Reference: <https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-2-resources/runtime-environments/getting-runtime-environment-details.html>

## Fix

Changes made:

1. **`internal/client/runtimes.go`** - All API paths changed from
   `BaseAPIPathV3 + "/runtimeEnvironments"` to `BaseAPIPathV2 + "/runtimeEnvironment"`
   (V2 uses singular resource names). Added `GetRuntimeEnvironmentByName` method using
   `/api/v2/runtimeEnvironment/name/<name>`. Structs fully rewritten to match actual API
   response: added `RuntimeEnvironmentAgent`, `ServerlessConfig`, `CloudProviderConfig`;
   updated `RuntimeEnvironment` with correct fields.

2. **`internal/client/runtimes_test.go`** - Tests updated to assert V2 paths; new test
   `TestGetRuntimeEnvironmentByName` added; test data updated to use real struct shape.

3. **`cmd/runtime.go`** - `runtime list` columns updated to show `federatedId`, agent
   count, `isShared`, `updateTime` (removed non-existent `type`/`status` columns).
   `runtime get` now supports `--name` (mutually exclusive with `--id`) and renders a
   tree-style detail view: environment attributes as a KV table followed by an agents
   table with color-coded `ACTIVE` (green/red) and `READY` (green/yellow) columns.

## Files Affected

- `internal/client/runtimes.go`
- `internal/client/runtimes_test.go`
- `cmd/runtime.go`
