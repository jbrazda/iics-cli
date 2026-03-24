# CR: Add PowerShell Variants to All Documentation Shell Examples

<!--
HOW TO USE THIS TEMPLATE
─────────────────────────
1. Fill in every section. Incomplete CRs lead to wrong assumptions and scope creep.
2. Save this file as docs/ChangeRequests/<YYYY-MM-DD>-<slug>.md.
3. When handing to Claude, say:
   "Implement the CR in docs/ChangeRequests/<file>.md. Read that file and docs/CLAUDE.md first."
4. Claude MUST read docs/CLAUDE.md before writing any code.
-->

---

## CR Type

> Tick exactly one. Determines which files need to change.

- [ ] **New resource** - add a brand-new `iics <resource>` command tree (all 4 files required)
- [ ] **New subcommand** - add a subcommand to an existing resource (client method + cmd wiring)
- [x] **Enhancement** - change behaviour of an existing command (modify specific files)
- [ ] **Output change** - add/remove/rename columns, change default format, fix display
- [ ] **Flag / config change** - add/rename/remove a CLI flag or config field
- [ ] **Refactor** - internal restructuring with no behaviour change (rare; justify below)

---

## Problem

Currently, all shell examples in the documentation use only bash syntax. Windows and PowerShell users need equivalent examples to understand how to use `iics-cli` in their environment. Without PowerShell variants, the documentation is incomplete for users working with Windows systems.

---

## Desired Change

Add PowerShell code samples as variants to all bash examples in the documentation files under `docs/documentation/*.md`. Each bash example should be followed by an equivalent PowerShell example shown in a separate code block with `powershell` language tag.

Examples of PowerShell equivalents:
- `iics user list` → `iics user list` (same, no translation needed)
- `iics user list --output json | jq '...'` → `iics user list --output json | ConvertFrom-Json | <PowerShell equivalent>`
- Other shell-specific features should be translated to PowerShell equivalents

---

## Scope

### Files to MODIFY

```text
docs/documentation/activitylog.md          # add PowerShell examples
docs/documentation/agent.md                # add PowerShell examples
docs/documentation/completion.md           # add PowerShell examples
docs/documentation/connection.md           # add PowerShell examples
docs/documentation/export.md               # add PowerShell examples
docs/documentation/folder.md               # add PowerShell examples
docs/documentation/import.md               # add PowerShell examples
docs/documentation/login.md                # add PowerShell examples
docs/documentation/logout.md               # add PowerShell examples
docs/documentation/lookup.md               # add PowerShell examples
docs/documentation/metering.md             # add PowerShell examples
docs/documentation/objects.md              # add PowerShell examples
docs/documentation/package.md              # add PowerShell examples
docs/documentation/permission.md           # add PowerShell examples
docs/documentation/privilege.md            # add PowerShell examples
docs/documentation/profile.md              # add PowerShell examples
docs/documentation/project.md              # add PowerShell examples
docs/documentation/publish.md              # add PowerShell examples
docs/documentation/role.md                 # add PowerShell examples
docs/documentation/runtime.md              # add PowerShell examples
docs/documentation/schedule.md             # add PowerShell examples
docs/documentation/securitylog.md          # add PowerShell examples
docs/documentation/sourcecontrol.md        # add PowerShell examples
docs/documentation/state.md                # add PowerShell examples
docs/documentation/tag.md                  # add PowerShell examples
docs/documentation/unpublish.md            # add PowerShell examples
docs/documentation/user.md                 # add PowerShell examples
docs/documentation/usergroup.md            # add PowerShell examples
```

### Files to READ (context only - do NOT modify)

```text
docs/CLAUDE.md                          # mandatory: patterns and rules
```

### Forbidden (do NOT touch)

```text
cmd/                                    # no command changes needed
internal/                               # no client/config changes needed
docs/documentation/comparison-informatica-cli.md   # no changes needed
```

---

## API Details

Not applicable - documentation update only, no API calls involved.

---

## Examples

### Current Format (bash only)
```bash
iics user list

iics user list --output json

# Find users by name using JSON + jq
iics user list --output json | jq '.[] | select(.userName | test("john"))'
```

### Desired Format (bash + PowerShell)
```bash
iics user list

iics user list --output json

# Find users by name using JSON + jq
iics user list --output json | jq '.[] | select(.userName | test("john"))'
```

```powershell
iics user list

iics user list --output json

# Find users by name using JSON + ConvertFrom-Json
$users = iics user list --output json | ConvertFrom-Json
$users | Where-Object { $_.userName -match "john" }
```

---

## Notes

- Do not modify any command behavior or code - only documentation.
- Each bash example should have a corresponding PowerShell example.
- PowerShell examples should accomplish the same goal using PowerShell-native cmdlets where applicable (e.g., `ConvertFrom-Json` instead of `jq`, `Where-Object` instead of `select()`, etc.).
- The presentation file `docs/documentation/iics-cli-presentation.pptx` should also be updated with examples of PowerShell usage where relevant.
