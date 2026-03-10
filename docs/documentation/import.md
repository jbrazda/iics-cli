# import

Import a ZIP package into IICS. Supports a full automated pipeline (`import run`) or individual steps for advanced control. Alias: `imp`.

## Synopsis

```bash
iics import <subcommand> [flags]
iics imp <subcommand> [flags]
```

## Subcommands

| Subcommand       | Description                                                |
| ---------------- | ---------------------------------------------------------- |
| `run`            | Full pipeline: upload, start, poll, download log          |
| `upload`         | Upload a ZIP package and get a job ID                     |
| `start`          | Start an import job from an uploaded package              |
| `status`         | Check import job status                                    |
| `download-log`   | Download the import log to a file                         |

---

## import run

The recommended command for most use cases. Uploads the ZIP, starts the import, polls until complete, and optionally prints the log.

### Flags

| Flag                    | Short | Type   | Required | Default         | Description                                                  |
| ----------------------- | ----- | ------ | -------- | --------------- | ------------------------------------------------------------ |
| `--zip-file`            | `-z`  | string | yes      |                 | ZIP package to import                                        |
| `--name`                |       | string |          | zip filename    | Import job name                                              |
| `--conflict-resolution` |       | string |          | `REUSE`         | How to handle conflicts: `REUSE` or `OVERWRITE`             |
| `--from-file`           |       | string |          |                 | JSON file with full import specification (overrides other flags) |
| `--polling-interval`    |       | int    |          | 10              | Seconds between status polls                                 |
| `--max-wait-time`       |       | int    |          | 300             | Maximum seconds to wait                                      |
| `--detailed-polling`    |       | bool   |          | false           | Print per-object status on every poll                        |
| `--print-import-log`    |       | bool   |          | false           | Print the import log after completion                        |
| `--expand`              |       | bool   |          | false           | Expand object list in final output                           |

All [global flags](../../README.md#global-flags) apply.

### Conflict resolution modes

| Mode        | Behaviour                                                       |
| ----------- | --------------------------------------------------------------- |
| `REUSE`     | (default) Keep existing objects; skip conflicting imports       |
| `OVERWRITE` | Replace existing objects with imported versions                 |

### Examples

```bash
# Basic import
iics import run --zip-file release-v2.zip

# Overwrite existing objects
iics import run --zip-file release-v2.zip --conflict-resolution OVERWRITE

# With verbose polling and log output
iics import run \
  --zip-file release-v2.zip \
  --verbose \
  --detailed-polling \
  --print-import-log \
  --expand

# Custom job name with extended timeout
iics import run \
  --zip-file big-package.zip \
  --name "Release 3.0 - Full Deploy" \
  --polling-interval 30 \
  --max-wait-time 1800

# Using a full import specification JSON
iics import run --from-file import-spec.json --zip-file package.zip

# CI/CD pipeline pattern
iics import run \
  --zip-file "$ARTIFACT_PATH" \
  --conflict-resolution OVERWRITE \
  --print-import-log \
  --expand \
  --profile prod
```

---

## import upload

Upload a ZIP package to IICS to get a job ID. Use `import start` to begin the actual import.

### Flags

| Flag     | Short | Type   | Required | Description         |
| -------- | ----- | ------ | -------- | ------------------- |
| `--file` | `-f`  | string | yes      | ZIP file to upload  |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics import upload --file release.zip

# Capture the job ID for subsequent commands
JOB_ID=$(iics import upload --file release.zip --output json | jq -r '.id')
```

---

## import start

Start an import job from a previously uploaded package.

### Flags

| Flag                    | Type   | Required | Default | Description                                                |
| ----------------------- | ------ | -------- | ------- | ---------------------------------------------------------- |
| `--id`                  | string | yes      |         | Job ID from `import upload`                                |
| `--name`                | string |          |         | Import job name                                            |
| `--conflict-resolution` | string |          | `REUSE` | `REUSE` or `OVERWRITE`                                     |
| `--from-file`           | string |          |         | JSON file with full import specification                   |
| `--wait`                | bool   |          | false   | Wait for the job to complete before returning              |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics import start --id "$JOB_ID" --conflict-resolution OVERWRITE --wait
```

---

## import status

Check the current status of an import job.

### Flags

| Flag       | Type   | Required | Description                          |
| ---------- | ------ | -------- | ------------------------------------ |
| `--id`     | string | yes      | Import job ID                        |
| `--expand` | bool   |          | Show per-object import status        |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column           | Description                                                    |
| ---------------- | -------------------------------------------------------------- |
| `id`             | Job ID                                                         |
| `name`           | Job name                                                       |
| `status.state`   | Current state: `SUCCESSFUL`, `FAILED`, `IN_PROGRESS`, etc.    |
| `status.message` | Status detail                                                  |

### Examples

```bash
iics import status --id <job-id>

# Show per-object results
iics import status --id <job-id> --expand
```

---

## import download-log

Download the import log file for a completed job.

### Flags

| Flag          | Type   | Required | Default                        | Description                              |
| ------------- | ------ | -------- | ------------------------------ | ---------------------------------------- |
| `--id`        | string | yes      |                                | Import job ID                            |
| `--log-path`  | string | yes      |                                | Directory to save the log file           |
| `--log-name`  | string |          | `<name>_<id>_<status>.log`    | Log filename                             |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics import download-log --id <job-id> --log-path ./logs

# Custom filename
iics import download-log --id <job-id> --log-path ./logs --log-name import-prod.log
```

## Job states

| State         | Meaning                          |
| ------------- | -------------------------------- |
| `QUEUED`      | Job is waiting to start          |
| `STARTING`    | Job is initialising              |
| `IN_PROGRESS` | Import is running                |
| `SUCCESSFUL`  | Import completed successfully    |
| `WARNINGS`    | Completed with warnings          |
| `FAILED`      | Import failed                    |
| `ERROR`       | System error                     |

## See also

- [export](export.md) - create a ZIP package to import
- [objects](objects.md) - verify imported objects
