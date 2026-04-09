---
id: BUG-0005
title: user create --interactive fails with HTTP 400 - limit parameter exceeds API max
status: pending
priority: high
affects: cmd/user.go
---

# BUG-0005: user create --interactive fails with HTTP 400 - limit exceeds API max

## Summary

Running `iics user create --interactive` fails immediately after the group/role
selection step with HTTP 400 `V3API_IDSError_003: Invalid value in limit parameter:
must be between 0 and 200`. The `runUserWizard`, `buildGroupMap`, and `buildRoleMap`
functions pass `Limit: 1000` to `ListUserGroups` and `ListRoles`, which exceeds the
API maximum of 200.

## Observed Behavior

```
Force password change on next login? [y/N]: n
Error: IICS API error (HTTP 400):
  "code": "V3API_IDSError_003",
  "message": "Invalid value in limit parameter: must be between 0 and 200"
```

## Root Cause

Three call sites in `cmd/user.go` use `Limit: 1000`:

- `runUserWizard` - `c.ListUserGroups(..., Limit: 1000)`
- `runUserWizard` - `c.ListRoles(..., Limit: 1000)`
- `buildGroupMap`  - `c.ListUserGroups(..., Limit: 1000)`
- `buildRoleMap`   - `c.ListRoles(..., Limit: 1000)`

## Fix

Change all four call sites to `Limit: 200` and add pagination loops in
`buildGroupMap` and `buildRoleMap` to retrieve more than 200 items when needed.
