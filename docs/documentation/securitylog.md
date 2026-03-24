# securitylog

Query the IICS security audit log. Alias: `auditlog`.

The security log records authentication events, object changes, and administrative actions.

## Synopsis

```bash
iics securitylog <subcommand> [flags]
iics auditlog <subcommand> [flags]
```

## Subcommands

| Subcommand | Description               |
| ---------- | ------------------------- |
| `list`     | List security log entries |

---

## securitylog list

### Flags

| Flag      | Type   | Default | Description                                       |
| --------- | ------ | ------- | ------------------------------------------------- |
| `--start` | string |         | Start time in ISO 8601 format, e.g. `2026-01-01T00:00:00Z` |
| `--end`   | string |         | End time in ISO 8601 format                       |
| `--limit` | int    | 200     | Max results                                       |
| `--skip`  | int    | 0       | Results to skip                                   |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column       | Description                              |
| ------------ | ---------------------------------------- |
| `entryTime`  | Timestamp of the audit event             |
| `userName`   | User who performed the action            |
| `action`     | Action taken (LOGIN, CREATE, UPDATE, DELETE, etc.) |
| `objectType` | Type of object affected                  |
| `status`     | Outcome: SUCCESS or FAILED               |
| `sourceIp`   | IP address of the client                 |

### Examples

```bash
# List recent security log entries (default: up to 200)
iics securitylog list

# Filter by time range
iics securitylog list \
  --start "2026-03-01T00:00:00Z" \
  --end   "2026-03-10T23:59:59Z"

# As JSON for analysis
iics securitylog list --output json

# Find failed login attempts
iics securitylog list --output json \
  | jq '.[] | select(.action == "LOGIN" and .status == "FAILED")'

# Audit a specific user's actions
iics securitylog list \
  --start "2026-03-01T00:00:00Z" \
  --output json \
  | jq '.[] | select(.userName == "jane.smith@company.com")'

# Export to CSV for compliance reporting
iics auditlog list \
  --start "2026-01-01T00:00:00Z" \
  --end   "2026-03-31T23:59:59Z" \
  --output csv > security-audit-q1-2026.csv
```

```powershell
# List recent security log entries (default: up to 200)
iics securitylog list

# Filter by time range
iics securitylog list `
  --start "2026-03-01T00:00:00Z" `
  --end   "2026-03-10T23:59:59Z"

# As JSON for analysis
iics securitylog list --output json

# Find failed login attempts
$logs = iics securitylog list --output json | ConvertFrom-Json
$logs | Where-Object { $_.action -eq "LOGIN" -and $_.status -eq "FAILED" }

# Audit a specific user's actions
$logs = iics securitylog list `
  --start "2026-03-01T00:00:00Z" `
  --output json | ConvertFrom-Json
$logs | Where-Object { $_.userName -eq "jane.smith@company.com" }

# Export to CSV for compliance reporting
iics auditlog list `
  --start "2026-01-01T00:00:00Z" `
  --end   "2026-03-31T23:59:59Z" `
  --output csv | Out-File security-audit-q1-2026.csv
```

## See also

- [metering](metering.md) - query usage and consumption data
