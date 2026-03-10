# schedule

Manage IICS schedules for triggering tasks and taskflows.

## Synopsis

```bash
iics schedule <subcommand> [flags]
```

## Subcommands

| Subcommand | Description             |
| ---------- | ----------------------- |
| `list`     | List schedules          |
| `get`      | Get a single schedule   |
| `create`   | Create a schedule       |
| `update`   | Update a schedule       |
| `delete`   | Delete a schedule       |

---

## schedule list

### Flags

| Flag      | Type | Default | Description     |
| --------- | ---- | ------- | --------------- |
| `--limit` | int  | 200     | Max results     |
| `--skip`  | int  | 0       | Results to skip |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column      | Description                       |
| ----------- | --------------------------------- |
| `id`        | Schedule ID                       |
| `name`      | Schedule name                     |
| `status`    | Active / Inactive                 |
| `interval`  | Run interval or cron expression   |
| `updateTime`| Last modification time            |

### Examples

```bash
iics schedule list

iics schedule list --output json

# Find all active schedules
iics schedule list --output json | jq '.[] | select(.status == "Active")'
```

---

## schedule get

### Flags

| Flag   | Type   | Required | Description  |
| ------ | ------ | -------- | ------------ |
| `--id` | string | yes      | Schedule ID  |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column      | Description                    |
| ----------- | ------------------------------ |
| `id`        | Schedule ID                    |
| `name`      | Schedule name                  |
| `status`    | Active / Inactive              |
| `interval`  | Run interval / cron expression |
| `timezone`  | Timezone for the schedule      |

### Examples

```bash
iics schedule get --id <schedule-id>
```

---

## schedule create

Create a schedule from a JSON definition file.

### Flags

| Flag          | Type   | Required | Description                        |
| ------------- | ------ | -------- | ---------------------------------- |
| `--from-file` | string | yes      | JSON file with schedule definition |

All [global flags](../../README.md#global-flags) apply.

### JSON definition example

```json
{
  "name": "Nightly ETL",
  "description": "Run ETL pipeline every night at 2am",
  "status": "Active",
  "interval": "DAILY",
  "startTime": "02:00:00",
  "timezone": "America/New_York",
  "frequency": 1
}
```

### Examples

```bash
iics schedule create --from-file nightly-etl.json
```

---

## schedule update

### Flags

| Flag          | Type   | Required | Description                     |
| ------------- | ------ | -------- | ------------------------------- |
| `--id`        | string | yes      | Schedule ID                     |
| `--from-file` | string | yes      | JSON file with updated fields   |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics schedule update --id <schedule-id> --from-file updated-schedule.json
```

---

## schedule delete

Delete a schedule. Prompts for confirmation unless `--yes` is given.

### Flags

| Flag    | Short | Type   | Required | Description              |
| ------- | ----- | ------ | -------- | ------------------------ |
| `--id`  |       | string | yes      | Schedule ID              |
| `--yes` | `-y`  | bool   |          | Skip confirmation prompt |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics schedule delete --id <schedule-id>

iics schedule delete --id <schedule-id> --yes
```

## See also

- [objects](objects.md) - list taskflows and tasks associated with schedules
