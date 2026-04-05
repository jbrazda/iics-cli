# auditlog

Query the IICS organization audit log (V2 API).

The audit log records all actions performed in the organization: logins, object
creation/deletion/update, publish events, connection changes, user management, and more.
It is distinct from the [securitylog](securitylog.md) command, which uses the V3 API and
focuses on authentication events.

## Synopsis

```bash
iics auditlog <subcommand> [flags]
```

## Subcommands

| Subcommand | Description             |
| ---------- | ----------------------- |
| `list`     | List audit log entries  |

---

## auditlog list

### Flags

| Flag       | Type   | Default                                          | Description |
| ---------- | ------ | ------------------------------------------------ | ----------- |
| `--limit`  | int    | 0                                                | Entries per page (maps to API `batchSize`). When 0, the API returns the most recent 200 entries. |
| `--skip`   | int    | 0                                                | Page number to retrieve, 0-based (maps to API `batchId`). Only used when `--limit > 0`. |
| `--fields` | string | `id,username,category,event,entryTimeUTC,objectName` | Comma-separated list of fields to display (table and CSV output only). |

All [global flags](../../README.md#global-flags) apply.

### Output columns

All available fields (selectable via `--fields`):

| Field (JSON tag) | Header      | Description                                         |
| ---------------- | ----------- | --------------------------------------------------- |
| `id`             | ID          | Audit log entry identifier (default)                |
| `username`       | USERNAME    | User who performed the action (default)             |
| `category`       | CATEGORY    | Audit category: AGENT, AUTH, CONNECTION, USER, etc. (default) |
| `event`          | EVENT       | Action type: CREATE, DELETE, UPDATE, RUN, PUBLISH, etc. (default) |
| `entryTimeUTC`   | TIME (UTC)  | Timestamp of the action in UTC (default)            |
| `objectName`     | OBJECT NAME | Name of the affected object (default)               |
| `entryTime`      | TIME (ET)   | Timestamp in Eastern Time (optional)                |
| `objectId`       | OBJECT ID   | Identifier of the affected object (optional)        |
| `eventParam`     | EVENT PARAM | Related objects, max 1024 characters (optional)     |
| `message`        | MESSAGE     | Additional context information (optional)           |
| `orgId`          | ORG ID      | Organization identifier (optional)                  |
| `version`        | VERSION     | Entry version number (optional)                     |

### Pagination

The underlying API uses batch-based pagination. The CLI flags map as follows:

| CLI flag  | API parameter | Meaning                                      |
| --------- | ------------- | -------------------------------------------- |
| `--limit` | `batchSize`   | Number of entries per page                   |
| `--skip`  | `batchId`     | Page number (0 = most recent batch)          |

### Examples

```bash
# List the most recent 200 audit log entries (default)
iics auditlog list

# First page of 50 entries (most recent)
iics auditlog list --limit 50

# Second page of 50 entries
iics auditlog list --limit 50 --skip 1

# Custom columns
iics auditlog list --fields id,username,category,event,entryTimeUTC

# Full output as JSON
iics auditlog list --output json

# Export to CSV
iics auditlog list \
  --limit 200 \
  --fields id,username,category,event,entryTimeUTC,objectName,message \
  --output csv > audit-log.csv

# Use a specific profile
iics auditlog list --profile prod
```

```powershell
# List the most recent 200 audit log entries (default)
iics auditlog list

# First page of 50 entries (most recent)
iics auditlog list --limit 50

# Second page of 50 entries
iics auditlog list --limit 50 --skip 1

# Custom columns
iics auditlog list --fields id,username,category,event,entryTimeUTC

# Full output as JSON
iics auditlog list --output json

# Export to CSV
iics auditlog list `
  --limit 200 `
  --fields id,username,category,event,entryTimeUTC,objectName,message `
  --output csv | Out-File audit-log.csv

# Filter JSON output for a specific user
$logs = iics auditlog list --output json | ConvertFrom-Json
$logs | Where-Object { $_.username -eq "jane.smith@company.com" }

# Filter for failed login events
$logs = iics auditlog list --output json | ConvertFrom-Json
$logs | Where-Object { $_.category -eq "AUTH" -and $_.event -eq "LOGIN_FAILED" }
```

## See also

- [securitylog](securitylog.md) - query security audit log (V3 API)
- [activitylog](activitylog.md) - query activity logs for completed jobs
