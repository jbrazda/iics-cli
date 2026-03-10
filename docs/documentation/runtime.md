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

## See also

- [agent](agent.md) - manage Secure Agents within runtime environments
