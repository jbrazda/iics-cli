# CR-0028: Secure Agent Installer Info and Download

## CR Type

- [x] Enhancement to existing command

## Problem

The `iics agent` command can manage running Secure Agents but provides no way to
obtain the installer needed to provision a new agent host. Operators must log in
to the IICS UI or hand-craft calls to the Platform REST API Version 2
`agentInstallerInfo` resource, then separately download and checksum-verify the
binary.

## Proposed Solution

Add two subcommands to `iics agent`:

### `iics agent installer-info --os <win64|linux64>`

Calls `GET /api/v2/agent/installerInfo/<platform>` and prints the response
(`downloadUrl`, `checksumDownloadUrl`, `installToken`).

- `--os` accepts `win64` or `linux64`. When omitted and the session is
  interactive, the value is prompted; otherwise the command errors.
- Default output is a vertical `FIELD`/`VALUE` table. `--output json|yaml|csv`
  renders the raw response.

### `iics agent installer-download`

Downloads the installer binary described by installer info.

Installer info resolution order:

1. `--installer-info <path>` JSON file.
2. Piped stdin JSON (e.g. from `installer-info --output json`).
3. Otherwise request installer info for the default or `--profile` account using
   `--os` (prompted when interactive).

- `--target <path>`: when it is an existing directory or ends with a path
  separator, the file name from the installer metadata is appended; otherwise it
  is the full destination path. Without `--target` the file goes to the system
  temp directory.
- `--verify`: download `checksumDownloadUrl`, compute the SHA-256 of the
  downloaded file, and fail (removing the partial download) when they do not
  match.
- Default output is a vertical `FIELD`/`VALUE` table describing the file. With
  `--verify` the expected/actual checksums and `verified` result are included.
  `--verbose` prints download and verification progress. `--output json|yaml|csv`
  renders the structured result.

## Implementation

- `internal/client/agents.go`: `AgentInstallerInfo` struct,
  `GetAgentInstallerInfo`, plus `DownloadFile` / `FetchText` helpers for the
  public CDN URLs (no session header).
- `internal/client/agents_test.go`: tests for the new methods.
- `cmd/agent_installer.go`: the two subcommands and helpers.
- `cmd/agent.go`: register the subcommands.
- Docs: `docs/documentation/agent.md`, `README.md` command table, completions.

## API Reference

Platform REST API Version 2, Secure Agents and services, `agentInstallerInfo`.
Sample response:

```json
{
  "@type": "agentInstallerInfo",
  "downloadUrl": "https://.../win64/agent64_install_ng_ext.6403.exe",
  "installToken": "PJ7NVrQ0...",
  "checksumDownloadUrl": "https://.../win64/agent64_install_ng_ext.6403_win64.sha256"
}
```
