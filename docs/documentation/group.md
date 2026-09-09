# group

Manage IICS user groups. Aliases: `usergroup`, `ug`.

## Synopsis

```bash
iics group <subcommand> [flags]
iics usergroup <subcommand> [flags]
iics ug <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                         |
| ---------- | ---------------------------------- |
| `list`     | List user groups                   |
| `get`      | Get a single user group            |
| `create`   | Create one or more user groups     |
| `update`   | Update one or more user groups     |
| `delete`   | Delete a user group                |

## API constraints

The v3 `userGroups` resource is limited:

- **Create** takes a `name`, an optional `description`, and a **non-empty**
  `roles` array (role IDs). `users` (user IDs) is optional.
- **Read** is only via the list endpoint - there is no get-by-id. `get --id`
  and `--name` both scan the list.
- **Update** can only change **role and user membership** (via add/remove
  operations). A group's **name and description cannot be changed** through the
  API - attempting to do so returns an error.
- **Delete** by ID.

---

## group list

### Flags

| Flag        | Short | Type   | Default | Description                                      |
| ----------- | ----- | ------ | ------- | ------------------------------------------------ |
| `--limit`   |       | int    | 200     | Max results                                      |
| `--skip`    |       | int    | 0       | Results to skip                                  |
| `--query`   | `-q`  | string |         | Server-side filter (e.g. `userGroupName=="Admins"`) |
| `--fields`  |       | string | (note)  | Comma-separated list of fields to display        |

Default fields (table): `id,userGroupName,updatedBy,updateTime,countMembers,countRoles`

Default fields (csv): `id,userGroupName,updatedBy,updateTime,description,countMembers,countRoles`

### Available fields

`id`, `userGroupName`, `description`, `updatedBy`, `updateTime`, `createdBy`,
`createTime`, `countMembers`, `countRoles`

### Examples

```bash
iics group list
iics ug list --query 'userGroupName=="Administrator"'
iics group list --output csv --fields id,userGroupName,countMembers,countRoles
```

---

## group get

Get a single user group by ID or name. With neither flag on a terminal, you are
prompted to pick one from a list.

### Flags

| Flag     | Type   | Description         |
| -------- | ------ | ------------------ |
| `--id`   | string | User group ID       |
| `--name` | string | User group name     |

### Output

Table mode prints three sections: a vertical `PROPERTY` / `VALUE` detail table
for the group, a `Roles (N):` table (name, id, description), and a
`Members (N):` table (username, id). `--output json` / `yaml` render the full
group record.

### Examples

```bash
iics group get --id <group-id>
iics group get --name "Data Engineering"
iics group get            # interactive picker
iics group get --name "Data Engineering" --output json
```

---

## group create

Create user groups from JSON (`--from-file` or piped stdin) or interactively.
A JSON **array** creates groups in bulk and prints a per-group result table
(exit code is non-zero if any element failed).

### Flags

| Flag                  | Type   | Description                                              |
| --------------------- | ------ | ------------------------------------------------------- |
| `--from-file`         | string | JSON file (object or array); omit to read piped stdin  |
| `--interactive`, `-i` | bool   | Prompt for name, description, and roles                 |

The wizard runs with `-i`, or when no input source is present on a terminal.

### JSON definition

```json
{
  "name": "Data Engineering",
  "description": "Data Engineering team",
  "roles": [ { "id": "<role-id>" } ],
  "users": [ { "id": "<user-id>" } ]
}
```

`roles` is required and must be non-empty. Role and user entries are objects with
an `id`.

### Examples

```bash
iics group create --from-file group.json
cat groups.json | iics group create          # array -> bulk
iics group create -i
```

---

## group update

Change a group's **role or user membership**. Name and description are immutable
via the API.

Single update - identify the group with `--id` or `--name` (or pick from a list
when neither is given on a terminal), then supply the desired state via
`--from-file` / stdin or `--interactive`. The desired `roles` / `users` sets are
diffed against the current ones and applied with add/remove operations.

Bulk update - pass a JSON array; each element is matched to an existing group by
its `id`, or by `name` / `userGroupName` when no `id` is present. Results are
printed as a table.

For updates, reference roles by `roleName` and users by `userName` (these are
the identifiers the add/remove endpoints use). A file produced by
`group get --output json` already contains them.

### Flags

| Flag                  | Type   | Description                                          |
| --------------------- | ------ | -------------------------------------------------- |
| `--id`                | string | User group ID                                       |
| `--name`              | string | User group name                                     |
| `--from-file`         | string | JSON file (object or array); omit to read piped stdin |
| `--interactive`, `-i` | bool   | Edit the role selection interactively               |

### Examples

```bash
# replace the role set of one group
echo '{"roles":[{"roleName":"Operator"},{"roleName":"Designer"}]}' \
  | iics group update --name "Data Engineering"

# interactive role editing
iics group update --name "Data Engineering" -i

# bulk: set roles on several groups
cat groups.json | iics group update
```

---

## group delete

Delete a user group. Identify it with `--id` or `--name`, or pick from a list
when neither is given on a terminal. Prompts for confirmation unless `--yes`.

### Flags

| Flag    | Short | Type   | Description               |
| ------- | ----- | ------ | ------------------------ |
| `--id`  |       | string | User group ID             |
| `--name`|       | string | User group name           |
| `--yes` | `-y`  | bool   | Skip the confirmation     |

### Examples

```bash
iics group delete --id <group-id>
iics group delete --name "Data Engineering" --yes
iics group delete            # interactive picker
```

## See also

- [user](user.md) - manage individual users
- [role](role.md) - manage roles assignable to groups
