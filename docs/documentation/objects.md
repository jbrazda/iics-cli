# objects

<!-- markdownlint-disable MD013 MD024 MD060 -->

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

| Flag                   | Short | Type   | Default                             | Description                                               |
| ---------------------- | ----- | ------ | ----------------------------------- | --------------------------------------------------------- |
| `--type`               |       | string |                                     | Filter by object type (see [types](#object-types))        |
| `--tag`                |       | string |                                     | Filter by tag name                                        |
| `-q, --query`          | `-q`  | string |                                     | Raw query filter, e.g. `location==ZZ_TEST_CLI`            |
| `--limit`              |       | int    | 0 (all)                             | Max results; 0 fetches all pages                          |
| `--skip`               |       | int    | 0                                   | Results to skip (only meaningful with `--limit`)          |
| `--output-fields`      |       | string | `id,path,type,updatedBy,updateTime` | Comma-separated fields shown on console                   |
| `--output-file`        |       | string |                                     | Write output to this file path                            |
| `--output-file-format` |       | string | `yaml`                              | Format of the output file: `yaml`, `json`, `csv`, `table` |
| `--output-file-fields` |       | string |                                     | Comma-separated fields written to the output file         |

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

```powershell
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
iics objects list --output-file objects.yaml `
  --output-file-format yaml `
  --output-file-fields id,path,type,description

# Get IDs for piping into export
iics objects list -q "location==ZZ_TEST_CLI" -o csv

# Count objects by type using JSON output
$objs = iics objects list --output json | ConvertFrom-Json
$objs | Group-Object -Property type | ForEach-Object { [PSCustomObject]@{type=$_.Name; count=$_.Count} }
```

---

## objects dependencies

Find what objects a given asset depends on, or which assets depend on it.
The command resolves full transitive closure by recursively traversing every discovered
dependency object until no new nodes are found.

`objects dependencies` and `package dependencies` now use the same transitive traversal
engine, so dependency recursion behavior is consistent across both commands.

### Flags

| Flag          | Short | Type     | Required | Default | Description                                                            |
| ------------- | ----- | -------- | -------- | ------- | ---------------------------------------------------------------------- |
| `--id`        |       | string   |          |         | Object ID to inspect (or pipe a JSON array from `objects list`)        |
| `--ref-type`  |       | string   |          |         | Reference direction: `uses` or `usedBy` (default: both)                |
| `--limit`     |       | int      |          | 0 (all) | Max results; 0 fetches all pages in batches of 50                      |
| `--skip`      |       | int      |          | 0       | Results to skip (only meaningful with `--limit`)                       |
| `--targets`   |       | []string |          |         | Comma-separated profiles to validate dependencies against              |
| `--publish`   |       | bool     |          | false   | Restrict output to publishable types only; sort by publish order       |
| `--filter`    |       | string   |          |         | Regex matched against `location` (`Explore/path.type`) for final output |
| `--output-file` |     | string   |          |         | Write output to a file |
| `--output-file-format` | | string |          | `yaml` | Output file format: `table`, `json`, `csv`, `yaml` |
| `--output-file-fields` | | string |          | `location,dependency,id,type,path,status,warning` | Comma-separated fields for output file rows |

All [global flags](../../README.md#global-flags) apply.

When `--id` is omitted and stdin is not a terminal, a JSON array is read from stdin
(e.g. piped from `objects list --output json`). Dependencies for all input objects
are collected and deduplicated.

With `--publish`, filtering is applied to output rows only; traversal still walks through
non-publishable dependency nodes so downstream publishable dependencies are included.

Rows include a `dependency` marker:

- `explicit`: seed objects provided by `--id` or stdin input list
- `transitive`: objects discovered through dependency traversal

### Output columns (standard mode)

| Column         | Description                                    |
| -------------- | ---------------------------------------------- |
| `location`     | Normalized asset identifier: `Explore/path.type` |
| `dependency`   | `explicit` (seed input) or `transitive` (resolved) |
| `id`           | Object ID (when available)                     |
| `type`         | Object type                                    |
| `path`         | Full path of the dependent object              |

### Output columns (--targets report mode)

| Column              | Description                                      |
| ------------------- | ------------------------------------------------ |
| `Location`          | Normalized asset identifier: `Explore/path.type` |
| `DEPENDENCY`        | `explicit` (seed input) or `transitive` (resolved) |
| `STATUS (<profile>)`| `found`, `missing`, or `unknown` per profile     |
| `WARNING (<profile>)`| Lookup warning message (CSV output only)        |

### Examples

```bash
# Find everything an object depends on
iics objects dependencies --id aLX7qnviqxJdmqpVsd17SG --ref-type uses

# Find what depends on an object (impact analysis)
iics objects dependencies --id aLX7qnviqxJdmqpVsd17SG --ref-type usedBy

# Resolve ID first, then find dependencies
# Note: lookup returns a JSON array - use .[0].id to extract the first result
ID=$(iics lookup --path "Sales/ETL/LoadOrders" --type MTT --output json | jq -r '.[0].id')
iics objects dependencies --id "$ID" --ref-type uses --output json

# Find all dependencies of tagged objects (multi-object mode via stdin pipe)
iics objects list -q "tag==Project_sprint_9" --output json \
  | iics objects dependencies --ref-type uses

# Validate dependencies across multiple target environments
iics objects list -q "tag==Project_sprint_9" --output json \
  | iics objects dependencies --ref-type uses --targets dev,qa

# Validate as JSON for scripting
iics objects list -q "tag==Project_sprint_9" --output json \
  | iics objects dependencies --ref-type uses --targets dev,qa --output json

# Find publishable dependencies and pipe directly to publish
iics objects list -q "tag==Project_sprint_9" --output json \
  | iics objects dependencies --ref-type uses --publish \
  | iics publish run

# Check which publishable dependencies are missing in QA, then publish them
iics objects list -q "tag==Project_sprint_9" --output json \
  | iics objects dependencies --ref-type uses --targets qa --publish \
  | iics publish run --profile qa

# Filter by normalized location and write selected fields to CSV
iics objects list -q "tag==Project_sprint_9" --output json \
  | iics objects dependencies --ref-type uses --filter '^Explore/Shared/' \
    --output-file deps.csv --output-file-format csv \
    --output-file-fields location,dependency,type,path,status,warning
```

```powershell
# Find everything an object depends on
iics objects dependencies --id aLX7qnviqxJdmqpVsd17SG --ref-type uses

# Find what depends on an object (impact analysis)
iics objects dependencies --id aLX7qnviqxJdmqpVsd17SG --ref-type usedBy

# Resolve ID first, then find dependencies
# Note: lookup returns a JSON array - index [0] to get the first result
$obj = (iics lookup --path "Sales/ETL/LoadOrders" --type MTT --output json | ConvertFrom-Json)[0]
iics objects dependencies --id $obj.id --ref-type uses --output json

# Find all dependencies of tagged objects (multi-object mode via stdin pipe)
iics objects list -q "tag==Project_sprint_9" --output json `
  | iics objects dependencies --ref-type uses

# Validate dependencies across multiple target environments
iics objects list -q "tag==Project_sprint_9" --output json `
  | iics objects dependencies --ref-type uses --targets dev,qa

# Find publishable dependencies and pipe directly to publish
iics objects list -q "tag==Project_sprint_9" --output json `
  | iics objects dependencies --ref-type uses --publish `
  | iics publish run

# Check which publishable dependencies are missing in QA, then publish them
iics objects list -q "tag==Project_sprint_9" --output json `
  | iics objects dependencies --ref-type uses --targets qa --publish `
  | iics publish run --profile qa
```

## See also

- [lookup](lookup.md) - resolve object paths to IDs
- [export](export.md) - export objects to a ZIP package
- [tag](tag.md) - assign tags to objects
- [permission](permission.md) - manage object-level permissions
