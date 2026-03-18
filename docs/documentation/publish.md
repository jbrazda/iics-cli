# publish

Publish Cloud Application Integration (CAI) assets to the IICS runtime.

## Synopsis

```
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

The `--from-file` file (or stdin) is auto-detected:

- **Plain text** (`.txt`): one asset path per line
- **JSON array**: `["Explore/...", "Explore/...", ...]`
- **CSV** (output of `iics objects list -o csv`): rows with `PATH` and `TYPE` columns are
  converted to asset paths; rows with non-publishable types are silently skipped

### Examples

```bash
iics publish start --asset "Explore/Default/MyProcess.PROCESS.xml"

iics publish start \
  --asset "Explore/Default/MyProcess.PROCESS.xml" \
  --asset "Explore/Default/MyConn.AI_CONNECTION.xml"

iics publish start --from-file assets.txt

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
| `--from-file` | string | | File with asset paths (txt/json/csv); omit to read from stdin | |
| `--cai-url` | string | | CAI base URL override | |
| `--name` | string | | Optional job label | |
| `--polling-interval` | int | 10 | Seconds between status polls | |
| `--max-wait-time` | int | 300 | Maximum seconds to wait for completion | |
| `--detailed-polling` | bool | false | Print totalCount/processedCount on each poll | |

### Output columns

Same as `publish status`.

### Exit codes

- `0` - all batches completed successfully
- `1` - at least one batch failed or timed out

### Examples

```bash
# Publish a single asset
iics publish run --asset "Explore/Default/MyProcess.PROCESS.xml"

# Publish from a text file with verbose progress
iics publish run --from-file assets.txt --verbose

# Publish from a CSV file (iics objects list output)
iics publish run --from-file objects.csv

# Pipe directly from objects list
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
