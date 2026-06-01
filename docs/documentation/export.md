# export

Export IICS objects to a ZIP package. Supports a full automated pipeline (`export run`) or individual steps for advanced control.

## Synopsis

```bash
iics export <subcommand> [flags]
```

## Subcommands

| Subcommand  | Description                                              |
| ----------- | -------------------------------------------------------- |
| `run`       | Full pipeline: resolve IDs, start job, poll, download   |
| `start`     | Resolve IDs and start job, return job ID only           |
| `status`    | Check export job status                                  |
| `download`  | Download completed export ZIP                            |
| `create`    | Start export with explicit object IDs                   |

---

## export run

The recommended command for most use cases. Reads an artifacts list, resolves paths to IDs, starts the export job, polls until complete, and downloads the ZIP.

### Flags

| Flag                    | Short | Type   | Required | Default    | Description                                                        |
| ----------------------- | ----- | ------ | -------- | ---------- | ------------------------------------------------------------------ |
| `--artifacts-file`      |       | string |          | stdin      | File listing objects to export (txt/json/yaml/csv); omit for stdin |
| `--export-file-path`    |       | string | yes      |            | Output ZIP file path                                               |
| `-n, --name`            | `-n`  | string |          | timestamp  | Export job name                                                    |
| `--include-tags`        |       | bool   |          | false      | Include tag data in the export                                     |
| `--exclude-dependencies`|       | bool   |          | false      | Do not include dependent objects                                   |
| `--polling-interval`    |       | int    |          | 10         | Seconds between status polls                                       |
| `--max-wait-time`       |       | int    |          | 300        | Maximum seconds to wait for the job                                |
| `--expand-status`       |       | bool   |          | false      | Print the per-object export status table                           |
| `--print-file-contents` |       | bool   |          | false      | List ZIP file contents after download                              |
| `--download-export-log` |       | string |          |            | Download export log to this path (default: `<zip>.log`)           |
| `--log-file`            |       | string |          |            | Append a Markdown release report section; use `--log-file=PATH` to override default path |

All [global flags](../../README.md#global-flags) apply.

### Artifacts file format

The artifacts file lists objects to export, one per line. Supported formats:

**Plain text (`.txt`)** - one object path per line:

```text
My Project/ETL/LoadOrders
My Project/ETL/TransformOrders
Shared/Connections/SalesforceConn
```

**CSV** - objects list output from `iics objects list -o csv`:

```bash
iics objects list -q "location==ZZ_TEST_CLI" -o csv | iics export run --export-file-path export.zip
```

**YAML/JSON** - structured list with `id` and `type` fields.

### Examples

```bash
# Export with an artifacts file
iics export run \
  --artifacts-file my-objects.txt \
  --export-file-path exports/my-export.zip

# Export with verbose polling output
iics export run \
  --artifacts-file my-objects.txt \
  --export-file-path exports/my-export.zip \
  --verbose \
  --expand-status

# Pipe objects list directly into export
iics objects list -q "location==MY_PROJECT" -o csv \
  | iics export run --export-file-path my-project-export.zip

# Export with a custom name and include tags
iics export run \
  --artifacts-file objects.txt \
  --export-file-path release-v2.zip \
  --name "Release v2.0" \
  --include-tags \
  --print-file-contents

# Long-running export with extended timeout
iics export run \
  --artifacts-file big-project.txt \
  --export-file-path big-project.zip \
  --polling-interval 30 \
  --max-wait-time 1800
```

```powershell
# Export with an artifacts file
iics export run `
  --artifacts-file my-objects.txt `
  --export-file-path exports/my-export.zip

# Export with verbose polling output
iics export run `
  --artifacts-file my-objects.txt `
  --export-file-path exports/my-export.zip `
  --verbose `
  --expand-status

# Get objects list and save to file, then export
iics objects list -q "location==MY_PROJECT" -o csv | Out-File objects.csv
iics export run --artifacts-file objects.csv --export-file-path my-project-export.zip

# Export with a custom name and include tags
iics export run `
  --artifacts-file objects.txt `
  --export-file-path release-v2.zip `
  --name "Release v2.0" `
  --include-tags `
  --print-file-contents

# Long-running export with extended timeout
iics export run `
  --artifacts-file big-project.txt `
  --export-file-path big-project.zip `
  --polling-interval 30 `
  --max-wait-time 1800
```

---

## export start

Resolve artifact paths to IDs, start the export job, and return the job ID without waiting.

### Flags

| Flag                    | Short | Type   | Required | Default   | Description                                      |
| ----------------------- | ----- | ------ | -------- | --------- | ------------------------------------------------ |
| `--artifacts-file`      |       | string |          | stdin     | Artifacts list file                              |
| `-n, --name`            | `-n`  | string |          | timestamp | Export job name                                  |
| `--include-tags`        |       | bool   |          | false     | Include tag data                                 |
| `--exclude-dependencies`|       | bool   |          | false     | Exclude dependencies                             |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Start export and capture the job ID
JOB_ID=$(iics export start --artifacts-file objects.txt --output json | jq -r '.id')
echo "Job started: $JOB_ID"

# Check status separately
iics export status --id "$JOB_ID"
```

```powershell
# Start export and capture the job ID
$job = iics export start --artifacts-file objects.txt --output json | ConvertFrom-Json
$jobId = $job.id
Write-Host "Job started: $jobId"

# Check status separately
iics export status --id $jobId
```

---

## export status

Check the status of an export job.

### Flags

| Flag       | Type   | Required | Description                                  |
| ---------- | ------ | -------- | -------------------------------------------- |
| `--id`     | string | yes      | Export job ID                                |
| `--expand` | bool   |          | Show per-object export status                |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column          | Description                                  |
| --------------- | -------------------------------------------- |
| `id`            | Job ID                                       |
| `name`          | Job name                                     |
| `status.state`  | Current state: `SUCCESSFUL`, `FAILED`, `IN_PROGRESS`, etc. |
| `status.message`| Status detail message                        |

### Examples

```bash
iics export status --id <job-id>

# Show per-object results
iics export status --id <job-id> --expand

# Poll until done (simple loop)
while true; do
  STATE=$(iics export status --id "$JOB_ID" --output json | jq -r '.status.state')
  echo "State: $STATE"
  [[ "$STATE" == "SUCCESSFUL" || "$STATE" == "FAILED" ]] && break
  sleep 10
done
```

```powershell
iics export status --id <job-id>

# Show per-object results
iics export status --id <job-id> --expand

# Poll until done (simple loop)
do {
  $status = iics export status --id $jobId --output json | ConvertFrom-Json
  $state = $status.status.state
  Write-Host "State: $state"
  if ($state -eq "SUCCESSFUL" -or $state -eq "FAILED") { break }
  Start-Sleep -Seconds 10
} while ($true)
```

---

## export download

Download the ZIP package for a completed export job.

### Flags

| Flag              | Short | Type   | Required | Default              | Description                |
| ----------------- | ----- | ------ | -------- | -------------------- | -------------------------- |
| `--id`            |       | string | yes      |                      | Export job ID              |
| `--output-file`   | `-f`  | string |          | `export_<id>.zip`   | Output file path           |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics export download --id <job-id> --output-file release.zip
```

```powershell
iics export download --id <job-id> --output-file release.zip
```

---

## export create

Start an export job with explicit comma-separated object IDs.

### Flags

| Flag              | Type   | Required | Description                                |
| ----------------- | ------ | -------- | ------------------------------------------ |
| `--name`          | string | yes      | Export job name                            |
| `--ids`           | string | yes      | Comma-separated object IDs to export       |
| `--include-deps`  | bool   |          | Include dependent objects                  |
| `--wait`          | bool   |          | Wait for the job to complete               |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics export create \
  --name "My Export" \
  --ids "aLX7qnviqxJdmqpVsd17SG,3OrYJxTOzUJdwgHskwZCcj" \
  --include-deps \
  --wait
```

## Job states

| State         | Meaning                           |
| ------------- | --------------------------------- |
| `QUEUED`      | Job is waiting to start           |
| `STARTING`    | Job is initialising               |
| `IN_PROGRESS` | Export is running                 |
| `SUCCESSFUL`  | Export completed successfully     |
| `WARNINGS`    | Completed with warnings           |
| `FAILED`      | Export failed                     |
| `ERROR`       | System error                      |

## See also

- [import](import.md) - import an export ZIP package
- [objects](objects.md) - list and filter objects to export
- [lookup](lookup.md) - resolve object paths to IDs
