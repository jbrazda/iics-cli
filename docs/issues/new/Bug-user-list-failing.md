# BUG: Error in command user list

## Symptoms

Getting error when executing the user list command

```shell
./iics user list
```

## Likely affected files

- internal/client/users.go

## Error Message

```text
Error: parsing response: json: cannot unmarshal object into Go struct field User.groups of type string
```

## Fix Instructions

- Review the [API documentation ] for User listing
- update the Command implementation to match the payload of the API
