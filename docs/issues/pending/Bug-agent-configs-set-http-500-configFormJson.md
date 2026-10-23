# Bug: agent configs set fails with HTTP 500 - form rows must be packed into configFormJson

## Summary

`iics agent configs set` (and `agent configs edit`, `agent rename`) fail with
HTTP 500 `CMN_001` when saving the agent form. Informatica changed the
Administrator UI form at `/runtime/app/agent.do` (internal ticket POP-88806):
the property rows are no longer submitted as flat form parameters but bundled
into one hidden JSON field, `configFormJson`, because the flat form exceeded
Tomcat's `maxParameterCount`. The CLI still sent about 1,100 flat parameters,
so the server rejected the POST.

## Error

```text
Changes for agent CAI-A03 (01D23X0800000000008G):
...
Error: IICS API error (HTTP 500):
HTTP 500 Internal Server Error
...
  {
    "error": {
      "code": "CMN_001",
      "message": "Internal system error. Contact Informatica Global Customer Support.",
      "requestId": "1Z71Pf2BP2gdRKnkdT7SIs"
    }
  }
```

The form GET still works; only the POST fails, with an upstream service time
of a few milliseconds.

## Root Cause

The UI's `doUpdate` now calls `pnAgent.packConfigFormParams(form)` from
`/runtime/js/pnAgent.js` before submitting. It collects every element named
in `configFormParamNames` (`sysConfig*`, `customConfig*`, `engineConfig*`,
`engineCustomConfigIdInput`, `isSensitiveConfig`), stores
`{"name": ["value", ...]}` in `configFormJson`, and disables the original
elements so they are not posted. Unchecked checkboxes are left out, and
names without elements are omitted. The server expands the JSON with
`ConfigFormRequestWrapper`.

`AgentConfigForm.encode()` in `internal/client/agentform.go` posted all rows as
flat parameters and left `configFormJson` empty.

## Fix

`encode()` now passes the submitted fields through `packConfigFields`. When the
form contains a `configFormJson` field, the row fields are moved into it as
JSON in the UI's key order, with LF line endings as the browser's
`element.value` has. Forms without the field are still sent flat.

Verified live on tst agent CAI-A03: setting
`process-engine:connectors:https-maxThreads=601` saves and reads back. All
other property values are unchanged. Four multi-line Process Server properties
(`server:ldap-properties`, `db:connection-properties`,
`db:logConnection-properties`, `logging:additional-logging`) whose values
equal their defaults were flagged overridden before the save and are now
reported as not overridden, the same as a save from the UI.

## Follow-up: whitespace-only overrides

The server flags a value as overridden when it differs from the default only
in line endings or whitespace. On tst CAI-A01 and CAI-A02 the four multi-line
Process Server properties above were listed by `--overridden` and exported by
`--export` for that reason alone.

`client.SameValue` compares values ignoring CRLF/LF differences, trailing
whitespace on a line, and leading and trailing blank lines. Indentation
counts, as it is significant in YAML. It is used:

- when the form is parsed, to clear the overridden flag of such values, so
  `get --overridden`, `--export`, `--reset-all` and the editor agree
- in `Apply`, so a whitespace-only `--set` is no change, a value equal to the
  default resets the property, and resetting a property that is not
  overridden is no change
- when verifying a save, and in the TUI editor's pending-change check

## Steps to Reproduce

```bash
iics --profile tst agent configs set --name CAI-A03 \
  --set "process-engine:connectors:https-maxThreads=601"
```

## Expected Behaviour

The property is saved and the restart hint is printed.

## Actual Behaviour

HTTP 500 `CMN_001` from `POST /runtime/app/agent.do`.
