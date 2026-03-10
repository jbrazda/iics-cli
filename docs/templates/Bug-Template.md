# BUG: [Short Title]

<!--
HOW TO USE THIS TEMPLATE
─────────────────────────
1. Fill in every section below. Incomplete reports slow down resolution.
2. Paste this file into docs/issues/<YYYY-MM-DD>-<slug>.md and link it from a GitHub issue.
3. When handing this to Claude, say:
   "Fix the bug described in docs/issues/<file>.md. Read that file and docs/CLAUDE.md first."
4. Claude must read docs/CLAUDE.md before touching any code.
-->

---

## Symptoms

> What the user sees. One or two sentences. Focus on observable behaviour, not causes.

_Example: `iics user list` prints an empty table even when users exist in the org._

---

## Command / Reproduction Steps

```bash
# Exact command(s) that trigger the bug
iics <subcommand> [flags]
```

> Add `--verbose` to capture HTTP-level detail and paste that output below.

---

## Expected Behaviour

> What should happen.

---

## Actual Behaviour

> What actually happens. Include the full terminal output, error message, and stack trace.

```text
<paste output here>
```

---

## Environment

| Field                      | Value                         |
| -------------------------- | ----------------------------- |
| OS                         | e.g. macOS 15.3, Ubuntu 24.04 |
| `iics --version`           | e.g. v0.4.1                   |
| Go version                 | e.g. 1.25.0                   |
| IICS region                | e.g. US, EMEA                 |
| Output format (`--output`) | e.g. table / json / csv       |

---

## Architecture Layer

> Tick the layer(s) where the bug most likely lives.
> This tells Claude which files to focus on.

- [ ] **`cmd/`** — flag parsing, command wiring, output formatting
- [ ] **`internal/client/`** — HTTP logic, API structs, request/response handling
- [ ] **`internal/config/`** — config file loading, session cache
- [ ] **`internal/output/`** — table / JSON / CSV renderer

---

## Likely Affected Files

> List files that probably need to change. Claude will read these first.
> Leave blank if unknown — Claude will locate them.

```text
internal/client/<resource>.go
cmd/<resource>.go
```

> Files Claude should read for context but NOT modify:

```text
internal/client/client.go      # HTTP do() / doJSON() helpers
docs/CLAUDE.md                 # Project conventions (mandatory read)
```

---

## API Details (if relevant)

> Fill in if the bug involves an API call returning unexpected data.

| Field          | Value                                       |
| -------------- | ------------------------------------------- |
| API version    | V2 (`api/v2`) / V3 (`public/core/v3`)       |
| HTTP method    | GET / POST / PUT / PATCH / DELETE           |
| Endpoint path  | e.g. `public/core/v3/users`                 |
| Session header | `icSessionId` (V2) / `INFA-SESSION-ID` (V3) |

**Actual API response (JSON):**

```json
// paste the raw API response here if available
// obtain with: curl -s ... | jq .
```

**Struct that maps this response** (file + line):

```text
internal/client/<resource>.go:<line>
```

> Common struct bugs: wrong JSON tag name, `[]string` used where `[]SomeStruct` is needed,
> `string` used where the API actually returns a number, missing `omitempty` on optional fields.

---

## Error Message / Stack Trace

```text
<paste full error here>
```

---

## Fix Instructions

> Precise, scoped instructions for Claude. Write these as if assigning a task.
> Be explicit about what to change and what to leave alone.

1. Read `docs/CLAUDE.md` and the affected files listed above before writing any code.
2. [Step-by-step description of the fix, e.g.:]
   - In `internal/client/users.go`, the `Email` field has JSON tag `"email"` but the API returns `"emails"` — change the tag.
   - Do **not** touch the `cmd/` layer unless a flag name or output column must change.
3. Add or update the test in `internal/client/<resource>_test.go` to cover the fixed case.
4. Run `/opt/local/bin/go test ./internal/client/...` and verify it passes.
5. Run `/opt/local/bin/go build ./...` to confirm no compilation errors.

---

## Acceptance Criteria

- [ ] The reproduction command now produces the expected output
- [ ] All existing tests still pass (`/opt/local/bin/go test ./...`)
- [ ] No unrelated code is refactored
- [ ] No new dependencies are introduced
- [ ] `go vet ./...` and `golangci-lint run ./...` report no new issues

---

## Do NOT

- Refactor, reformat, or add comments to code outside the fix scope
- Change function signatures or struct names not directly involved in the bug
- Add error handling for scenarios unrelated to this bug
- Switch tablewriter to v0.x API patterns (always use v1.x: `NewTable`, `Header`, `Append([]interface{}{...})`, `Render() error`)
- Call `os.Exit()` — always return errors from `RunE`
- Never Guess JSON field names — verify against the API docs or the raw response pasted in bug report

---

## Fix (filled in after resolution)

> Claude fills this section in when done.

**Root cause:**

**Files changed:**

```text
<file>:<line range> — <one-line description of change>
```

**Test added / updated:**

```text
internal/client/<resource>_test.go — <describe test>
```
