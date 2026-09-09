# runtime

Manage IICS runtime environments. Alias: `rt`.

Runtime environments are groups of one or more Secure Agents that execute data integration tasks.

## Synopsis

```bash
iics runtime <subcommand> [flags]
iics rt <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                      |
| ---------- | -------------------------------- |
| `list`     | List runtime environments        |
| `get`      | Get a single runtime environment |
| `create`   | Create a runtime environment     |
| `update`   | Update a runtime environment     |
| `configs`  | Manage Secure Agent group service properties |

---

## runtime list

### Flags

| Flag      | Type | Default | Description     |
| --------- | ---- | ------- | --------------- |
| `--limit` | int  | 200     | Max results     |
| `--skip`  | int  | 0       | Results to skip |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column   | Description                     |
| -------- | ------------------------------- |
| `id`     | Runtime environment ID          |
| `name`   | Runtime environment name        |
| `type`   | Type (CLOUD, HYBRID, LOCAL)     |
| `status` | Current status                  |

### Examples

```bash
iics runtime list

iics rt list --output json

# List only HYBRID environments
iics runtime list --output json | jq '.[] | select(.type == "HYBRID")'
```

```powershell
iics runtime list

iics rt list --output json

# List only HYBRID environments
$runtimes = iics runtime list --output json | ConvertFrom-Json
$runtimes | Where-Object { $_.type -eq "HYBRID" }
```

---

## runtime get

### Flags

| Flag   | Type   | Required | Description             |
| ------ | ------ | -------- | ----------------------- |
| `--id` | string | yes      | Runtime environment ID  |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column        | Description                  |
| ------------- | ---------------------------- |
| `id`          | Runtime environment ID       |
| `name`        | Name                         |
| `type`        | Type                         |
| `status`      | Current status               |
| `description` | Description                  |

### Examples

```bash
iics runtime get --id <runtime-id>

iics rt get --id <runtime-id> --output json
```

```powershell
iics runtime get --id <runtime-id>

iics rt get --id <runtime-id> --output json
```

---

## runtime create

Create a runtime environment from a JSON definition file.

### Flags

| Flag          | Type   | Required | Description                                    |
| ------------- | ------ | -------- | ---------------------------------------------- |
| `--from-file` | string | yes      | JSON file with runtime environment definition  |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics runtime create --from-file my-runtime.json
```

---

## runtime update

### Flags

| Flag          | Type   | Required | Description                      |
| ------------- | ------ | -------- | -------------------------------- |
| `--id`        | string | yes      | Runtime environment ID           |
| `--from-file` | string | yes      | JSON file with updated fields    |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics runtime update --id <runtime-id> --from-file updated-runtime.json
```

---

## runtime configs

Manage Secure Agent group service property overrides. Uses
`GET` / `PUT /api/v2/runtimeEnvironment/<id>/configs`. The Secure Agent group ID
is the runtime environment ID.

### runtime configs get

Show the service property overrides for a group.

| Flag        | Type   | Description |
| ----------- | ------ | ----------- |
| `--id`      | string | Secure Agent group (runtime environment) ID |
| `--name`    | string | Secure Agent group (runtime environment) name |
| `--service` | string | Show only this service's properties |

Exactly one of `--id` / `--name` is required.

In table mode, each service is printed as a section of property/value rows. An
empty result prints `No service property overrides.`. `--output json` / `yaml`
render the raw document.

```bash
iics runtime configs get --id <groupId>
iics runtime configs get --name "My Group" --service Data_Integration_Server --output json
```

### runtime configs set

Replace the service property overrides for a group from a JSON file shaped as
`{"<Service_Name>":[{ ...settings... }]}`.

| Flag           | Type   | Description |
| -------------- | ------ | ----------- |
| `--id`         | string | Secure Agent group (runtime environment) ID |
| `--name`       | string | Secure Agent group (runtime environment) name |
| `--from-file`  | string | JSON file with the overrides (required) |
| `--yes` / `-y` | bool   | Skip the confirmation prompt |

Exactly one of `--id` / `--name` is required.

```bash
iics runtime configs set --id <groupId> --from-file props.json --yes
```

## See also

- [agent](agent.md) - manage Secure Agents within runtime environments
