# role

Manage IICS roles. Roles group privileges and are assigned to users and user groups.

## Synopsis

```bash
iics role <subcommand> [flags]
```

## Subcommands

| Subcommand | Description        |
| ---------- | ------------------ |
| `list`     | List roles         |
| `get`      | Get a single role  |
| `create`   | Create a role      |
| `update`   | Update a role      |
| `delete`   | Delete a role      |

---

## role list

### Flags

| Flag      | Type | Default | Description     |
| --------- | ---- | ------- | --------------- |
| `--limit` | int  | 200     | Max results     |
| `--skip`  | int  | 0       | Results to skip |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column       | Description                       |
| ------------ | --------------------------------- |
| `id`         | Role ID                           |
| `name`       | Role name                         |
| `systemRole` | Whether this is a built-in role   |
| `description`| Role description                  |

### Examples

```bash
iics role list

# List as JSON
iics role list --output json

# List only custom (non-system) roles
iics role list --output json | jq '.[] | select(.systemRole == false)'
```

```powershell
iics role list

# List as JSON
iics role list --output json

# List only custom (non-system) roles
$roles = iics role list --output json | ConvertFrom-Json
$roles | Where-Object { $_.systemRole -eq $false }
```

---

## role get

### Flags

| Flag   | Type   | Required | Description |
| ------ | ------ | -------- | ----------- |
| `--id` | string | yes      | Role ID     |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics role get --id <role-id>

# JSON to see all role fields including privileges
iics role get --id <role-id> --output json
```

```powershell
iics role get --id <role-id>

# JSON to see all role fields including privileges
iics role get --id <role-id> --output json
```

---

## role create

Create a role from a JSON definition file.

### Flags

| Flag          | Type   | Required | Description                    |
| ------------- | ------ | -------- | ------------------------------ |
| `--from-file` | string | yes      | JSON file with role definition |

All [global flags](../../README.md#global-flags) apply.

### JSON definition example

```json
{
  "name": "ETL Developer",
  "description": "Can create and run mappings",
  "privileges": [
    { "id": "<privilege-id>", "name": "dataIntegration.mapping.create" },
    { "id": "<privilege-id>", "name": "dataIntegration.mapping.run" }
  ]
}
```

Use `iics privilege list` to find available privilege IDs.

### Examples

```bash
iics role create --from-file etl-developer-role.json
```

```powershell
iics role create --from-file etl-developer-role.json
```

---

## role update

### Flags

| Flag          | Type   | Required | Description                     |
| ------------- | ------ | -------- | ------------------------------- |
| `--id`        | string | yes      | Role ID                         |
| `--from-file` | string | yes      | JSON file with updated fields   |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics role update --id <role-id> --from-file updated-role.json
```

```powershell
iics role update --id <role-id> --from-file updated-role.json
```

---

## role delete

Delete a role. Prompts for confirmation unless `--yes` is given.

### Flags

| Flag    | Short | Type   | Required | Description              |
| ------- | ----- | ------ | -------- | ------------------------ |
| `--id`  |       | string | yes      | Role ID                  |
| `--yes` | `-y`  | bool   |          | Skip confirmation prompt |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics role delete --id <role-id>

iics role delete --id <role-id> --yes
```

```powershell
iics role delete --id <role-id>

iics role delete --id <role-id> --yes
```
```

## See also

- [privilege](privilege.md) - list available privileges to assign to roles
- [user](user.md) - assign roles to users
- [usergroup](usergroup.md) - assign roles to user groups
