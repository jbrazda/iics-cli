# agent

Manage IICS Secure Agents. Agents are on-premises processes that execute data integration tasks.

## Synopsis

```bash
iics agent <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                          |
| ---------- | ------------------------------------ |
| `list`     | List Secure Agents                   |
| `get`      | Get a single agent                   |
| `details`  | Get agent service engine details     |
| `start`    | Start an agent service               |
| `stop`     | Stop an agent service                |

---

## agent list

### Flags

| Flag           | Type | Default | Description                             |
| -------------- | ---- | ------- | --------------------------------------- |
| `--limit`      | int  | 200     | Max results                             |
| `--skip`       | int  | 0       | Results to skip                         |
| `--unassigned` | bool | false   | Include only agents not in a group      |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column         | Description                        |
| -------------- | ---------------------------------- |
| `id`           | Agent ID                           |
| `name`         | Agent name                         |
| `agentHost`    | Hostname of the agent machine      |
| `active`       | Whether the agent is active        |
| `readyToRun`   | Whether the agent can run tasks    |
| `platform`     | OS platform (linux64, win64, etc.) |
| `agentVersion` | Installed agent version            |
| `agentGroupId` | Runtime environment group ID       |

### Examples

```bash
iics agent list

iics agent list --output json

# Find agents that are active and ready
iics agent list --output json | jq '.[] | select(.active == true and .readyToRun == true)'

# List unassigned agents
iics agent list --unassigned
```

```powershell
iics agent list

iics agent list --output json

# Find agents that are active and ready
$agents = iics agent list --output json | ConvertFrom-Json
$agents | Where-Object { $_.active -eq $true -and $_.readyToRun -eq $true }

# List unassigned agents
iics agent list --unassigned
```

---

## agent get

Get full details for a single Secure Agent.

### Flags

| Flag   | Type   | Required | Description |
| ------ | ------ | -------- | ----------- |
| `--id` | string | yes      | Agent ID    |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column          | Description                     |
| --------------- | ------------------------------- |
| `id`            | Agent ID                        |
| `name`          | Agent name                      |
| `agentHost`     | Hostname                        |
| `active`        | Active status                   |
| `readyToRun`    | Ready-to-run status             |
| `platform`      | OS platform                     |
| `agentVersion`  | Agent version string            |
| `upgradeStatus` | Upgrade status                  |
| `agentGroupId`  | Runtime environment group ID    |
| `createdBy`     | Creator                         |
| `createTime`    | Creation timestamp              |
| `updateTime`    | Last modification timestamp     |

### Examples

```bash
iics agent get --id <agent-id>

iics agent get --id <agent-id> --output json
```

```powershell
iics agent get --id <agent-id>

iics agent get --id <agent-id> --output json
```

---

## agent details

Get the service engine details for an agent, including per-service status.

### Flags

| Flag   | Type   | Required | Description |
| ------ | ------ | -------- | ----------- |
| `--id` | string | yes      | Agent ID    |

All [global flags](../../README.md#global-flags) apply.

### Output

Prints a summary of the agent followed by a table of services:

| Column           | Description                       |
| ---------------- | --------------------------------- |
| `appDisplayName` | Service display name              |
| `appname`        | Internal service name             |
| `appversion`     | Service version                   |
| `status`         | Current status (running, stopped) |
| `subState`       | Sub-state detail                  |

### Examples

```bash
iics agent details --id <agent-id>

# Show all service statuses for a specific agent
iics agent details --id <agent-id> --verbose
```

```powershell
iics agent details --id <agent-id>

# Show all service statuses for a specific agent
iics agent details --id <agent-id> --verbose
```

---

## agent start

Start a specific service on a Secure Agent.

### Flags

| Flag        | Type   | Required | Description                    |
| ----------- | ------ | -------- | ------------------------------ |
| `--id`      | string | yes      | Agent ID                       |
| `--service` | string | yes      | Service name to start          |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Start the Data Integration Server service
iics agent start --id <agent-id> --service "Data Integration Server"
```

```powershell
# Start the Data Integration Server service
iics agent start --id <agent-id> --service "Data Integration Server"
```

---

## agent stop

Stop a specific service on a Secure Agent.

### Flags

| Flag        | Type   | Required | Description                  |
| ----------- | ------ | -------- | ---------------------------- |
| `--id`      | string | yes      | Agent ID                     |
| `--service` | string | yes      | Service name to stop         |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics agent stop --id <agent-id> --service "Data Integration Server"

# Restart a service (stop then start)
iics agent stop  --id <agent-id> --service "Data Integration Server"
iics agent start --id <agent-id> --service "Data Integration Server"
```

```powershell
iics agent stop --id <agent-id> --service "Data Integration Server"

# Restart a service (stop then start)
iics agent stop  --id <agent-id> --service "Data Integration Server"
iics agent start --id <agent-id> --service "Data Integration Server"
```

## See also

- [runtime](runtime.md) - manage runtime environments that contain agents
