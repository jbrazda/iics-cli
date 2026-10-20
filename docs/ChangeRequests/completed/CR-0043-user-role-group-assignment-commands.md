# CR-0043: User role and user group assignment commands

## CR Type

- [x] New command
- [x] Enhancement to existing command

## Problem

`user update` cannot change user properties: the V3 API has no update
endpoint and the V2 update fails (see
`docs/issues/new/2026-09-30-user-update-v2-403.md`). Role and user group
assignments can be changed through documented V3 endpoints, but the CLI only
reached them through the failing `UpdateUser` flow.

References:

- [Updating role assignments](https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/users/updating-role-assignments.html)
- [Updating user group assignments](https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/users/updating-user-group-assignments.html)

## Proposed Solution

- `user update-roles` and `user update-groups`:
  - `--id` / `--username` (aliases `--uid` / `--uname`), one required.
  - `--add`, `--remove` (combinable), `--replace` (exclusive; empty removes
    all).
  - Inputs are de-duplicated and matched case-insensitively; unknown names
    and names in both `--add` and `--remove` fail before any change.
  - Additions before removals; no-op requests reported and skipped.
  - Resulting user printed in the `--output` format.
- `user update --interactive` edits only group and role assignments and
  applies them through the same endpoints.
- Follow-up: `user update` is renamed to `user edit` (terminal only, no
  `--interactive` switch needed). `user update` stays as a hidden,
  deprecated alias. `--from-file` is removed together with the V2 scalar
  update (`Client.UpdateUser`, `updateUserV2Request`, `doXML`), which never
  worked against the API.

## Implementation

- `internal/client/usermembership.go` - `AddUserRoles`, `RemoveUserRoles`,
  `AddUserGroups`, `RemoveUserGroups`, `PlanMembership`, `MemberName`,
  `RoleMemberName`.
- `cmd/user_membership.go` - both commands, `applyUserMembership`, shared
  `printUser` (also used by `user get`).
- `internal/tui/userwizard.go` - update mode shows only the Membership page.
- Finding verified on `dev`: role assignment endpoints match the role
  display name, not `roleName`; the commands send the display name and
  accept either as input.

## Acceptance Criteria

- [x] Add, remove, combined add/remove and replace work on `dev` for roles
      and user groups
- [x] Validation: duplicates, case, unknown names, add/remove conflict,
      replace exclusivity, missing user flag
- [x] Output honors `-o table|json|yaml|csv`
- [x] Interactive update applies group and role changes only
- [x] `user edit` replaces `user update`; `update` kept as deprecated alias;
      `--from-file` update and V2 client code removed
- [x] `go build`, `go vet`, `golangci-lint`, `go test ./...` pass
