# state

Fetch or load the state (configuration snapshot) of an IICS object. Useful for backups, migrations, and comparison.

## Synopsis

```bash
iics state <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                              |
| ---------- | ---------------------------------------- |
| `fetch`    | Fetch the current state of an object     |
| `load`     | Load a previously saved state into an object |

---

## state fetch

Retrieve the current state of an object as JSON. Output is written to a file or to stdout.

### Flags

| Flag              | Short | Type   | Required | Description                             |
| ----------------- | ----- | ------ | -------- | --------------------------------------- |
| `--object-id`     |       | string | yes      | Object ID                               |
| `--output-file`   | `-f`  | string |          | File path to save state JSON (default: stdout) |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Print state to stdout
iics state fetch --object-id <object-id>

# Save state to a file
iics state fetch --object-id <object-id> --output-file my-mapping.json

# Resolve path to ID, then fetch
ID=$(iics lookup --path "My Project/ETL/LoadOrders" --type MTT --output json | jq -r '.id')
iics state fetch --object-id "$ID" --output-file LoadOrders-backup.json

# Backup all mappings in a project
for ID in $(iics objects list --type MTT --output json | jq -r '.[].id'); do
  iics state fetch --object-id "$ID" --output-file "backups/${ID}.json"
done
```

```powershell
# Print state to stdout
iics state fetch --object-id <object-id>

# Save state to a file
iics state fetch --object-id <object-id> --output-file my-mapping.json

# Resolve path to ID, then fetch
$obj = iics lookup --path "My Project/ETL/LoadOrders" --type MTT --output json | ConvertFrom-Json
iics state fetch --object-id $obj.id --output-file LoadOrders-backup.json

# Backup all mappings in a project
$objs = iics objects list --type MTT --output json | ConvertFrom-Json
foreach ($obj in $objs) {
  iics state fetch --object-id $obj.id --output-file "backups/$($obj.id).json"
}
```

---

## state load

Load a previously fetched state JSON back into an object.

### Flags

| Flag            | Short | Type   | Required | Description                         |
| --------------- | ----- | ------ | -------- | ----------------------------------- |
| `--object-id`   |       | string | yes      | Object ID to load state into        |
| `--input-file`  | `-f`  | string | yes      | JSON file containing the state      |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics state load --object-id <object-id> --input-file my-mapping.json

# Restore from backup
iics state load --object-id "$ID" --input-file backups/${ID}.json
```

```powershell
iics state load --object-id <object-id> --input-file my-mapping.json

# Restore from backup
iics state load --object-id $ID --input-file "backups/$ID.json"
```

## Backup and restore pattern

```bash
# Backup
ID=$(iics lookup --path "Finance/ETL/LoadLedger" --type MTT --output json | jq -r '.id')
iics state fetch --object-id "$ID" --output-file LoadLedger-$(date +%Y%m%d).json

# Restore
iics state load --object-id "$ID" --input-file LoadLedger-20260310.json
```

```powershell
# Backup
$obj = iics lookup --path "Finance/ETL/LoadLedger" --type MTT --output json | ConvertFrom-Json
$date = Get-Date -Format "yyyyMMdd"
iics state fetch --object-id $obj.id --output-file "LoadLedger-$date.json"

# Restore
iics state load --object-id $obj.id --input-file "LoadLedger-20260310.json"
```

## See also

- [sourcecontrol](sourcecontrol.md) - full source control integration (checkout, checkin, commit)
- [export](export.md) - export objects as a deployable ZIP package
- [lookup](lookup.md) - resolve object paths to IDs
