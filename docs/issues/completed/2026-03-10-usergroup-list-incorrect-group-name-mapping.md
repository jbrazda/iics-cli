# BUG: Usergroup list command is not printing name

---

## Symptoms

`iics usergroup list` displays incorrect or missing group names in the output table.

---

## Command / Reproduction Steps

```bash
iics usergroup list
iics usergroup list --verbose
```

> Add `--verbose` to capture HTTP-level detail and paste that output below.

---

## Expected Behaviour

The `NAME` column should display the correct user group name for each group returned by the API.

Example Http request

```text
GET /saas/public/core/v3/userGroups?q=userGroupName=="Administrator"&expand=privileges HTTP/1.1
Host: use4.dm-us.informaticacloud.com
INFA-SESSION-ID: *****
Accept: application/json
```

---

## Actual Behaviour

The `NAME` column is empty or shows an incorrect value. The raw API response contains the correct group name, but the Go struct field mapping does not align with the actual JSON field name returned by the API.

### Command

```bash
 ./iics usergroup list --limit 10
```

### Output

```text
┌────────────────────────┬──────┬──────────────────────────────┬──────────────────────────┐
│           ID           │ NAME │          UPDATED BY          │         UPDATED          │
├────────────────────────┼──────┼──────────────────────────────┼──────────────────────────┤
│ 79LozIvF04mhpIokbC37KQ │      │ Hidden for privacy           │ 2026-03-03T19:49:12.000Z │
│ f8MdsqAdpysfXbHDASmpax │      │ Hidden for privacy           │ 2019-06-26T20:32:08.000Z │
│ 8vdbUg9DzefdZx26OVwpQU │      │ Hidden for privacy           │ 2022-04-29T19:26:43.000Z │
│ koUrXnWaheihK1iCLCSIY0 │      │ Hidden for privacy           │ 2018-11-20T15:42:53.000Z │
│ 3xlbXU9fFAkexXpoSsbv0W │      │ Hidden for privacy           │ 2018-11-20T00:32:15.000Z │
│ 7nsKzzbO7mZlD6y3g0HA6s │      │ Hidden for privacy           │ 2019-09-16T16:16:41.000Z │
│ iSm8ZlgilIZlyoEIy5B7Ud │      │ Hidden for privacy           │ 2025-03-31T18:41:19.000Z │
│ kEfvuO2LcK4f1cSXZZSlwB │      │ Hidden for privacy           │ 2024-05-03T20:35:28.000Z │
│ cKe0fKfUawTdAV4PvyeQWr │      │ Hidden for privacy           │ 2023-06-06T13:42:13.000Z │
│ 1a7PobyctMKglCVbdmo0oy │      │ Hidden for privacy           │ 2023-04-21T17:05:06.000Z │
└────────────────────────┴──────┴──────────────────────────────┴──────────────────────────┘
```

---

## Environment

| Field                      | Value                                               |
| -------------------------- | --------------------------------------------------- |
| OS                         |                                                     |
| `iics --version`           | iics version ab9e88b (commit: none, built: unknown) |
| Go version                 | 1.25.0                                              |
| IICS region                |                                                     |
| Output format (`--output`) | table                                               |

---

## Architecture Layer

- [ ] **`cmd/`** - flag parsing, command wiring, output formatting
- [x] **`internal/client/`** - HTTP logic, API structs, request/response handling
- [ ] **`internal/config/`** - config file loading, session cache
- [ ] **`internal/output/`** - table / JSON / CSV renderer

---

## Likely Affected Files

```text
internal/client/usergroups.go
cmd/usergroup.go
```

> Files Claude should read for context but NOT modify:

```text
internal/client/client.go      # HTTP do() / doJSON() helpers
docs/CLAUDE.md                 # Project conventions (mandatory read)
```

---

## API Details (if relevant)

| Field          | Value                                       |
| -------------- | ------------------------------------------- |
| API version    | V3 (`public/core/v3`)                       |
| HTTP method    | GET                                         |
| Endpoint path  | `public/core/v3/userGroups`                 |
| Session header | `INFA-SESSION-ID`                           |

**Actual API response (JSON):**
The actual jsonm otput contains following
Only 1 record included for simplicity to demonstrate the issue
Note that the grou name is not name but userGroupName

```json
[
    {
        "id": "79LozIvF04mhpIokbC37KQ",
        "orgId": "iGefnn2sAmxbnCx8WdS5mb",
        "createdBy": "OrgMigUser_1542674026095",
        "updatedBy": "admin.dev@acme.com",
        "createTime": "2018-11-20T00:31:07.000Z",
        "updateTime": "2026-03-03T19:49:12.000Z",
        "userGroupName": "Administrator",
        "description": "",
        "roles": [
            {
                "id": "9gedBDoYQoQibNMohf5KCh",
                "roleName": "Admin",
                "description": "Role for performing administrative tasks for an organization. Has full access to all licensed services.",
                "displayName": "Admin",
                "displayDescription": "Role for performing administrative tasks for an organization. Has full access to all licensed services."
            },
            {
                "id": "24tgNRrSBMziVgl3X9FXKE",
                "roleName": "Governance Administrator",
                "description": "Governance Administrator role for Metadata Command Center and Data Governance and Catalog application",
                "displayName": "Governance Administrator",
                "displayDescription": "Governance Administrator role for Metadata Command Center and Data Governance and Catalog application"
            }
        ],
        "users": [
            {
                "id": "0LWjYwBFdk0kPhQ6EPnLWa",
                "userName": "user1.dev@natl.com",
                "description": ""
            },
            {
                "id": "0WcqpyAicwEjBcWzHqhIcd",
                "userName": "user2.dev@natl.com",
                "description": ""
            }
        ]
    }
]
```

**Struct that maps this response** (file + line):

```text
internal/client/usergroups.go
```

> Common struct bugs: wrong JSON tag name, `[]string` used where `[]SomeStruct` is needed,
> `string` used where the API actually returns a number, missing `omitempty` on optional fields.
> incorrect fild name mapping

---

## Error Message / Stack Trace

```text
No error, missing data
```

---

## Fix Instructions

1. Read `docs/CLAUDE.md` and the affected files listed above before writing any code.
2. In `internal/client/usergroups.go`, verify the JSON tag for the group name field against the actual API response. The field is likely tagged `"name"` but the returns `"userGroupName"`.
3. verify completeness of api and field mapping based on the [Documentation](https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/users/getting-user-details.html)
4. Add support to potantially priint all `root` fields in csv and table output controlled by --fields  parameter  defaulting to     id, name, status,updated, updatedBy, description
5. Also support clalculate fields countMembers to display number user group members and countRoles to display number of group roles
6. Add support for --query string  short q paraneter

    String
    Query filter. You can filter using one of the following fields:
    userId. Unique identifier for the user.
    userName. Informatica Intelligent Cloud Services user name.

7. Update all related documentation and command completions
8. In `cmd/usergroup.go`, verify the `Column.Field` value for the name column matches the corrected JSON tag exactly.
9. Do **not** touch any other fields or commands.
10. Add or update the test in `internal/client/usergroups_test.go` to assert the group name is correctly populated.
11. Run `/opt/local/bin/go test ./internal/client/...` and verify it passes.
12. Run `/opt/local/bin/go build ./...` to confirm no compilation errors.

---

## Acceptance Criteria

- [ ] The reproduction command now produces the expected output
- [ ] All existing tests still pass (`/opt/local/bin/go test ./...`)
- [ ] No unrelated code is refactored
- [ ] No new dependencies are introduced
- [ ] `go vet ./...` and `golangci-lint run ./...` report no new issues

---

## Do NOT

- Refactor, reformat, or add comments to code outside the fix scope
- Change function signatures or struct names not directly involved in the bug
- Add error handling for scenarios unrelated to this bug
- Never gues JSON field names - verify against the API docs or the raw response pasted above

---

## Fix (filled in after resolution)

**Root cause:**

`UserGroup.Name` had JSON tag `"name"` but the IICS v3 API returns the field as `"userGroupName"`.
All commands using the `name` field produced empty output because Go's JSON decoder found no matching key.

**Files changed:**

```text
internal/client/usergroups.go:20    - Renamed Name->UserGroupName, corrected JSON tag to "userGroupName"
internal/client/usergroups.go:13-17 - Added UserGroupMember struct for the users array
internal/client/usergroups.go:29    - Added Users []UserGroupMember field to UserGroup
internal/client/usergroups.go:31-32 - Added CountMembers/CountRoles computed fields to UserGroup
internal/client/usergroups.go:36    - Added Query string to UserGroupListOptions
internal/client/usergroups.go:51-53 - Pass Query as q query parameter
internal/client/usergroups.go:57-60 - Populate CountMembers/CountRoles after list fetch
cmd/usergroup.go:17-29              - Added allUsergroupColumns map with all available column definitions
cmd/usergroup.go:31-38              - Added columnsFromFields helper for --fields flag parsing
cmd/usergroup.go:71                 - Added --query/-q flag to list command
cmd/usergroup.go:72                 - Added --fields flag defaulting to id,userGroupName,updatedBy,updateTime,description
cmd/usergroup.go:101                - Fixed get command column Field "name"->"userGroupName"
cmd/usergroup.go:126                - Fixed create message to use created.UserGroupName
cmd/usergroup.go:158                - Fixed update message to use updated.UserGroupName
```

**Test added / updated:**

```text
internal/client/usergroups_test.go - Updated TestListUserGroups: uses UserGroupName, asserts CountMembers=2 and CountRoles=1
internal/client/usergroups_test.go - Added TestListUserGroupsQuery: verifies q query param is passed correctly
```
