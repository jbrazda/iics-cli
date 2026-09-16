# CR-0037: `agent details --services`

## CR Type

- [x] Enhancement to existing command

## Problem

`agent details` prints the agent summary followed by a vertical status listing
per service (and per-service config tables with `--full`). For a quick overview
of "what is running on this agent" that is a lot of scrolling.

## Proposed Solution

Add `--services` to `agent details`: print only the `Agent:` summary and a
single horizontal `Services (N):` table - one row per service engine with
columns `SERVICE` (`appDisplayName`), `APP NAME` (`appname`), `VERSION`
(`appversion`), `STATUS`, `DESIRED` (`desiredStatus`), `SUBSTATE`, `REPLACE`
(`replacePolicy`), `UPDATED` (`updateTime`).

Mutually exclusive with `--full`. `--output json` / `yaml` are unchanged (full
document). No API change - the data is already in the details response.

## Implementation

- `cmd/agent.go` - `agentServiceCols` var; `--services` flag on
  `newAgentDetailsCmd`; early return in the table branch that prints the summary
  + the one services table; `MarkFlagsMutuallyExclusive("full", "services")`.
- Docs: `docs/documentation/agent.md`.
