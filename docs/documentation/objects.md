# objects

List and inspect IICS assets (objects) in your organisation.

## Synopsis

```bash
iics objects <subcommand> [flags]
```

## Subcommands

| Subcommand       | Description                              |
| ---------------- | ---------------------------------------- |
| `list`           | List objects with optional filters       |
| `dependencies`   | Find what an object uses or is used by   |

---

## objects list

List assets in the organisation. Without `--limit` all pages are fetched automatically (batches of 200).

### Flags

| Flag                   | Short | Type   | Default                          | Description                                                   |
| ---------------------- | ----- | ------ | -------------------------------- | ------------------------------------------------------------- |
| `--type`               |       | string |                                  | Filter by object type (see [types](#object-types))            |
| `--tag`                |       | string |                                  | Filter by tag name                                            |
| `-q, --query`          | `-q`  | string |                                  | Raw query filter, e.g. `location==ZZ_TEST_CLI`                |
| `--limit`              |       | int    | 0 (all)                          | Max results; 0 fetches all pages                              |
| `--skip`               |       | int    | 0                                | Results to skip (only meaningful with `--limit`)              |
| `--output-fields`      |       | string | `id,path,type,updatedBy,updateTime` | Comma-separated fields shown on console                    |
| `--output-file`        |       | string |                                  | Write output to this file path                                |
| `--output-file-format` |       | string | `yaml`                           | Format of the output file: `yaml`, `json`, `csv`, `table`    |
| `--output-file-fields` |       | string |                                  | Comma-separated fields written to the output file             |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column       | Description                                    |
| ------------ | ---------------------------------------------- |
| `id`         | Object ID                                      |
| `path`       | Full path within the org                       |
| `type`       | Object type (MTT, DTEMPLATE, DSS, etc.)        |
| `location`   | Computed as `Explore/<path>.<type>`            |
| `description`| Object description                             |
| `updatedBy`  | Last user who modified the object              |
| `updateTime` | Last modification timestamp                    |

### Object types

Common values for `--type`:

| Type        | Description                        |
| ----------- | ---------------------------------- |
| `MTT`       | Mapping / Mapping Task             |
| `DTEMPLATE` | Data Integration Task template     |
| `DSS`       | Synchronization Task               |
| `BSERVICE`  | Business Service                   |
| `FWCONFIG`  | Fixed-width configuration          |
| `TASKFLOW`  | Taskflow                           |
| `WORKFLOW`  | Cloud Data Integration workflow    |
| `AI_CONNECTION` | AI Connection                  |

### Examples

```bash
# List all objects (auto-paginated)
iics objects list

# List all mappings
iics objects list --type MTT

# Filter by a query expression
iics objects list -q "location==ZZ_TEST_CLI"

# First 50 objects, skip 100
iics objects list --type MTT --limit 50 --skip 100

# Output as JSON
iics objects list --type DTEMPLATE --output json

# Save a full CSV inventory to a file
iics objects list --output-file inventory.csv --output-file-format csv

# Save YAML with specific fields
iics objects list --output-file objects.yaml \
  --output-file-format yaml \
  --output-file-fields id,path,type,description

# Pipe IDs into export
iics objects list -q "location==ZZ_TEST_CLI" -o csv \
  | iics export run --export-file-path export.zip

# Count objects by type using JSON output
iics objects list --output json | jq 'group_by(.type) | map({type: .[0].type, count: length})'
```

---

## objects dependencies

Find what objects a given asset depends on, or which assets depend on it.

### Flags

| Flag         | Short | Type   | Required | Default | Description                                |
| ------------ | ----- | ------ | -------- | ------- | ------------------------------------------ |
| `--id`       |       | string | yes      |         | Object ID to inspect                       |
| `--ref-type` |       | string |          |         | Reference direction: `uses` or `usedBy`    |
| `--limit`    |       | int    |          | 200     | Max results                                |
| `--skip`     |       | int    |          | 0       | Results to skip                            |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column         | Description                          |
| -------------- | ------------------------------------ |
| `appContextId` | Dependent object ID                  |
| `type`         | Object type                          |
| `path`         | Full path of the dependent object    |
| `updatedBy`    | Last modifier                        |

### Examples

```bash
# Find everything an object depends on
iics objects dependencies --id aLX7qnviqxJdmqpVsd17SG --ref-type uses

# Find what depends on an object (impact analysis)
iics objects dependencies --id aLX7qnviqxJdmqpVsd17SG --ref-type usedBy

# Resolve ID first, then find dependencies
ID=$(iics lookup --path "Sales/ETL/LoadOrders" --type MTT --output json | jq -r '.id')
iics objects dependencies --id "$ID" --ref-type uses --output json
```

## See also

- [lookup](lookup.md) - resolve object paths to IDs
- [export](export.md) - export objects to a ZIP package
- [tag](tag.md) - assign tags to objects
- [permission](permission.md) - manage object-level permissions
