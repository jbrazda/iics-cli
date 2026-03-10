# permission

Manage object-level permissions in IICS. Alias: `perm`.

Permissions control which users and groups can read, write, or execute a specific object.

## Synopsis

```bash
iics permission <subcommand> [flags]
iics perm <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                     |
| ---------- | ------------------------------- |
| `get`      | Get permissions for an object   |
| `set`      | Set permissions on an object    |
| `delete`   | Delete permissions from an object |

---

## permission get

Retrieve the current permission assignments for an object.

### Flags

| Flag          | Type   | Required | Description   |
| ------------- | ------ | -------- | ------------- |
| `--object-id` | string | yes      | Object ID     |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column          | Description                                |
| --------------- | ------------------------------------------ |
| `principalId`   | User or group ID                           |
| `principalType` | Type: `USER` or `GROUP`                    |
| `principalName` | User or group name                         |
| `permission`    | Permission level: `READ`, `WRITE`, `EXECUTE` |

### Examples

```bash
iics permission get --object-id <object-id>

iics perm get --object-id <object-id> --output json

# Resolve object ID from path first
ID=$(iics lookup --path "My Project/ETL/LoadOrders" --type MTT --output json | jq -r '.id')
iics permission get --object-id "$ID"
```

---

## permission set

Set or replace permissions on an object from a JSON file.

### Flags

| Flag          | Type   | Required | Description                        |
| ------------- | ------ | -------- | ---------------------------------- |
| `--object-id` | string | yes      | Object ID                          |
| `--from-file` | string | yes      | JSON file with permission definitions |

All [global flags](../../README.md#global-flags) apply.

### JSON definition example

```json
[
  {
    "principalId": "<user-id>",
    "principalType": "USER",
    "permission": "READ"
  },
  {
    "principalId": "<group-id>",
    "principalType": "GROUP",
    "permission": "WRITE"
  }
]
```

### Examples

```bash
iics permission set --object-id <object-id> --from-file permissions.json

iics perm set --object-id <object-id> --from-file permissions.json
```

---

## permission delete

Remove all permissions from an object. Prompts for confirmation unless `--yes` is given.

### Flags

| Flag          | Short | Type   | Required | Description              |
| ------------- | ----- | ------ | -------- | ------------------------ |
| `--object-id` |       | string | yes      | Object ID                |
| `--yes`       | `-y`  | bool   |          | Skip confirmation prompt |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics permission delete --object-id <object-id>

iics perm delete --object-id <object-id> --yes
```

## See also

- [lookup](lookup.md) - resolve object paths to IDs
- [user](user.md) - manage users referenced in permissions
- [usergroup](usergroup.md) - manage groups referenced in permissions
