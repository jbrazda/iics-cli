# user

Manage IICS users.

## Synopsis

```bash
iics user <subcommand> [flags]
```

## Subcommands

| Subcommand | Description          |
| ---------- | -------------------- |
| `list`     | List users           |
| `get`      | Get a single user    |
| `create`   | Create a user        |
| `update`   | Update a user        |
| `delete`   | Delete a user        |

---

## user list

### Flags

| Flag      | Type | Default | Description     |
| --------- | ---- | ------- | --------------- |
| `--limit` | int  | 200     | Max results     |
| `--skip`  | int  | 0       | Results to skip |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column      | Description            |
| ----------- | ---------------------- |
| `id`        | User ID                |
| `userName`  | Username / login name  |
| `email`     | Email address          |
| `state`     | Account state (Active/Inactive) |
| `updateTime`| Last modification time |

### Examples

```bash
iics user list

iics user list --output json

# Find users by name using JSON + jq
iics user list --output json | jq '.[] | select(.userName | test("john"))'
```

---

## user get

### Flags

| Flag   | Type   | Required | Description |
| ------ | ------ | -------- | ----------- |
| `--id` | string | yes      | User ID     |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics user get --id <user-id>

iics user get --id <user-id> --output json
```

---

## user create

Create a user from a JSON definition file.

### Flags

| Flag          | Type   | Required | Description                      |
| ------------- | ------ | -------- | -------------------------------- |
| `--from-file` | string | yes      | JSON file with user definition   |

All [global flags](../../README.md#global-flags) apply.

### JSON definition example

```json
{
  "userName": "jane.smith@company.com",
  "firstName": "Jane",
  "lastName": "Smith",
  "title": "Data Engineer",
  "phone": "+1-555-000-0001",
  "timezone": "America/New_York",
  "roles": [
    { "id": "<role-id>", "name": "Designer" }
  ],
  "groups": [
    { "id": "<group-id>", "userGroupName": "Data Engineering" }
  ]
}
```

### Examples

```bash
iics user create --from-file new-user.json
```

---

## user update

### Flags

| Flag          | Type   | Required | Description                   |
| ------------- | ------ | -------- | ----------------------------- |
| `--id`        | string | yes      | User ID                       |
| `--from-file` | string | yes      | JSON file with updated fields |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics user update --id <user-id> --from-file updated-user.json
```

---

## user delete

Delete a user. Prompts for confirmation unless `--yes` is given.

### Flags

| Flag    | Short | Type   | Required | Description              |
| ------- | ----- | ------ | -------- | ------------------------ |
| `--id`  |       | string | yes      | User ID                  |
| `--yes` | `-y`  | bool   |          | Skip confirmation prompt |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics user delete --id <user-id>

# Non-interactive
iics user delete --id <user-id> --yes
```

## See also

- [usergroup](usergroup.md) - manage user groups
- [role](role.md) - manage roles assignable to users
- [privilege](privilege.md) - list available privileges
