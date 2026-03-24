# lookup

Resolve an object by ID, path, or name. Useful for converting between human-readable paths and the IDs required by other commands.

## Synopsis

```bash
iics lookup [flags]
```

## Flags

| Flag     | Type   | Required | Description                                                  |
| -------- | ------ | -------- | ------------------------------------------------------------ |
| `--id`   | string | *        | Object ID to look up                                         |
| `--path` | string | *        | Object path (requires `--type`)                              |
| `--type` | string | *        | Object type: `PROJECT`, `FOLDER`, `DTEMPLATE`, `MTT`, `DSS`, `CONNECTION`, etc. |

\* Provide exactly one of `--id` or `--path`+`--type`.

All [global flags](../../README.md#global-flags) apply.

## Output columns

| Column        | Description             |
| ------------- | ----------------------- |
| `id`          | Object ID               |
| `type`        | Object type             |
| `path`        | Full path               |
| `description` | Object description      |
| `updateTime`  | Last modification time  |

## Examples

```bash
# Look up by ID
iics lookup --id aLX7qnviqxJdmqpVsd17SG

# Look up a mapping by path
iics lookup --path "My Project/My Folder/My Mapping" --type MTT

# Look up a connection by path
iics lookup --path "Shared/SalesforceConn" --type CONNECTION

# Get JSON output for scripting
iics lookup --id aLX7qnviqxJdmqpVsd17SG --output json

# Extract just the ID from a path lookup
iics lookup --path "My Project/ETL/Load Orders" --type DTEMPLATE --output json \
  | jq -r '.id'
```

```powershell
# Look up by ID
iics lookup --id aLX7qnviqxJdmqpVsd17SG

# Look up a mapping by path
iics lookup --path "My Project/My Folder/My Mapping" --type MTT

# Look up a connection by path
iics lookup --path "Shared/SalesforceConn" --type CONNECTION

# Get JSON output for scripting
iics lookup --id aLX7qnviqxJdmqpVsd17SG --output json

# Extract just the ID from a path lookup
$obj = iics lookup --path "My Project/ETL/Load Orders" --type DTEMPLATE --output json | ConvertFrom-Json
$obj.id
```

## See also

- [objects](objects.md)
- [connection](connection.md)
