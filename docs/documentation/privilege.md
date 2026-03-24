# privilege

List all privileges available in the organisation. Privileges can be assigned to roles.

## Synopsis

```bash
iics privilege list [flags]
```

## Subcommands

| Subcommand | Description         |
| ---------- | ------------------- |
| `list`     | List all privileges |

## privilege list

### Flags

No command-specific flags. All [global flags](../../README.md#global-flags) apply.

### Output columns

| Column        | Description                          |
| ------------- | ------------------------------------ |
| `id`          | Privilege ID                         |
| `name`        | Privilege name                       |
| `service`     | Service the privilege applies to     |
| `description` | Description of what the privilege grants |

### Examples

```bash
# List all privileges in table format
iics privilege list

# List as JSON for scripting
iics privilege list --output json

# Find privileges for a specific service
iics privilege list --output json | jq '.[] | select(.service == "DataIntegration")'
```

```powershell
# List all privileges in table format
iics privilege list

# List as JSON for scripting
iics privilege list --output json

# Find privileges for a specific service
$privileges = iics privilege list --output json | ConvertFrom-Json
$privileges | Where-Object { $_.service -eq "DataIntegration" }
```

## See also

- [role](role.md)
- [user](user.md)
