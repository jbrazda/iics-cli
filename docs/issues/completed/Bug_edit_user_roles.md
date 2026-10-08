# BUG: removing a user's last direct role fails with V3API_IDSError_044

---

## Symptoms

Removing a role from a user fails with HTTP 400:

```text
V3API_IDSError_048: Failed to remove role(s) from the user 3PH7U8cFuoxbnZov19KWE0
  V3API_IDSError_044: The group must have at least one role.Selected operation
  will result in Group without any role
```

---

## Command / Reproduction Steps

Observed on the `dev` profile on 2026-09-30. The user had one direct role
(`CommandTask`) and was a member of the `Developer` user group, which also
grants `CommandTask`.

```bash
iics user update-roles --username test.user.dev@natl.com --remove CommandTask
```

The same happens in `iics user edit` when the last role is unchecked.

---

## Analysis

The error is about the user's direct roles, not the `Developer` group.
Once the user had a second direct role (`DummyTest`), removing `CommandTask`
succeeded, although `Developer` still grants it. Adding and then removing
`CustomerAPI` next to `DummyTest` also succeeded.

IICS appears to keep a user's direct roles in a hidden per-user group that
must keep at least one role, so `removeRoles` cannot leave a user with no
direct role, whatever the user's groups grant.

Assigning a role directly that a user group already grants has no effect on
access and only adds to the confusion.

---

## Fix

- `client.CheckKeepsDirectRole` rejects a change that would leave no direct
  role. `user update-roles` and `user edit` (`applyUserMembership`) call it
  before any API call; the edit wizard review cannot be applied in that case.
- `client.InheritedRoles` computes the roles granted by user groups.
  `user update-roles` skips additions of inherited roles (reported on stderr);
  the user wizard hides roles granted by the selected groups (except roles
  assigned directly now) and lists them in the field description.
- Docs: `docs/documentation/user.md` (`user create`, `user edit`,
  `user update-roles`, `user update-groups`).

---

## Verification

```bash
iics user update-roles --id 3PH7U8cFuoxbnZov19KWE0 --remove DummyTest
# Error: cannot remove every role assigned directly to test.user.dev@natl.com ...
iics user update-roles --id 3PH7U8cFuoxbnZov19KWE0 --add CommandTask
# Inherited from user group (skipped): CommandTask
```

The wizard role list was covered by unit tests only; it was not driven in a
terminal.

---

## Initial Analysis (superseded)

The report as first filed. Its explanation (removing a role inherited from a
group) did not hold up; see Analysis above.

```text
That error indicates you're not removing a role directly from the user. You're attempting to remove a role assignment that comes from a group membership, and the operation would leave that group with zero roles assigned. The IICS API is blocking the change.

Key error:

JSON
{
"code": "V3API_IDSError_044",
"message": "The group must have at least one role. Selected operation will result in Group without any role"
}
Show more lines

This means:

The user is a member of one or more groups.
The role you're removing is likely assigned through that group.
IICS enforces a rule that a group must always have at least one role.
Removing the last role from the group is not allowed.

The error is returned while processing the user request:

JSON
V3API_IDSError_048
Failed to remove role(s) from the user
``
Show more lines

but the underlying cause is the group validation failure. The platform exposes user, group, and role management through the same API area.

Things to check

Get the user's group memberships.

For each group, inspect the roles assigned to the group.

Determine whether the role you're removing is:

Directly assigned to the user, or
Inherited through a group.

If it's the only role on that group, you must either:

Assign another role to the group first, or
Remove the user from the group instead of removing the group's role.
```
