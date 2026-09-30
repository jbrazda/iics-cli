# user

Manage IICS users.

## Synopsis

```bash
iics user <subcommand> [flags]
```

## Subcommands

| Subcommand         | Description                                     |
| ------------------ | ----------------------------------------------- |
| `list`             | List users                                      |
| `get`              | Get a single user                               |
| `create`           | Create a user                                   |
| `edit`             | Edit group and role assignments (interactive)   |
| `update-roles`     | Add, remove or replace a user's roles           |
| `update-groups`    | Add, remove or replace a user's user groups     |
| `delete`           | Delete a user                                   |
| `change-password`  | Change a user password                          |
| `reset-password`   | Reset a user password using the security answer |

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

```powershell
iics user list

iics user list --output json

# Find users by name using PowerShell
$users = iics user list --output json | ConvertFrom-Json
$users | Where-Object { $_.userName -match "john" }
```

---

## user get

Show a user's details (profile fields, groups and roles).

### Flags

| Flag         | Type   | Required | Description                                  |
| ------------ | ------ | -------- | -------------------------------------------- |
| `--id`       | string | one of*  | User ID                                      |
| `--username` | string | one of*  | User name (exact match, case-insensitive)    |
| `--fields`   | string |          | Fields for CSV output                        |

\* On a terminal both can be omitted: you are asked for a
[user search](#finding-a-user).

All [global flags](../../README.md#global-flags) apply.

### Finding a user

Without `--id` / `--username` on a terminal (also used by `user edit` and
`user delete`), one prompt accepts a user name, a user ID, or part of a name
or email:

1. An exact user name or user ID is looked up on the server
   (`q=userName==` / `q=userId==`).
2. Otherwise the text is matched (case-insensitive) against user name, first
   name, last name, full name and email. One match is used directly; several
   are shown in a filterable list with user name, name, email and state.
3. No match asks again; an empty answer cancels.

The IICS API only filters exact user names and IDs, so partial matching
lists the users page by page (200 per request). An organization has at most
1000 users, user groups and roles combined, so this stays within a few
requests.

### Examples

```bash
iics user get --id <user-id>

iics user get --username jdoe@example.com --output json

# Search interactively
iics user get
```

```powershell
iics user get --id <user-id>

iics user get --username jdoe@example.com --output json

iics user get
```

------ | ------ | -------- | ----------- |
| `--id` | string | yes      | User ID     |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics user get --id <user-id>

iics user get --id <user-id> --output json
```

```powershell
iics user get --id <user-id>

iics user get --id <user-id> --output json
```

---

## user create

Create a user from a definition file, or interactively with `--interactive`.

### Flags

| Flag            | Type   | Required | Description                                            |
| --------------- | ------ | -------- | ------------------------------------------------------ |
| `--from-file`   | string | one of   | JSON, YAML or CSV file with user(s); `-` for stdin     |
| `--interactive` | bool   | one of   | Launch the interactive creation wizard (terminal only) |

### Interactive wizard

`--interactive` opens a paged form (`Enter` next field, `Shift+Tab` back):

1. **Identity** - authentication (Native or SSO), first name, last name,
   user name and email (validated). User name and email show a generated
   value in gray once first and last name are entered; `→` or `Ctrl+E`
   fills it in for editing, and leaving the field empty uses it. The values
   come from the active profile's [new user patterns](#new-user-patterns).
2. **Single sign-on** - alias name in the identity provider; shown only for
   SSO, where the API requires it.
3. **Details** - phone, title, description, force password change.
4. **Membership** - user groups and roles as filterable checklists
   (`/` filter, `Space` toggle).
5. **Review** - summary with **Create user**, **Back to editing** or
   **Cancel**. At least one user group or role is required.

The create API does not accept a time zone, so the wizard does not ask for
one. See [Interactive prompts](interactive.md) for keys.

### New user patterns

The suggested user name and email are built from patterns in the active
profile (`~/.iics/config.yaml`):

```yaml
profiles:
  dev:
    username: admin@acme.com
    newUser:
      domain: acme.com                                                  # default: domain of the profile username
      userNamePattern: "{firstName}.{lastName}.{profileName}@{domain}"  # default
      emailPattern: "{firstName}.{lastName}@{domain}"                   # default
```

| Placeholder      | Value                                                   |
| ---------------- | ------------------------------------------------------- |
| `{firstName}`    | First name, lowercase, spaces removed                   |
| `{lastName}`     | Last name, lowercase, spaces removed                    |
| `{firstInitial}` | First letter of the first name, lowercase               |
| `{lastInitial}`  | First letter of the last name, lowercase                |
| `{profileName}`  | Name of the active profile                              |
| `{domain}`       | `newUser.domain`, or the domain of the profile username |

With the defaults, Jane Doe in profile `dev` (login `admin@acme.com`) gets
user name `jane.doe.dev@acme.com` and email `jane.doe@acme.com`. An unknown
placeholder or a missing value (for example no `@` in the profile username
and no `newUser.domain`) shows a note under the field instead of a
suggestion. `iics profile show <name>` lists the effective patterns.

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

iics user create --interactive
```

```powershell
iics user create --from-file new-user.json

iics user create --interactive
```

---

## user edit

Edit a user's user group and role assignments in an interactive form. Requires
a terminal; for scripts use [user update-roles](#user-update-roles) and
[user update-groups](#user-update-groups).

The form shows the user's name, email, authentication and state read-only,
then the user groups and roles as filterable checklists with the current
assignments pre-checked, and ends with a review (`+ Group: ...`,
`- Role: ...`). Changes are applied through the V3 `addGroups` /
`removeGroups` / `addRoles` / `removeRoles` endpoints (additions first), and
the resulting user is printed. See [Interactive prompts](interactive.md) for
keys.

User properties (name, email, phone, title, description, time zone) cannot be
changed: the IICS REST API has no working update endpoint for them. The V3
`PUT /public/core/v3/users/{id}` returns HTTP 405, and the V2
`POST /api/v2/user/{id}` never succeeded in testing (HTTP 400, later HTTP 403
`REPO_10704`).

> `user update` is a deprecated, hidden alias of `user edit`. Its former
> `--from-file` option was removed because it never worked against the API.
> `--interactive` is accepted and ignored.

### Flags

| Flag         | Type   | Required | Description             |
| ------------ | ------ | -------- | ----------------------- |
| `--id`       | string |          | User ID                 |
| `--username` | string |          | User name (exact match) |

Without `--id` or `--username`, you are asked to [find the user](#finding-a-user).

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics user edit --username user@example.com

# Search for the user first
iics user edit
```

```powershell
iics user edit --username user@example.com

iics user edit
```

---

## user update-roles

Add, remove or replace the roles assigned to a user. Uses
`PUT /public/core/v3/users/<id>/addRoles` and `.../removeRoles`.

### Flags

| Flag                   | Type     | Required | Description                                                |
| ---------------------- | -------- | -------- | ---------------------------------------------------------- |
| `--id` (`--uid`)       | string   | one of   | User ID                                                    |
| `--username` (`--uname`) | string | one of   | User name (exact match)                                    |
| `--add`                | string[] |          | Roles to assign; comma-separated or repeated               |
| `--remove`             | string[] |          | Roles to unassign; comma-separated or repeated             |
| `--replace`            | string[] |          | Exact list of roles; all other roles are removed           |
| `--fields`             | string   |          | Fields for CSV output (same as `user get`)                 |

Rules:

- `--id` and `--username` are mutually exclusive; one is required.
- `--add` and `--remove` can be combined. `--replace` cannot be combined with
  either. `--replace=` (empty) removes all roles.
- Names match case-insensitively, by role name or display name. Duplicates are
  ignored. Unknown names, or a name in both `--add` and `--remove`, fail
  before any change is made.
- Roles already assigned (for `--add`) or not assigned (for `--remove`) are
  skipped and reported on stderr.
- Additions are applied before removals.
- The role assignment endpoints match the role's **display name** (for
  example `Data Integration Data Previewer` for the `Data Preview` role); the
  command sends the display name, so either name can be typed.
- The API rejects changes when the organization maps SAML groups and roles.

After the change the resulting user is printed in the `--output` format
(table sections, JSON, YAML, or CSV with `--fields`). Progress messages go to
stderr, so JSON output can be piped.

### Examples

```bash
iics user update-roles --username jdoe@example.com --add "Designer,Monitor"

iics user update-roles --id <user-id> --add Designer --remove "Data Preview"

# Set the exact role list and print the user as JSON
iics user update-roles --username jdoe@example.com --replace Designer -o json
```

```powershell
iics user update-roles --username jdoe@example.com --add "Designer,Monitor"

iics user update-roles --id <user-id> --add Designer --remove "Data Preview"

iics user update-roles --username jdoe@example.com --replace Designer -o json
```

---

## user update-groups

Add, remove or replace the user groups assigned to a user. Uses
`PUT /public/core/v3/users/<id>/addGroups` and `.../removeGroups`. Flags and
rules are the same as [user update-roles](#user-update-roles), with user group
names instead of role names.

### Examples

```bash
iics user update-groups --username jdoe@example.com --add "Data Engineering"

# Remove the user from all groups
iics user update-groups --uid <user-id> --replace=
```

```powershell
iics user update-groups --username jdoe@example.com --add "Data Engineering"

iics user update-groups --uid <user-id> --replace=
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

```powershell
iics user delete --id <user-id>

# Non-interactive
iics user delete --id <user-id> --yes
```

---

## user change-password

Change a user password. To change your own password, provide `--old-password`.
An administrator can change another user's password by providing `--id` instead.

> **Security note:** Passwords passed as flags appear in shell history. For
> automated scripts, consider reading the value from a variable or a secrets
> manager and passing it via shell substitution rather than typing it directly.

### Flags

| Flag             | Type   | Required    | Description                                            |
| ---------------- | ------ | ----------- | ------------------------------------------------------ |
| `--new-password` | string | yes         | New password                                           |
| `--old-password` | string | conditional | Current password (required when changing own password) |
| `--id`           | string | conditional | User ID (required when admin changes another user)     |

At least one of `--old-password` or `--id` must be provided.

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
# Change your own password
iics user change-password --old-password <current> --new-password <new>

# Admin changes another user's password
iics user change-password --id <user-id> --new-password <new>
```

```powershell
# Change your own password
iics user change-password --old-password <current> --new-password <new>

# Admin changes another user's password
iics user change-password --id <user-id> --new-password <new>
```

---

## user reset-password

Reset a user password using the user's security question answer. Use this when
the password has expired or been forgotten.

> **Security note:** Passwords and security answers passed as flags appear in
> shell history. See the note in `change-password` above.

### Flags

| Flag                | Type   | Required | Description                             |
| ------------------- | ------ | -------- | --------------------------------------- |
| `--id`              | string | yes      | User ID                                 |
| `--security-answer` | string | yes      | Answer to the user's security question  |
| `--new-password`    | string | yes      | New password                            |

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics user reset-password --id <user-id> --security-answer <answer> --new-password <new>
```

```powershell
iics user reset-password --id <user-id> --security-answer <answer> --new-password <new>
```

---

## See also

- [Interactive prompts](interactive.md) - keys, accessible mode and piped input for wizards and pickers
- [group](group.md) - manage user groups
- [role](role.md) - manage roles assignable to users
- [privilege](privilege.md) - list available privileges
