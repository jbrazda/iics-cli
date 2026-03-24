# folder

Manage IICS folders. Folders organise assets within a project.

## Synopsis

```bash
iics folder <subcommand> [flags]
```

## Subcommands

| Subcommand | Description           |
| ---------- | --------------------- |
| `create`   | Create a new folder   |
| `update`   | Update a folder       |
| `delete`   | Delete a folder       |

---

## folder create

### Flags

| Flag              | Type   | Required | Description                                              |
| ----------------- | ------ | -------- | -------------------------------------------------------- |
| `--name`          | string | yes      | Folder name                                              |
| `--description`   | string |          | Folder description                                       |
| `--project-id`    | string |          | Parent project ID (mutually exclusive with `--project-name`) |
| `--project-name`  | string |          | Parent project name (mutually exclusive with `--project-id`) |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Create folder using project ID
iics folder create --name "Staging" --project-id <project-id>

# Create folder using project name
iics folder create --name "Staging" --project-name "Finance ETL"

# With description
iics folder create --name "Staging" --project-name "Finance ETL" \
  --description "Staging area for raw data loads"
```

```powershell
# Create folder using project ID
iics folder create --name "Staging" --project-id <project-id>

# Create folder using project name
iics folder create --name "Staging" --project-name "Finance ETL"

# With description
iics folder create --name "Staging" --project-name "Finance ETL" `
  --description "Staging area for raw data loads"
```

---

## folder update

### Flags

| Flag              | Type   | Required | Description                                                |
| ----------------- | ------ | -------- | ---------------------------------------------------------- |
| `--id`            | string |          | Folder ID                                                  |
| `--name`          | string |          | New folder name                                            |
| `--description`   | string |          | New description                                            |
| `--project-id`    | string |          | Parent project ID                                          |
| `--project-name`  | string |          | Parent project name                                        |
| `--folder-name`   | string |          | Current folder name (use with `--project-name` to identify) |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Update by folder ID
iics folder update --id <folder-id> --name "Production"

# Update by project and folder name
iics folder update --project-name "Finance ETL" --folder-name "Staging" \
  --name "Prod" --description "Production-ready pipelines"
```

```powershell
# Update by folder ID
iics folder update --id <folder-id> --name "Production"

# Update by project and folder name
iics folder update --project-name "Finance ETL" --folder-name "Staging" `
  --name "Prod" --description "Production-ready pipelines"
```

---

## folder delete

Delete a folder. Prompts for confirmation unless `--yes` is given.

### Flags

| Flag    | Short | Type   | Required | Description              |
| ------- | ----- | ------ | -------- | ------------------------ |
| `--id`  |       | string | yes      | Folder ID                |
| `--yes` | `-y`  | bool   |          | Skip confirmation prompt |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics folder delete --id <folder-id>

# Non-interactive
iics folder delete --id <folder-id> --yes
```

```powershell
iics folder delete --id <folder-id>

# Non-interactive
iics folder delete --id <folder-id> --yes
```

## See also

- [project](project.md) - manage parent projects
- [objects](objects.md) - list assets within a folder
- [lookup](lookup.md) - resolve a folder path to its ID
