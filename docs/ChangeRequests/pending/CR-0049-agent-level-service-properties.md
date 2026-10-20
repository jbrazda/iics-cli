# CR: Agent-level Secure Agent service properties (agent configs)

---

## CR Type

- [ ] **New resource**
- [x] **New subcommand** - `iics agent configs get` and `iics agent configs set`
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

- `iics agent configs get --id|--fid|--name|--hostname` lists the agent's
  properties with value, default and overridden flag. Filters: `--service`,
  `--type` (`type/subType`), `--search` (regex on the name), `--overridden`.
- `iics agent configs set` changes properties with `--set
  service:type:name=value` (value `@file` reads a file), `--reset
  service:type:name` and `--from-file` (JSON list of `{property, value, reset}`).
  It prints a CURRENT / NEW table, asks for confirmation (`--yes` skips,
  `--dry-run` only shows), saves, and reads back to verify.
- `set` refuses agents whose group has group-level properties unless `--force`,
  because the docs say group settings replace agent settings.
- `set` refuses the agent's own properties (`agent:` Agent Core, Agent Manager,
  Tomcat), because the server ignores changes to them (see notes).
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
cmd/agent_configs.go                # agent configs get / set
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
  - After a save, the agent redeploys the affected service. OI Data Collector
    went NEED_RUNNING, then DEPLOYING, then RUNNING. Process Server
    (`REPLACE_VERSION`) started a new version next to the running one.
- Setting a property to its current value is a no-op, so defaults do not turn
  into overrides.
- `--debug` now masks the `Cookie` and `Set-Cookie` headers, which carry the
  session.
