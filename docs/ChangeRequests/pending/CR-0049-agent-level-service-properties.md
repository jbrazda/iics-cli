# CR: Agent-level Secure Agent service properties (agent configs)

---

## CR Type

- [ ] **New resource**
- [x] **New subcommand** - `iics agent configs get|set|edit`, `iics agent rename`
- [ ] **Enhancement**
- [ ] **Output change**
- [ ] **Flag / config change**
- [ ] **Refactor**

---

## Problem

Some Secure Agent service properties must differ per agent, for example the
Process Server cluster `primary-node` flag, `host-name` / `key-alias`, and the
Process Server database URLs and connection properties. The public REST API
only configures service properties for a whole Secure Agent group
(`/api/v2/runtimeEnvironment/<groupId>/configs`, already covered by
`iics environment configs`). Passing an agent ID to those endpoints fails with
HTTP 403 `REPO_54014` "Access denied to perform operation on Agent". The `agent`
resource documents only GET and DELETE, so agent-level properties could not be
changed from the CLI.

---

## Desired Change

- The commands work for any agent regardless of which services it runs, for
  example Process Server and OI Data Collector on CAI agents, or Data
  Integration Server, Mass Ingestion, Database Ingestion, CMI Streaming Agent,
  IDMC Data Gateway and Metadata services on CDI agents.
- `iics agent configs get --id|--fid|--name|--hostname` lists the agent's
  properties with value, default and overridden flag. Filters: `--service`,
  `--type` (`type/subType`), `--search` (regex on the name), `--overridden`.
  `--export` writes the filtered properties as a change file (YAML, or JSON
  with `-o json`).
- `iics agent configs set` changes properties with `--set
  service:type:name=value` (value `@file` reads a file), `--reset
  service:type:name` and `--from-file`. The file is JSON or YAML, either a
  nested `service -> type -> name: value` map (`null` resets), as written by
  `get --export`, or a list of `{property, value, reset}`. Only the listed
  properties change.
  It prints a CURRENT / NEW table, asks for confirmation (`--yes` skips,
  `--dry-run` only shows), saves, and reads back to verify. It then lists the
  changed services with the `iics agent restart` command for each, because
  services such as Process Server only mark themselves as needing a restart.
- `set` refuses agents whose group has group-level properties unless `--force`,
  because the docs say group settings replace agent settings.
- `set` refuses the agent's own properties (`agent:` Agent Core, Agent Manager,
  Tomcat), because the server ignores changes to them (see notes). `get` hides
  them unless `--include-hidden` or `--service agent` is given.
- `iics agent configs edit` opens an interactive editor in the style of
  `role edit`: services, then properties, then a property editor, then review
  and apply. One-line values are edited in an input (YAML single quotes are
  added back automatically, sensitive values masked). Multi-value properties
  (lists of maps such as LDAP, database connection properties, additional
  logging) open in an entry editor with edit, add, remove, edit-as-text
  (`$EDITOR` via ctrl+e), reset and undo. The review lists entry-level
  changes. Menu entry "Edit agent properties".
- Custom properties (Custom Configuration Details), for example
  `maxDTMProcesses` for the Data Integration Server Tomcat on CDI agents:
  `get` lists them (marked `custom`), `set --add service:type:name=value` adds
  one with a type the service offers, `--set` changes it, `--remove` (or
  `--reset`) removes it, `--add-missing` adds missing properties from a change
  file, and `agent configs edit` has **+ Add custom property** and remove.
  Type labels from the UI (`Tomcat`, `DTM`) are accepted in property names.
- `iics agent rename --id|--fid|--name|--hostname --new-name <name>` renames an
  agent, for example from the host name the installer registers to a logical
  name such as `CAI-A03`. The name is trimmed, at most 100 characters, and must
  be unique in the org (case-insensitive). It is read back with the agent API.
- Both commands are marked experimental.

**Informatica docs:**

- Group-level API: `https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-2-resources/runtime-environments/configuring-secure-agent-service-properties-for-secure-agent-gro.html`
- Agent resource (GET/DELETE only): `https://onlinehelp.informatica.com/IICS/prod/CDI/en/tt-cloud-rest-api/Secure_Agents_and_services.html`

---

## Scope

### Files to CREATE

```text
internal/client/agentform.go        # form read/parse/apply/save/verify
internal/client/agentform_test.go   # parser, apply, filter, GET/POST, re-login tests
internal/client/agentconfigfile.go  # change files (JSON/YAML map or list) and export
internal/client/agentcustom.go      # custom property rows: parse, add, remove, submit
internal/client/agentcustom_test.go
internal/client/agentconfigfile_test.go
cmd/agent_configs.go                # agent configs get / set
cmd/agent_rename.go                 # agent rename
internal/tui/agentconfigeditor.go   # interactive property editor
internal/tui/agentconfigvalue.go    # quoting, list-of-maps parsing/writing, entry diff
internal/tui/agentconfigeditor_test.go
internal/menu/entries.go            # "Edit agent properties" menu entry
```

### Files to MODIFY

```text
cmd/agent.go                        # register `agent configs`
internal/client/client.go           # mask Cookie and Set-Cookie in --debug output
docs/documentation/agent.md         # agent configs section
docs/documentation/environment.md   # pointer to agent configs, REPO_54014 note
README.md                           # agent row lists `configs`
completions/                        # regenerated
```

---

## API Details

There is no REST API. The Administrator UI uses a server-rendered form,
verified with a browser HAR capture and live calls on the tst org.

| Field          | Value                                                                                 |
| -------------- | ------------------------------------------------------------------------------------- |
| Read           | `GET https://<pod host>/runtime/app/agent.do?actionMethod=displayUpdate&id=<agentId>` |
| Write          | `POST https://<pod host>/runtime/app/agent.do`, `application/x-www-form-urlencoded`   |
| Authentication | Cookie `USER_SESSION=<REST session ID>` (the v2/v3 `icSessionId` is accepted)         |
| CSRF           | Double submit: cookie `XSRF_TOKEN` must equal form field `csrfToken`                  |
| Response       | HTML page; `var hasErrors = "true"` and `#InfErrorDiv` carry errors; HTTP 200 always  |

Form layout (`<form id="agentForm">`, about 1,000 fields, all submitted on save):

- `name`, `id`, `version`, `actionMethod=update`, `csrfToken`, plus UI state
  fields (`isAnySensitiveField`, `customConfig*`, `uiUpgradeStatus`,
  `crossIFrameCallState`, `iframeWorkspaceId`, `launchPagePath`).
- System rows, six fields each: `sysConfigType`, `sysConfigSubType`,
  `sysConfigName`, `sysConfigValue`, `sysConfigDefault`, `sysConfigOverriden`.
- Engine rows, four fields each: `engineConfigKey`, `engineConfigValue`,
  `engineConfigDefault`, `engineConfigOverriden`, plus checkbox
  `engineConfigSensitive` (sent only when checked). The key is
  `<service key><type><name>`, for example
  `process-engine15124581clusterprimary-node`. Each row is wrapped in
  `<tr data-engine="<service key>" data-config-type="<type>">`, and the
  `#engineConfigSel` select maps service keys to display names.
- Editing a property sets its value and `...Overriden=true`. Resetting sets the
  value to the default and `...Overriden=false` (from `pnAgent.js`).
- Browsers send newlines in values as CRLF.

---

## Acceptance Criteria

- [x] `agent configs get` lists all engine and system properties of an agent
- [x] `agent configs set` changes and resets engine properties and verifies them
- [x] Multi-line values via `@file` round-trip unchanged
- [x] Errors from the form (invalid value) are returned as command errors
- [x] Unknown, ambiguous and duplicate properties are rejected before saving
- [x] Client methods have tests; `go build`, `go test`, `go vet`, `golangci-lint` pass
- [x] Two-layer rule respected; no new dependencies

---

## Implementation Notes (filled in by Claude during implementation)

- No HTML parser dependency was added. A small tokenizer handles input,
  textarea and select fields the way a browser serializes a form: checked
  checkboxes only, skips disabled fields, uses the selected option, drops the
  first newline in a textarea, and sends CRLF newlines. Against a live page and
  the user's HAR capture, the encoded body matched the browser's field for
  field (1,041 fields).
- The page only renders a `csrfToken` value when an `XSRF_TOKEN` cookie is
  sent. The CLI generates a random token for both. The `JSESSIONID` cookie from
  the GET is sent back on the POST.
- If the page has no agent form (session not accepted), the client logs in
  again once and retries the GET.
- Live test on tst, agent CAI-A03 (group CAI, no group-level properties):
  - `OpsInsightsDataCollector:OpsInsights:scriptLogRetentionCount` set to `'101'`
    and reset. Both confirmed through the form and the REST `agent details`.
  - `process-engine:logging:additional-logging` (multi-line) changed through
    `@file` and restored. An earlier invalid value was rejected by the server
    with "For configuration additional-logging invalid value specified ...",
    and nothing was saved.
  - System rows (`agent:AGENTCORE_LOG4J/INFO` and `/DEBUG`
    `log4j.appender.aclog.MaxBackupIndex`): the server returned no error but did
    not store the change, also after waiting 80 seconds. `set` therefore
    refuses `agent:` properties. Supporting them needs a HAR capture of a system
    property save in the UI.
  - After a save, OI Data Collector went NEED_RUNNING, then DEPLOYING, then
    RUNNING by itself. Process Server does not restart by itself after a
    configuration change: it keeps the old values and the Administrator UI
    shows that it needs a restart, which is done manually (developer
    confirmation). `set` therefore prints restart commands for the changed
    services.
- Read-only checks on CDI-G01-A01 in dev and tst (12 services, 426 and 435
  properties) found every property with a unique `service:type:name` address,
  including Data Integration Server types such as `TOMCAT_JRE`, `PMRDTM_CFG`
  and `SYSTEM_CFG`. A dry-run of a Data Integration Server change from a JSON
  file showed only that property.
- Change files are parsed as YAML nodes, so scalars keep their written text
  (`true`, `512`, `1.0`) and `null` means reset. YAML removes one level of
  quotes, so Process Server values need `"'4G'"`. `--export` writes values
  with quotes double-quoted and multi-line values as block scalars; an
  unchanged export of CAI-A03's Process Server overrides and of a CDI agent's
  `TOMCAT_JRE` properties applied as "Nothing to change".
- Setting a property to its current value is a no-op, so defaults do not turn
  into overrides.
- `--debug` now masks the `Cookie` and `Set-Cookie` headers, which carry the
  session.
- The agent's own (`sysConfig*`) rows sit in a form section (`#agentSysConfig`)
  that `pnAgent.js` shows only when the Service list has an `agentCoreSelOpt`
  option. No checked page has one (CAI-A03 and CDI-G01-A01 in tst, CAI-A01 in
  dev), and the UI screenshot shows only the agent's services. So these rows
  are hidden, not editable, which matches the server ignoring them.
- The agent name is the form's `name` text field (`maxlength="100"`). Live test
  on CAI-A03: renaming to an existing name (`cai-a01`) stopped before saving;
  renaming to `CAI A03 Rename Test` (spaces) worked, and `agent get --name`
  and `agent list` found it; renaming back via `--hostname cletinfacld07`
  worked; renaming to the current name did nothing. The 14 overrides were
  unchanged and no service restarted (service update times unchanged).
- The entry editor keeps the value as a YAML node tree and changes only the
  edited scalars, so unchanged entries keep their text. All 8 list values on
  CAI-A03 (current and default of LDAP, DB and log DB connection properties,
  additional logging) round-trip byte for byte. Edited values are written
  plain when YAML allows (`300000`, `true`) and quoted only when needed.
- The editor was driven in a pseudo-terminal on tst CAI-A03: editing entry 2
  and adding an entry to `db:connection-properties` showed the entry diff in
  review, then Cancel saved nothing; changing
  `OpsInsightsDataCollector:OpsInsights:scriptLogRetentionCount` by typing
  `101` stored `'101'` (quotes added), verified, and was reset afterwards.
- Table columns `NAME` (get) and `PROPERTY` (set diff) no longer use
  `ShrinkWrap`, which the table renderer moves to the last position.
- Custom property rows (`#customConfigTable`) submit, in order:
  `customConfigEngine` (service key), `customConfigType` (`-1`),
  `customConfigTypeEngine` (type), `customConfigSubType` (`-1`),
  `customConfigName` (maxlength 255), `customConfigValue` (maxlength 512),
  `customConfigSensitive` (checkbox, only when checked),
  `engineCustomConfigIdInput` (row ID) and `isSensitiveConfig`. The types per
  service are the `data-engine` options of `customConfigTypeEngine`. From
  `pnAgent.js`: a new row gets `Date.now()` as its ID; removing the last row
  leaves an empty template row with `-1` values, which the form always
  submits when there are no custom properties. The client keeps the parsed
  fields as they are and applies additions and removals when it builds the
  request, so field positions stay valid.
- Read-only on tst CDI-G01-A01: the 5 custom properties from the UI
  (`maxDTMProcesses`, `csv-reader-disable-safety-switch`, `maxUrlLength`,
  `INFA_CODEPAGENAME`, `MatchQuotesPastEndOfLine`) are listed, exported, and
  a dry run of add, set and remove shows the expected changes.
- Live on tst CAI-A03 (OI Data Collector custom property, which replaces
  the empty template row): `set --add`, `--set`, a refused duplicate `--add`,
  and `--remove`, each verified by reading back; then add and remove through
  `agent configs edit` in a pseudo-terminal. The agent has no custom
  properties afterwards, as before.
