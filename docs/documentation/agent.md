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
| `installer-info`     | Get Secure Agent installer download information |
| `installer-download` | Download the Secure Agent installer (optional checksum verify) |

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

---

## agent installer-info

Retrieve the Secure Agent installer download URL, install token, and checksum download URL
for a platform.

### Flags

| Flag   | Type   | Required | Description                          |
| ------ | ------ | -------- | ------------------------------------ |
| `--os` | string | yes      | Operating system: `win64` or `linux64` |

When `--os` is omitted and the session is interactive, the command prompts for it.

All [global flags](../../README.md#global-flags) apply.

### Output

By default the fields are printed as a vertical `FIELD`/`VALUE` table. With
`--output json`, `--output yaml`, or `--output csv` the raw response is rendered.

| Field                 | Description                                      |
| --------------------- | ------------------------------------------------ |
| `type`                | Resource type (`agentInstallerInfo`)             |
| `downloadUrl`         | URL of the Secure Agent installer binary         |
| `checksumDownloadUrl` | URL of the installer SHA-256 checksum file       |
| `installToken`        | Token used to register the agent after install   |

### Examples

```bash
iics agent installer-info --os win64

iics agent installer-info --os linux64 --output json
```

```powershell
iics agent installer-info --os win64

iics agent installer-info --os linux64 --output json
```

---

## agent installer-download

Download the Secure Agent installer binary, optionally verifying it against the
published SHA-256 checksum.

Installer info can be supplied three ways:

1. Piped as JSON on stdin (for example from `iics agent installer-info --output json`).
2. From a file via `--installer-info <path>`.
3. Omitted entirely, in which case the info is requested for the default or
   `--profile` account using `--os` (prompted when interactive).

### Flags

| Flag               | Type   | Default              | Description                                                        |
| ------------------ | ------ | -------------------- | ------------------------------------------------------------------ |
| `--os`             | string |                      | Operating system: `win64` or `linux64` (used when no info supplied) |
| `--installer-info` | string |                      | Path to installer info JSON (omit to read piped stdin)             |
| `--target`         | string | system temp directory | Destination file or directory                                     |
| `--verify`         | bool   | false                | Download the checksum file and verify the installer               |

`--target` behavior:

- Not provided: the file is written to the system temp directory using the file
  name from the installer metadata.
- An existing directory or a value ending with a path separator: the metadata file
  name is appended.
- Any other value: treated as the full destination file path.

All [global flags](../../README.md#global-flags) apply.

### Output

By default a vertical `FIELD`/`VALUE` table describes the downloaded file. With
`--verify` the expected and actual checksums and the `verified` result are included.
Use `--verbose` to print download and verification progress. `--output json`,
`yaml`, and `csv` render the structured result.

| Field                 | Description                                       |
| --------------------- | ------------------------------------------------- |
| `file`                | Full path of the downloaded file                  |
| `fileName`            | Installer file name                               |
| `size`                | Downloaded size in bytes                          |
| `downloadUrl`         | Source URL                                        |
| `checksumDownloadUrl` | Checksum file URL (when known)                    |
| `expectedChecksum`    | SHA-256 digest from the checksum file (`--verify`) |
| `actualChecksum`      | SHA-256 digest of the downloaded file (`--verify`) |
| `verified`            | Whether the checksums matched (`--verify`)        |

If the checksums do not match, the partial download is removed and the command
exits with an error.

### Examples

```bash
# Request info and download to a directory
iics agent installer-download --os linux64 --target ./downloads/

# Pipe info from installer-info and verify
iics agent installer-info --os win64 --output json | iics agent installer-download --verify

# Use a saved info file and an explicit file path
iics agent installer-download --installer-info info.json --target /tmp/agent.exe --verify
```

```powershell
iics agent installer-download --os linux64 --target ./downloads/

iics agent installer-info --os win64 --output json | iics agent installer-download --verify

iics agent installer-download --installer-info info.json --target C:\Temp\agent.exe --verify
```

## See also

- [runtime](runtime.md) - manage runtime environments that contain agents
