# BUG: role commands do not match the v3 Roles API

---

## Symptoms

`iics role get --id <role-id>` calls `GET /public/core/v3/roles/{id}`, which is not part of the
IICS v3 Roles API. The command cannot look up a role by name and cannot return the privileges
assigned to a role.

Follow-up review of the other role commands against the
[Roles](https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/roles.html)
documentation found:

- `role create` sent the `Role` struct (`roleName`, privilege objects), but the API expects
  `name`, `description` and `privileges` as an array of privilege IDs.
- `role update` sent `PUT /roles/{id}`, which does not exist. The API only supports
  `PUT /roles/{id}/addPrivileges` and `PUT /roles/{id}/removePrivileges` (or the
  `/roles/name/{name}/...` variants), with privilege names in the body.
- `role delete` (`DELETE /roles/{id}`) matched the API but could not target a role by name.

---

## Command / Reproduction Steps

```bash
iics role get --id <role-id> --verbose
```

---

## Expected Behaviour

Per the [Getting role details](https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/roles/getting-role-details.html)
documentation, a single role is retrieved with a query filter on `roleId` or `roleName`, and
privileges are included with `expand=privileges`:

```text
GET /public/core/v3/roles?q=roleName=="Business Manager"&expand=privileges
```

- `iics role get --id <role-id>` queries `q=roleId=="<role-id>"`
- `iics role get --name "<role-name>"` queries `q=roleName=="<role-name>"`
- `--privileges` adds `expand=privileges` and shows the privileges

---

## Architecture Layer

- [x] **`cmd/`** - flag parsing, command wiring, output formatting
- [x] **`internal/client/`** - HTTP logic, API structs, request/response handling
- [ ] **`internal/config/`** - config file loading, session cache
- [ ] **`internal/output/`** - table / JSON / CSV renderer

---

## API Details

| Field          | Value                       |
| -------------- | --------------------------- |
| API version    | V3 (`public/core/v3`)       |
| HTTP method    | GET                         |
| Endpoint path  | `public/core/v3/roles`      |
| Session header | `INFA-SESSION-ID`           |

The `privileges` array contains objects (`id`, `name`, `description`, `service`, `status`), but
`Role.Privileges` was typed `[]string`, so decoding an expanded response would fail.

---

## Fix (filled in after resolution)

**Root cause:**

`GetRole` built the URL `/roles/{id}`, which the v3 Roles API does not provide. The role struct
also typed `privileges` as `[]string` instead of an array of objects. Create reused the response
struct as the request body, and update targeted a non-existent endpoint.

**Files changed:**

```text
internal/client/roles.go          - Added RolePrivilege struct; Role.Privileges is now []RolePrivilege
internal/client/roles.go          - Added RoleGetOptions (ID, Name, ExpandPrivileges)
internal/client/roles.go          - GetRole queries /roles with q=roleId==/roleName== and optional expand=privileges
cmd/role.go                       - role get: --id / --name (mutually exclusive), --privileges flag
cmd/role.go                       - Table output renders role details plus a privileges table
cmd/role.go                       - role get prompts for a role when --id and --name are omitted (terminal only)
cmd/role.go                       - role get table output asks "Display role privileges? [y/Q]" when --privileges is omitted
cmd/role_prompt.go                - pickRole picker, privilege prompt, privilege sort (service, name), status colors
internal/client/roles.go          - Added CreateRoleRequest (name, description, privilege IDs); CreateRole uses it
internal/client/roles.go          - Removed UpdateRole; added RoleRef, AddRolePrivileges, RemoveRolePrivileges
internal/client/roles.go          - Added ResolvePrivileges (privilege name or ID to Privilege)
cmd/role.go                       - role create: --name, --description, --privilege (name or ID), --from-file
cmd/role.go                       - Replaced role update with role add-privileges / remove-privileges
cmd/role.go                       - role delete: added --name
docs/documentation/role.md        - Updated get, create, delete; added add-privileges, remove-privileges
README.md                         - Updated role subcommands
```

**Test added / updated:**

```text
internal/client/roles_test.go - TestGetRoleByName, TestGetRoleByID, TestGetRoleNotFound, TestGetRoleRequiresSelector
internal/client/roles_test.go - TestCreateRole, TestAddRolePrivileges, TestAddRolePrivilegesValidation
internal/client/roles_test.go - TestResolvePrivileges, TestDeleteRole
```
