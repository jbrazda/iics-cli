# BUG: permission commands do not match the v3 object permissions API

---

## Symptoms

`iics permission get`, `permission set` and `permission delete` use request
and response structs that do not match the
[Object permissions](https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/object-permissions.html)
API. `permission set` calls an endpoint that does not exist.

---

## Command / Reproduction Steps

```bash
iics permission get --object-id <object-id> --verbose
iics permission set --object-id <object-id> --from-file perms.json --verbose
```

---

## Expected Behaviour

Per the documentation:

| Operation | Method and URI |
| --------- | -------------- |
| List ACLs of an object | `GET /public/core/v3/objects/<object ID>/permissions` |
| Get one ACL | `GET /public/core/v3/objects/<object ID>/permissions/<ACL ID>` |
| Create an ACL (one principal) | `POST /public/core/v3/objects/<object ID>/permissions` |
| Update an ACL | `PUT /public/core/v3/objects/<object ID>/permissions/<ACL ID>` |
| Delete one ACL | `DELETE /public/core/v3/objects/<object ID>/permissions/<ACL ID>` |
| Delete all ACLs | `DELETE /public/core/v3/objects/<object ID>/permissions` |
| Check own access | `GET /public/core/v3/objects/<object ID>/permissions/checkAccess[?type=<asset type>]` |

`GET` returns an array of ACLs:

```json
[
    {
        "id": "4D6ER3yic8cjjE1GmxEKEi",
        "principal": {
            "type": "USER",
            "name": "saki@infa.com"
        },
        "permissions": {
            "read": true,
            "update": true,
            "delete": true,
            "execute": true,
            "changePermission": true
        }
    }
]
```

POST and PUT bodies use the same `principal` (`type` is `USER` or `GROUP`,
`name`) and `permissions` objects.

---

## Actual Behaviour

`internal/client/permissions.go` models the response as a single
`ObjectPermission` object with `objectId`, `objectType` and a
`permissions` array of `{principalId, principalType, principalName,
permission}` entries, which does not match the API. `SetObjectPermissions`
sends `PUT /objects/<id>/permissions`, which the API does not define.

---

## Architecture Layer

- [x] **`cmd/`** - flag parsing, command wiring, output formatting
- [x] **`internal/client/`** - HTTP logic, API structs, request/response handling
- [ ] **`internal/config/`** - config file loading, session cache
- [ ] **`internal/output/`** - table / JSON / CSV renderer

---

## Likely Affected Files

```text
internal/client/permissions.go
cmd/permission.go
docs/documentation/permission.md
```

---

## Fix Instructions

1. Replace the structs with `ObjectACL {id, principal {type, name},
   permissions {read, update, delete, execute, changePermission}}`.
2. Client methods: `ListObjectACLs`, `GetObjectACL`, `CreateObjectACL`,
   `UpdateObjectACL`, `DeleteObjectACL`, `DeleteAllObjectACLs`,
   `CheckObjectAccess`.
3. Commands: `permission get` (table: principal type, name, R/U/D/X/P),
   `permission add`, `permission update`, `permission delete` (`--acl-id` or
   `--all`), `permission check`. Keep `permission set --from-file` as a
   declarative apply (diff file against current ACLs; create/update/delete).
4. Verify against a live object on `dev` before committing.
5. Tests with `newTestClient` for every method; update docs and completions.

The interactive editor for permissions is tracked separately in CR-0042 and
builds on this fix.

---

## Fix (filled in after resolution)

**Root cause:** the client modeled a response shape and a `PUT
/objects/<id>/permissions` endpoint that the API does not have.

**Files changed:**

```text
internal/client/permissions.go      - ObjectACL/ACLPrincipal/ACLPermissions, ObjectAccess; List/Get/Create/
                                      Update/Delete/DeleteAll ObjectACL(s), CheckObjectAccess; DiffACLs,
                                      ParseACLPermissions, PrincipalKey
internal/client/permissions_test.go - tests for every method, DiffACLs and permission parsing
cmd/permission.go                   - get, add, update, delete (--acl-id/--user/--group/--all), set
                                      (--from-file, --prune, --dry-run), check; --object-id or --path/--type
docs/documentation/permission.md    - rewritten
```

**Verified on `dev`** (throwaway project): get on an empty object, add,
duplicate add rejected, update, check, set dry-run/apply/no-change,
prune (declined and confirmed), delete by group, delete --all, and a
`get -o json | set --from-file -` round-trip.
