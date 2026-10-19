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

| Flag             | Type   | Required | Description                                                  |
| ---------------- | ------ | -------- | ------------------------------------------------------------ |
| `--name`         | string | yes      | Folder name                                                  |
| `--description`  | string |          | Folder description                                           |
| `--project-id`   | string |          | Parent project ID (mutually exclusive with `--project-name`) |
| `--project-name` | string |          | Parent project name (mutually exclusive with `--project-id`) |

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

Change the name and/or description of a folder (`PATCH`). Identify the folder
in one of these ways; IDs and names cannot be mixed:

| Identify by | Flags                                | URI                                              |
| ----------- | ------------------------------------ | ------------------------------------------------ |
| ID          | `--id`                               | `/folders/<folder ID>`                           |
| Names       | `--project-name` and `--folder-name` | `/projects/name/<project>/folders/name/<folder>` |

Folder IDs are unique in the organization, so `--id` works for folders in any
project. `--project-id` is deprecated and ignored: the documented
`/projects/<project ID>/folders/<folder ID>` URI returns HTTP 404.

Names are percent-encoded, so names with spaces work as typed. A description
cannot be cleared.

With `--interactive` (or with no identifier when run in a terminal), pick a
project, then one of its folders. Lists with more than 10 entries first ask for
a name filter (substring, ignores case; empty lists all). Then edit the name and
description, pre-filled with the current values, and confirm. The
[menu](menu.md) entry "Edit folder" runs this flow.

### Flags

| Flag             | Short | Type   | Required | Description                                     |
| ---------------- | ----- | ------ | -------- | ----------------------------------------------- |
| `--id`           |       | string |          | Folder ID                                       |
| `--project-id`   |       | string |          | Deprecated and ignored; `--id` is enough        |
| `--project-name` |       | string |          | Parent project name (use with `--folder-name`)  |
| `--folder-name`  |       | string |          | Current folder name (use with `--project-name`) |
| `--name`         |       | string |          | New folder name                                 |
| `--description`  |       | string |          | New description                                 |
| `--interactive`  | `-i`  | bool   |          | Pick the folder and edit it interactively       |

Without `--interactive`, at least one of `--name` and `--description` is required.

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Update by folder ID
iics folder update --id <folder-id> --name "Production"

# Update by project and folder name
iics folder update --project-name "Finance ETL" --folder-name "Staging" \
  --name "Prod" --description "Production-ready pipelines"

# Pick the project and folder, then edit
iics folder update -i
```

```powershell
# Update by folder ID
iics folder update --id <folder-id> --name "Production"

# Update by project and folder name
iics folder update --project-name "Finance ETL" --folder-name "Staging" `
  --name "Prod" --description "Production-ready pipelines"

# Pick the project and folder, then edit
iics folder update -i
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
