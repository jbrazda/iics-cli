# CR-0033: Rename `runtime` command to `environment`; add `environment delete`

## CR Type

- [x] Enhancement to existing command
- [x] New subcommand

## Problem

The command that manages IICS runtime environments / Secure Agent groups is
called `runtime`, which reads as "runtime configuration" rather than "the
environment resource". There is also no way to delete a runtime environment from
the CLI, even though `DELETE /api/v2/runtimeEnvironment/<id>` works.

## Proposed Solution

1. Rename the command to `environment`. Keep `runtime`, `rt`, and add `env` as
   aliases - fully backward compatible; every existing `iics runtime ...` and
   `iics rt ...` invocation keeps working.
2. Add `environment delete`:
   - `DELETE /api/v2/runtimeEnvironment/<id>` (verified live: returns 200).
   - `--id` or `--name` (mutually exclusive; `--name` resolves via
     `GetRuntimeEnvironmentByName`), `--yes` / `-y` to skip the confirmation.

## Implementation

- `internal/client/runtimes.go` - `DeleteRuntimeEnvironment(ctx, id)`.
- `internal/client/runtimes_test.go` - `TestDeleteRuntimeEnvironment`.
- `cmd/runtime.go` - `Use: "environment"`, `Aliases: ["runtime","rt","env"]`,
  `newRuntimeDeleteCmd` (reuses `resolveRuntimeID` from `cmd/runtime_configs.go`
  and the standard delete-confirmation pattern). Example strings updated to
  `iics environment ...`.
- Docs: `docs/documentation/runtime.md` renamed to `environment.md` and updated;
  `README.md` command table row; `docs/documentation/agent.md` "See also" link;
  `make completions`.

Internal Go identifiers and file names keep the `runtime` prefix (no functional
impact).
