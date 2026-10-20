# CR-0045: Configurable user name and email patterns for new users

## CR Type

- [x] Enhancement to existing command

## Problem

The `user create --interactive` wizard suggests a user name
`first.last@<domain>`, where the domain is taken from the active profile's
login user name. The format is fixed, the email is not suggested at all, and
the suggestion can only be used by leaving the field empty.

Organizations use naming conventions per environment, for example
`jane.doe.dev@acme.com` for a user in the dev org and `jane.doe@acme.com` as
the email address.

## Proposed Solution

### Profile settings

```yaml
profiles:
  dev:
    username: admin@acme.com
    newUser:
      domain: acme.com                                        # default: domain of the profile username
      userNamePattern: "{firstName}.{lastName}.{profileName}@{domain}"  # default
      emailPattern: "{firstName}.{lastName}@{domain}"                   # default
```

Placeholders:

| Placeholder      | Value                                                   |
| ---------------- | ------------------------------------------------------- |
| `{firstName}`    | First name, lowercase, spaces removed                   |
| `{lastName}`     | Last name, lowercase, spaces removed                    |
| `{firstInitial}` | First letter of the first name, lowercase               |
| `{lastInitial}`  | First letter of the last name, lowercase                |
| `{profileName}`  | Name of the active profile                              |
| `{domain}`       | `newUser.domain`, or the domain of the profile username |

An unknown placeholder, or a pattern that needs a missing value (for example
`{domain}` without a domain), produces no suggestion and a note in the
wizard.

### Wizard

- User name and email fields show the generated value as a suggestion
  (ghost text).
- `→` (right arrow) or `Ctrl+E` completes the suggestion into the field for
  editing; submitting an empty field uses the suggestion.
- `profile show` lists the effective patterns and domain.

## Implementation

- `internal/config`: `Profile.NewUser` (`NewUserConfig`), `NewUserPatterns`
  (effective values with defaults), `ExpandUserPattern` with placeholder
  validation; tests.
- `internal/tui/userwizard.go`: pattern-based suggestions for user name and
  email, right-arrow accept key; tests.
- `cmd/user.go`: pass the effective patterns; `cmd/profile.go`: show them.
- Docs: `user.md`, `profile.md`, README config sample.

## Acceptance Criteria

- [x] Defaults produce `{firstName}.{lastName}.{profileName}@{domain}` and
      `{firstName}.{lastName}@{domain}` with the domain of the profile user
- [x] Custom patterns and domain from the profile are used
- [x] Unknown placeholders are reported, no suggestion is made
- [x] `→` / `Ctrl+E` completes; empty submit uses the suggestion
- [x] `profile show` lists the effective settings
- [x] `go build`, `go vet`, `golangci-lint`, `go test ./...` pass

## Implementation Notes

- huh's built-in input suggestions only complete text that was already
  typed, so the wizard wraps the input (`suggestField`): the generated value
  is the placeholder (gray), and `→` / `Ctrl+E` on an empty field fills it in.
  Typed prefixes still get the built-in completion.
- Verified in a pty on `dev`: placeholders after entering names, `→` fills
  user name and email, value editable afterwards; wizard canceled (no user
  created).

## Follow-up: existing user check (2026-09-30)

- The User name field of `user create --interactive` checks the effective
  name (typed or suggested) with `client.FindUserByName` (server-side
  `q=userName==`) when the field is left, and shows
  `user "<name>" already exists (ID <id>)`. Results are cached per name;
  lookup errors do not block (the create call reports conflicts).
- Verified on `dev`: existing login rejected with its ID, free name accepted.
