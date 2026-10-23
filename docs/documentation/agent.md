# agent

Manage IICS Secure Agents. Agents are on-premises processes that execute data integration tasks.

## Synopsis

```bash
iics agent <subcommand> [flags]
```

## Subcommands

| Subcommand           | Description                                                     |
| -------------------- | --------------------------------------------------------------- |
| `list`               | List Secure Agents                                              |
| `get`                | Get a single agent                                              |
| `details`            | Get agent service engine details                                |
| `delete`             | Delete a Secure Agent                                           |
| `start`              | Start an agent service                                          |
| `stop`               | Stop an agent service                                           |
| `installer-info`     | Get Secure Agent installer download information                 |
| `installer-download` | Download the Secure Agent installer (optional checksum verify)  |
| `configs get`        | Show an agent's service properties (experimental)               |
| `configs set`        | Change or reset an agent's service properties (experimental)    |
| `configs edit`       | Edit an agent's service properties interactively (experimental) |
| `rename`             | Rename an agent, for example from host name to `CAI-A03`        |

---

## agent list

### Flags

| Flag           | Type         | Default | Description                                                                                       |
| -------------- | ------------ | ------- | ------------------------------------------------------------------------------------------------- |
| `--unassigned` | bool         | false   | Include only agents not in a group                                                                |
| `--basic-info` | bool         | false   | Include package and configuration details in the response                                         |
| `--fields`     | string       |         | Comma-separated columns to display, by technical field name                                       |
| `--filter`     | string array |         | Client-side filter, e.g. `agentHost==host01` or `active!=true`. Repeatable; conditions are AND-ed |
| `--limit`      | int          | 200     | Accepted for compatibility; the API currently ignores it                                          |
| `--skip`       | int          | 0       | Accepted for compatibility; the API currently ignores it                                          |

All [global flags](../../README.md#global-flags) apply.

### Filtering

`--filter` matches on the raw API field names (the same names accepted by
`--fields`). Only `==` and `!=` are supported. String comparison is
case-insensitive. Booleans and numbers are compared by value. Dot notation
selects nested fields. A boolean field with a `false` value is omitted from the
API response, so `field==false` will not match; filter on the value that is
present (for example `active==true`).

### Output columns

Default columns: `id`, `name`, `agentHost`, `active`, `readyToRun`, `platform`,
`agentVersion`, `agentGroupId`.

`--fields` accepts any of: `id`, `orgId`, `name`, `description`, `agentHost`,
`active`, `readyToRun`, `platform`, `agentVersion`, `upgradeStatus`,
`agentGroupId`, `federatedId`, `serverUrl`, `spiUrl`, `proxyHost`, `createdBy`,
`updatedBy`, `createTime`, `updateTime`, `createTimeUTC`, `updateTimeUTC`,
`lastStatusChange`, `lastUpgraded`, `lastUpgradeCheck`, `configUpdateTime`.
Unknown names are ignored. `--output json` / `yaml` always emit the full agent
record regardless of `--fields`.

### Examples

```bash
iics agent list

iics agent list --fields name,agentHost,agentVersion,configUpdateTime

# Client-side filtering on technical field names
iics agent list --filter agentHost==devinfacld01
iics agent list --filter active==true --filter platform==win64

iics agent list --unassigned

iics agent list --output json
```

```powershell
iics agent list

iics agent list --fields name,agentHost,agentVersion,configUpdateTime

iics agent list --filter agentHost==devinfacld01
iics agent list --filter active==true --filter platform==win64

iics agent list --unassigned
```

---

## agent get

Get a single Secure Agent by ID or name.

### Flags

| Flag     | Type   | Description                                       |
| -------- | ------ | ------------------------------------------------- |
| `--id`   | string | Agent ID                                          |
| `--name` | string | Agent name (uses `GET /api/v2/agent/name/<name>`) |

Exactly one of `--id` / `--name` is required; they are mutually exclusive.

All [global flags](../../README.md#global-flags) apply.

### Interactive mode

With `--interactive` (`-i`), or without an agent flag in a terminal, the
command lists the agents (name, host, ID; `/` filters) and then asks for the
view:

| View                                   | Same as                                  |
| -------------------------------------- | ---------------------------------------- |
| Services summary                       | `agent details --services`               |
| Details with service status            | `agent details`                          |
| Full details with configuration        | `agent details --full`                   |
| Service properties with defaults       | `agent configs get`                      |

With an agent flag, `-i` only asks for the view; `--services` or `--full`
skip the view question. The main menu entry **Show agent details** runs
`agent details -i`.

### Output

In table mode the agent is printed as a vertical `PROPERTY` / `VALUE` listing.
`--output json`, `yaml`, and `csv` render the agent record directly.

### Examples

```bash
iics agent get --id <agent-id>

iics agent get --name "My Agent"

iics agent get --id <agent-id> --output json
```

```powershell
iics agent get --id <agent-id>

iics agent get --name "My Agent"

iics agent get --id <agent-id> --output json
```

---

## agent details

Get the service engine details for an agent: the agent summary, then each
service (engine) with its status and, optionally, its configuration properties.

### Flags

| Flag                  | Type   | Description                                                                                         |
| --------------------- | ------ | --------------------------------------------------------------------------------------------------- |
| `--id`                | string | Agent ID                                                                                            |
| `--fid`               | string | Agent federated ID                                                                                  |
| `--name`              | string | Agent name                                                                                          |
| `--hostname`          | string | Agent host name                                                                                     |
| `--full`              | bool   | Include agent-level and per-service configuration properties (adds `onlyStatus=false`)              |
| `--services`          | bool   | Show only the agent summary and one horizontal table of services (mutually exclusive with `--full`) |
| `--interactive`, `-i` | bool   | Pick the agent (when no agent flag is given) and the view from lists                                |

One of `--id` / `--fid` / `--name` / `--hostname` is required, unless the
command runs interactively; they are mutually exclusive. Only `--id` maps to the API directly; `--name` uses the
by-name endpoint, and `--fid` / `--hostname` are resolved to an ID by scanning
the agent list (with a runtime-environment fallback for `--fid`).

All [global flags](../../README.md#global-flags) apply.

### Output

Table mode prints, in order:

1. `Agent:` - a vertical `PROPERTY` / `VALUE` listing of the agent summary.
2. `Agent Config:` (only with `--full`, when present) - a table of agent-level
   configuration properties.
3. For each service: a `Service: <display name> (<appname> v<version>)` header,
   a vertical status listing, and (with `--full`) a configuration table.

Configuration tables have columns `TYPE`, `NAME`, `VALUE`, `DEFAULT`,
`OVERRIDDEN`. Long `VALUE` / `DEFAULT` cells are wrapped onto multiple lines.
`OVERRIDDEN` follows [`agent configs get`](#agent-configs-get): `custom` for a
custom property (Custom Configuration Details), `yes` for a property whose
value differs from its default (ignoring line endings and whitespace), empty
otherwise. The API's own `customized` flag marks custom properties only, so it
is not used for overrides.

With `--services`, only the `Agent:` summary and a single `Services (N):` table
are printed, one row per service with columns `SERVICE`, `APP NAME`, `VERSION`,
`STATUS`, `DESIRED`, `SUBSTATE`, `REPLACE`, `UPDATED`.

`--output json` / `yaml` render the full nested details document. Each
configuration property also has a computed `overridden` field, next to the
API's `customized` field.

### Examples

```bash
iics agent details --id <agent-id>

iics agent details --id <agent-id> --services

iics agent details --hostname devinfacld01 --full

iics agent details --name "My Agent" --output json

# Pick the agent and the view from lists
iics agent details -i
```

```powershell
iics agent details --id <agent-id>

iics agent details --id <agent-id> --services

iics agent details --hostname devinfacld01 --full
```

---

## agent configs

Show and change the service properties of **one agent**, for example the
Process Server cluster settings (`primary-node`, `load-balance-url`), its
database URLs and connection properties, or JVM heap sizes. The Secure Agent
group equivalent is [environment configs](environment.md#environment-configs).

The commands work on whatever services the agent runs: Process Server and
OI Data Collector on a CAI agent, or Data Integration Server, Mass Ingestion,
Database Ingestion, CMI Streaming Agent, IDMC Data Gateway, Metadata services
and others on a CDI agent. Each agent's own form lists its properties.

> **Experimental.** The public REST API only configures properties for a whole
> Secure Agent group. Agent-level properties are read and saved through the
> Administrator UI form `/runtime/app/agent.do`, which is undocumented and may
> change without notice.

How it works:

- The CLI session is sent as the `USER_SESSION` cookie. The form's CSRF check
  compares the `XSRF_TOKEN` cookie with the `csrfToken` field, so the CLI sends
  a random token in both.
- `set` reads the whole form, changes only the requested properties and submits
  every field back, as the UI's Save button does. It then reads the form again
  and fails if a change was not stored.
- How a service picks up new values depends on the service. Process Server
  does **not** restart by itself: it keeps running with the old values and the
  Administrator UI shows that it needs a restart. Restart it with
  [agent restart](#agent-restart). `set` lists the changed services and the
  restart commands after saving.

### Property names

Properties are named `service:type:name`:

| Part      | Values                                                                                                                           |
| --------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `service` | Service name without its version (`process-engine`, `OpsInsightsDataCollector`), the display name (`Process Server`), or `agent` |
| `type`    | Property type, for example `server`, `cluster`, `jvm`, `db`, `logging`, `TOMCAT_CFG`, or the UI label (`Tomcat`, `DTM`)          |
| `name`    | Property name, for example `primary-node`                                                                                        |

Matching is case-insensitive. Use `type/subType` (for example
`AGENTCORE_LOG4J/DEBUG`) when a name exists for both the `INFO` and `DEBUG`
subtypes. `agent configs get` shows the names to use.

`agent` stands for the agent's own Agent Core, Agent Manager and Tomcat
properties (a legacy section of the form). The Administrator UI never shows
them: its Service list holds only the agent's services. They are not editable;
the server ignores changes to them, so `set` refuses them. `get` hides them
unless `--include-hidden` or `--service agent` is given. On CDI agents the
editable Tomcat and JVM settings are under the Data Integration Server
service, for example `Data_Integration_Server:TOMCAT_JRE:INFA_MEMORY`.

### Custom properties

Services also take **custom properties** (Custom Configuration Details in the
Administrator UI): properties that are not in the service's standard list,
for example `maxDTMProcesses` or `maxUrlLength` for the Data Integration
Server Tomcat, or `INFA_CODEPAGENAME` for its JRE. Each service offers its own
types, for example:

| Service                 | Custom property types                                                                                                          |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| Data Integration Server | `TOMCAT_CFG` (Tomcat), `TOMCAT_JRE` (Tomcat JRE), `TOMCAT_LOG4J`, `PMRDTM_CFG` (DTM), `PLATFORM_CFG`, `SYSTEM_CFG`, `IDQ`, ... |
| Mass Ingestion          | `AGENT_RUNTIME_SETTINGS`, `PGP_SETTINGS`, `JVM_SETTINGS`                                                                       |
| Process Server          | `server`, `cluster`, `jvm`, `connectors`, `db`, `logging`                                                                      |

The types come from the agent's own form, so they always match the services
the agent runs. A wrong type is rejected with the list of valid types.

- `get` lists custom properties with the standard ones, marked `custom` in the
  `OVERRIDDEN` column (`"custom": true` in JSON). `--overridden` includes them.
- `set --add service:type:name=value` adds one; `set --set` changes its value;
  `set --remove service:type:name` (or `--reset`) removes it.
- Names can have up to 255 characters, values up to 512, on one line.
- `agent configs edit` lists them with a `+` mark and offers
  **+ Add custom property** for each service.
- `get --export` includes them. When applying such a file to another agent,
  `--add-missing` adds the custom properties that agent does not have yet.

### agent configs get

Show an agent's properties with their defaults and whether they are overridden.

| Flag               | Type   | Description                                                                       |
| ------------------ | ------ | --------------------------------------------------------------------------------- |
| `--id`             | string | Agent ID                                                                          |
| `--fid`            | string | Agent federated ID                                                                |
| `--name`           | string | Agent name                                                                        |
| `--hostname`       | string | Agent host name                                                                   |
| `--service`        | string | Only this service: name, display name, or `agent`                                 |
| `--type`           | string | Only this property type, optionally `type/subType`                                |
| `--search`         | string | Regular expression on the property name (case-insensitive)                        |
| `--overridden`     | bool   | Only properties whose value is overridden (differs from the default)              |
| `--export`         | bool   | Write the properties as a `--from-file` change file: YAML, or JSON with `-o json` |
| `--include-hidden` | bool   | Also show the agent's own (`agent`) properties, which the UI hides                |

Exactly one of `--id` / `--fid` / `--name` / `--hostname` is required.

`--export` writes the filtered properties as a nested service / type / name
map, the format `agent configs set --from-file` reads. Values that contain
quotes are written as `"'4G'"` and multi-line values as YAML block scalars, so
the file reads back unchanged. `agent` properties are left out because they
cannot be set.

Table mode prints the layout of `agent details --full`: an `Agent: <name>
(<id>)` line, then for each service a `Service: <display name> (<service>)`
header and a table with columns `TYPE` (with `/subType` for `agent`
properties), `NAME`, `VALUE`, `DEFAULT`, `OVERRIDDEN`. Long values are wrapped.

A value that differs from its default only in line endings, trailing
whitespace on a line, or leading and trailing blank lines is shown as not
overridden, even when the server flags it as overridden. Indentation counts,
as it is significant in YAML values.

`--output csv` prints one flat table with an extra `SERVICE` column.
`--output json` / `yaml` list the properties with `service`, `serviceName`,
`type`, `subType`, `name`, `value`, `defaultValue`, `overridden`,
`sensitive` and `custom`.

```text
Agent: CAI-A03 (01D23X0800000000008G)

Service: Process Server (process-engine)

  TYPE      NAME           VALUE     DEFAULT   OVERRIDDEN
  cluster   enabled        true      false     yes
  cluster   primary-node   false     true      yes
  jvm       max-heap       '10G'     '1536M'   yes
```

```bash
iics agent configs get --name CAI-A03 --overridden
iics agent configs get --name CAI-A03 --service "Process Server" --type cluster
iics agent configs get --hostname myhost --service process-engine --search '^db|url' -o json
iics agent configs get --name CDI-G01-A01 --service "Data Integration Server" --type TOMCAT_JRE

# Save the Process Server overrides of one agent as a change file
iics agent configs get --name CAI-A01 --service process-engine --overridden --export > pe.yaml
```

### agent configs set

Change or reset properties of one agent. The changes are shown as a
`PROPERTY` / `CURRENT` / `NEW` table (values that are not overridden are
marked `(default)`, sensitive values are shown as `***`) and applied after
confirmation.

| Flag            | Type   | Description                                                                              |
| --------------- | ------ | ---------------------------------------------------------------------------------------- |
| `--id`          | string | Agent ID                                                                                 |
| `--fid`         | string | Agent federated ID                                                                       |
| `--name`        | string | Agent name                                                                               |
| `--hostname`    | string | Agent host name                                                                          |
| `--set`         | string | `service:type:name=value`; a value `@file` reads the value from a file. Repeatable       |
| `--reset`       | string | `service:type:name` to reset to the default; removes a custom property. Repeatable       |
| `--add`         | string | `service:type:name=value` to add a custom property (`@file` reads the value). Repeatable |
| `--remove`      | string | `service:type:name` of a custom property to remove. Repeatable                           |
| `--add-missing` | bool   | Add properties from `--set` or `--from-file` that do not exist, as custom properties     |
| `--reset-all`   | bool   | Reset every overridden standard property to its default, like the UI's **Reset All**     |
| `--service`     | string | With `--reset-all`: only this service                                                    |
| `--from-file`   | string | JSON or YAML change file (see below); only the properties in the file change             |
| `--dry-run`     | bool   | Show the changes without applying them                                                   |
| `--yes`, `-y`   | bool   | Apply without asking for confirmation                                                    |
| `--force`       | bool   | Change the agent even if its Secure Agent group has group-level properties               |

Rules:

- Values are stored exactly as given. Process Server values keep their YAML
  quoting, for example `'4G'` or `'jdbc:postgresql://host:5432/activevos'`, as
  `get` shows them. Multi-line values (connection properties, logging) are
  easiest to pass with `@file`; the file content is used as is.
- `--reset-all` works like the Administrator UI's **Reset All** button: every
  overridden standard property of every service (or of `--service`) goes back
  to its default. Custom properties are kept. Sensitive properties are left
  unchanged (the UI disables Reset All for them) and reported; reset them one
  by one with `--reset`. It can be combined with other changes, which apply
  on top. Check the list with `--dry-run` first.
- Setting a property to its current value changes nothing, so a default does
  not turn into an override. Use `--reset` to go back to the default.
  Values are compared ignoring line endings, trailing whitespace on a line,
  and leading and trailing blank lines, so a YAML file with an extra trailing
  newline is not a change.
- Setting a property to its default value resets it, and resetting a
  property that is not overridden changes nothing.
- A property can be changed only once per command.
- If the agent's Secure Agent group has group-level properties, the group's
  settings replace the agent's when the agent is added to or restarted in the
  group, so `set` stops with an error. Remove the group-level properties first
  (`iics environment configs`), or use `--force`.
- Errors reported by the form, such as an invalid value, are returned as the
  command error and nothing is changed.

### Change files

`--from-file` reads JSON or YAML in one of two shapes. Only the properties in
the file change; all others keep their values. `--set` and `--reset` can be
combined with a file.

A nested service / type / name map, as written by `get --export`. A `null`
value resets the property to its default:

```yaml
process-engine:
  cluster:
    primary-node: "true"
  jvm:
    max-heap: "'4G'"
    min-heap: null
  db:
    connection-properties: |
      - key: timeBetweenEvictionRunsMillis
        value: 300000
      - key: testOnBorrow
        value: true
```

```json
{
  "Data_Integration_Server": {
    "TOMCAT_JRE": { "INFA_MEMORY": "'-Xms512m -Xmx4096m'" }
  }
}
```

A list of changes:

```json
[
  { "property": "process-engine:cluster:primary-node", "value": "false" },
  { "property": "process-engine:jvm:max-heap", "value": "'4G'" },
  { "property": "process-engine:jvm:min-heap", "reset": true },
  { "property": "Data_Integration_Server:TOMCAT_CFG:maxDTMProcesses", "value": "5", "add": true },
  { "property": "Data_Integration_Server:PMRDTM_CFG:MatchQuotesPastEndOfLine", "remove": true }
]
```

List items take `property` and one of `value`, `reset: true`, `add: true`
(with `value`, optionally `sensitive: true`) or `remove: true`. In the nested
map, `null` resets a standard property or removes a custom one; properties
that do not exist are rejected unless `--add-missing` is given.

Notes on values:

- Values are used as written. YAML removes one level of quotes, so a
  Process Server value `'4G'` must be written as `"'4G'"` in YAML, the same
  as in JSON. Unquoted `true` or `512` become the strings `true` and
  `512`.
- In YAML and JSON double-quoted strings a backslash must be doubled
  (`C:\\Windows`).
- Service, type and name are matched like `--set` names, so the display name
  (`"Process Server"`) also works as the service key.

### After saving

`set` prints the services whose properties changed, with the command to
restart each one, for example:

```text
Changed services: Process Server. Services such as Process Server use the new values only after a restart:
  iics agent restart --id 01D23X0800000000008G --service "Process Server" --blocking
```

Process Server keeps running with the old values until it is restarted.

### Examples for set

```bash
# Preview a change
iics agent configs set --name CAI-A03 --set "process-engine:cluster:primary-node=false" --dry-run

# Make the agent the cluster primary node
iics agent configs set --name CAI-A03 --set "process-engine:cluster:primary-node=true"

# Point Process Server at another database, connection properties from a file
iics agent configs set --name CAI-A03 \
  --set "process-engine:db:url='jdbc:postgresql://dbhost:5432/activevos'" \
  --set "process-engine:db:connection-properties=@db-props.yaml" --yes

# Back to the default heap size
iics agent configs set --name CAI-A03 --reset process-engine:jvm:max-heap --yes

# Several changes from a file
iics agent configs set --hostname myhost --from-file changes.yaml

# Copy Process Server overrides from one agent to another, then review
iics agent configs get --name CAI-A01 --service process-engine --overridden --export > pe.yaml
# edit agent-specific values such as host-name, key-alias and primary-node
iics agent configs set --name CAI-A02 --from-file pe.yaml --dry-run

# Custom properties on a CDI agent (type value or UI label)
iics agent configs set --name CDI-G01-A01 \
  --add "Data Integration Server:Tomcat:maxDTMProcesses=5" \
  --add "Data_Integration_Server:TOMCAT_JRE:INFA_CODEPAGENAME=UTF-8"
iics agent configs set --name CDI-G01-A01 --set "Data_Integration_Server:TOMCAT_CFG:maxDTMProcesses=8"
iics agent configs set --name CDI-G01-A01 --remove "Data_Integration_Server:PMRDTM_CFG:MatchQuotesPastEndOfLine"

# Copy a CDI agent's Data Integration Server overrides and custom properties to a new agent
iics agent configs get --name CDI-G01-A01 --service Data_Integration_Server --overridden --export > dis.yaml
iics agent configs set --name CDI-G01-A02 --from-file dis.yaml --add-missing --dry-run

# Restart Process Server to use the new values
iics agent restart --name CAI-A03 --service "Process Server" --blocking
```

```powershell
iics agent configs set --name CAI-A03 --set "process-engine:cluster:primary-node=false" --dry-run

iics agent configs set --name CAI-A03 --set "process-engine:jvm:max-heap='4G'" --yes
```

### agent configs edit

Edit an agent's service properties in an interactive editor, in the style of
`iics role edit`. Without an agent flag, the agent is picked from a list. The
menu entry **Edit agent properties** runs this command.

| Flag         | Type   | Description                                                                 |
| ------------ | ------ | --------------------------------------------------------------------------- |
| `--id`       | string | Agent ID                                                                    |
| `--fid`      | string | Agent federated ID                                                          |
| `--name`     | string | Agent name                                                                  |
| `--hostname` | string | Agent host name                                                             |
| `--force`    | bool   | Edit the agent even if its Secure Agent group has group-level properties    |

Screens:

1. **Services** - each service with its number of properties, overridden and
   custom properties and pending changes, then **Review and apply**,
   **↺ Reset all to defaults** and **Cancel**.
2. **Properties** of a service - type, name and a value preview. `*` marks an
   overridden property, `+` a custom property, `✎` a pending change, `✗` a
   custom property to remove. `/` filters. **+ Add custom property** asks for
   the type (from the service's types), name and value. **↺ Reset all
   &lt;service&gt; properties to defaults** resets this service only.
3. **Property editor**:
   - One-line values open in an input field. Values stored in YAML single
     quotes (`'10G'`) are edited without the quotes, which are added back.
     Sensitive values are masked.
   - Multi-value properties - lists of small maps such as LDAP properties
     (`ldap-properties`), database connection properties
     (`connection-properties`, `logConnection-properties`) or
     `additional-logging` - open as a list of entries, for example
     `key=testOnBorrow  value=true`. Pick an entry to edit or remove it, or
     use **+ Add entry** (new entries get the same fields, such as `key` and
     `value`). Unchanged entries keep their exact text.
   - **Edit as text** edits the whole value as YAML; `ctrl+e` opens it in
     `$EDITOR`.
   - **Reset to default** (shown for overridden properties) sets the value back
     to the default, like the reset icon in the UI; **Undo changes to this
     property** drops pending edits.
   - Custom properties: **Edit value**, **Remove custom property** (or
     **Keep this property** to undo a removal).
4. **Review** - all pending changes; list properties show entry changes
   (`~` changed, `+` added, `-` removed). **Apply** saves.

Reset all (for all services or one) works like the UI's **Reset All**: it
lists the overridden standard properties it will reset and asks first (the
default answer is **< Back**). Custom properties are kept, sensitive ones are
left unchanged, and nothing is saved until **Apply** in the review.

After applying, the command prints the change table, saves and verifies like
`agent configs set`, and lists the restart commands for the changed services.

Keys: `↑`/`↓` move, `enter` selects, `/` filters (enter while filtering selects
the highlighted item), `esc` goes back one screen and cancels on the service
list. In entry forms `enter` moves to the next field and saves on the last one.

```bash
iics agent configs edit
iics agent configs edit --name CAI-A03 -p tst
```

---

## agent rename

Rename a Secure Agent. The installer registers a new agent under its host
name; use `rename` to give it the logical name used across all orgs, for
example `CAI-A03`.

> **Experimental.** The name is saved through the same Administrator form as
> [agent configs](#agent-configs), together with the agent's unchanged service
> properties. Renaming does not restart any service.

| Flag          | Type   | Description                               |
| ------------- | ------ | ----------------------------------------- |
| `--id`        | string | Agent ID                                  |
| `--fid`       | string | Agent federated ID                        |
| `--name`      | string | Current agent name                        |
| `--hostname`  | string | Agent host name                           |
| `--new-name`  | string | New agent name (required)                 |
| `--yes`, `-y` | bool   | Rename without asking for confirmation    |

Exactly one of `--id` / `--fid` / `--name` / `--hostname` is required.
`--hostname` is handy right after installation, when the name is still the
host name.

Rules:

- The new name is trimmed, must not be empty and can have at most 100
  characters (the form's limit). Spaces are allowed; quote the name.
- The name must be unique in the organization (compared case-insensitively);
  otherwise the command stops before changing anything.
- The new name is read back with the agent API; the command fails if it was
  not stored.
- Renaming to the current name does nothing.

```bash
# Right after installation: name the agent registered as its host name
iics agent rename --hostname cletinfacld07 --new-name CAI-A03

iics agent rename --name "cletinfacld07.corp.example.com" --new-name CAI-A03 --yes
iics agent rename --id <agent-id> --new-name CDI-G01-A02 -p prd
```

```powershell
iics agent rename --hostname cletinfacld07 --new-name CAI-A03
```

---

## agent delete

Delete a Secure Agent.

### Flags

| Flag           | Type   | Description                  |
| -------------- | ------ | ---------------------------- |
| `--id`         | string | Agent ID                     |
| `--name`       | string | Agent name                   |
| `--hostname`   | string | Agent host name              |
| `--yes` / `-y` | bool   | Skip the confirmation prompt |

Exactly one of `--id` / `--name` / `--hostname` is required; they are mutually
exclusive.

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics agent delete --id <agent-id>

iics agent delete --hostname devinfacld01 --yes
```

```powershell
iics agent delete --id <agent-id>

iics agent delete --hostname devinfacld01 --yes
```

---

## agent start / agent stop

Start or stop a service on a Secure Agent. Uses
`POST public/core/v3/agent/service`.

### Flags

| Flag                  | Type   | Default | Description                                                                              |
| --------------------- | ------ | ------- | ---------------------------------------------------------------------------------------- |
| `--id`                | string |         | Agent ID                                                                                 |
| `--name`              | string |         | Agent name                                                                               |
| `--hostname`          | string |         | Agent host name                                                                          |
| `--service`           | string |         | Service display name, e.g. `"Data Integration Server"` (required unless `--interactive`) |
| `--interactive`, `-i` | bool   | false   | Select the agent and service interactively                                               |
| `--blocking`          | bool   | false   | Poll the service status until it has started/stopped, printing each poll                 |
| `--poll-interval`     | int    | 10      | Seconds between status polls (with `--blocking`)                                         |
| `--max-wait-time`     | int    | 1200    | Maximum seconds to wait (with `--blocking`); `stop` defaults to 300                      |

Exactly one of `--id` / `--name` / `--hostname` is required, unless
`--interactive` is used. `--interactive` lists the Secure Agents to choose from,
asks for confirmation, and (unless `--blocking` was passed explicitly) asks
`Wait for service "<name>" to start/stop`.

- `stop -i` lists the agent's running services to pick from.
- `start -i` prompts for the service **display name as free text** (the running
  services are shown for reference). There is no API that lists an agent's
  stopped services, so a stopped service cannot be offered in a menu.

With `--blocking` the command returns only once the service is fully started
(`RUNNING` with `subState 0`) or stopped (gone from the listing), or
`--max-wait-time` elapses (non-zero exit, with the last state seen). A service
in `ERROR` also fails the wait. The default wait is 20 minutes for `start` and
5 minutes for `stop`, because services such as Process Server can take well
over five minutes to become operational.

Each poll prints the status, desired status and subState. Known subStates are
labeled:

| subState | Label          | Meaning                                                                                                    |
| -------- | -------------- | ---------------------------------------------------------------------------------------------------------- |
| `-1`     | `Initializing` | The service has been deployed and is initializing                                                          |
| `2`      | `Starting`     | The service is still starting; the Administrator UI shows a warning state, but the wait continues          |
| `0`      | `Operational`  | The service is fully started                                                                               |

```text
[2026-10-23 09:55:20.451] Process Server: RUNNING (desired RUNNING, subState -1 - Initializing)
[2026-10-23 09:56:24.102] Process Server: RUNNING (desired RUNNING, subState 2 - Starting)
[2026-10-23 10:01:44.530] Process Server: RUNNING (desired RUNNING, subState 0 - Operational)
[2026-10-23 10:01:44.530] Service "Process Server" started.
```

All [global flags](../../README.md#global-flags) apply.

### Examples

```bash
iics agent stop  --id <agent-id> --service "Data Integration Server"
iics agent start --hostname devinfacld01 --service "Data Integration Server"

iics agent start --id <agent-id> --service "Process Server" --blocking --poll-interval 5

iics agent stop -i
```

```powershell
iics agent stop  --id <agent-id> --service "Data Integration Server"
iics agent start --hostname devinfacld01 --service "Data Integration Server"
```

---

## agent restart

Restart a service on a Secure Agent: stop it, wait for it to leave the agent's
service listing, then start it again.

### Flags

Same as `agent start` / `agent stop`
(`--id` / `--name` / `--hostname`, `--service`, `--interactive` / `-i`,
`--blocking`, `--poll-interval`, `--max-wait-time`).

The wait for the **stop** to complete is always performed. The wait for the
**start** to complete happens with `--blocking`, or after answering yes to the
interactive `Wait for service "<name>" to start` prompt. `--max-wait-time`
bounds the whole operation (stop wait + start wait combined) and defaults to
1200 seconds (20 minutes). "Fully started" means the engine reports `RUNNING`
with `subState 0`; see [subState labels](#agent-start--agent-stop).

### Examples

```bash
iics agent restart --id <agent-id> --service "Data Integration Server" --blocking

iics agent restart -i
```

```powershell
iics agent restart --id <agent-id> --service "Data Integration Server" --blocking
```

---

## agent installer-info

Retrieve the Secure Agent installer download URL, install token, and checksum download URL
for a platform.

### Flags

| Flag   | Type   | Required | Description                            |
| ------ | ------ | -------- | -------------------------------------- |
| `--os` | string | yes      | Operating system: `win64` or `linux64` |

When `--os` is omitted and the session is interactive, the command shows a
selection list (`win64`/`linux64`); press `Ctrl+C` to cancel. See
[Interactive prompts](interactive.md).

All [global flags](../../README.md#global-flags) apply.

### Output

By default the fields are printed as a vertical `PROPERTY`/`VALUE` table. With
`--output json`, `--output yaml`, or `--output csv` the raw response is rendered.

| Field                 | Description                                      |
| --------------------- | ------------------------------------------------ |
| `type`                | Resource type (`agentInstallerInfo`)             |
| `downloadUrl`         | URL of the Secure Agent installer binary         |
| `checksumDownloadUrl` | URL of the installer SHA-256 checksum file       |
| `installToken`        | Token used to register the agent after install   |

### Examples

```bash
iics agent installer-info --os win64

iics agent installer-info --os linux64 --output json
```

```powershell
iics agent installer-info --os win64

iics agent installer-info --os linux64 --output json
```

---

## agent installer-download

Download the Secure Agent installer binary, optionally verifying it against the
published SHA-256 checksum.

Installer info can be supplied three ways:

1. Piped as JSON on stdin (for example from `iics agent installer-info --output json`).
2. From a file via `--installer-info <path>`.
3. Omitted entirely, in which case the info is requested for the default or
   `--profile` account using `--os` (shown as a selection list when interactive;
   `Ctrl+C` cancels).

### Flags

| Flag                  | Type   | Default               | Description                                                                                |
| --------------------- | ------ | --------------------- | ------------------------------------------------------------------------------------------ |
| `--os`                | string |                       | Operating system: `win64` or `linux64` (used when no info supplied)                        |
| `--installer-info`    | string |                       | Path to installer info JSON (omit to read piped stdin)                                     |
| `--target`            | string | system temp directory | Destination file or directory                                                              |
| `--verify`            | bool   | false                 | Download the checksum file and verify the installer                                        |
| `--progress`          | bool   | false                 | Print live download progress to stderr                                                     |
| `--interactive`, `-i` | bool   | false                 | Prompt for the OS, the destination and whether to replace an existing file (terminal only) |
| `--force`             | bool   | false                 | Download even when the destination file already matches                                    |

`--target` behavior:

- Not provided: the file is written to the system temp directory using the file
  name from the installer metadata.
- An existing directory or a value ending with a path separator: the metadata file
  name is appended.
- Any other value: treated as the full destination file path.

### Existing destination file

When the destination file already exists, the command compares its SHA-256
with the installer checksum (`checksumDownloadUrl`) and reports the result:

```text
File already exists: /tmp/agent64_install_ng_ext.7617.bin (364.6MB, modified 2026-09-30 00:41)
  It matches the checksum of the current installer.
```

| Mode              | File matches                                             | File differs or cannot be checked |
| ----------------- | -------------------------------------------------------- | --------------------------------- |
| Default (scripts) | Kept, download skipped                                   | Downloaded again (replaced)       |
| `--interactive`   | Asks "Download again and overwrite it?" (default **No**) | Asks (default **Yes**)            |
| `--force`         | Downloaded again                                         | Downloaded again                  |

When the download is skipped, the result describes the existing file
(including `verified` when the checksum could be checked).

### Interactive mode

`--interactive` asks for the operating system (unless `--os` is given), then
"Download to", pre-filled with the default destination (the system temp
directory plus the installer file name, or `--target`). Enter a directory
(existing, or ending with `/`) or a full file path; `~/` is expanded. The
[main menu](menu.md) runs `iics agent installer-download --interactive --progress`.

All [global flags](../../README.md#global-flags) apply.

### Output

By default a vertical `PROPERTY`/`VALUE` table describes the downloaded file. With
`--verify` the expected and actual checksums and the `verified` result are included.
Use `--verbose` to print download and verification progress messages, or `--progress`
for a live, updating transfer line (bytes transferred, percentage when the server
reports a content length, and transfer rate) written to stderr. `--output json`,
`yaml`, and `csv` render the structured result.

| Field                 | Description                                                 |
| --------------------- | ----------------------------------------------------------- |
| `file`                | Full path of the downloaded file                            |
| `fileName`            | Installer file name                                         |
| `size`                | Downloaded size in bytes                                    |
| `downloadUrl`         | Source URL                                                  |
| `checksumDownloadUrl` | Checksum file URL (when known)                              |
| `installToken`        | Token used to register the agent after install (when known) |
| `expectedChecksum`    | SHA-256 digest from the checksum file (`--verify`)          |
| `actualChecksum`      | SHA-256 digest of the downloaded file (`--verify`)          |
| `verified`            | Whether the checksums matched (`--verify`)                  |

If the checksums do not match, the partial download is removed and the command
exits with an error.

### Examples

```bash
# Request info and download to a directory
iics agent installer-download --os linux64 --target ./downloads/

# Pipe info from installer-info and verify
iics agent installer-info --os win64 --output json | iics agent installer-download --verify

# Use a saved info file and an explicit file path
iics agent installer-download --installer-info info.json --target /tmp/agent.exe --verify

# Show live download progress
iics agent installer-download --os linux64 --target ./downloads/ --progress
```

```powershell
iics agent installer-download --os linux64 --target ./downloads/

iics agent installer-info --os win64 --output json | iics agent installer-download --verify

iics agent installer-download --installer-info info.json --target C:\Temp\agent.exe --verify
```

## See also

- [Interactive prompts](interactive.md) - keys, accessible mode and piped input for wizards and pickers
- [environment](environment.md) - manage runtime environments (Secure Agent groups) that contain agents
