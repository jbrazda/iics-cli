# permission

Manage object permissions in IICS. Alias: `perm`.

Each access control list (ACL) entry grants one user or user group a set of
permissions on one object (project, folder or asset): `read`, `update`,
`delete`, `execute` and `changePermission`. When an object has no ACLs, access
follows the users' roles; once ACLs exist, they restrict access to the listed
principals (administrators keep the right to change permissions).

Uses the [Object permissions](https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/object-permissions.html)
API (`/public/core/v3/objects/<object ID>/permissions`).

## Synopsis

```bash
iics permission <subcommand> [flags]
iics perm <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                                          |
| ---------- | ---------------------------------------------------- |
| `get`      | List the ACLs of an object                           |
| `add`      | Grant permissions to a user or user group            |
| `update`   | Set the permissions of an existing ACL               |
| `delete`   | Delete one ACL, or all ACLs of an object             |
| `set`      | Make an object's ACLs match a JSON file              |
| `check`    | Show your own access to an object                    |
| `edit`     | Edit an object's ACLs in an interactive grid         |

## Selecting the object

Every subcommand takes the object as either:

| Flag               | Description                                                        |
| ------------------ | ------------------------------------------------------------------ |
| `--object-id`      | Object ID                                                          |
| `--path`, `--type` | Object path and type, resolved with [lookup](lookup.md), e.g. `--path "Default/Sales" --type Folder` |

Common types: `Project`, `Folder`, `DTEMPLATE` (mapping), `MTT` (mapping task),
`TASKFLOW`, `PROCESS`.

## Permission names

`--grant` takes a comma-separated list (or the flag repeated): `read`,
`update`, `delete`, `execute`, `changePermission` (also `perm`), `all`,
`none`. Names are case-insensitive.

---

## permission get

### Output columns

| Column        | Description                               |
| ------------- | ----------------------------------------- |
| `TYPE`        | `USER` or `GROUP`                         |
| `NAME`        | User name or user group name              |
| `READ` ... `CHANGE PERM` | Granted permissions (`true`/`false`) |
| `ACL ID`      | ACL ID, used by `update` / `delete --acl-id` |

`-o json` returns the API shape (see [permission set](#permission-set)).

### Examples

```bash
iics permission get --object-id <object-id>

iics perm get --path "Default/Sales" --type Folder -o json
```

```powershell
iics permission get --object-id <object-id>

iics perm get --path "Default/Sales" --type Folder -o json
```

---

## permission add

Create an ACL for a user or user group. Fails if the principal already has an
ACL on the object; use `permission update` instead.

### Flags

| Flag      | Type     | Required | Description                         |
| --------- | -------- | -------- | ----------------------------------- |
| `--user`  | string   | one of   | User name                           |
| `--group` | string   | one of   | User group name                     |
| `--grant` | string[] | yes      | Permissions to grant                |

### Examples

```bash
iics permission add --object-id <id> --group "Data Engineering" --grant read,execute

iics perm add --path "Default/Sales" --type Folder --user jdoe@example.com --grant all
```

```powershell
iics permission add --object-id <id> --group "Data Engineering" --grant read,execute

iics perm add --path "Default/Sales" --type Folder --user jdoe@example.com --grant all
```

---

## permission update

Replace the permissions of an existing ACL. `--grant` is the complete new set.

### Flags

| Flag       | Type     | Required | Description                     |
| ---------- | -------- | -------- | ------------------------------- |
| `--acl-id` | string   | one of   | ACL ID                          |
| `--user`   | string   | one of   | User name                       |
| `--group`  | string   | one of   | User group name                 |
| `--grant`  | string[] | yes      | Complete permission set         |

### Examples

```bash
iics permission update --object-id <id> --group "Data Engineering" --grant read

iics permission update --object-id <id> --acl-id <acl-id> --grant read,update,execute
```

```powershell
iics permission update --object-id <id> --group "Data Engineering" --grant read

iics permission update --object-id <id> --acl-id <acl-id> --grant read,update,execute
```

---

## permission delete

Delete one ACL (by `--acl-id`, `--user` or `--group`) or every ACL of the
object (`--all`). Prompts for confirmation unless `--yes` is given.

### Flags

| Flag         | Short | Type   | Required | Description                 |
| ------------ | ----- | ------ | -------- | --------------------------- |
| `--acl-id`   |       | string | one of   | ACL ID                      |
| `--user`     |       | string | one of   | User name                   |
| `--group`    |       | string | one of   | User group name             |
| `--all`      |       | bool   | one of   | Delete every ACL            |
| `--yes`      | `-y`  | bool   |          | Skip confirmation prompt    |

### Examples

```bash
iics permission delete --object-id <id> --group "Data Engineering"

iics perm delete --object-id <id> --all --yes
```

```powershell
iics permission delete --object-id <id> --group "Data Engineering"

iics perm delete --object-id <id> --all --yes
```

---

## permission set

Make an object's ACLs match a JSON file:

- principals in the file without an ACL are added (`+`),
- principals whose permissions differ are updated (`~`),
- with `--prune`, ACLs of principals not in the file are deleted (`-`); this
  asks for confirmation unless `--yes` is given.

Principals match by type and case-insensitive name. The planned changes are
printed to stderr; `--dry-run` stops there. Changes are applied in the order
add, update, delete; on error the command stops and reports how many changes
were applied.

### Flags

| Flag          | Short | Type   | Required | Description                                   |
| ------------- | ----- | ------ | -------- | --------------------------------------------- |
| `--from-file` |       | string | yes      | JSON file with the desired ACLs; `-` for stdin |
| `--prune`     |       | bool   |          | Delete ACLs of principals not in the file     |
| `--dry-run`   |       | bool   |          | Show changes without applying them            |
| `--yes`       | `-y`  | bool   |          | Skip the confirmation for `--prune`           |

### JSON file

The same shape as `permission get -o json`; `id` values are ignored.

```json
[
  {
    "principal": { "type": "GROUP", "name": "Data Engineering" },
    "permissions": {
      "read": true,
      "update": true,
      "delete": false,
      "execute": true,
      "changePermission": false
    }
  },
  {
    "principal": { "type": "USER", "name": "jdoe@example.com" },
    "permissions": {
      "read": true,
      "update": false,
      "delete": false,
      "execute": false,
      "changePermission": false
    }
  }
]
```

### Examples

```bash
# Preview, then apply
iics permission set --object-id <id> --from-file acls.json --dry-run
iics permission set --object-id <id> --from-file acls.json

# Copy the ACLs of one folder to another
iics perm get --path "Default/Sales" --type Folder -o json \
  | iics perm set --path "Default/Marketing" --type Folder --from-file - --prune --yes
```

```powershell
iics permission set --object-id <id> --from-file acls.json --dry-run
iics permission set --object-id <id> --from-file acls.json

iics perm get --path "Default/Sales" --type Folder -o json |
  iics perm set --path "Default/Marketing" --type Folder --from-file - --prune --yes
```

---

## permission check

Show the current user's access to an object. With `--asset-type` on a project
or folder, `CREATE` reports whether that asset type can be created there.

### Flags

| Flag           | Type   | Required | Description                                  |
| -------------- | ------ | -------- | -------------------------------------------- |
| `--asset-type` | string |          | Asset type to check create access for        |

### Examples

```bash
iics permission check --object-id <id>

iics perm check --path "Default" --type Project --asset-type DTEMPLATE
```

```powershell
iics permission check --object-id <id>

iics perm check --path "Default" --type Project --asset-type DTEMPLATE
```

## permission edit

Edit an object's ACLs in an interactive grid (terminal only). User groups and
users are rows; `READ`, `UPDATE`, `DELETE`, `EXEC` and `PERM` (change
permission) are columns.

```text
Permissions: Default/Sales (Folder)

  PRINCIPAL                  READ    UPDATE  DELETE  EXEC    PERM
  Data Engineering group     [x]     [+]     [ ]     [x]     [ ]
> Developer_ReadOnly group   [-]     [ ]     [ ]     [ ]     [ ]
  jdoe@example.com user(new) [+]     [ ]     [ ]     [ ]     [ ]
```

| Mark  | Meaning                        |
| ----- | ------------------------------ |
| `[x]` | Granted, unchanged             |
| `[+]` | Will be granted                |
| `[-]` | Will be revoked                |
| `[ ]` | Not granted                    |

| Key                 | Action                                                  |
| ------------------- | ------------------------------------------------------- |
| Arrows or `h j k l` | Move                                                    |
| `Space` or `x`      | Toggle the permission under the cursor                  |
| `a`                 | Toggle every permission in the row                      |
| `c`                 | Toggle the column for all visible rows                  |
| `d`                 | Clear the row; the principal's ACL is deleted           |
| `n`                 | Add a user group or user (filterable list)              |
| `/`, `Ctrl+U`       | Filter rows, clear the filter                           |
| `Enter`, `Esc`, `q` | Finish editing and review                               |

A principal left with no permissions has its ACL deleted. The review lists the
changes (`+` add, `~` update, `-` delete) with **Apply changes**, **Back to
editing** or **Cancel**, and warns when the change removes change permission
from your own user. Changes are applied in the order add, update, delete, and
the resulting ACLs are printed.

Without `--object-id` or `--path`, pick a project from a filterable list, then
the project itself, a folder (to open it), or an asset.

The command checks first that you have change permission on the object.

> Once an object has ACLs, only the listed principals have access. Removing
> your own groups can take away even your create access; administrators keep
> change permission and can delete the ACLs again.

### Examples

```bash
iics permission edit --path "Default/Sales" --type Folder

iics perm edit --object-id <id>

# Pick the object interactively
iics perm edit
```

```powershell
iics permission edit --path "Default/Sales" --type Folder

iics perm edit
```

See [Interactive prompts](interactive.md) for general keys.

## See also

- [lookup](lookup.md) - resolve object paths to IDs
- [user](user.md) - manage users referenced in permissions
- [group](group.md) - manage groups referenced in permissions
