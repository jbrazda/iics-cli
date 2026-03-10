# connection

Manage IICS connections (data sources and targets). Alias: `conn`.

## Synopsis

```bash
iics connection <subcommand> [flags]
iics conn <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                   |
| ---------- | ----------------------------- |
| `list`     | List connections              |
| `get`      | Get a single connection       |
| `create`   | Create a new connection       |
| `update`   | Update an existing connection |
| `delete`   | Delete a connection           |

---

## connection list

### Flags

| Flag      | Type   | Default | Description                    |
| --------- | ------ | ------- | ------------------------------ |
| `--type`  | string |         | Filter by connection type      |
| `--name`  | string |         | Filter by connection name      |
| `--limit` | int    | 200     | Max results                    |
| `--skip`  | int    | 0       | Results to skip                |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column      | Description              |
| ----------- | ------------------------ |
| `id`        | Connection ID            |
| `name`      | Connection name          |
| `type`      | Connection type          |
| `updatedBy` | Last modifier            |
| `updateTime`| Last modification time   |

### Examples

```bash
# List all connections
iics connection list

# Filter by type
iics connection list --type Salesforce

# Filter by name substring
iics connection list --name "Prod"

# Output as JSON
iics connection list --output json

# Get IDs of all Snowflake connections
iics connection list --type Snowflake --output json | jq -r '.[].id'
```

---

## connection get

Get the full details of a single connection.

### Flags

| Flag   | Type   | Required | Description     |
| ------ | ------ | -------- | --------------- |
| `--id` | string | yes      | Connection ID   |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics connection get --id abc123

# View as JSON to see all fields
iics connection get --id abc123 --output json
```

---

## connection create

Create a new connection from a JSON definition file.

### Flags

| Flag          | Type   | Required | Description                          |
| ------------- | ------ | -------- | ------------------------------------ |
| `--from-file` | string | yes      | Path to a JSON file with the connection definition |

All [global flags](../../README.md#global-flags) apply.

### JSON definition example

```json
{
  "name": "My Snowflake Prod",
  "type": "Snowflake",
  "description": "Production Snowflake connection",
  "runtimeEnvironmentId": "<runtime-env-id>",
  "connectionParams": {
    "account": "myorg.snowflakecomputing.com",
    "database": "PROD_DB",
    "warehouse": "COMPUTE_WH",
    "username": "etl_user",
    "password": ""
  }
}
```

### Examples

```bash
iics connection create --from-file snowflake_prod.json
```

---

## connection update

Update an existing connection from a JSON file. Only fields present in the file are changed.

### Flags

| Flag          | Type   | Required | Description                          |
| ------------- | ------ | -------- | ------------------------------------ |
| `--id`        | string | yes      | Connection ID to update              |
| `--from-file` | string | yes      | Path to a JSON file with the updated fields |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics connection update --id abc123 --from-file snowflake_prod_updated.json
```

---

## connection delete

Delete a connection. Prompts for confirmation unless `--yes` is provided.

### Flags

| Flag       | Short | Type   | Required | Description                |
| ---------- | ----- | ------ | -------- | -------------------------- |
| `--id`     |       | string | yes      | Connection ID to delete    |
| `--yes`    | `-y`  | bool   |          | Skip confirmation prompt   |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Interactive - will prompt "Are you sure?"
iics connection delete --id abc123

# Non-interactive (CI/CD)
iics connection delete --id abc123 --yes
```

## See also

- [lookup](lookup.md) - resolve connection paths to IDs
- [objects](objects.md) - list assets that use a connection
