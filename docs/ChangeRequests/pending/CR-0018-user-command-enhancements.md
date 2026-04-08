# CR-0018: User Command Enhancements

## Type

- [ ] New resource
- [x] New subcommand
- [x] Enhancement - change behaviour of an existing command
- [x] Output change - add/remove/rename columns, change default format, fix display
- [x] Flag / config change - add/rename/remove a CLI flag or config field

---

## Problem

The `user` command family has several gaps:

1. `user get` displays a flat single-row table that loses nested `groups` and `roles` data.
2. `user reset-password` requires all arguments up-front with no interactive fallback;
   it cannot discover users interactively and does not validate the `authentication` type.
3. `user change-password` has no confirmation step; new password is never verified.
4. `user create` requires a pre-authored JSON file; there is no interactive wizard,
   no YAML support, and no bulk-creation path from CSV or stdin.
5. `user update` cannot interactively add/remove roles or groups; it requires a full
   replacement JSON body.
6. `user delete` does not support lookup by `userName`; only `--id` is accepted.
7. No sample test data exists for any of the above input paths.

---

## Desired Change

### 1 - `user get`: multi-section table rendering

For `--output table` (default), render three sections instead of a single flat row.

**Section 1 - User Details** (property/value pairs):

| Property        | Value                        |
|-----------------|------------------------------|
| id              | 9IB4JZitmQDhj2cLTgWS64       |
| orgId           | iGefnn2sAmxbnCx8WdS5mb       |
| userName        | jaroslav.brazda.dev@natl.com |
| firstName       | Jaroslav                     |
| lastName        | Brazda                       |
| email           | jaroslav.brazda@natl.com     |
| phone           | 1234567890                   |
| state           | Enabled                      |
| timeZoneId      | America/New_York             |
| title           | App Analyst Dev Consultant   |
| authentication  | Native                       |
| lastLoginMode   | API                          |
| lastLoginTime   | 2026-04-08T20:37:55Z         |
| maxLoginAttempts| 10                           |
| createTime      | 2021-05-03T18:11:59.000Z     |
| createdBy       | paul.luc@natl.com.dev        |
| updateTime      | 2026-04-08T11:45:51.000Z     |
| updatedBy       | System built-in user         |

**Section 2 - Groups:**

| id | userGroupName | description |
|----|---------------|-------------|
| ...| ...           | ...         |

**Section 3 - Roles:**

| id | displayName | displayDescription |
|----|-------------|--------------------|
| ...| ...         | ...                |

For `--output csv`, add a `--fields` flag selecting which top-level properties to include.
Default fields: `id,userName,firstName,lastName,email,state,authentication,lastLoginTime`.
Nested `groups` and `roles` are rendered as pipe-separated IDs in a single column each
(e.g. `groups`, `roles`), appended after the selected scalar fields when present in
`--fields`.

For `--output json` and `--output yaml`, output the full `User` struct unchanged.

---

### 2 - `user reset-password`: interactive lookup and guided prompts

API: `POST /public/core/v3/Users/ResetPassword`
Docs: <https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/passwords/resetting-a-password.html>

**Fully-inlined (non-TTY / CI):**
When `--id` or `--username`, `--security-answer`, and `--new-password` are all provided,
call the API immediately without prompts.

**Partial flags on TTY:**
When `--id` or `--username` is provided but interactive fields are missing, prompt for
the remaining required inputs (security answer, new password, confirm new password).
Repeat the password pair prompt until the two entries match or the user exits.

**No flags on TTY:**
Present an interactive lookup loop:

1. Prompt: `Search by: [1] User Name  [2] ID  [0] Exit`
2. Prompt for the search value.
   - For ID: require exact match via `GetUser(id)`.
   - For User Name: check whether the IICS v3 API supports partial match on
     `GET /public/core/v3/users` (see API docs linked above). If partial match is
     supported, list results and let the user pick one; otherwise require an exact match.
3. If no match found, return to step 1 (re-show field selection, include exit option).
4. If the matched user has `authentication != "Native"`, print an error
   (`reset-password is only supported for Native authentication users`) and return to step 1.
5. Prompt: `Security Answer:` (masked input)
6. Prompt: `New password:` (masked input)
7. Prompt: `Confirm new password:` (masked input)
8. If passwords do not match, repeat steps 6-7.
9. Call the reset-password API and report success or error.
10. All prompts must offer an exit path (empty input or `q`).

**Non-TTY with partial flags:**
Return a usage error listing which required flags are missing.
Do not block waiting for stdin.

---

### 3 - `user change-password`: add confirmation prompt

API: `POST /public/core/v3/Users/ChangePassword`
Docs: <https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/passwords/changing-a-password.html>

Apply the same interactive pattern as reset-password except:

- Requires `--old-password` (own password change) or `--id`/`--username` (admin change).
- Does not require a security answer.
- Prompts: `Old password:`, `New password:`, `Confirm new password:`.
- Repeat new-password pair until they match or user exits.
- When all flags are supplied non-interactively, call the API immediately.
- Non-TTY with partial flags returns a usage error.

---

### 4 - `user create`: interactive wizard and bulk input

API: `POST /public/core/v3/users`
Docs: <https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/users/creating-a-user.html>

**`--interactive` flag - guided wizard (single user):**

1. Prompt: `Authentication type: [1] Native  [2] SSO  [0] Exit`
2. Prompt: `First name:`
3. Prompt: `Last name:`
4. Prompt: `Username:` (default: `{firstName}.{lastName}@{domain-of-current-session-user}`)
5. Prompt remaining supported fields from the API (email, phone, title, timeZoneId, etc.);
   mark required fields per API docs.
6. Fetch available user groups and prompt for selection of one or more.
   Pre-select any group names listed in a new `defaults.userGroups` config field
   (comma-separated group names; IDs resolved at runtime).
7. Fetch available roles and prompt for selection of one or more.
   Pre-select any roles listed in a new `defaults.userRoles` config field
   (comma-separated role names; IDs resolved at runtime).
8. At least one group or role must be selected before proceeding.
9. All steps offer an exit path.
10. After creation, display the created user using the same multi-section layout as
    `user get` (see section 1), or respect `--output` flag.

**`--from-file` flag:**

- Accepts JSON or YAML (auto-detected by file extension or content).
- Single-object input: creates one user, output as `user get` layout.
- Array input: bulk creation; for each user call `POST /public/core/v3/users`.
  Display a summary table after all attempts:

  | id | userName | firstName | lastName | email | state | status |
  |----|----------|-----------|----------|-------|-------|--------|

  `status` column: `Success` (green) or `Error: <message>` (red).
  With `--verbose`, print id, userName, status, and elapsed time per user at INFO level.

- For CSV input, `groups` and `roles` columns are pipe-separated names; names are resolved
  to IDs by fetching the group/role list from the profile before the bulk run.

**Stdin input:**
When `--from-file -` is specified (or input is piped), read from stdin.
Format is inferred from the first non-whitespace character (`{`/`[` = JSON, `-`/letter = YAML,
comma-detection = CSV).

**Error handling:**
Bulk errors are printed to stderr. Stdout receives only the summary table.

---

### 5 - `user update`: interactive role/group management

API docs:
- Update user: <https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/users/updating-a-user.html>
- Update role assignments: <https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/users/updating-role-assignments.html>

Apply the same lookup pattern from reset-password (sections --id / --username / interactive
search) to identify the user to update.

**`--interactive` flag:**
1. Resolve the target user (via `--id`, `--username`, or interactive search).
2. Display current field values; prompt to update each field (press Enter to keep current).
3. Show current groups; prompt to add or remove groups (multi-select diff).
4. Show current roles; prompt to add or remove roles (multi-select diff).
5. Apply field updates via the update-user API.
6. Apply role-assignment changes via add/remove role-assignment API calls as needed
   (do not send a full replacement body; compute the diff and issue only the minimum calls).
7. Print a change report summarising fields modified and roles/groups added or removed.
8. With `--verbose`, print each API call and its result at INFO level.

**`--from-file` / stdin:**
Same JSON/YAML/CSV support as `user create`. Single-object updates one user; array input
performs bulk updates. For CSV, groups and roles columns are pipe-separated names resolved
to IDs.

---

### 6 - Test data samples

Create sample input files in `testdata/user/create/`:

| File | Description |
|------|-------------|
| `single_native.json` | Single Native user, JSON format |
| `single_native.yaml` | Single Native user, YAML format |
| `bulk_native.json` | Array of 3 Native users, JSON |
| `bulk_native.csv` | 3 users, CSV with pipe-separated groups/roles columns |
| `single_sso.json` | Single SSO user, JSON |

Samples must use placeholder values (no real credentials, no real org IDs).

---

### 7 - `user delete`: support lookup by userName

API: `DELETE /public/core/v3/users/{id}`
Docs: <https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/users/deleting-a-user.html>

- Add `--username` flag as an alternative to `--id`.
- When `--username` is provided, resolve to an ID using the list/search approach
  (exact or partial match, same as reset-password lookup, section 2).
- When neither flag is provided on a TTY, fall into the same interactive search loop.
- Existing `--yes` flag skips the confirmation prompt.
- With `--verbose`, print each step (lookup, confirmation, delete call, result) at INFO level.

---

## Implementation Notes

- Interactive prompts must use the same TTY-detection approach as `internal/config/prompt.go`
  (`IsTerminal` / `go-isatty`). All prompt helpers must live in `internal/config/prompt.go`
  or a new `internal/config/user_prompt.go` file - never in `cmd/`.
- The `--fields` flag for CSV output is added to `user get` only; other commands use the
  existing `--output` flag.
- New config fields (`defaults.userGroups`, `defaults.userRoles`) extend the existing
  `Config` struct in `internal/config/config.go`.
- Username default pattern for create wizard (`{firstName}.{lastName}@{domain}`) is
  computed in the client or prompt layer; the domain is extracted from the logged-in user's
  `userName` field in the session.
- Bulk creation/update processes users sequentially (not concurrently) to avoid rate-limit
  issues on the IICS API.
- All new interactive paths must support exit at every prompt step; exiting mid-wizard
  must not leave partial state.

---

## Files Affected

| File | Change |
|------|--------|
| `cmd/user.go` | Update all subcommands per sections 1-7 |
| `internal/client/users.go` | Add role-assignment add/remove methods; extend `UserListOptions` if API supports name filter |
| `internal/client/users_test.go` | Tests for new client methods |
| `internal/config/prompt.go` or `internal/config/user_prompt.go` | Interactive lookup and wizard helpers |
| `internal/config/config.go` | Add `defaults.userGroups` and `defaults.userRoles` fields |
| `docs/documentation/user.md` | Update command reference |
| `README.md` | Update commands table if new flags added |
| `testdata/user/create/` | New sample files (section 6) |
| `completions/` | Regenerate after flag changes |
