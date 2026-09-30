# role

Manage IICS roles. Roles group privileges and are assigned to users and user groups.

## Synopsis

```bash
iics role <subcommand> [flags]
```

## Subcommands

| Subcommand          | Description                          |
| ------------------- | ------------------------------------ |
| `list`              | List roles                           |
| `get`               | Get a single role by ID or name, or pick one interactively |
| `create`            | Create a custom role                 |
| `add-privileges`    | Add privileges to a custom role      |
| `remove-privileges` | Remove privileges from a custom role |
| `delete`            | Delete a custom role                 |

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

## role create

Create a custom role. The API requires a name and at least one privilege.

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
| `--from-file`   | string   |          | JSON file with `name`, `description` and `privileges`        |

\* Can be supplied through `--from-file` instead. Flags override `name` and
`description` from the file; `--privilege` values are added to the file's
`privileges`.

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
iics role create --name "CAI Viewer" --description "View CAI assets" \
  --privilege view.ai.designer --privilege view.ai.assets

iics role create --from-file cai-viewer-role.json
```

```powershell
iics role create --name "CAI Viewer" --description "View CAI assets" `
  --privilege view.ai.designer --privilege view.ai.assets

iics role create --from-file cai-viewer-role.json
```

---

## role add-privileges

Add privileges to a custom role. Sends
`PUT /public/core/v3/roles/<id>/addPrivileges`, or
`PUT /public/core/v3/roles/name/<name>/addPrivileges` when `--name` is used.

### Flags

| Flag          | Type     | Required | Description                                         |
| ------------- | -------- | -------- | --------------------------------------------------- |
| `--id`        | string   | one of   | Role ID                                             |
| `--name`      | string   | one of   | Role name                                           |
| `--privilege` | string[] | yes      | Privilege name or ID; repeatable or comma-separated |

`--id` and `--name` are mutually exclusive. Privilege IDs are converted to
names, which is what the endpoint expects.

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
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

### Flags

| Flag          | Type     | Required | Description                                         |
| ------------- | -------- | -------- | --------------------------------------------------- |
| `--id`        | string   | one of   | Role ID                                             |
| `--name`      | string   | one of   | Role name                                           |
| `--privilege` | string[] | yes      | Privilege name or ID; repeatable or comma-separated |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
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

- [privilege](privilege.md) - list available privileges to assign to roles
- [user](user.md) - assign roles to users
- [group](group.md) - assign roles to user groups
