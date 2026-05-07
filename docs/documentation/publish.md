# publish

Publish Cloud Application Integration (CAI) assets to the IICS runtime.

## Synopsis

```text
iics publish <subcommand> [flags]
```

## Subcommands

| Subcommand | Description |
| ---------- | ----------- |
| `start` | Submit a publish job and return the job ID (fire-and-forget) |
| `status` | Retrieve the current status of a publish job by ID |
| `run` | Full workflow: resolve inputs, auto-batch, submit, poll to completion, print summary |

## CAI URL Auto-detection

The publish API uses a CAI-specific service host. The host is derived automatically from
the login response `products[].baseApiUrl` field (scheme+host only). No configuration is
required for normal use.

To override, set any of the following (in decreasing precedence):

- `--cai-url` flag
- `IICS_CAI_URL` environment variable
- `caiUrl` field in the profile config (`~/.iics/config.yaml`)

Example profile entry:

```yaml
profiles:
  dev:
    username: user@example.com
    password: secret
    region: US
    caiUrl: https://na1.ai.dm-us.informaticacloud.com
```

## Asset Path Format

Asset paths must follow the format: `Explore/<folder-path>/<asset-name>.<TYPE>.xml`

Example: `Explore/MyProject/Processes/OrderProcess.PROCESS.xml`

The `location` field returned by `iics objects list` already contains the
`Explore/<path>.<TYPE>` prefix. Appending `.xml` gives the exact value the publish API
expects. This is the recommended source for generating publish lists.

### Supported asset types

| Type suffix            | Asset type                         |
|------------------------|------------------------------------|
| `AI_SERVICE_CONNECTOR` | Service connector                  |
| `AI_CONNECTION`        | Application integration connection |
| `PROCESS`              | Process                            |
| `GUIDE`                | Guide                              |
| `TASKFLOW`             | Taskflow                           |

## Publish Order

Assets are automatically sorted in dependency order before being submitted to the API,
regardless of the order they appear in the input file or piped input. This ensures that
dependencies are published before the assets that depend on them.

| Order | Type suffix            | Asset type                         |
|-------|------------------------|------------------------------------|
| 1     | `AI_SERVICE_CONNECTOR` | Service connector                  |
| 2     | `AI_CONNECTION`        | Application integration connection |
| 3     | `PROCESS`              | Process                            |
| 4     | `GUIDE`                | Guide                              |
| 5     | `TASKFLOW`             | Taskflow                           |

Within each type group the original relative order from the input file is preserved
(stable sort). Assets of unknown type are placed at the end.

## Batch Limit

The publish API accepts at most 199 assets per request. When more than 199 assets are
provided, `publish run` and `publish start` automatically split them into sequential
batches of up to 199 each.

---

## publish start

Submit a publish job and print the job ID. Does not wait for completion.

### Flags

| Flag | Type | Default | Description | Required |
| ---- | ---- | ------- | ----------- | -------- |
| `--asset` | string (repeatable) | | Asset path(s) to publish | one input required |
| `--from-file` | string | | File with asset paths (txt/json/csv); omit to read from stdin | |
| `--cai-url` | string | | CAI base URL override | |
| `--name` | string | | Optional job label for verbose output | |

### Input format

`--from-file` uses the file extension to select a parser. When reading from stdin the format
is auto-detected from content.

| Extension / source | Behaviour |
| ------------------ | --------- |
| `.txt` | One asset path per line, no header. Lines starting with `#` are comments. Paths without a `.xml` suffix have it appended automatically. |
| `.csv` | CSV with header row. Resolution order per row: (1) `LOCATION` column - appends `.xml`; (2) `PATH` + `TYPE` - builds `Explore/<PATH>.<TYPE>.xml`; (3) `PATH` only - appends `.xml` if missing. At least one of `LOCATION` or `PATH` is required. |
| `.json` | JSON array of strings (direct paths) or array of objects from `iics objects list -o json`. Per object: `location` field preferred (appends `.xml`); falls back to `path`+`type` conversion; then `path` with `.xml` appended if missing. |
| `.yaml` / `.yml` | Same as `.json` but YAML format (output of `iics objects list -o yaml`). |
| stdin | Auto-detected: starts with `[` - JSON; commas in first line or first line is `PATH` or `LOCATION` - CSV; otherwise plain text. |

Assets whose type is not one of the supported publishable types (`AI_SERVICE_CONNECTOR`,
`AI_CONNECTION`, `PROCESS`, `GUIDE`, `TASKFLOW`) are silently skipped regardless of input
format or source, including the `--asset` flag. Type is derived from the path suffix
(`Name.TYPE.xml`).

#### Generating an asset list from `objects list`

The `location` field is the recommended source. It is already in `Explore/<path>.<TYPE>`
format - appending `.xml` gives the exact asset path the publish API expects.

```bash
# Recommended: CSV using the location field (single-column, no type conversion needed)
iics objects list -q "location==MyProject/Processes" -o csv --output-fields location \
  > assets.csv
iics publish start --from-file assets.csv

# Pipe directly using location (stdin auto-detect)
iics objects list -q "type=='PROCESS'" -o csv --output-fields location | iics publish start

# JSON (location field is included by default in objects list JSON output)
iics objects list -q "location==MyProject/Processes" -o json > assets.json
iics publish start --from-file assets.json

# Alternative: CSV with PATH + TYPE (requires both columns)
iics objects list -q "location==MyProject/Processes" -o csv --output-fields path,type \
  > assets.csv
iics publish start --from-file assets.csv

# Plain text file (paths must already be in Explore/...xml format)
iics publish start --from-file assets.txt
```

### Examples

```bash
iics publish start --asset "Explore/Default/MyProcess.PROCESS.xml"

iics publish start \
  --asset "Explore/Default/MyProcess.PROCESS.xml" \
  --asset "Explore/Default/MyConn.AI_CONNECTION.xml"

iics publish start --from-file assets.txt
iics publish start --from-file assets.csv
iics publish start --from-file assets.json

iics objects list -q "type=='PROCESS'" -o csv | iics publish start
```

```powershell
iics publish start --asset "Explore/Default/MyProcess.PROCESS.xml"

iics publish start `
  --asset "Explore/Default/MyProcess.PROCESS.xml" `
  --asset "Explore/Default/MyConn.AI_CONNECTION.xml"

iics publish start --from-file assets.txt
iics publish start --from-file assets.csv
iics publish start --from-file assets.json

iics objects list -q "type=='PROCESS'" -o csv | iics publish start
```

---

## publish status

Retrieve the current status of a publish job.

### Flags

| Flag | Type | Default | Description | Required |
| ---- | ---- | ------- | ----------- | -------- |
| `--id` | string | | Publish job ID | yes |
| `--cai-url` | string | | CAI base URL override | |
| `--full` | bool | false | Fetch and display per-asset item detail | |

### Output columns

| Header | Description |
| ------ | ----------- |
| ID | Job ID |
| TYPE | "publish" |
| STATE | Job state: NOT_STARTED, PROCESSING, COMPLETED, FAILED |
| TOTAL | Total number of assets in the job |
| PROCESSED | Number of assets processed so far |
| STARTED | Job start timestamp (ISO 8601) |
| BY | Username that started the job |

### Examples

```bash
iics publish status --id <job-id>
iics publish status --id <job-id> --full
iics publish status --id <job-id> --output json
```

```powershell
iics publish status --id $jobId
iics publish status --id $jobId --full
iics publish status --id $jobId --output json
```

---

## publish run

Full workflow: resolve inputs, auto-batch, submit each batch, poll to completion,
and print a detailed summary. Mirrors the `import run` pattern.

### Flags

| Flag | Type | Default | Description | Required |
| ---- | ---- | ------- | ----------- | -------- |
| `--asset` | string (repeatable) | | Asset path(s) to publish | one input required |
| `--from-file` | string | | File with asset paths (.txt/.csv/.json/.yaml); omit to read from stdin | |
| `--cai-url` | string | | CAI base URL override | |
| `--name` | string | | Optional job label | |
| `--polling-interval` | int | 10 | Seconds between status polls | |
| `--max-wait-time` | int | 300 | Maximum seconds to wait for completion | |
| `--detailed-polling` | bool | false | Print per-asset item detail table on each poll (requires `--verbose`) | |
| `--item-fields` | string | see below | Comma-separated item detail fields to display | |

### Input format

Same as `publish start`. See the [Input format](#input-format) section above.

### Output

On each status poll, the command always prints:

```text
[HH:MM:SS] Published X out of Y asset(s). State: <state> elapsed: <duration>
```

When `--verbose` and `--detailed-polling` are both set, a per-asset item detail table is
printed after each poll line.

After all batches complete, a summary is printed. The layout depends on the number of batches:

**Single batch** - vertical field/value table:

| Field | Description |
| ----- | ----------- |
| Job ID | Publish job ID |
| State | Final job state |
| Start Date | Job start timestamp |
| End Date | Job end timestamp |
| Duration | Total elapsed time (human-readable) |
| Total | Total number of assets |
| Published | Number of successfully published assets |
| Errors | Number of assets that failed |

**Multiple batches** - horizontal table with one row per batch plus a `TOTAL` aggregate row:

| Column | Description |
| ------ | ----------- |
| BATCH | Batch number or `TOTAL` for the aggregate row |
| JOB ID | Publish job ID for this batch |
| STATE | Final job state for this batch; aggregate is `ERROR` if any batch failed |
| TOTAL | Number of assets in this batch |
| PUBLISHED | Number of successfully published assets |
| ERRORS | Number of failed assets |
| START DATE | Batch start timestamp |
| END DATE | Batch end timestamp |
| DURATION | Total elapsed time for this batch (aggregate: first start to last end) |

When `--verbose` is set, a combined per-asset item detail table is printed after the summary.
For multi-batch runs the table includes a `BATCH` column. The `DETAIL` column is not included
in the items table.

If any assets failed with a non-empty status detail message, a separate **Errors** section is
printed after the items table showing: batch number (multi-batch only), index, GUID, asset
path, and detail message.

### Exit codes

- `0` - all batches completed (including `ERROR`/`WARNING` final state or timeout)
- `1` - an API call failed or the input is invalid

### Item detail fields

Default: `itemIndex,itemGUID,assetPath,itemState,itemStartDate,itemEndDate,duration`

Override with `--item-fields` using any combination of:

| Field name | Description |
| ---------- | ----------- |
| `itemIndex` | Asset index within the batch |
| `itemGUID` | Published asset GUID |
| `assetPath` | Full `Explore/...xml` asset path |
| `itemState` | Per-asset result state (`SUCCESS`, `BAD_REQUEST`, etc.) |
| `itemStatusDetail` | Status detail message (non-empty on failure) |
| `itemStartDate` | Asset processing start timestamp |
| `itemEndDate` | Asset processing end timestamp |
| `duration` | Processing duration (calculated, human-readable) |

### Examples

```bash
# Publish a single asset
iics publish run --asset "Explore/Default/MyProcess.PROCESS.xml"

# Publish from a text file with verbose progress and item details
iics publish run --from-file assets.txt --verbose --detailed-polling
```

```powershell
# Publish a single asset
iics publish run --asset "Explore/Default/MyProcess.PROCESS.xml"

# Publish from a text file with verbose progress and item details
iics publish run --from-file assets.txt --verbose --detailed-polling

# Recommended: publish from CSV using location field
iics objects list -q "location==MyProject/Processes" -o csv --output-fields location \
  > assets.csv
iics publish run --from-file assets.csv --verbose

# Pipe directly from objects list (stdin auto-detect)
iics objects list -q "type=='PROCESS'" -o csv --output-fields location | iics publish run
```

```powershell
# Recommended: publish from CSV using location field
iics objects list -q "location==MyProject/Processes" -o csv --output-fields location `
  | Out-File assets.csv
iics publish run --from-file assets.csv --verbose

# Pipe directly from objects list (stdin auto-detect)
iics objects list -q "type=='PROCESS'" -o csv --output-fields location | iics publish run

# Custom polling interval, timeout, and item field selection
iics publish run \
  --from-file assets.csv \
  --polling-interval 15 \
  --max-wait-time 600 \
  --detailed-polling \
  --item-fields "itemIndex,assetPath,itemState,duration" \
  --verbose

# JSON output for CI/CD
iics publish run --from-file assets.txt --output json
```

```powershell
# Custom polling interval, timeout, and item field selection
iics publish run `
  --from-file assets.csv `
  --polling-interval 15 `
  --max-wait-time 600 `
  --detailed-polling `
  --item-fields "itemIndex,assetPath,itemState,duration" `
  --verbose

# JSON output for CI/CD
iics publish run --from-file assets.txt --output json
```
