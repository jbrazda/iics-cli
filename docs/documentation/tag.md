# tag

Assign or remove tags on IICS objects. Tags can be used to filter objects with `iics objects list --tag`.

## Synopsis

```bash
iics tag <subcommand> [flags]
```

## Subcommands

| Subcommand | Description                  |
| ---------- | ---------------------------- |
| `assign`   | Assign tags to an object     |
| `remove`   | Remove tags from an object   |

---

## tag assign

Assign one or more tags to an object.

### Flags

| Flag          | Type   | Required | Description                        |
| ------------- | ------ | -------- | ---------------------------------- |
| `--object-id` | string | yes      | ID of the object to tag            |
| `--tags`      | string | yes      | Comma-separated list of tag names  |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Assign a single tag
iics tag assign --object-id aLX7qnviqxJdmqpVsd17SG --tags "prod"

# Assign multiple tags at once
iics tag assign --object-id aLX7qnviqxJdmqpVsd17SG --tags "prod,reviewed,v2"

# Resolve object ID from path first, then tag
ID=$(iics lookup --path "Sales/ETL/LoadOrders" --type MTT --output json | jq -r '.id')
iics tag assign --object-id "$ID" --tags "sales,etl"
```

---

## tag remove

Remove one or more tags from an object.

### Flags

| Flag          | Type   | Required | Description                        |
| ------------- | ------ | -------- | ---------------------------------- |
| `--object-id` | string | yes      | ID of the object to untag          |
| `--tags`      | string | yes      | Comma-separated list of tag names  |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Remove a single tag
iics tag remove --object-id aLX7qnviqxJdmqpVsd17SG --tags "draft"

# Remove multiple tags
iics tag remove --object-id aLX7qnviqxJdmqpVsd17SG --tags "draft,wip"
```

## See also

- [objects](objects.md) - use `--tag` to filter objects by tag
- [lookup](lookup.md) - resolve object paths to IDs
