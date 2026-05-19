# metering

Query and download IICS platform usage and consumption data.

## Synopsis

```bash
iics metering <subcommand> [flags]
```

## Subcommands

| Subcommand  | Description                          |
| ----------- | ------------------------------------ |
| `get`       | Retrieve metering data               |
| `download`  | Download a metering report to a file |

---

## metering get

Retrieve metering data for a given type and time range.

### Flags

| Flag      | Type   | Required | Description                                      |
| --------- | ------ | -------- | ------------------------------------------------ |
| `--type`  | string |          | Metering type (e.g. `IPU`, `CDI`, `CAI`)         |
| `--start` | string |          | Start date, e.g. `2026-01-01`                    |
| `--end`   | string |          | End date, e.g. `2026-03-31`                      |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column      | Description                    |
| ----------- | ------------------------------ |
| `id`        | Report/record ID               |
| `type`      | Metering type                  |
| `startDate` | Start of the metering period   |
| `endDate`   | End of the metering period     |

### Examples

```bash
# Get all metering data
iics metering get

# Filter by type and date range
iics metering get --type IPU --start 2026-01-01 --end 2026-03-31

# Output as JSON
iics metering get --output json
```

```powershell
# Get all metering data
iics metering get

# Filter by type and date range
iics metering get --type IPU --start 2026-01-01 --end 2026-03-31

# Output as JSON
iics metering get --output json
```

---

## metering download

Download a metering report to a CSV file.

### Flags

| Flag            | Short | Type   | Required | Default             | Description      |
|-----------------|-------|--------|----------|---------------------|------------------|
| `--id`          |       | string | yes      |                     | Report ID        |
| `--output-file` | `-f`  | string |          | `metering_<id>.csv` | Output file path |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Download with default filename
iics metering download --id <report-id>

# Download to a specific path
iics metering download --id <report-id> --output-file reports/ipu-q1-2026.csv

# Get report ID then download
REPORT_ID=$(iics metering get --type IPU --output json | jq -r '.[0].id')
iics metering download --id "$REPORT_ID" --output-file ipu-report.csv
```

```powershell
# Download with default filename
iics metering download --id <report-id>

# Download to a specific path
iics metering download --id <report-id> --output-file reports/ipu-q1-2026.csv

# Get report ID then download
$report = iics metering get --type IPU --output json | ConvertFrom-Json | Select-Object -First 1
iics metering download --id $report.id --output-file ipu-report.csv
```

## See also

- [securitylog](securitylog.md) - query the security audit log
