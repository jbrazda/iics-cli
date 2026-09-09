# CR-0031: Runtime environment detail polish + shared vertical-table helper (Phase 3)

## CR Type

- [x] Enhancement to existing command (`runtime`)
- [x] Refactor (consolidate duplicated vertical-table code)
- [x] Documentation fix (`runtime.md` is stale)

## Problem

- `runtime get` (table mode) omits fields that the API returns: `orgUUID`,
  `createTimeUTC` / `updateTimeUTC`, the serverless configuration, and on the
  nested agents `serverUrl`, `federatedId`, `upgradeStatus`, `agentGroupId`.
- The vertical property/value table pattern is re-implemented in three places
  (`cmd/agent.go`, `cmd/agent_installer.go`, `cmd/runtime.go`) with a private
  2-field struct each. CR-0029 added `output.KVRow` / `output.KVCols`; the older
  call sites should use it.
- `docs/documentation/runtime.md` documents `type` / `status` columns that the
  command has not produced since the v2 API fix.

## Scope

1. `internal/output/kv.go` (added in CR-0029) - add `kv_test.go`.
2. Refactor `cmd/agent_installer.go` (`installerInfoRow`) and `cmd/runtime.go`
   (`rtAttr` / `rtAttrCols`) to `output.KVRow` / `output.KVCols`. Leave
   `cmd/user.go` and `cmd/activitylog.go` (out of scope; note for a later pass).
3. `internal/client/runtimes.go` - add `OrgUUID` to `RuntimeEnvironment`.
4. `cmd/runtime.go`:
   - `runtimeEnvAttrs`: add `orgUUID`, `createTimeUTC`, `updateTimeUTC`.
   - `runtime get`: print a `Serverless Config:` section when present.
   - `runtimeAgentCols`: add `serverUrl`, `federatedId`, `upgradeStatus`,
     `agentGroupId` (all verified populated on nested agents).
   - `runtime list`: add `--filter` (client-side, `internal/filter`).
5. Full rewrite of `docs/documentation/runtime.md`; `README.md`; `make completions`.

## Verified (live, 2026-09-09)

Runtime env response keys: `id, orgId, name, createTime, updateTime, createdBy,
updatedBy, agents, isShared, federatedId, createTimeUTC, updateTimeUTC,
serverlessConfig, orgUUID`. Nested agent keys include `serverUrl, federatedId,
upgradeStatus, agentGroupId` (`serverUrl` empty on non-serverless agents; the
rest populated). `serverlessConfig` is present but empty (`{}`) on
non-serverless environments; the `Serverless Config:` section is shown only when
it carries a platform/status/applicationType.

## Known limitation

`--filter` compares against the decoded JSON. A boolean field with a `false`
value is dropped by `omitempty`, so `field==false` does not match. Documented in
both command pages; filter on the value that is present.
