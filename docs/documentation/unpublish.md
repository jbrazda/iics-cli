# unpublish

Unpublish Cloud Application Integration (CAI) assets from the IICS runtime.

## Synopsis

```text
iics unpublish <subcommand> [flags]
```

## Subcommands

| Subcommand | Description |
| ---------- | ----------- |
| `start` | Submit an unpublish job and return the job ID (fire-and-forget) |
| `status` | Retrieve the current status of an unpublish job by ID |
| `run` | Full workflow: resolve inputs, auto-batch, submit, poll to completion, print summary |

## CAI URL Auto-detection

The unpublish API uses the same CAI-specific service host as `publish`. The host is derived
automatically from the login response `products[].baseApiUrl` field (scheme+host only).
No configuration is required for normal use.

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
`Explore/<path>.<TYPE>` prefix. Appending `.xml` gives the exact value the unpublish API
expects. This is the recommended source for generating unpublish lists.

### Supported asset types

| Type suffix | Asset type |
| ----------- | ---------- |
| `AI_SERVICE_CONNECTOR` | Service connector |
| `AI_CONNECTION` | Application integration connection |
| `PROCESS` | Process |
| `GUIDE` | Guide |
| `TASKFLOW` | Taskflow |

## Unpublish Order

Assets are automatically sorted in reverse dependency order before being submitted to the API.
This ensures that dependents are unpublished before the assets they depend on, avoiding
failures caused by active dependencies. Note that unpublishing `AI_SERVICE_CONNECTOR` or
`AI_CONNECTION` will fail if any published asset still depends on them.

| Order | Type suffix | Asset type |
| ----- | ----------- | ---------- |
| 1 | `TASKFLOW` | Taskflow |
| 2 | `GUIDE` | Guide |
| 3 | `PROCESS` | Process |
| 4 | `AI_CONNECTION` | Application integration connection |
| 5 | `AI_SERVICE_CONNECTOR` | Service connector |

Within each type group the original relative order from the input file is preserved
(stable sort). Assets of unknown type are placed at the beginning.

## Batch Limit

The unpublish API accepts at most 199 assets per request. When more than 199 assets are
provided, `unpublish run` and `unpublish start` automatically split them into sequential
batches of up to 199 each.

---

## unpublish start

Submit an unpublish job and print the job ID. Does not wait for completion.

### Flags

| Flag | Type | Default | Description | Required |
| ---- | ---- | ------- | ----------- | -------- |
| `--asset` | string (repeatable) | | Asset path(s) to unpublish | one input required |
| `--from-file` | string | | File with asset paths (txt/json/csv); omit to read from stdin | |
| `--cai-url` | string | | CAI base URL override | |
| `--name` | string | | Optional job label for verbose output | |

### Input format

`--from-file` uses the file extension to select a parser. When reading from stdin the format
is auto-detected from content.

| Extension / source | Behaviour |
| ------------------ | --------- |
| `.txt` | One asset path per line, no header. Lines starting with `#` are comments. Paths without a `.xml` suffix have it appended automatically. |
| `.csv` | CSV with header row. Resolution order per row: (1) `LOCATION` column - appends `.xml`; (2) `PATH` + `TYPE` - builds `Explore/<PATH>.<TYPE>.xml`; (3) `PATH` only - appends `.xml` if missing. At least one of `LOCATION` or `PATH` is required. Non-publishable types are skipped silently. |
| `.json` | JSON array of strings (direct paths) or array of objects from `iics objects list -o json`. Per object: `location` field preferred (appends `.xml`); falls back to `path`+`type` conversion; then `path` with `.xml` appended if missing. |
| `.yaml` / `.yml` | Same as `.json` but YAML format (output of `iics objects list -o yaml`). |
| stdin | Auto-detected: starts with `[` - JSON; commas in first line or first line is `PATH` or `LOCATION` - CSV; otherwise plain text. |

#### Generating an asset list from `objects list`

The `location` field is the recommended source. It is already in `Explore/<path>.<TYPE>`
format - appending `.xml` gives the exact asset path the unpublish API expects.

```bash
# Recommended: CSV using the location field
iics objects list -q "location==MyProject/Processes" -o csv --output-fields location \
  > assets.csv
iics unpublish start --from-file assets.csv

# Pipe directly using location (stdin auto-detect)
iics objects list -q "type=='PROCESS'" -o csv --output-fields location | iics unpublish start

# JSON (location field included by default)
iics objects list -q "location==MyProject/Processes" -o json > assets.json
iics unpublish start --from-file assets.json
```

```powershell
# Recommended: CSV using the location field
iics objects list -q "location==MyProject/Processes" -o csv --output-fields location `
  | Out-File assets.csv
iics unpublish start --from-file assets.csv

# Pipe directly using location (stdin auto-detect)
iics objects list -q "type=='PROCESS'" -o csv --output-fields location | iics unpublish start

# JSON (location field included by default)
iics objects list -q "location==MyProject/Processes" -o json | Out-File assets.json
iics unpublish start --from-file assets.json
```

### Examples

```bash
iics unpublish start --asset "Explore/Default/MyProcess.PROCESS.xml"

iics unpublish start \
  --asset "Explore/Default/MyProcess.PROCESS.xml" \
  --asset "Explore/Default/MyConn.AI_CONNECTION.xml"

iics unpublish start --from-file assets.txt
iics unpublish start --from-file assets.csv
iics unpublish start --from-file assets.json

iics objects list -q "type=='PROCESS'" -o csv | iics unpublish start
```

```powershell
iics unpublish start --asset "Explore/Default/MyProcess.PROCESS.xml"

iics unpublish start `
  --asset "Explore/Default/MyProcess.PROCESS.xml" `
  --asset "Explore/Default/MyConn.AI_CONNECTION.xml"

iics unpublish start --from-file assets.txt
iics unpublish start --from-file assets.csv
iics unpublish start --from-file assets.json

iics objects list -q "type=='PROCESS'" -o csv | iics unpublish start
```

---

## unpublish status

Retrieve the current status of an unpublish job.

### Flags

| Flag | Type | Default | Description | Required |
| ---- | ---- | ------- | ----------- | -------- |
| `--id` | string | | Unpublish job ID | yes |
| `--cai-url` | string | | CAI base URL override | |
| `--full` | bool | false | Fetch and display per-asset item detail | |

### Output columns

| Header | Description |
| ------ | ----------- |
| ID | Job ID |
| TYPE | "unpublish" |
| STATE | Job state: NOT_STARTED, PROCESSING, COMPLETED, FAILED |
| TOTAL | Total number of assets in the job |
| PROCESSED | Number of assets processed so far |
| STARTED | Job start timestamp (ISO 8601) |
| BY | Username that started the job |

### Examples

```bash
iics unpublish status --id <job-id>
iics unpublish status --id <job-id> --full
iics unpublish status --id <job-id> --output json
```

```powershell
iics unpublish status --id $jobId
iics unpublish status --id $jobId --full
iics unpublish status --id $jobId --output json
```

---

## unpublish run

Full workflow: resolve inputs, auto-batch, submit each batch, poll to completion,
and print a detailed summary.

### Flags

| Flag | Type | Default | Description | Required |
| ---- | ---- | ------- | ----------- | -------- |
| `--asset` | string (repeatable) | | Asset path(s) to unpublish | one input required |
| `--from-file` | string | | File with asset paths (.txt/.csv/.json/.yaml); omit to read from stdin | |
| `--cai-url` | string | | CAI base URL override | |
| `--name` | string | | Optional job label | |
| `--polling-interval` | int | 10 | Seconds between status polls | |
| `--max-wait-time` | int | 300 | Maximum seconds to wait for completion | |
| `--detailed-polling` | bool | false | Print per-asset item detail table on each poll (requires `--verbose`) | |
| `--item-fields` | string | see publish run | Comma-separated item detail fields to display | |

### Input format

Same as `unpublish start`. See the [Input format](#input-format) section above.

### Output

Same polling progress, summary, and item detail behaviour as `publish run`. See
[publish run Output](publish.md#output) for field descriptions and `--item-fields` values.

### Exit codes

- `0` - all batches completed successfully
- `1` - at least one batch failed or timed out

### Examples

```bash
# Unpublish a single asset
iics unpublish run --asset "Explore/Default/MyProcess.PROCESS.xml"

# Unpublish from a text file (pre-formatted Explore/...xml paths)
iics unpublish run --from-file assets.txt --verbose
```

```powershell
# Unpublish a single asset
iics unpublish run --asset "Explore/Default/MyProcess.PROCESS.xml"

# Unpublish from a text file (pre-formatted Explore/...xml paths)
iics unpublish run --from-file assets.txt --verbose

# Unpublish from CSV with automatic path conversion (PATH + TYPE)
iics objects list -q "location==MyProject/Processes" -o csv --output-fields path,type \
  > assets.csv
iics unpublish run --from-file assets.csv

# Unpublish from JSON
iics objects list -q "location==MyProject/Processes" -o json > assets.json
iics unpublish run --from-file assets.json

# Pipe directly from objects list (stdin auto-detect)
iics objects list -q "type=='PROCESS'" -o csv | iics unpublish run
```

```powershell
# Unpublish from CSV with automatic path conversion (PATH + TYPE)
iics objects list -q "location==MyProject/Processes" -o csv --output-fields path,type `
  | Out-File assets.csv
iics unpublish run --from-file assets.csv

# Unpublish from JSON
iics objects list -q "location==MyProject/Processes" -o json | Out-File assets.json
iics unpublish run --from-file assets.json

# Pipe directly from objects list (stdin auto-detect)
iics objects list -q "type=='PROCESS'" -o csv | iics unpublish run

# Custom polling and timeout
iics unpublish run \
  --from-file assets.txt \
  --polling-interval 15 \
  --max-wait-time 600 \
  --detailed-polling \
  --verbose

# JSON output for CI/CD
iics unpublish run --from-file assets.txt --output json
```

```powershell
# Custom polling and timeout
iics unpublish run `
  --from-file assets.txt `
  --polling-interval 15 `
  --max-wait-time 600 `
  --detailed-polling `
  --verbose

# JSON output for CI/CD
iics unpublish run --from-file assets.txt --output json
```
