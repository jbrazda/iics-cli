# CR-0022: Release plan target credential resolution

## Summary

`iics release plan` must support target credential resolution in two modes:

1. Local usage with configured CLI profiles.
2. CI usage where profiles may not exist and credentials come from environment variables.

The command must support explicit target profile selection via CLI flags, env-based
target profile selection, and clear errors when an explicitly requested profile is
not found.

## Problem statement

Current release planning assumes target environment names map directly to configured
profiles. This fails in CI agents where `~/.iics/config.yaml` may be absent, and
where credentials are provided as pipeline variables such as:

- `IICS_USER_TST`, `IICS_PWD_TST`
- `IICS_USER_QA`, `IICS_PWD_QA`
- `IICS_USER_STAGE`, `IICS_PWD_STAGE`
- `IICS_USER_PROD`, `IICS_PWD_PROD`

We need deterministic resolution rules for profile selection and credential source.

## Goals

1. Allow explicit target profile mapping from CLI for release planning.
2. Allow target profile mapping from env vars.
3. Support CI credential-only mode without config profiles.
4. Fail fast with actionable errors when explicit mappings are invalid.
5. Preserve existing behavior as fallback when no new options are provided.

## Non-goals

1. No changes to non-release commands.
2. No changes to profile file schema.
3. No secret persistence to disk.

## Proposed behavior

### 1. Target to profile resolution

For each target in `manifest.options.targets`, resolve profile name using:

1. `--target-profile-map` (highest precedence), format `TST=tstprof,QA=qaprof`.
2. `IICS_TARGET_PROFILE_MAP` env var, same format.
3. implicit target name match (case-insensitive), such as `TST` -> profile `tst`.

If a profile is explicitly provided by CLI or env map and not found, return:

- `target profile "<name>" not found in config for target "<TARGET>"`

### 2. Credential source resolution per target

After target to profile resolution:

1. If resolved profile exists in config, use `ResolveTargetProfile(...)`.
2. If not found, try CI-style env credentials for that target key:
   - `IICS_USER_<TARGET>`
   - `IICS_PWD_<TARGET>`
   - optional: `IICS_REGION_<TARGET>` or `IICS_LOGIN_URL_<TARGET>`
3. If neither is available, fail with guidance listing accepted env names.

### 3. Backward compatibility

If no new map flag/env is set, current target-name matching behavior remains.

## CLI and env additions

### New flag

- `iics release plan --target-profile-map`

### New env var

- `IICS_TARGET_PROFILE_MAP`

### CI credential env vars (target-scoped)

- `IICS_USER_<TARGET>`
- `IICS_PWD_<TARGET>`
- `IICS_REGION_<TARGET>` (optional)
- `IICS_LOGIN_URL_<TARGET>` (optional)

## Implementation plan

1. Add target profile map parser in `internal/release`:
   - parse `TARGET=profile` pairs
   - normalize target keys to uppercase
   - reject malformed input
2. Add target credential resolver in `internal/release`:
   - resolve profile name by precedence
   - if profile missing, build ephemeral target credentials from env
3. Update `cmd/release.go` (`release plan`):
   - add `--target-profile-map`
   - route per-target existence checks through new resolver
4. Add tests:
   - map parsing and precedence
   - explicit missing profile error path
   - CI env-only credential resolution path
5. Update docs:
   - `docs/documentation/release.md`
   - `README.md` env var table and examples
6. Regenerate completions.

## Acceptance criteria

1. `release plan` works locally with profile-based targets.
2. `release plan` works in CI with no profiles when `IICS_USER_<TARGET>` and
   `IICS_PWD_<TARGET>` are present.
3. `--target-profile-map` overrides implicit profile matching.
4. `IICS_TARGET_PROFILE_MAP` works when flag is not set.
5. Explicitly mapped missing profile produces a clear error.
6. Existing workflows without new settings continue to work.

