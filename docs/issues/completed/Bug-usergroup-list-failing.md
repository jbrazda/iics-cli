# BUG: Error in command usergroup list

## Symptoms

Getting error when executing the usergroup list command

```shell
./iics usergroup list
```

## Likely affected files

- internal/client/users.go

## Error Message

```text
Error: parsing response: json: cannot unmarshal object into Go struct field UserGroup.roles of type string
```

## Fix Instructions

- Review the [API documentation ] for User listing
- Update the Command implementation to match the payload of the API
- Update Documentation
- Commit Changes

## Fix

Changed `UserGroup.Roles` from `[]string` to `[]UserRole` in `internal/client/usergroups.go`.
The IICS API returns roles as an array of objects, not strings.
Added `internal/client/usergroups_test.go` to cover the fixed behaviour.
