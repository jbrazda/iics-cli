# CR-0029: Secure Agent API Coverage (Phase 1) - list/get/details/delete

## CR Type

- [x] Enhancement to existing command
- [x] Bug fix (agent details struct is wrong; renders empty)

## Problem

`iics agent` exposes only `list / get / details / start / stop`. Gaps and defects:

- `agent details` models `/api/v2/agent/details/<id>` as flat
  `agentEngineStatus[]` + `agentEngineConfigs[]`. The real payload is
  `{ "@type":"agentdetails", packages[], agentConfigs[], agentEngines:[ { agentEngineStatus:{...}, agentEngineConfigs:[...] } ] }`.
  The command prints an empty table.
- `agent get` / `agent details` accept only `--id` and print a wide single-row
  table that is unreadable for a detail view.
- No `agent delete`.
- No client-side filtering or column selection on `agent list`.

## Scope (Phase 1)

1. New `internal/filter` package: client-side `key==value` / `key!=value`
   predicates (repeatable, AND-ed; JSON-tag keys, dot notation;
   case-insensitive string compare; bool/number aware).
2. Multi-line table cell support in `internal/output` + `wrapCell` helper.
3. Rewrite the agent details structs to the verified nested shape; add
   `full bool` to `GetAgentDetails` (adds `onlyStatus=false` only when set).
4. New client methods: `DeleteAgent`, `GetAgentByName`, `FindAgent`
   (`AgentSelector`); `BasicInfo` on `AgentListOptions`.
5. `agent list`: `--fields` (override columns), `--filter` (repeatable),
   `--basic-info`; refined default column set.
6. `agent get`: `--id | --name`; vertical property/value output in table mode.
7. `agent details`: `--id | --fid | --name | --hostname` (only `--id` hits the
   API; the rest resolve via list+match); `--full`; multi-section output - agent
   KV table, optional agent-config table, then per-engine status KV table and
   (with `--full`) an `agentEngineConfigs` table whose `value` / `defaultValue`
   cells wrap onto multiple lines.
8. `agent delete`: `--id | --name | --hostname` + `--yes/-y` confirmation.

## Out of scope (later CRs)

- Fixing `agent start` / `agent stop` (broken v3 path) - CR-0030.
- Secure Agent group service properties - CR-0030.
- Runtime environment detail polish + shared KV helper refactor - CR-0031.

## Verified API facts (live, 2026-09-09)

- `GET /api/v2/agent` - `limit` / `skip` query params are ignored by the server
  (returns all); `basicInfo`, `includeUnassignedOnly` work.
- `GET /api/v2/agent/name/<name>` works.
- `GET /api/v2/agent/details/<id>` - omitting `onlyStatus` == `onlyStatus=true`
  == status only; `onlyStatus=false` includes `agentConfigs` +
  per-engine `agentEngineConfigs`. `<groupId>` -> HTTP 403.
- `DELETE /api/v2/agent/<id>`.
- Agent list objects do not carry `federatedId`; `runtimeEnvironment.agents[]` do.

## Docs

`docs/documentation/agent.md`, `README.md`, `make completions`.
