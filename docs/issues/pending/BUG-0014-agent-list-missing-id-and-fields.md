# BUG-0014: agent list drops the ID column and omits API fields in json/yaml

## Status

- [x] Fixed (pending developer confirmation)

## Symptoms

Introduced by CR-0029 (commit `1666e5a`):

1. `iics agent list` with no `--fields` no longer shows the agent `id`. The
   pre-CR-0029 default table led with the ID column.
2. `iics agent list -o json` / `-o yaml` omits fields the API returns:
   `@type`, `serverUrl`, `spiUrl`, `federatedId`, `createTimeUTC`,
   `updateTimeUTC`.

## Cause

1. `agentListDefaultFields` (`cmd/agent.go`) was set to
   `name,agentHost,active,readyToRun,platform,agentVersion,upgradeStatus,agentGroupId`
   with no `id`.
2. The `client.Agent` struct (`internal/client/agents.go`) never modeled the
   extra keys. The client unmarshals the `GET /api/v2/agent` array into
   `[]Agent` and re-marshals it for output, so unmapped fields are lost.

## Fix

- `client.Agent`: add `Type` (`@type`), `ServerURL` (`serverUrl`), `SpiURL`
  (`spiUrl`), `FederatedID` (`federatedId`), `CreateTimeUTC`, `UpdateTimeUTC`.
  Remove the now-redundant `Type` / `ServerURL` from `AgentDetails` (inherited
  from the embedded `Agent`).
- `FindAgent`: the `--fid` path now scans `ListAgents` directly (the list
  carries `federatedId`), keeping the runtime-environment scan only as a
  fallback.
- `cmd/agent.go`: `agentListDefaultFields` -> `id,name,agentHost,active,readyToRun,platform,agentVersion,agentGroupId`;
  add `federatedId` / `serverUrl` / `spiUrl` / `createTimeUTC` / `updateTimeUTC`
  to `agentColumnMap`; add `federatedId` / `spiUrl` / `serverUrl` to
  `agentToKVRows`.
- Docs: `docs/documentation/agent.md` (default columns, `--fields` names,
  `--fid` resolution note).

## Verification

```bash
iics agent list | head -3                    # ID column present
iics agent list -o json | jq '.[0] | keys'   # @type, spiUrl, federatedId, UTC times
iics agent list --fields id,name,federatedId,spiUrl
iics agent get --name <agent> | grep -E 'federatedId|spiUrl'
iics agent details --fid <federatedId>        # still resolves
```
