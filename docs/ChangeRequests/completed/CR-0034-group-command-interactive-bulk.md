# CR-0034: Rename `usergroup` to `group`; interactive + bulk create/update; name lookup

## CR Type

- [x] Enhancement to existing command
- [x] Bug fix (create/update used the wrong request shape and a non-existent endpoint)

## Problem

- The command is `usergroup`; a shorter `group` name is wanted.
- `usergroup create` / `update` were never exercised live. They:
  - sent `{"userGroupName": ..., "roles": [{...}]}` - the v3 create endpoint
    wants `{"name": ..., "roles": ["<roleId>", ...]}` (role IDs, non-empty).
  - `update` did `PUT /userGroups/<id>` - that path only allows `DELETE`
    (HTTP 405). Membership is changed via `addRoles` / `removeRoles` /
    `addUsers` / `removeUsers` sub-endpoints (role/user **names**).
  - `get --id` did `GET /userGroups/<id>` - also HTTP 405; the v3 resource has
    no get-by-id.
- No interactive mode; `update` / `delete` only accepted `--id`.
- No bulk (array) create/update.

## Verified API (live, 2026-09-09)

| Op | Method + path | Body |
| --- | --- | --- |
| Create | `POST /public/core/v3/userGroups` | `{name, description?, roles:[roleId,...], users:[userId,...]}` - roles required, non-empty |
| List / read | `GET /public/core/v3/userGroups` (`?q=`) | only way to read; no get-by-id |
| Delete | `DELETE /public/core/v3/userGroups/<id>` | - |
| Add/remove roles | `PUT /public/core/v3/userGroups/<id>/addRoles` \| `/removeRoles` | `{"roles":["<roleName>", ...]}` |
| Add/remove users | `PUT /public/core/v3/userGroups/<id>/addUsers` \| `/removeUsers` | `{"users":["<userName>", ...]}` |

Name and description are **immutable** after creation.

## Solution

1. Rename command to `group`; keep `usergroup`, `ug` as aliases.
2. `internal/client/usergroups.go`:
   - `GetUserGroup` / `GetUserGroupByName` scan the list.
   - `CreateUserGroup` builds `{name, description, roles:[ids], users:[ids]}`
     and requires a name + at least one role.
   - `UpdateUserGroup` fetches current, diffs `roles` (by name) and `users`
     (by name), calls the add/remove sub-endpoints (add before remove);
     rejects name/description changes.
   - `AddUserGroupRoles` / `RemoveUserGroupRoles` / `AddUserGroupUsers` /
     `RemoveUserGroupUsers`.
3. `cmd/usergroup.go` + `cmd/usergroup_prompt.go`:
   - `get` / `update` / `delete`: `--id` | `--name`, or an interactive picker
     when neither is given on a terminal.
   - `create` / `update`: `--from-file` **or piped stdin**; a JSON **array**
     runs a bulk operation and prints a per-item result table (non-zero exit on
     any failure).
   - `--interactive` / `-i`: create wizard (name, description, roles);
     update wizard (roles only - name/description are immutable).
4. Docs: `docs/documentation/usergroup.md` -> `group.md`; `README.md`;
   `user.md` / `role.md` / `permission.md` "See also" links; `make completions`.
