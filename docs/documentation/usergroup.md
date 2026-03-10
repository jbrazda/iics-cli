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

| Flag      | Type | Default | Description     |
| --------- | ---- | ------- | --------------- |
| `--limit` | int  | 200     | Max results     |
| `--skip`  | int  | 0       | Results to skip |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column      | Description            |
| ----------- | ---------------------- |
| `id`        | User group ID          |
| `name`      | Group name             |
| `updatedBy` | Last modifier          |
| `updateTime`| Last modification time |

### Examples

```bash
iics usergroup list

iics ug list --output json
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
  "name": "Data Engineering",
  "description": "Data Engineering team",
  "roles": [
    { "id": "<role-id>", "name": "Designer" }
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
