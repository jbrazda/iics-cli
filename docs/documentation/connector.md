# connector

List the connectors available to the organization and the connection attributes
each connector takes. Use it to build the JSON definition for
[`connection create`](connection.md#connection-create).

Connectors use the V2 API (`/api/v2/connector`) with JSON.

## Synopsis

```bash
iics connector <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                                   |
| ---------- | --------------------------------------------- |
| `list`     | List the connectors available to the org      |
| `metadata` | Show the connection attributes of a connector |

---

## connector list

### Flags

No command-specific flags. All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column             | Description                                    |
| ------------------ | ---------------------------------------------- |
| `name`             | Connector name (use with `connector metadata`) |
| `shortName`        | Connector short name                           |
| `sdkType`          | Connector SDK, for example `CTK` or `CCI`      |
| `connectorVersion` | Connector version                              |
| `isPublic`         | Whether the connector is public                |
| `updateTime`       | Last update time                               |

`--output json` also returns `displayName`, `description`, `publisher`, `type`,
`pluginId`, `guid` and `orgId` when the API provides them.

### Examples

```bash
iics connector list

# Names of all CCI connectors
iics connector list --output json | jq -r '.[] | select(.sdkType == "CCI") | .name'
```

```powershell
iics connector list

# Names of all CCI connectors
(iics connector list --output json | ConvertFrom-Json) | Where-Object sdkType -eq 'CCI' | ForEach-Object name
```

---

## connector metadata

Show the connection attributes of a connector.

`--name` is the connector name from `connector list` (for example `SqlServer`,
`Salesforce`, `Amazon S3 v2`), not the short name. The API matches it as a
prefix, so `Oracle` also returns `Oracle Autonomous Database` and others:

- One match (or one exact match with `--exact`): the attributes are listed.
- Several matches: a summary of the matching connectors is shown; repeat with
  the full name and `--exact`.

CCI connectors describe their attributes in a CCI definition (`cciMeta` in the
JSON output); the table lists the attributes from that definition, including
their labels and whether they are encrypted.

### Flags

| Flag      | Type   | Required | Description                                                       |
| --------- | ------ | -------- | ----------------------------------------------------------------- |
| `--name`  | string | yes      | Connector name                                                    |
| `--exact` | bool   |          | Keep only the connector whose name matches exactly (ignores case) |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column         | Description                                                                   |
| -------------- | ----------------------------------------------------------------------------- |
| `name`         | Attribute name                                                                |
| `label`        | Display label (CCI connectors)                                                |
| `dataType`     | Numeric connector data type code, or the CCI type such as `STRING`, `BOOLEAN` |
| `mandatory`    | Whether the attribute is required                                             |
| `encrypted`    | Whether the attribute is stored encrypted (CCI connectors)                    |
| `defaultValue` | Default value                                                                 |
| `values`       | Allowed values, when restricted                                               |

`--output json` returns the raw API response: an array of objects with `type`,
`isStandardConnType`, `attributes` and, for CCI connectors, `cciMeta`.

### Examples

```bash
iics connector metadata --name SqlServer

# Several connectors start with "Oracle"; pick the exact one
iics connector metadata --name Oracle --exact

iics connector metadata --name "Amazon S3 v2" --output json
```

```powershell
iics connector metadata --name SqlServer

# Several connectors start with "Oracle"; pick the exact one
iics connector metadata --name Oracle --exact

iics connector metadata --name "Amazon S3 v2" --output json
```

## See also

- [connection](connection.md) - create, update and test connections
