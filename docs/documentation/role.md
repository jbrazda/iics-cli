# role

Manage IICS roles. Roles group privileges and are assigned to users and user groups.

## Synopsis

```bash
iics role <subcommand> [flags]
```

## Subcommands

| Subcommand          | Description                                                   |
| ------------------- | ------------------------------------------------------------- |
| `list`              | List roles                                                    |
| `get`               | Get a single role by ID or name, or pick one interactively    |
| `create`            | Create a custom role (interactive wizard on a terminal)       |
| `edit`              | Add and remove privileges in the interactive editor           |
| `add-privileges`    | Add privileges to a custom role                               |
| `remove-privileges` | Remove privileges from a custom role                          |
| `delete`            | Delete a custom role                                          |

The IICS v3 Roles API has no general update endpoint. The only change you can
make to an existing custom role is adding or removing privileges.

---

## role list

### Flags

| Flag      | Type | Default | Description     |
| --------- | ---- | ------- | --------------- |
| `--limit` | int  | 200     | Max results     |
| `--skip`  | int  | 0       | Results to skip |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column       | Description                       |
| ------------ | --------------------------------- |
| `id`         | Role ID                           |
| `name`       | Role name                         |
| `systemRole` | Whether this is a built-in role   |
| `description`| Role description                  |

### Examples

```bash
iics role list

# List as JSON
iics role list --output json

# List only custom (non-system) roles
iics role list --output json | jq '.[] | select(.systemRole == false)'
```

```powershell
iics role list

# List as JSON
iics role list --output json

# List only custom (non-system) roles
$roles = iics role list --output json | ConvertFrom-Json
$roles | Where-Object { $_.systemRole -eq $false }
```

---

## role get

Get the details of a single role, looked up by ID or by name. The IICS v3 API
has no `/roles/{id}` endpoint, so the command queries
`GET /public/core/v3/roles?q=roleId=="<id>"` or `q=roleName=="<name>"`.
With `--privileges`, `expand=privileges` is added to the request and the
privileges assigned to the role are included in the output.

When neither `--id` nor `--name` is given and stdin is a terminal, the command
lists all roles alphabetically and prompts you to pick one. Without a terminal
(for example in CI), `--id` or `--name` is required.

In `table` output on a terminal, when `--privileges` is not given, the command
prints the role details and then asks `Display role privileges? [y/Q]`.
Answering `y` fetches and prints the privileges; any other answer (`q`,
Enter) quits. No prompt is shown when `--privileges` is set explicitly
(including `--privileges=false`), for other output formats, or when stdin or
stdout is not a terminal.

### Flags

| Flag           | Type   | Required | Description                               |
| -------------- | ------ | -------- | ----------------------------------------- |
| `--id`         | string |          | Role ID                                    |
| `--name`       | string |          | Role name (for example `Business Manager`) |
| `--privileges` | bool   |          | Include privileges assigned to the role; prompts when omitted (table output on a terminal) |

`--id` and `--name` are mutually exclusive. If both are omitted, you are
prompted to select a role.

All [global flags](../../README.md#global-flags) apply.

### Output

In `table` format the role is shown as a vertical detail table (`id`, `orgId`,
`displayName`, `description`, `systemRole`, `status`, `createdBy`,
`createTime`, `updatedBy`, `updateTime`). With `--privileges`, a second table
lists the privileges, sorted by service and then privilege name:

| Column        | Description                                      |
| ------------- | ------------------------------------------------ |
| `NAME`        | Privilege name                                   |
| `SERVICE`     | IDMC service that uses the privilege             |
| `STATUS`      | Privilege status (color-coded, see below)        |
| `ID`          | Privilege ID                                     |
| `DESCRIPTION` | Privilege description                            |

On colored themes the `STATUS` column is color-coded:

| Status       | Color  |
| ------------ | ------ |
| `Enabled`    | Green  |
| `Unassigned` | Orange |
| `Disabled`   | Red    |
| `Default`    | Blue   |

Colors are omitted with `--no-color`, `NO_COLOR`, the `plain`, `markdown` and
`gh` themes, and when output is not a terminal.

`json` and `yaml` formats return the full role object, including the
`privileges` array (in the same sort order) when `--privileges` is set.

`csv` output returns one row per privilege (`NAME`, `SERVICE`, `STATUS`, `ID`,
`DESCRIPTION`) when `--privileges` is set, otherwise a single role row (`ID`,
`NAME`, `DISPLAY NAME`, `SYSTEM`, `STATUS`, `DESCRIPTION`, `UPDATED BY`,
`UPDATED`).

### Examples

```bash
# Pick a role from a list
iics role get --privileges

iics role get --id <role-id>

iics role get --name "Business Manager"

# Include privileges assigned to the role
iics role get --name "Business Manager" --privileges

# Privilege names only
iics role get --name "Business Manager" --privileges --output json | jq -r '.privileges[].name'
```

```powershell
# Pick a role from a list
iics role get --privileges

iics role get --id <role-id>

iics role get --name "Business Manager"

# Include privileges assigned to the role
iics role get --name "Business Manager" --privileges

# Privilege names only
(iics role get --name "Business Manager" --privileges --output json | ConvertFrom-Json).privileges.name
```

---

## Interactive privilege editor

`role edit`, `role create` (wizard), and `role add-privileges` /
`role remove-privileges` without `--privilege` open an interactive editor on a
terminal. Privileges are grouped in two levels:

1. **Service** - a filterable list with the number of privileges, how many are
   selected, and pending changes (`+added -removed`) per service.
2. **Object and action grid** - privilege names of the form
   `<action>.<object>` (for example `create.data.transfer.task`) become one row
   per object with a column per action. Names that do not follow this pattern
   (for example `feature.mcp.SuperAdmin`, `PROFILE.viewResults`) are listed
   individually under **Other**. Privileges without a service are grouped as
   `(no service)`.

```text
Role: CAI Viewer > ApplicationIntegration (14)

  OBJECT                 VIEW    CREATE  UPDATE  DELETE  EXEC    PERM
> ai.assets              [-]     [ ]     [ ]     [ ]     [ ]     [ ]
  ai.console             [x]      .       .       .       .       .
  ai.designer            [x]      .       .       .       .       .
  -- Other --
  ai.admin               [+]

ai.admin - Administration (Enabled)
arrows move  space toggle  a row  c column  / filter  ctrl+u clear filter  enter/esc back
```

| Mark  | Meaning                                  |
| ----- | ---------------------------------------- |
| `[x]` | Assigned, unchanged                      |
| `[+]` | Will be added                            |
| `[-]` | Will be removed                          |
| `[ ]` | Not assigned                             |
| `.`   | The action does not exist for the object |

Grid keys:

| Key                   | Action                                              |
| --------------------- | --------------------------------------------------- |
| Arrows or `h j k l`   | Move                                                |
| `Space` or `x`        | Toggle the privilege under the cursor               |
| `a`                   | Toggle every action in the row                      |
| `c`                   | Toggle the column for all visible (filtered) rows   |
| `/`                   | Filter by object, privilege name or description     |
| `Ctrl+U`              | Clear the filter                                    |
| `PgUp` / `PgDn`, `g` / `G` | Page, jump to top / bottom                     |
| `Enter`, `Esc`, `q`   | Back to the service list                            |

In the service list, choose **Review and apply** to see the pending changes
grouped by service and confirm them, or **Cancel** to leave without changes.
Additions are applied before removals. A change that would leave the role with
no privileges is blocked at review.

Modes:

- `role edit` - add and remove.
- `role add-privileges` - add only; assigned privileges are locked.
- `role remove-privileges` - remove only; only assigned privileges are shown.

System roles cannot be edited. Without a terminal, the commands require
`--privilege` (or `--name` and privileges for `create`).

---

## role edit

Open the interactive privilege editor for a custom role. Without `--id` or
`--name`, a filterable role list is shown first. Requires a terminal.

### Flags

| Flag     | Type   | Required | Description |
| -------- | ------ | -------- | ----------- |
| `--id`   | string |          | Role ID     |
| `--name` | string |          | Role name   |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Pick a role, then edit its privileges
iics role edit

iics role edit --name "CAI Viewer"
```

```powershell
iics role edit

iics role edit --name "CAI Viewer"
```

---

## role create

Create a custom role. The API requires a name and at least one privilege.

On a terminal, when `--name` is missing or no privileges are given (no
`--privilege`, `--from-role` or `--from-file`), a wizard asks for the role
name (duplicate names are rejected), description, and an optional existing
role to copy privileges from, then opens the
[interactive privilege editor](#interactive-privilege-editor). `--from-role`
and `--privilege` values pre-select privileges in the wizard.

`--from-role` copies the privileges of an existing role, looked up by name
first and then by ID. The source description is used unless `--description`
is given. `--privilege` values are added on top.

Privileges can be given as names (for example `view.ai.assets`) or IDs. The
command looks them up with `iics privilege list` data and sends privilege IDs,
as the create endpoint requires. Unknown privileges are reported before the
role is created.

### Flags

| Flag            | Type     | Required | Description                                                  |
| --------------- | -------- | -------- | ------------------------------------------------------------ |
| `--name`        | string   | yes*     | Role name                                                    |
| `--description` | string   |          | Role description                                             |
| `--privilege`   | string[] | yes*     | Privilege name or ID; repeatable or comma-separated          |
| `--from-role`   | string   |          | Copy privileges and description from a role (name or ID)     |
| `--from-file`   | string   |          | JSON file with `name`, `description` and `privileges`        |

\* Can be supplied through `--from-file` instead, and privileges through
`--from-role`. On a terminal, missing values are collected by the wizard.
Flags override `name` and `description` from the file; `--privilege` values
are added to the file's `privileges`.

All [global flags](../../README.md#global-flags) apply.

### JSON definition example

The file uses the same shape as the API request body. `privileges` may hold
privilege names or IDs.

```json
{
  "name": "CAIviewer",
  "description": "A role to view Application Integration designer and assets",
  "privileges": ["view.ai.designer", "view.ai.assets"]
}
```

### Examples

```bash
# Interactive wizard
iics role create

iics role create --name "CAI Viewer" --description "View CAI assets" \
  --privilege view.ai.designer --privilege view.ai.assets

# Clone an existing role and add one privilege
iics role create --name "Designer Plus" --from-role Designer \
  --privilege view.ai.console

iics role create --from-file cai-viewer-role.json
```

```powershell
# Interactive wizard
iics role create

iics role create --name "CAI Viewer" --description "View CAI assets" `
  --privilege view.ai.designer --privilege view.ai.assets

# Clone an existing role and add one privilege
iics role create --name "Designer Plus" --from-role Designer `
  --privilege view.ai.console

iics role create --from-file cai-viewer-role.json
```

---

## role add-privileges

Add privileges to a custom role. Sends
`PUT /public/core/v3/roles/<id>/addPrivileges`, or
`PUT /public/core/v3/roles/name/<name>/addPrivileges` when `--name` is used.

Without `--privilege` on a terminal, the
[interactive privilege editor](#interactive-privilege-editor) opens in
add-only mode (with a role picker when `--id` and `--name` are omitted).

### Flags

| Flag          | Type     | Required | Description                                         |
| ------------- | -------- | -------- | --------------------------------------------------- |
| `--id`        | string   | one of*  | Role ID                                             |
| `--name`      | string   | one of*  | Role name                                           |
| `--privilege` | string[] | yes*     | Privilege name or ID; repeatable or comma-separated |

\* Optional on a terminal, where the interactive editor is used instead.
`--id` and `--name` are mutually exclusive. Privilege IDs are converted to
names, which is what the endpoint expects.

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Interactive editor (add-only)
iics role add-privileges --name "CAI Viewer"

iics role add-privileges --name "CAI Viewer" --privilege view.ai.console

iics role add-privileges --id <role-id> \
  --privilege create.data.transfer.task,delete.data.transfer.task
```

```powershell
iics role add-privileges --name "CAI Viewer" --privilege view.ai.console

iics role add-privileges --id <role-id> `
  --privilege create.data.transfer.task,delete.data.transfer.task
```

---

## role remove-privileges

Remove privileges from a custom role. Sends
`PUT /public/core/v3/roles/<id>/removePrivileges`, or
`PUT /public/core/v3/roles/name/<name>/removePrivileges` when `--name` is
used. A role must keep at least one privilege.

Without `--privilege` on a terminal, the
[interactive privilege editor](#interactive-privilege-editor) opens in
remove-only mode.

### Flags

| Flag          | Type     | Required | Description                                         |
| ------------- | -------- | -------- | --------------------------------------------------- |
| `--id`        | string   | one of*  | Role ID                                             |
| `--name`      | string   | one of*  | Role name                                           |
| `--privilege` | string[] | yes*     | Privilege name or ID; repeatable or comma-separated |

\* Optional on a terminal, where the interactive editor is used instead.

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Interactive editor (remove-only)
iics role remove-privileges --name "CAI Viewer"

iics role remove-privileges --name "CAI Viewer" --privilege view.ai.console
```

```powershell
iics role remove-privileges --name "CAI Viewer" --privilege view.ai.console
```

---

## role delete

Delete a custom role. Sends `DELETE /public/core/v3/roles/<id>`. With
`--name`, the role ID is looked up first. Prompts for confirmation unless
`--yes` is given.

### Flags

| Flag     | Short | Type   | Required | Description              |
| -------- | ----- | ------ | -------- | ------------------------ |
| `--id`   |       | string | one of   | Role ID                  |
| `--name` |       | string | one of   | Role name                |
| `--yes`  | `-y`  | bool   |          | Skip confirmation prompt |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics role delete --id <role-id>

iics role delete --name "CAI Viewer" --yes
```

```powershell
iics role delete --id <role-id>

iics role delete --name "CAI Viewer" --yes
```

## See also

- [Interactive prompts](interactive.md) - keys, accessible mode and piped input for wizards and pickers
- [privilege](privilege.md) - list available privileges to assign to roles
- [user](user.md) - assign roles to users
- [group](group.md) - assign roles to user groups
