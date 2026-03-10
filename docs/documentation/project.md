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

---

## project update

### Flags

| Flag            | Type   | Required | Description        |
| --------------- | ------ | -------- | ------------------ |
| `--id`          | string | yes      | Project ID         |
| `--name`        | string |          | New project name   |
| `--description` | string |          | New description    |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics project update --id <project-id> --name "Finance Data Engineering"

iics project update --id <project-id> --description "Updated description"
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

## See also

- [folder](folder.md) - manage folders within projects
- [objects](objects.md) - list assets within a project
- [lookup](lookup.md) - resolve a project name to its ID
