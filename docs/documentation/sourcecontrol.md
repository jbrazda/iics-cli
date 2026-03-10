# sourcecontrol

Manage source control operations for IICS objects. Alias: `sc`.

Source control integration allows tracking changes to IICS assets in a Git repository configured in the Administrator settings.

## Synopsis

```bash
iics sourcecontrol <subcommand> [flags]
iics sc <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                                              |
| ---------- | -------------------------------------------------------- |
| `checkout` | Check out an object for editing                         |
| `checkin`  | Check in an object with a commit comment                |
| `pull`     | Pull latest changes from the source control repository  |
| `commit`   | Commit pending changes to the repository                |

---

## sourcecontrol checkout

Check out an object to allow editing. This marks the object as locked for the current user.

### Flags

| Flag          | Type   | Required | Description              |
| ------------- | ------ | -------- | ------------------------ |
| `--object-id` | string | yes      | Object ID to check out   |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column     | Description           |
| ---------- | --------------------- |
| `objectId` | Object ID             |
| `status`   | Checkout status       |
| `message`  | Status message        |

### Examples

```bash
iics sourcecontrol checkout --object-id <object-id>

iics sc checkout --object-id <object-id>

# Resolve path to ID, then checkout
ID=$(iics lookup --path "My Project/ETL/LoadOrders" --type MTT --output json | jq -r '.id')
iics sc checkout --object-id "$ID"
```

---

## sourcecontrol checkin

Check in an object after editing, adding a commit comment.

### Flags

| Flag          | Type   | Required | Description                     |
| ------------- | ------ | -------- | ------------------------------- |
| `--object-id` | string | yes      | Object ID to check in           |
| `--comment`   | string |          | Commit comment for source control |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column     | Description         |
| ---------- | ------------------- |
| `objectId` | Object ID           |
| `status`   | Checkin status      |
| `message`  | Status message      |

### Examples

```bash
iics sourcecontrol checkin --object-id <object-id> --comment "Fix null handling in filter step"

iics sc checkin --object-id <object-id>
```

---

## sourcecontrol pull

Pull the latest changes from the source control repository into IICS.

### Flags

No command-specific flags. All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column    | Description     |
| --------- | --------------- |
| `status`  | Pull status     |
| `message` | Status message  |

### Examples

```bash
iics sourcecontrol pull

iics sc pull --verbose
```

---

## sourcecontrol commit

Commit pending changes from IICS to the source control repository.

### Flags

| Flag        | Type   | Required | Description      |
| ----------- | ------ | -------- | ---------------- |
| `--comment` | string |          | Commit message   |

All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column    | Description      |
| --------- | ---------------- |
| `status`  | Commit status    |
| `message` | Status message   |

### Examples

```bash
iics sourcecontrol commit --comment "Release v2.0 changes"

iics sc commit --comment "Deploy Finance ETL pipeline"
```

## Typical workflow

```bash
# 1. Pull latest from source control
iics sc pull

# 2. Resolve object path to ID
ID=$(iics lookup --path "My Project/ETL/LoadOrders" --type MTT --output json | jq -r '.id')

# 3. Check out the object
iics sc checkout --object-id "$ID"

# (make changes in the IICS Designer)

# 4. Check in with a comment
iics sc checkin --object-id "$ID" --comment "Optimised filter logic"

# 5. Commit all pending changes
iics sc commit --comment "Sprint 12 changes"
```

## See also

- [state](state.md) - fetch and load object state (alternative to source control for snapshots)
- [lookup](lookup.md) - resolve object paths to IDs
