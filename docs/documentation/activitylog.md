# activitylog

Query the IICS activity log for completed job runs.

The activity log records execution history for tasks including row counts, run state,
timing, and error messages. Uses the Platform REST API v2.

## Synopsis

```bash
iics activitylog <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                           |
| ---------- | ------------------------------------- |
| `list`     | List activity log entries             |
| `get`      | Get a single activity log entry by ID |

---

## activitylog list

### Flags

| Flag        | Type   | Default | Description                                     |
| ----------- | ------ | ------- | ----------------------------------------------- |
| `--task-id` | string |         | Filter by task ID                               |
| `--run-id`  | int    | 0       | Filter by run ID (requires `--task-id`)         |
| `--offset`  | int    | 0       | Number of rows to skip                          |
| `--limit`   | int    | 200     | Max results returned (API max: 1000)            |
| `--fields`  | string | (1)     | Field names for table/csv output                |

(1) Default fields: `id,objectName,type,state,runId,objectId,startTimeUtc,endTimeUtc,startedBy`

All [global flags](../../README.md#global-flags) apply.

`--fields` controls which columns appear in `table` and `csv` output. `json` and `yaml` output
always includes all fields regardless of `--fields`.

### Available fields

| Field name             | Header          | Description                                                              |
| ---------------------- | --------------- | ------------------------------------------------------------------------ |
| `id`                   | ID              | Log entry identifier (default)                                           |
| `objectName`           | NAME            | Task name (default)                                                      |
| `type`                 | TYPE            | Task type: DMASK, DRS, DSS, MTT, PCS, WORKFLOW (default)                 |
| `state`                | STATE           | Job outcome: SUCCESS, ERRORS, FAILED, NOT STARTED (default)              |
| `runId`                | RUN ID          | Task run identifier (default)                                            |
| `objectId`             | TASK ID         | Task asset identifier (default)                                          |
| `startTimeUtc`         | START TIME      | Job start timestamp in UTC (default)                                     |
| `endTimeUtc`           | END TIME        | Job end timestamp in UTC (default)                                       |
| `startedBy`            | STARTED BY      | User who initiated the task (default)                                    |
| `agentId`              | AGENT ID        | Executing Secure Agent identifier                                        |
| `runtimeEnvironmentId` | RUNTIME ENV     | Runtime environment identifier                                           |
| `startTime`            | START TIME (ET) | Job start timestamp in Eastern Time                                      |
| `endTime`              | END TIME (ET)   | Job end timestamp in Eastern Time                                        |
| `failedSourceRows`     | FAILED SRC      | Source rows that could not be read                                       |
| `successSourceRows`    | SUCCESS SRC     | Source rows successfully read                                            |
| `failedTargetRows`     | FAILED TGT      | Target rows that could not be written                                    |
| `successTargetRows`    | SUCCESS TGT     | Target rows successfully written                                         |
| `totalSuccessRows`     | TOTAL SUCCESS   | Total successfully processed rows                                        |
| `totalFailedRows`      | TOTAL FAILED    | Total failed rows                                                        |
| `scheduleName`         | SCHEDULE        | Schedule name if the job was triggered by a schedule                     |
| `errorMsg`             | ERROR           | Error message if the job failed                                          |
| `runContextType`       | RUN CONTEXT     | How the job was launched: ICS_UI, SCHEDULER, REST-API, OUTBOUND MESSAGE  |
| `isStopped`            | STOPPED         | Whether the job was manually stopped                                     |
| `stopOnError`          | STOP ON ERR     | Whether the job was configured to stop on first error                    |
| `contextExternalId`    | CONTEXT ID      | Parent task identifier for sub-tasks                                     |

### Examples

```bash
# List recent activity log entries (default: up to 200)
iics activitylog list

# Filter by task ID
iics activitylog list --task-id abc123

# Filter by a specific run of a task
iics activitylog list --task-id abc123 --run-id 42

# Page through results
iics activitylog list --offset 200 --limit 200

# Select specific fields for table output
iics activitylog list --fields id,objectName,state,errorMsg

# Add row counts to the default columns
iics activitylog list --fields id,objectName,state,runId,successTargetRows,failedTargetRows,startTimeUtc

# As JSON for scripting (all fields always included)
iics activitylog list --output json

# Find failed jobs
iics activitylog list --output json \
  | jq '.[] | select(.state == 3)'

# Export to CSV with selected fields
iics activitylog list \
  --task-id abc123 \
  --fields id,objectName,state,runId,startTimeUtc,endTimeUtc,errorMsg \
  --output csv > task-history.csv
```

```powershell
# List recent activity log entries (default: up to 200)
iics activitylog list

# Filter by task ID
iics activitylog list --task-id abc123

# Filter by a specific run of a task
iics activitylog list --task-id abc123 --run-id 42

# Page through results
iics activitylog list --offset 200 --limit 200

# Select specific fields for table output
iics activitylog list --fields id,objectName,state,errorMsg

# Add row counts to the default columns
iics activitylog list --fields id,objectName,state,runId,successTargetRows,failedTargetRows,startTimeUtc

# As JSON for scripting (all fields always included)
iics activitylog list --output json

# Find failed jobs
$logs = iics activitylog list --output json | ConvertFrom-Json
$logs | Where-Object { $_.state -eq 3 }

# Export to CSV with selected fields
iics activitylog list `
  --task-id abc123 `
  --fields id,objectName,state,runId,startTimeUtc,endTimeUtc,errorMsg `
  --output csv | Out-File task-history.csv
```

---

## activitylog get

Get a single activity log entry by its ID. For `table` output, displays the main entry followed
by separate labeled sections for any non-empty nested data (child entries, sub-task entries,
item attributes, session variables, transformation entries). For `json` and `yaml` output,
the full nested structure is included.

### Arguments

| Argument | Description  |
| -------- | ------------ |
| `<id>`   | Log entry ID |

### Flags

| Flag       | Type   | Default | Description                                      |
| ---------- | ------ | ------- | ------------------------------------------------ |
| `--fields` | string | (1)     | Comma-separated field names for table/csv output |

(1) Default fields: `id,objectName,type,state,runId,objectId,startTimeUtc,endTimeUtc,startedBy`

All [global flags](../../README.md#global-flags) apply.

### Nested sections (table output only)

| Section label             | Content                                                        |
| ------------------------- | -------------------------------------------------------------- |
| `Entries:`                | Child task entries using the same `--fields` columns           |
| `Sub-task Entries:`       | Sub-task entries (workflow sub-tasks) using `--fields` columns |
| `Item Attributes:`        | Key-value task metadata (compute units, session log, etc.)     |
| `Session Variables:`      | Advanced session properties as key-value pairs                 |
| `Transformation Entries:` | Per-transformation row counts (SOURCE and TARGET)              |

### Examples

```bash
# Get a specific log entry (table with nested sections)
iics activitylog get abc123

# Full JSON including all nested objects
iics activitylog get abc123 --output json
```

```powershell
# Get a specific log entry (table with nested sections)
iics activitylog get abc123

# Full JSON including all nested objects
iics activitylog get abc123 --output json
```

# Table with additional row count fields
iics activitylog get abc123 \
  --fields id,objectName,state,runId,successTargetRows,failedTargetRows,errorMsg
```

---

## State values

| Code | Label       |
| ---- | ----------- |
| 1    | SUCCESS     |
| 2    | ERRORS      |
| 3    | FAILED      |
| 4    | NOT STARTED |

## See also

- [securitylog](securitylog.md) - query the security audit log
- [metering](metering.md) - query usage and consumption data
- [state](state.md) - fetch and load object state snapshots
