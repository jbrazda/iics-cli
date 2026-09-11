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
| `delete`   | Delete a Secure Agent                |
| `start`    | Start an agent service               |
| `stop`     | Stop an agent service                |
| `installer-info`     | Get Secure Agent installer download information |
| `installer-download` | Download the Secure Agent installer (optional checksum verify) |

---

## agent list

### Flags

| Flag           | Type         | Default | Description                                                              |
| -------------- | ------------ | ------- | ---------------------------------------------------------------------- |
| `--unassigned` | bool         | false   | Include only agents not in a group                                      |
| `--basic-info` | bool         | false   | Include package and configuration details in the response              |
| `--fields`     | string       |         | Comma-separated columns to display, by technical field name             |
| `--filter`     | string array |         | Client-side filter, e.g. `agentHost==host01` or `active!=true`. Repeatable; conditions are AND-ed |
| `--limit`      | int          | 200     | Accepted for compatibility; the API currently ignores it               |
| `--skip`       | int          | 0       | Accepted for compatibility; the API currently ignores it               |

All [global flags](../../README.md#global-flags) apply.

### Filtering

`--filter` matches on the raw API field names (the same names accepted by
`--fields`). Only `==` and `!=` are supported. String comparison is
case-insensitive. Booleans and numbers are compared by value. Dot notation
selects nested fields. A boolean field with a `false` value is omitted from the
API response, so `field==false` will not match; filter on the value that is
present (for example `active==true`).

### Output columns

Default columns: `id`, `name`, `agentHost`, `active`, `readyToRun`, `platform`,
`agentVersion`, `agentGroupId`.

`--fields` accepts any of: `id`, `orgId`, `name`, `description`, `agentHost`,
`active`, `readyToRun`, `platform`, `agentVersion`, `upgradeStatus`,
`agentGroupId`, `federatedId`, `serverUrl`, `spiUrl`, `proxyHost`, `createdBy`,
`updatedBy`, `createTime`, `updateTime`, `createTimeUTC`, `updateTimeUTC`,
`lastStatusChange`, `lastUpgraded`, `lastUpgradeCheck`, `configUpdateTime`.
Unknown names are ignored. `--output json` / `yaml` always emit the full agent
record regardless of `--fields`.

### Examples

```bash
iics agent list

iics agent list --fields name,agentHost,agentVersion,configUpdateTime

# Client-side filtering on technical field names
iics agent list --filter agentHost==devinfacld01
iics agent list --filter active==true --filter platform==win64

iics agent list --unassigned

iics agent list --output json
```

```powershell
iics agent list

iics agent list --fields name,agentHost,agentVersion,configUpdateTime

iics agent list --filter agentHost==devinfacld01
iics agent list --filter active==true --filter platform==win64

iics agent list --unassigned
```

---

## agent get

Get a single Secure Agent by ID or name.

### Flags

| Flag     | Type   | Description |
| -------- | ------ | ----------- |
| `--id`   | string | Agent ID    |
| `--name` | string | Agent name (uses `GET /api/v2/agent/name/<name>`) |

Exactly one of `--id` / `--name` is required; they are mutually exclusive.

All [global flags](../../README.md#global-flags) apply.

### Output

In table mode the agent is printed as a vertical `PROPERTY` / `VALUE` listing.
`--output json`, `yaml`, and `csv` render the agent record directly.

### Examples

```bash
iics agent get --id <agent-id>

iics agent get --name "My Agent"

iics agent get --id <agent-id> --output json
```

```powershell
iics agent get --id <agent-id>

iics agent get --name "My Agent"

iics agent get --id <agent-id> --output json
```

---

## agent details

Get the service engine details for an agent: the agent summary, then each
service (engine) with its status and, optionally, its configuration properties.

### Flags

| Flag         | Type   | Description |
| ------------ | ------ | ----------- |
| `--id`       | string | Agent ID    |
| `--fid`      | string | Agent federated ID |
| `--name`     | string | Agent name  |
| `--hostname` | string | Agent host name |
| `--full`     | bool   | Include agent-level and per-service configuration properties (adds `onlyStatus=false`) |
| `--services` | bool   | Show only the agent summary and one horizontal table of services (mutually exclusive with `--full`) |

Exactly one of `--id` / `--fid` / `--name` / `--hostname` is required; they are
mutually exclusive. Only `--id` maps to the API directly; `--name` uses the
by-name endpoint, and `--fid` / `--hostname` are resolved to an ID by scanning
the agent list (with a runtime-environment fallback for `--fid`).

All [global flags](../../README.md#global-flags) apply.

### Output

Table mode prints, in order:

1. `Agent:` - a vertical `PROPERTY` / `VALUE` listing of the agent summary.
2. `Agent Config:` (only with `--full`, when present) - a table of agent-level
   configuration properties.
3. For each service: a `Service: <display name> (<appname> v<version>)` header,
   a vertical status listing, and (with `--full`) a configuration table.

Configuration tables have columns `TYPE`, `NAME`, `VALUE`, `DEFAULT`,
`CUSTOMIZED`. Long `VALUE` / `DEFAULT` cells are wrapped onto multiple lines.

With `--services`, only the `Agent:` summary and a single `Services (N):` table
are printed, one row per service with columns `SERVICE`, `APP NAME`, `VERSION`,
`STATUS`, `DESIRED`, `SUBSTATE`, `REPLACE`, `UPDATED`.

`--output json` / `yaml` render the full nested details document.

### Examples

```bash
iics agent details --id <agent-id>

iics agent details --id <agent-id> --services

iics agent details --hostname devinfacld01 --full

iics agent details --name "My Agent" --output json
```

```powershell
iics agent details --id <agent-id>

iics agent details --id <agent-id> --services

iics agent details --hostname devinfacld01 --full
```

---

## agent delete

Delete a Secure Agent.

### Flags

| Flag         | Type   | Description |
| ------------ | ------ | ----------- |
| `--id`       | string | Agent ID    |
| `--name`     | string | Agent name  |
| `--hostname` | string | Agent host name |
| `--yes` / `-y` | bool | Skip the confirmation prompt |

Exactly one of `--id` / `--name` / `--hostname` is required; they are mutually
exclusive.

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics agent delete --id <agent-id>

iics agent delete --hostname devinfacld01 --yes
```

```powershell
iics agent delete --id <agent-id>

iics agent delete --hostname devinfacld01 --yes
```

---

## agent start / agent stop

Start or stop a service on a Secure Agent. Uses
`POST public/core/v3/agent/service`.

### Flags

| Flag                  | Type   | Default | Description |
| --------------------- | ------ | ------- | ----------- |
| `--id`                | string |         | Agent ID    |
| `--name`              | string |         | Agent name  |
| `--hostname`          | string |         | Agent host name |
| `--service`           | string |         | Service display name, e.g. `"Data Integration Server"` (required unless `--interactive`) |
| `--interactive`, `-i` | bool   | false   | Select the agent and service interactively |
| `--blocking`          | bool   | false   | Poll the service status until it has started/stopped, printing each poll |
| `--poll-interval`     | int    | 10      | Seconds between status polls (with `--blocking`) |
| `--max-wait-time`     | int    | 300     | Maximum seconds to wait (with `--blocking`) |

Exactly one of `--id` / `--name` / `--hostname` is required, unless
`--interactive` is used. `--interactive` lists the Secure Agents to choose from,
asks for confirmation, and (unless `--blocking` was passed explicitly) asks
`Wait for service "<name>" to start/stop`.

- `stop -i` lists the agent's running services to pick from.
- `start -i` prompts for the service **display name as free text** (the running
  services are shown for reference). There is no API that lists an agent's
  stopped services, so a stopped service cannot be offered in a menu.

With `--blocking` the command returns only once the service is fully started
(`RUNNING` with `subState 0`) or stopped (gone from the listing), or
`--max-wait-time` elapses (non-zero exit). A service in `ERROR` also fails the
wait.

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics agent stop  --id <agent-id> --service "Data Integration Server"
iics agent start --hostname devinfacld01 --service "Data Integration Server"

iics agent start --id <agent-id> --service "Process Server" --blocking --poll-interval 5

iics agent stop -i
```

```powershell
iics agent stop  --id <agent-id> --service "Data Integration Server"
iics agent start --hostname devinfacld01 --service "Data Integration Server"
```

---

## agent restart

Restart a service on a Secure Agent: stop it, wait for it to leave the agent's
service listing, then start it again.

### Flags

Same as `agent start` / `agent stop`
(`--id` / `--name` / `--hostname`, `--service`, `--interactive` / `-i`,
`--blocking`, `--poll-interval`, `--max-wait-time`).

The wait for the **stop** to complete is always performed. The wait for the
**start** to complete happens with `--blocking`, or after answering yes to the
interactive `Wait for service "<name>" to start` prompt. `--max-wait-time`
bounds the whole operation (stop wait + start wait combined). "Fully started"
means the engine reports `RUNNING` with `subState 0`.

### Examples

```bash
iics agent restart --id <agent-id> --service "Data Integration Server" --blocking

iics agent restart -i
```

```powershell
iics agent restart --id <agent-id> --service "Data Integration Server" --blocking
```

---

## agent installer-info

Retrieve the Secure Agent installer download URL, install token, and checksum download URL
for a platform.

### Flags

| Flag   | Type   | Required | Description                          |
| ------ | ------ | -------- | ------------------------------------ |
| `--os` | string | yes      | Operating system: `win64` or `linux64` |

When `--os` is omitted and the session is interactive, the command shows a numbered
selection menu (`win64`/`linux64`); enter `0`, press Enter, or type `q` to cancel.

All [global flags](../../README.md#global-flags) apply.

### Output

By default the fields are printed as a vertical `PROPERTY`/`VALUE` table. With
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
   `--profile` account using `--os` (shown as a selection menu when interactive;
   `0`, Enter, or `q` cancels).

### Flags

| Flag               | Type   | Default              | Description                                                        |
| ------------------ | ------ | -------------------- | ------------------------------------------------------------------ |
| `--os`             | string |                      | Operating system: `win64` or `linux64` (used when no info supplied) |
| `--installer-info` | string |                      | Path to installer info JSON (omit to read piped stdin)             |
| `--target`         | string | system temp directory | Destination file or directory                                     |
| `--verify`         | bool   | false                | Download the checksum file and verify the installer               |
| `--progress`       | bool   | false                | Print live download progress to stderr                            |

`--target` behavior:

- Not provided: the file is written to the system temp directory using the file
  name from the installer metadata.
- An existing directory or a value ending with a path separator: the metadata file
  name is appended.
- Any other value: treated as the full destination file path.

All [global flags](../../README.md#global-flags) apply.

### Output

By default a vertical `PROPERTY`/`VALUE` table describes the downloaded file. With
`--verify` the expected and actual checksums and the `verified` result are included.
Use `--verbose` to print download and verification progress messages, or `--progress`
for a live, updating transfer line (bytes transferred, percentage when the server
reports a content length, and transfer rate) written to stderr. `--output json`,
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

# Show live download progress
iics agent installer-download --os linux64 --target ./downloads/ --progress
```

```powershell
iics agent installer-download --os linux64 --target ./downloads/

iics agent installer-info --os win64 --output json | iics agent installer-download --verify

iics agent installer-download --installer-info info.json --target C:\Temp\agent.exe --verify
```

## See also

- [environment](environment.md) - manage runtime environments (Secure Agent groups) that contain agents
