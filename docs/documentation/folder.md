# folder

Manage IICS folders. Folders organise assets within a project.

## Synopsis

```bash
iics folder <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                                   |
| ---------- | --------------------------------------------- |
| `list`     | List folders with created and updated details |
| `create`   | Create a new folder                           |
| `update`   | Update a folder                               |
| `delete`   | Delete a folder                               |

---

## folder list

List folders with their full path (`project/folder`), description, created and
updated time and user, sorted by path.

- Without a project, all folders in the org are listed.
- `--project-name` or `--project-id` lists the folders of one project.
- `--pick-project` picks the project from a list, the same picker as
  `folder update -i` (more than 10 projects first ask for a name filter).

`--name` filters by a regular expression matched against the full path
(case-insensitive), so it can match the project name, the folder name or both;
`--interactive` asks for it.

The [menu](menu.md) entries "List folders" (`folder list -i`) and "List folders
in a project" (`folder list --pick-project`) run these flows.

Created time and created by come from the internal FRS API that the IICS web
UI uses (`/frs/api/v1`). It is not part of the published REST API, so the CLI
falls back to the documented objects API when it fails: a note is printed to
stderr and the `createTime` and `createdBy` columns are left out. User IDs are
resolved to user names; users that are not in the org user list (such as
Informatica system users) are shown by ID.

### Flags

| Flag             | Short | Type   | Description                                                     |
| ---------------- | ----- | ------ | --------------------------------------------------------------- |
| `--name`         |       | string | Regular expression matched against the full path (ignores case) |
| `--project-name` |       | string | List only the folders of this project                           |
| `--project-id`   |       | string | List only the folders of this project ID                        |
| `--pick-project` |       | bool   | Pick the project from a list                                    |
| `--interactive`  | `-i`  | bool   | Ask for the name pattern                                        |

`--project-name`, `--project-id` and `--pick-project` are mutually exclusive.

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column        | Description                                |
| ------------- | ------------------------------------------ |
| `path`        | Full path: `project/folder`                |
| `description` | Description                                |
| `createTime`  | Created time (FRS only)                    |
| `createdBy`   | User who created the folder (FRS only)     |
| `updateTime`  | Last update time                           |
| `updatedBy`   | User who last updated the folder           |
| `id`          | Folder ID                                  |

`--output json` also returns `name` and `type`.

### Examples

```bash
# All folders
iics folder list

# Folders in the Salesforce project whose name starts with "Acc"
iics folder list --name '^salesforce/acc'

# Any folder ending in _v1
iics folder list --name '_v1$'

# Folders of one project
iics folder list --project-name "Finance ETL"

# Pick the project, then show its folders
iics folder list --pick-project
```

```powershell
# All folders
iics folder list

# Folders in the Salesforce project whose name starts with "Acc"
iics folder list --name '^salesforce/acc'

# Any folder ending in _v1
iics folder list --name '_v1$'

# Folders of one project
iics folder list --project-name "Finance ETL"

# Pick the project, then show its folders
iics folder list --pick-project
```

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
