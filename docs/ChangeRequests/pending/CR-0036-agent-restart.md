# CR-0036: `agent restart` + fix interactive `agent start` service selection

## CR Type

- [x] New subcommand
- [x] Enhancement to existing command

## Problem

- No `agent restart`; restarting a service means running `stop` then `start` by
  hand with no wait-for-full-stop.
- Interactive `agent start`'s service picker was built from `agent details`,
  which only returns **running** engines - useless for starting a stopped one.

## API investigation

- No `agent details` query param returns stopped services.
- No API lists an agent's configured/stoppable services by engine display name.
  `GET /api/v2/runtimeEnvironment/<groupId>/selections/details` lists service
  *categories* ("Data Integration", "Data Quality") with an `enabled` flag -
  these do not map cleanly to engine display names ("Data Integration Server",
  "DV Processor").
- `agentEngineStatus.subState`: `"0"` = fully operational; `"-1"` (observed
  during startup) = still initializing. A freshly started engine reports
  `RUNNING` with `subState -1` for ~1-2 minutes, then flips to `0`.

## Solution

### `agent restart`

New `newAgentRestartCmd`. Flags identical to `start`/`stop`
(`--id`/`--name`/`--hostname`, `--service`, `-i`, `--blocking`,
`--poll-interval` 10, `--max-wait-time` 300).

1. Resolve agent + service (interactive service picker uses the *running*
   engines - you can only restart something running).
2. Interactive: confirm; ask `Wait for service "<name>" to start`.
3. Stop -> **always** poll until the service leaves the listing.
4. Start -> poll until `RUNNING` + `subState 0`, if waiting.
5. `--max-wait-time` is **one total deadline** for the whole operation.

### Interactive `agent start`

Replace the running-services picker with a free-text
`Service to start (display name)` prompt, printing `(already running: ...)` for
reference.

### `subState 0` check everywhere

`serviceReached("start", ...)` now requires the `RUNNING` engine's `subState`
to be `"0"`. `agent start --blocking` and `agent restart` both use it.

## Implementation

- `cmd/agent_service.go` (new) - moved the service helpers out of `cmd/agent.go`
  and added `agentServiceFlags` + `bind`, `resolveServiceTarget`,
  `newAgentRestartCmd`, `promptStartService`. `waitForAgentService` now takes an
  explicit `deadline time.Time`.
- `cmd/agent.go` - register `restart`; drop the now-unused `io`/`time`/`config`
  imports.
- Docs: `docs/documentation/agent.md`, `README.md`, `make completions`.

Verified live: full `agent restart --blocking` of `GitRepoConnectApp`
(stop -> wait for gone -> start -> polled through `NEED_RUNNING`/`DEPLOYING`/
`RUNNING subState -1` -> concluded at `RUNNING subState 0`).
