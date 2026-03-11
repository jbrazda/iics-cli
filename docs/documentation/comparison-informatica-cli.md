# Why `iics-cli` Supersedes the Informatica Asset Management CLI V2

The Informatica Intelligent Cloud Services (IICS) platform ships an official command-line utility
called the **IICS Asset Management CLI V2** (documented in Informatica KB article DOC-18245).
That utility has not received updates since its initial release and covers only a narrow slice of
what the IICS REST API can do. This document explains why `iics-cli` is a better choice for
automation, CI/CD integration, and day-to-day administrative work.

---

## The Official Tool Is No Longer Maintained

The Informatica-provided CLI V2 utility is abandoned. Its source repository
(`github.com/InformaticaCloudApplicationIntegration/Tools`) has received no significant updates,
and there is no indication that Informatica plans further development. Bugs, API changes, and new
IICS features are not reflected in it.

`iics-cli` is actively maintained and tracks the current IICS REST API v3 surface.

---

## POSIX Compliance - Exit Codes

The official tool does not return meaningful exit codes. A failed export or import returns exit
code `0` (success) regardless of outcome, making it impossible for shell scripts and CI/CD
pipelines to detect failures automatically:

```bash
# Official CLI - always exits 0, even on error
iics export -u user -p pass -r us -a "Project/Asset.Process"
echo $?   # 0, even if export failed
```

`iics-cli` follows POSIX conventions precisely:

| Exit code | Meaning                                  |
| --------- | ---------------------------------------- |
| `0`       | Success                                  |
| `1`       | Runtime or API error                     |
| `2`       | Usage error (bad flags, unknown command) |

```bash
iics export create --name my-export --project MyProject
echo $?   # 1 if the API returned an error, 2 if flags were wrong
```

This makes `iics-cli` suitable for use in `set -e` scripts and CI/CD pipelines that gate
deployments on command success.

---

## Pipeable, Machine-Readable Output

The official tool prints human-readable text with no structured output option. You cannot pipe
its output into `jq`, `grep`, or other standard UNIX tools reliably.

`iics-cli` supports three output formats via the global `--output` / `-o` flag:

```bash
# Table format (default) - human readable
iics user list

# JSON - pipe into jq, parse in scripts
iics user list -o json | jq '.[].userName'

# CSV - import into spreadsheets, pass to other tools
iics connection list -o csv > connections.csv
```

Every command supports all three formats, enabling standard UNIX pipeline composition:

```bash
# Find all schedules that are enabled and export their names
iics schedule list -o json | jq -r '.[] | select(.status == "ENABLED") | .name'

# Count objects in a project
iics objects list --project MyProject -o json | jq length
```

---

## Credentials on the Command Line vs. Secure Configuration

The official tool requires passing credentials as command-line flags on every invocation:

```bash
iics export -u user@company.com -p MyPassword123 -r us -a "..."
```

Credentials passed as arguments are visible in the system process list (`ps aux`), shell history,
and CI/CD build logs. This is a significant security risk.

`iics-cli` stores credentials in a YAML config file (`~/.iics/config.yaml`) and reads the
password from an environment variable at runtime:

```yaml
defaultProfile: prod
profiles:
  prod:
    region: US
    username: user@company.com
```

```bash
export IICS_PASSWORD=MyPassword123
iics export create --name nightly-backup
```

Credentials never appear on the command line. The config file supports file-system permissions
(`chmod 600`) for additional protection.

---

## Named Profiles for Multiple Environments

The official tool has no concept of environments or profiles. Switching between a development org
and a production org requires changing every `-u`, `-p`, and `-r` argument on every command.

`iics-cli` uses named profiles:

```yaml
defaultProfile: dev
profiles:
  dev:
    region: US
    username: dev-user@company.com
  prod:
    region: US
    username: prod-admin@company.com
  emea:
    region: EMEA
    username: emea-admin@company.com
```

Switch environments with a single flag or environment variable:

```bash
iics user list --profile dev
iics user list --profile prod
IICS_PROFILE=emea iics export create --name backup
```

This makes it straightforward to write scripts that target multiple orgs without maintaining
separate credential sets or wrapper scripts.

---

## Session Caching

The official tool authenticates on every command, adding latency to each invocation and
increasing the risk of account lockout during automated runs with many commands.

`iics-cli` caches the session token in `~/.iics/sessions.yaml` with a 30-minute TTL. A
sequence of commands reuses the same authenticated session:

```bash
iics login                          # authenticates once
iics user list                      # reuses session
iics role list                      # reuses session
iics connection list --profile prod # separate cached session per profile
```

---

## Much Broader API Coverage

The official tool covers only five operations: export, import, publish, list assets, and version.
It cannot manage users, connections, schedules, roles, or any other administrative resource.

`iics-cli` covers the full IICS REST API v3 surface:

| Resource        | Operations                        |
| --------------- | --------------------------------- |
| `objects`       | list, dependencies                |
| `lookup`        | resolve IDs, names, paths         |
| `connection`    | list, get, create, update, delete |
| `export`        | create, status, download          |
| `import`        | upload, start, status, log        |
| `schedule`      | list, get, create, update, delete |
| `project`       | create, update, delete            |
| `folder`        | create, update, delete            |
| `user`          | list, get, create, update, delete |
| `usergroup`     | list, get, create, update, delete |
| `role`          | list, get, create, update, delete |
| `privilege`     | list                              |
| `runtime`       | list, get, create, update         |
| `agent`         | list, start, stop                 |
| `tag`           | assign, remove                    |
| `permission`    | get, set, delete                  |
| `securitylog`   | list                              |
| `metering`      | get, download                     |
| `sourcecontrol` | checkout, checkin, pull, commit   |
| `state`         | fetch, load                       |

---

## Comprehensive Region Support

The official tool supports only three region values: `us`, `eu`, and `ap`. IICS has many more
regional pod deployments (USW1, USW1-1, USW1-2, USE2, USW3, USW3-1, USE4, USW5, USE6, CAC1,
APSE1, APJ, APNE1, EMEA, EMWE1, and others).

`iics-cli` includes a built-in pod registry covering all known IICS regions and their login
endpoints, and also accepts a `loginUrl` override for custom or future pods.

---

## Better Error Reporting

The official tool provides minimal error output. When an API call fails you typically receive
a generic message with little context.

`iics-cli` reports structured API errors to `stderr`, including the HTTP status code, the
Informatica error code, and the full response body when `--debug` is set:

```
Error: API error 400: invalid artifact type 'PROCESS_OBJECT'
```

With `--debug`:

```
Error: API error 400: invalid artifact type 'PROCESS_OBJECT'
  HTTP 400 Bad Request
  {
    "error": {
      "code": "ICS-012345",
      "message": "invalid artifact type 'PROCESS_OBJECT'"
    }
  }
```

Errors always go to `stderr`, leaving `stdout` clean for piping.

## Structured Subcommand Hierarchy

The official tool uses a flat command model: one binary invocation per operation with a mix of
short flags. There is no discoverability - you must read the documentation to know what the tool
can do.

`iics-cli` uses a `resource verb` hierarchy that follows the same convention as `kubectl`, `gh`,
and `aws`:

```bash
iics user list
iics user get --id abc123
iics connection create --from-file conn.json
iics export create --name my-export --project Prod
```

Built-in help is available at every level:

```bash
iics --help
iics user --help
iics export create --help
```

---

## Better Documentation

The official tool is documented only through a single Informatica KB article (DOC-18245) that
provides a high-level overview without complete flag references or examples.

`iics-cli` ships with per-command documentation in `docs/documentation/`, covering flags,
examples, configuration, and common workflows for every command group.

---

## Summary

| Capability                          | Informatica CLI V2 | `iics-cli` |
| ----------------------------------- | ------------------ | ---------- |
| Maintained and updated              | No                 | Yes        |
| POSIX exit codes                    | No                 | Yes        |
| Machine-readable output (JSON/CSV)  | No                 | Yes        |
| Pipe-friendly stdout / stderr split | No                 | Yes        |
| Credentials in config file          | No                 | Yes        |
| Named environment profiles          | No                 | Yes        |
| Session caching                     | No                 | Yes        |
| User and group management           | No                 | Yes        |
| Role and privilege management       | No                 | Yes        |
| Connection CRUD                     | No                 | Yes        |
| Schedule CRUD                       | No                 | Yes        |
| Agent and runtime management        | No                 | Yes        |
| Tag and permission management       | No                 | Yes        |
| Security log access                 | No                 | Yes        |
| Metering and usage data             | No                 | Yes        |
| Source control operations           | No                 | Yes        |
| All IICS regions supported          | No                 | Yes        |
| Structured subcommand hierarchy     | No                 | Yes        |
| Built-in contextual help            | Minimal            | Yes        |
| Per-command reference documentation | No                 | Yes        |
