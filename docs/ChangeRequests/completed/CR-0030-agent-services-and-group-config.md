# CR-0030: Fix agent start/stop + Secure Agent group service properties (Phase 2)

## CR Type

- [x] Bug fix (`agent start` / `agent stop` use a 404 endpoint)
- [x] Enhancement to existing command (`runtime`)

## Problem

- `agent start` / `agent stop` `POST public/core/v3/agent/<id>/services`, which
  returns HTTP 404. The command reports success while doing nothing.
- There is no way to read or set Secure Agent **group** (runtime environment)
  service properties, which is how Secure Agent service configuration is managed
  in IICS.

## Verified API facts (live, 2026-09-09)

- Start/stop: `POST public/core/v3/agent/service` (`allow: POST`), body
  `{"agentId":"<id>","serviceName":"<service display name>","serviceAction":"start"|"stop"}`.
  Endpoint and body confirmed against the official docs and by schema probing
  (renaming or dropping any of the three fields returns `V3API_007`). The old
  `.../services` path 404s.
  RESOLVED (CR-0035): the `agentId` field must be the agent's **federatedId**,
  not its v2 `id`. Passing the federatedId works end-to-end (start/stop verified
  live). The `serviceName` is the service display name. Fixed in
  `SetAgentServiceState` + `cmd/agent.go`.
- Group service properties: `GET` / `PUT api/v2/runtimeEnvironment/<id>/configs`.
  GET returns `{}` when the group has no property overrides. PUT body is keyed by
  service name: `{"<Service_Name>":[{...settings...}]}`.

## Scope

### `agent start` / `agent stop`

- Replace `StartAgentService` / `StopAgentService` with one
  `AgentServiceAction(ctx, agentID, serviceName string, action)` posting to the
  correct v3 endpoint and body.
- Accept `--id | --name | --hostname` (resolved via `resolveAgentID`) plus the
  existing `--service`.

### `runtime configs`

New subcommand group:

- `runtime configs get --id <groupId> [--name <envName>] [--service <name>]` -
  `GET .../runtimeEnvironment/<id>/configs`. Table mode prints one section per
  service key; `--service` narrows to one. `--output json|yaml` prints the raw
  document.
- `runtime configs set --id <groupId> --from-file <json> [--yes]` -
  `PUT .../runtimeEnvironment/<id>/configs` with the file contents. Confirms
  unless `--yes`.

`--name` resolves to an ID via `GetRuntimeEnvironmentByName`.

## Out of scope

- Per-agent config writes - no such endpoint exists (confirmed with developer);
  agent-level config is visible via `agent details --full` (CR-0029).
- Runtime environment detail polish / shared KV helper refactor - CR-0031.

## Implementation

- `internal/client/agents.go` - replace the two service methods.
- `internal/client/agentconfigs.go` (+ test) - `AgentGroupServiceConfig`,
  `GetAgentGroupConfigs`, `UpdateAgentGroupConfigs`.
- `cmd/agent.go` - update start/stop wiring.
- `cmd/runtime.go` - `runtime configs get|set`.
- Docs: `docs/documentation/agent.md`, `docs/documentation/runtime.md`,
  `README.md`, `make completions`.
