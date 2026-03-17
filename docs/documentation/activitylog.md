# activitylog

Query the IICS activity log for completed job runs.

The activity log records execution history for tasks including row counts, run state,
timing, and error messages. Uses the Platform REST API v2.

## Synopsis

```bash
iics activitylog <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                              |
| ---------- | ---------------------------------------- |
| `list`     | List activity log entries                |
| `get`      | Get a single activity log entry by ID    |

---

## activitylog list

### Flags

| Flag            | Type   | Default | Description                                |
| --------------- | ------ | ------- | ------------------------------------------ |
| `--task-id`     | string |         | Filter by task ID                          |
| `--run-id`      | int    | 0       | Filter by run ID (requires `--task-id`)    |
| `--offset`      | int    | 0       | Number of rows to skip                     |
| `--limit`       | int    | 200     | Max results returned (API max: 1000)       |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column          | Description                                                        |
| --------------- | ------------------------------------------------------------------ |
| `id`            | Log entry identifier                                               |
| `objectName`    | Task name                                                          |
| `type`          | Task type: DMASK, DRS, DSS, MTT, PCS, WORKFLOW                     |
| `state`         | Status code: 1=success, 2=errors, 3=failed, 4=not started         |
| `startTimeUtc`  | Job start timestamp (UTC)                                          |
| `endTimeUtc`    | Job end timestamp (UTC)                                            |
| `startedBy`     | User who initiated the task                                        |
| `runContextType`| How the task was launched: ICS_UI, SCHEDULER, REST-API, OUTBOUND MESSAGE |

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

# As JSON for scripting
iics activitylog list --output json

# Find failed jobs
iics activitylog list --output json \
  | jq '.[] | select(.state == 3)'

# Export to CSV
iics activitylog list \
  --task-id abc123 \
  --output csv > task-history.csv
```

---

## activitylog get

Get a single activity log entry by its ID. Includes child entries for workflows.

### Arguments

| Argument | Description      |
| -------- | ---------------- |
| `<id>`   | Log entry ID     |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Get a specific log entry
iics activitylog get abc123

# As JSON (includes child entries and row counts)
iics activitylog get abc123 --output json
```

---

## State codes

| Code | Meaning     |
| ---- | ----------- |
| 1    | Success     |
| 2    | Errors      |
| 3    | Failed      |
| 4    | Not started |

## See also

- [securitylog](securitylog.md) - query the security audit log
- [metering](metering.md) - query usage and consumption data
- [state](state.md) - fetch and load object state snapshots
