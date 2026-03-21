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

Asset paths must follow the format: `Explore/<folder-path>/<asset-name>.<type-suffix>`

### Supported asset type suffixes

| Suffix | Asset type |
| ------ | ---------- |
| `.PROCESS.xml` | Process |
| `.AI_SERVICE_CONNECTOR.xml` | Service connector |
| `.AI_CONNECTION.xml` | Application integration connection |
| `.DTEMPLATE.xml` | Mapping |
| `.GUIDE.xml` | Guide |
| `.PROCESS_OBJECT.xml` | Process object |

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
| `.txt` | One asset path per line, no header. Lines starting with `#` are comments. |
| `.csv` | CSV with header row. `PATH` column required; `TYPE` column optional. When both are present rows are converted to `Explore/<PATH>.<TYPE>.xml` and non-publishable types are skipped silently. When only `PATH` is present the value is used as-is. |
| `.json` | JSON array of strings (direct asset paths) or array of objects with `path` and optional `type` (output of `iics objects list -o json`). When `type` is present, rows are converted; non-publishable types are skipped. |
| `.yaml` / `.yml` | Same as `.json` but YAML format (output of `iics objects list -o yaml`). |
| stdin | Auto-detected: starts with `[` - JSON; commas in first line or first line is `PATH` - CSV; otherwise plain text. |

#### Generating an asset list from `objects list`

```bash
# CSV with automatic path conversion (PATH + TYPE both present)
iics objects list -q "location==MyProject/Processes" -o csv --output-fields path,type \
  > assets.csv
iics publish start --from-file assets.csv

# Pipe directly (stdin auto-detect)
iics objects list -q "type=='PROCESS'" -o csv | iics publish start

# JSON with automatic path conversion
iics objects list -q "location==MyProject/Processes" -o json > assets.json
iics publish start --from-file assets.json

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

---

## publish status

Retrieve the current status of a publish job.

### Flags

| Flag | Type | Default | Description | Required |
| ---- | ---- | ------- | ----------- | -------- |
| `--id` | string | | Publish job ID | yes |
| `--cai-url` | string | | CAI base URL override | |
| `--full` | bool | false | Fetch full job object including asset list | |

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
| `--detailed-polling` | bool | false | Print totalCount/processedCount on each poll | |

### Input format

Same as `publish start`. See the [Input format](#input-format) section above.

### Output columns

Same as `publish status`.

### Exit codes

- `0` - all batches completed successfully
- `1` - at least one batch failed or timed out

### Examples

```bash
# Publish a single asset
iics publish run --asset "Explore/Default/MyProcess.PROCESS.xml"

# Publish from a text file (pre-formatted Explore/...xml paths)
iics publish run --from-file assets.txt --verbose

# Publish from CSV with automatic path conversion (PATH + TYPE)
iics objects list -q "location==MyProject/Processes" -o csv --output-fields path,type \
  > assets.csv
iics publish run --from-file assets.csv

# Publish from JSON
iics objects list -q "location==MyProject/Processes" -o json > assets.json
iics publish run --from-file assets.json

# Pipe directly from objects list (stdin auto-detect)
iics objects list -q "type=='PROCESS'" -o csv | iics publish run

# Custom polling and timeout
iics publish run \
  --from-file assets.txt \
  --polling-interval 15 \
  --max-wait-time 600 \
  --detailed-polling \
  --verbose

# JSON output for CI/CD
iics publish run --from-file assets.txt --output json
```
