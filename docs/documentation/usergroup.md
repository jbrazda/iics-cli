# usergroup

Manage IICS user groups. Alias: `ug`.

## Synopsis

```bash
iics usergroup <subcommand> [flags]
iics ug <subcommand> [flags]
```

## Subcommands

| Subcommand | Description             |
| ---------- | ----------------------- |
| `list`     | List user groups        |
| `get`      | Get a single user group |
| `create`   | Create a user group     |
| `update`   | Update a user group     |
| `delete`   | Delete a user group     |

---

## usergroup list

### Flags

| Flag        | Short | Type   | Default | Description                                      |
| ----------- | ----- | ------ | ------- | ------------------------------------------------ |
| `--limit`   |       | int    | 200     | Max results                                      |
| `--skip`    |       | int    | 0       | Results to skip                                  |
| `--query`   | `-q`  | string |         | Filter query (e.g. `userGroupName=="Admins"`)    |
| `--fields`  |       | string | (note)  | Comma-separated list of fields to display        |

Default fields for `--output table`: `id,userGroupName,updatedBy,updateTime,countMembers,countRoles`

Default fields for `--output csv`: `id,userGroupName,updatedBy,updateTime,description,countMembers,countRoles`

All [global flags](../../README.md#global-flags) apply.

### Available fields

| Field           | Description                                       |
| --------------- | ------------------------------------------------- |
| `id`            | User group ID                                     |
| `userGroupName` | Group name                                        |
| `description`   | Group description                                 |
| `updatedBy`     | Last modifier                                     |
| `updateTime`    | Last modification time                            |
| `createdBy`     | Creator                                           |
| `createTime`    | Creation time                                     |
| `countMembers`  | Number of users in the group (computed)           |
| `countRoles`    | Number of roles assigned to the group (computed)  |

### Examples

```bash
# List all user groups (table, default columns)
iics usergroup list

# Filter by name using server-side query
iics ug list --query 'userGroupName=="Administrator"'

# JSON output (all fields available via jq)
iics ug list --output json

# CSV export with custom fields
iics ug list --output csv --fields id,userGroupName,description,countMembers,countRoles

# Show member and role counts
iics ug list --fields id,userGroupName,countMembers,countRoles
```

```powershell
# List all user groups (table, default columns)
iics usergroup list

# Filter by name using server-side query
iics ug list --query 'userGroupName=="Administrator"'

# JSON output (all fields available)
iics ug list --output json

# CSV export with custom fields
iics ug list --output csv --fields id,userGroupName,description,countMembers,countRoles

# Show member and role counts
iics ug list --fields id,userGroupName,countMembers,countRoles
```

---

## usergroup get

### Flags

| Flag   | Type   | Required | Description    |
| ------ | ------ | -------- | -------------- |
| `--id` | string | yes      | User group ID  |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics usergroup get --id <group-id>

iics usergroup get --id <group-id> --output json
```

```powershell
iics usergroup get --id <group-id>

iics usergroup get --id <group-id> --output json
```

---

## usergroup create

Create a user group from a JSON definition file.

### Flags

| Flag          | Type   | Required | Description                           |
| ------------- | ------ | -------- | ------------------------------------- |
| `--from-file` | string | yes      | JSON file with user group definition  |

All [global flags](../../README.md#global-flags) apply.

### JSON definition example

```json
{
  "userGroupName": "Data Engineering",
  "description": "Data Engineering team",
  "roles": [
    { "id": "<role-id>", "roleName": "Designer" }
  ],
  "users": [
    { "id": "<user-id>", "userName": "jane.smith@company.com" }
  ]
}
```

### Examples

```bash
iics usergroup create --from-file data-engineering-group.json
```

```powershell
iics usergroup create --from-file data-engineering-group.json
```

---

## usergroup update

### Flags

| Flag          | Type   | Required | Description                      |
| ------------- | ------ | -------- | -------------------------------- |
| `--id`        | string | yes      | User group ID                    |
| `--from-file` | string | yes      | JSON file with updated fields    |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics usergroup update --id <group-id> --from-file updated-group.json
```

```powershell
iics usergroup update --id <group-id> --from-file updated-group.json
```

---

## usergroup delete

Delete a user group. Prompts for confirmation unless `--yes` is given.

### Flags

| Flag    | Short | Type   | Required | Description              |
| ------- | ----- | ------ | -------- | ------------------------ |
| `--id`  |       | string | yes      | User group ID            |
| `--yes` | `-y`  | bool   |          | Skip confirmation prompt |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics usergroup delete --id <group-id>

iics ug delete --id <group-id> --yes
```

## See also

- [user](user.md) - manage individual users
- [role](role.md) - manage roles assignable to groups
