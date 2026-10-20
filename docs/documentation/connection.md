# connection

Manage IICS connections (data sources and targets). Alias: `conn`.

Connections use the V2 API (`/api/v2/connection`). The CLI sends and receives JSON;
JSON request bodies carry `"@type": "connection"`, which the CLI adds when missing.

## Synopsis

```bash
iics connection <subcommand> [flags]
iics conn <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                                                            |
| ---------- | ---------------------------------------------------------------------- |
| `list`     | List connections                                                       |
| `get`      | Get a single connection by ID or name                                  |
| `create`   | Create a new connection                                                |
| `update`   | Update an existing connection (full or partial)                        |
| `delete`   | Delete a connection                                                    |
| `test`     | Test a connection                                                      |
| `search`   | List connections of a type available to a runtime environment or agent |
| `objects`  | List the source or target objects of a connection                      |
| `metadata` | Show the source or target runtime attributes of a connection           |

See also [connector](connector.md) for the attributes each connection type takes.

---

## connection list

The API returns every connection in the organization; `--type` and `--name` are
applied by the CLI.

### Flags

| Flag      | Type   | Default | Description                                        |
| --------- | ------ | ------- | -------------------------------------------------- |
| `--type`  | string |         | Filter by connection type (exact, ignores case)    |
| `--name`  | string |         | Filter by connection name substring (ignores case) |
| `--limit` | int    | 200     | Max results                                        |
| `--skip`  | int    | 0       | Results to skip                                    |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column       | Description            |
| ------------ | ---------------------- |
| `id`         | Connection ID          |
| `name`       | Connection name        |
| `type`       | Connection type        |
| `updatedBy`  | Last modifier          |
| `updateTime` | Last modification time |

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

```powershell
# List all connections
iics connection list

# Filter by type
iics connection list --type Salesforce

# Filter by name substring
iics connection list --name "Prod"

# Output as JSON
iics connection list --output json

# Get IDs of all Snowflake connections
$conns = iics connection list --type Snowflake --output json | ConvertFrom-Json
$conns | ForEach-Object { $_.id }
```

---

## connection get

Get the full details of a single connection. Use `--output json` to see every
attribute, including connector-specific `connParams`. The password is returned
masked (`********`).

### Flags

| Flag     | Type   | Required | Description     |
| -------- | ------ | -------- | --------------- |
| `--id`   | string | one of   | Connection ID   |
| `--name` | string | one of   | Connection name |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column                 | Description             |
| ---------------------- | ----------------------- |
| `id`                   | Connection ID           |
| `name`                 | Connection name         |
| `type`                 | Connection type         |
| `host`                 | Host name               |
| `runtimeEnvironmentId` | Runtime environment ID  |
| `updateTime`           | Last modification time  |
| `description`          | Description             |

### Examples

```bash
iics connection get --id abc123

# By name, as JSON to see all fields
iics connection get --name "Salesforce Prod" --output json
```

```powershell
iics connection get --id abc123

# By name, as JSON to see all fields
iics connection get --name "Salesforce Prod" --output json
```

---

## connection create

Create a new connection from a JSON definition file. The file is sent as-is
(plus `@type`), so any connector-specific attribute can be included. Attributes
that are not top-level connection fields go in `connParams`. Use
[`connector metadata`](connector.md#connector-metadata) to list the attributes
of a connection type.

### Flags

| Flag          | Type   | Required | Description                                        |
| ------------- | ------ | -------- | -------------------------------------------------- |
| `--from-file` | string | yes      | Path to a JSON file with the connection definition |

All [global flags](../../README.md#global-flags) apply.

### JSON definition example

```json
{
  "@type": "connection",
  "name": "Actgdata Dev",
  "type": "SqlServer2019",
  "description": "Accounting database in dev",
  "runtimeEnvironmentId": "<runtime-env-id>",
  "host": "devsql01",
  "port": 1433,
  "database": "Actgdata",
  "schema": "dbo",
  "authenticationType": "SqlServer",
  "username": "etl_user",
  "password": "<password>",
  "codepage": "MS1252",
  "connParams": {
    "encryptionMethod": "None"
  }
}
```

### Examples

```bash
iics connection create --from-file actgdata_dev.json
```

```powershell
iics connection create --from-file actgdata_dev.json
```

---

## connection update

Update an existing connection from a JSON file (`POST /api/v2/connection/<id>`).

- Without `--partial` the file replaces the connection definition, so it must
  contain every attribute.
- With `--partial` (header `Update-Mode: PARTIAL`) only the attributes in the
  file change.

`connection get` returns the password masked. Do not send its output back as a
full update; use `--partial` and leave `password` out unless you are changing it.

### Flags

| Flag          | Type   | Required | Description                                           |
| ------------- | ------ | -------- | ----------------------------------------------------- |
| `--id`        | string | yes      | Connection ID to update                               |
| `--from-file` | string | yes      | Path to a JSON file with the connection attributes    |
| `--partial`   | bool   |          | Change only the attributes in the file                |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Change only the description
echo '{"description": "Accounting database (dev)"}' > desc.json
iics connection update --id abc123 --partial --from-file desc.json

# Full replacement
iics connection update --id abc123 --from-file actgdata_dev.json
```

```powershell
# Change only the description
'{"description": "Accounting database (dev)"}' | Set-Content desc.json
iics connection update --id abc123 --partial --from-file desc.json

# Full replacement
iics connection update --id abc123 --from-file actgdata_dev.json
```

---

## connection delete

Delete a connection. Prompts for confirmation unless `--yes` is provided.

### Flags

| Flag    | Short | Type   | Required | Description              |
| ------- | ----- | ------ | -------- | ------------------------ |
| `--id`  |       | string | yes      | Connection ID to delete  |
| `--yes` | `-y`  | bool   |          | Skip confirmation prompt |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Interactive - will prompt "Are you sure?"
iics connection delete --id abc123

# Non-interactive (CI/CD)
iics connection delete --id abc123 --yes
```

```powershell
# Interactive - will prompt "Are you sure?"
iics connection delete --id abc123

# Non-interactive (CI/CD)
iics connection delete --id abc123 --yes
```

---

## connection test

Test a connection from its runtime environment. Prints the API's success
message; a failed test is returned as an error (exit code 1).

### Flags

| Flag     | Type   | Required | Description     |
| -------- | ------ | -------- | --------------- |
| `--id`   | string | one of   | Connection ID   |
| `--name` | string | one of   | Connection name |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics connection test --name "Salesforce Prod"

# In CI/CD: fail the step when the connection does not work
iics connection test --id abc123 || exit 1
```

```powershell
iics connection test --name "Salesforce Prod"

# In CI/CD: fail the step when the connection does not work
iics connection test --id abc123
if ($LASTEXITCODE -ne 0) { exit 1 }
```

---

## connection search

List the connections of a type that a runtime environment (or Secure Agent) can use.

### Flags

| Flag           | Type   | Required | Description                                                     |
| -------------- | ------ | -------- | --------------------------------------------------------------- |
| `--type`       | string | yes      | Connection type, for example `Salesforce`, `CSVFile`, `TOOLKIT` |
| `--runtime-id` | string | one of   | Runtime environment ID (takes precedence over `--agent-id`)     |
| `--agent-id`   | string | one of   | Secure Agent ID                                                 |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column                 | Description            |
| ---------------------- | ---------------------- |
| `id`                   | Connection ID          |
| `name`                 | Connection name        |
| `type`                 | Connection type        |
| `runtimeEnvironmentId` | Runtime environment ID |
| `updateTime`           | Last modification time |

### Examples

```bash
iics connection search --type Salesforce --runtime-id <runtime-environment-id>
```

```powershell
iics connection search --type Salesforce --runtime-id <runtime-environment-id>
```

---

## connection objects

List the source objects (or target objects with `--target`) available through a
connection, such as database tables or Salesforce objects. The API returns at
most 200 objects; use `--search-pattern` to narrow the list.

### Flags

| Flag               | Type   | Required | Description                                     |
| ------------------ | ------ | -------- | ----------------------------------------------- |
| `--id`             | string | one of   | Connection ID                                   |
| `--name`           | string | one of   | Connection name                                 |
| `--target`         | bool   |          | List target objects instead of source objects   |
| `--search-pattern` | string |          | Substring filter on object names (ignores case) |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column | Description                                   |
| ------ | --------------------------------------------- |
| `id`   | Position of the object in the result          |
| `name` | Object name                                   |

### Examples

```bash
iics connection objects --name "Actgdata Dev" --search-pattern billing

iics connection objects --id abc123 --target --output json | jq -r '.[].name'
```

```powershell
iics connection objects --name "Actgdata Dev" --search-pattern billing

(iics connection objects --id abc123 --target --output json | ConvertFrom-Json).name
```

---

## connection metadata

Show the source (or target with `--target`) runtime attributes of a connection,
such as Salesforce batch size or upsert options. Connection types without
runtime attributes return the API error `APP_13405 The connection has no object.`

### Flags

| Flag       | Type   | Required | Description                                         |
| ---------- | ------ | -------- | --------------------------------------------------- |
| `--id`     | string | one of   | Connection ID                                       |
| `--name`   | string | one of   | Connection name                                     |
| `--target` | bool   |          | Show target attributes instead of source attributes |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column           | Description                          |
| ---------------- | ------------------------------------ |
| `name`           | Attribute name                       |
| `dataType`       | `STRING`, `NUMBER` or `BOOLEAN`      |
| `mandatory`      | Whether the attribute is required    |
| `defaultValue`   | Default value                        |
| `possibleValues` | Allowed values, when restricted      |

### Examples

```bash
iics connection metadata --name "Salesforce Prod" --target
```

```powershell
iics connection metadata --name "Salesforce Prod" --target
```

## See also

- [connector](connector.md) - list connectors and the attributes a connection type takes
- [lookup](lookup.md) - resolve connection paths to IDs
- [objects](objects.md) - list assets that use a connection
