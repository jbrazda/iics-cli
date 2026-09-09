# BUG-0015: runtime create/update send no @type, API returns APP_20149

## Status

- [x] Fixed (create) - pending developer confirmation
- [ ] `runtime update` still returns a server-side 500 (see below)

## Symptom

```
$ iics runtime create --from-file TEST_ENV.json
Error: IICS API error APP_20149 (HTTP 400):
  "description": "Missing required request body:Malformed or missing request body."
```

`--debug` shows the request body as `{"name": "TEST_ENV"}`.

## Cause

`cmd/runtime.go` `newRuntimeCreateCmd` unmarshals the `--from-file` JSON into
`client.RuntimeEnvironment` and `CreateRuntimeEnvironment`
(`internal/client/runtimes.go`) re-marshals it. The struct had **no `@type`
field**, so the discriminator was dropped and the wire body was
`{"name": "TEST_ENV"}`. The v2 `POST /api/v2/runtimeEnvironment` endpoint
requires `"@type": "runtimeEnvironment"` in the body.

Verified live: the same body with `@type` added succeeds (creates the
environment and returns its id).

## Fix

- `internal/client/runtimes.go`: add `Type string json:"@type,omitempty"` to
  `RuntimeEnvironment`; `CreateRuntimeEnvironment` / `UpdateRuntimeEnvironment`
  copy the struct and set `Type = "runtimeEnvironment"` when empty (a user
  `@type` in the file is preserved).
- `internal/client/runtimes_test.go`: assert the request body carries `@type`.
- `docs/documentation/runtime.md`: note the automatic discriminator.

## Known follow-up (not fixed here)

`iics runtime update --id <id> --from-file <file>` now gets past the 400 but the
v2 `PUT /api/v2/runtimeEnvironment/<id>` returns `HTTP 500 CMN_001 "Internal
system error"` even with a full-object body (verified live with a raw request).
Renaming a runtime environment through this endpoint appears unsupported or
broken on the server side. `runtime update` needs separate investigation
(possibly the wrong endpoint/verb, or the operation is not supported by the v2
API). `DELETE /api/v2/runtimeEnvironment/<id>` works (returns 200) - a
`runtime delete` command could be added.

## Related

`connection create` / `connection update` (`internal/client/connections.go`)
have the same shape - a struct with no `@type`, sent via `doJSON`. If the v2
connection endpoints also require `@type: "connection"`, they share this latent
bug. Not investigated here.

## Verification

```bash
iics runtime create --from-file testdata/runtime/TEST_ENV.json   # succeeds
iics runtime get --name TEST_ENV
iics runtime create --from-file testdata/runtime/TEST_ENV.json --debug  # body has @type
# delete the test env via the IICS UI or DELETE /api/v2/runtimeEnvironment/<id>
```
