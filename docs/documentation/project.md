# project

Manage IICS projects. Projects are top-level containers for organising assets and folders.

## Synopsis

```bash
iics project <subcommand> [flags]
```

## Subcommands

| Subcommand | Description            |
| ---------- | ---------------------- |
| `create`   | Create a new project   |
| `update`   | Update a project       |
| `delete`   | Delete a project       |

---

## project create

### Flags

| Flag            | Type   | Required | Description          |
| --------------- | ------ | -------- | -------------------- |
| `--name`        | string | yes      | Project name         |
| `--description` | string |          | Project description  |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics project create --name "Data Engineering"

iics project create --name "Finance ETL" --description "Finance team data pipelines"
```

```powershell
iics project create --name "Data Engineering"

iics project create --name "Finance ETL" --description "Finance team data pipelines"
```

---

## project update

Change the name and/or description of a project (`PATCH /public/core/v3/projects/<id>`
or `/projects/name/<name>`). Identify the project by `--id` or by its current
name with `--project-name`. A description cannot be cleared.

With `--interactive` (or with no identifier when run in a terminal), pick the
project from a list. When there are more than 10 projects you are first asked
for a name filter (substring, ignores case; empty lists all). Then edit the
name and description, pre-filled with the current values, and confirm.
The [menu](menu.md) entry "Edit project" runs this flow.

### Flags

| Flag             | Short | Type   | Required | Description                                |
| ---------------- | ----- | ------ | -------- | ------------------------------------------ |
| `--id`           |       | string | one of   | Project ID                                 |
| `--project-name` |       | string | one of   | Current project name                       |
| `--name`         |       | string |          | New project name                           |
| `--description`  |       | string |          | New description                            |
| `--interactive`  | `-i`  | bool   |          | Pick the project and edit it interactively |

Without `--interactive`, at least one of `--name` and `--description` is required.

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics project update --id <project-id> --name "Finance Data Engineering"

iics project update --project-name "Finance ETL" --description "Finance team data pipelines"

# Pick the project and edit it
iics project update -i
```

```powershell
iics project update --id <project-id> --name "Finance Data Engineering"

iics project update --project-name "Finance ETL" --description "Finance team data pipelines"

# Pick the project and edit it
iics project update -i
```

---

## project delete

Delete a project. Prompts for confirmation unless `--yes` is given.

### Flags

| Flag    | Short | Type   | Required | Description              |
| ------- | ----- | ------ | -------- | ------------------------ |
| `--id`  |       | string | yes      | Project ID               |
| `--yes` | `-y`  | bool   |          | Skip confirmation prompt |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Interactive confirmation
iics project delete --id <project-id>

# Non-interactive (CI/CD)
iics project delete --id <project-id> --yes
```

```powershell
# Interactive confirmation
iics project delete --id <project-id>

# Non-interactive (CI/CD)
iics project delete --id <project-id> --yes
```

## See also

- [folder](folder.md) - manage folders within projects
- [objects](objects.md) - list assets within a project
- [lookup](lookup.md) - resolve a project name to its ID
