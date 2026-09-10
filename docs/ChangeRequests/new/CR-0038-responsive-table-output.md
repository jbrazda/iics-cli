# CR-0038: Responsive table output (adapt to terminal width)

## CR Type

- [x] Enhancement to existing subsystem (`internal/output/`)

## Status

Design captured from a grilling session. **Implementation deferred.** Open
questions in the section at the end must be answered before coding.

## Problem

`internal/output/table.go` `computeColWidths` sizes every column to
`max(header, widest cell, Column.Width)` with no ceiling. On a narrow terminal
wide tables hard-wrap into an unreadable mess. The worst offenders:

- **Opaque GUID columns** (`id`, `federatedId`, `agentGroupId`, `orgId`,
  `objectId`, `agentId`, `runtimeEnvironmentId`, `contextExternalId`,
  `itemGUID`) - always 24 chars, ~40 uses. Cannot be truncated (a partial UUID
  is useless; the IICS API does not accept ID prefixes) and cannot be wrapped.
- **Unbounded free text** (`description`, `message`, `status.message`,
  `errorMsg`, `eventParam`, `itemStatusDetail`, `detail`, `warning`, config
  `value`/`defaultValue`) - ~30 uses, often with no `Width:` set so they grow
  to the widest cell.
- **Paths / URLs** (`path`, `assetPath`, `sourcePath`, `targetPath`,
  `location`, `serverUrl`, `spiUrl`, `downloadUrl`, `endpoint`) - 40-60 wide,
  the tail (filename) matters most.

Buckets 1 and 2 are the entire problem and need opposite treatments: bucket 2
wraps or truncates fine, bucket 1 can only be dropped.

## Column analysis and priority tiers

Drop order under width pressure: P5 first, then P4, then wrap/truncate P3/P2,
then vertical fallback if P1 alone overflows.

| Tier | Behavior under pressure | Columns |
| ---- | ----------------------- | ------- |
| P1 Essential | never drop, never shrink below content | primary `name`/`userName`/`objectName`, `state`/`status`, and the column that is the point of the command (e.g. `startTime` for `activitylog`, `permission` for `permission`) |
| P2 Core | keep; truncate with ellipsis only as last resort | `type`, secondary names, `email`, counts, `interval`, `duration`, timestamps |
| P3 Context | truncate or wrap freely | paths, URLs, `updatedBy`/`createdBy`, short `description` |
| P4 Detail | wrap to remaining space, then drop | `message`, `errorMsg`, `eventParam`, `detail`, `warning`, long `description`, config `value` |
| P5 Reference | drop first; emit a hint to use `--wide` or `-o json` | opaque GUID columns that are not the primary key of the row |

## Decisions made

### D1 - Adaptation strategy (confirmed)

Tier-based. Assign every column P1-P5. Under width pressure:

1. Drop from P5 upward.
2. Wrap P4, then P3.
3. Truncate P2.
4. Fall back to vertical key/value layout only if P1 columns alone overflow.

Emit a one-line **stderr** hint when columns were dropped, naming them:

```text
# 3 columns hidden (ID, FEDERATED ID, GROUP ID) - use --wide or -o json
```

This is worth an ADR (touches every `cmd/` file; "where did the ID column go?"
is surprising; real trade-off against always-vertical and truncate-everything).

### D2 - Width detection and non-TTY behavior (confirmed)

Effective width precedence: `--width N` flag > `IICS_WIDTH` env > detected TTY
size (`term.GetSize` on `os.Stdout` fd) > `$COLUMNS` env > `80`.

No adaptation at all (full width, nothing dropped or truncated) when **any** of:

- output writer is not a TTY;
- `--output` is `csv` / `json` / `yaml`;
- table theme is `markdown` or `gh`.

Add a `--wide` persistent bool flag: detect width for layout but never drop or
truncate. Covers "wide terminal piped into `less -S`".

`golang.org/x/term` is already an indirect dependency (used in
`internal/config/prompt.go`).

## Proposed implementation (subject to open questions)

### `internal/output/formatter.go` - extend `Column`

```go
type ShrinkMode int

const (
    ShrinkTruncate     ShrinkMode = iota // default: right-truncate with ellipsis
    ShrinkTruncateLeft                   // keep the tail (paths, URLs)
    ShrinkWrap                           // hard-wrap onto multiple lines
    ShrinkNever                          // squeeze not allowed; drop instead
)

type Column struct {
    Header   string
    Field    string
    Width    int        // existing meaning: minimum / floor
    MaxWidth int        // new: 0 = unbounded
    Priority int         // new: 1..5; 0 = unset -> treated as P3
    Shrink   ShrinkMode  // new
    Func     func(v interface{}) string
}
```

### `internal/output/table.go`

- New `planColumns(rows, columns, termWidth, wide bool) (kept []Column, widths []int, dropped []string)`.
- Heuristic defaults when `Priority == 0` (see open question Q4):
  - `Field == "id"` or `Header` contains `ID` with `Width == 24` -> P5, `ShrinkNever`
  - `Width == 0` and content long -> P4, `ShrinkWrap`
  - short enum widths (<= 16) -> P2, `ShrinkNever`
  - first name-ish column -> P1
- `renderTable` consumes the plan; wrapped cells already supported via
  `splitCellLines` / `WrapCell`.
- Hint written to `os.Stderr` (not `f.w`) when `len(dropped) > 0`.

### `cmd/root.go`

- `--width int` and `--wide bool` persistent flags; `IICS_WIDTH` env binding.
- Plumb resolved width + wide into `TableStyle` (new fields `Width int`,
  `Wide bool`), or pass via a new `output.New` parameter.

### `internal/config/config.go`

- Add `style.responsiveTables bool` (default `true`) - the single config toggle.
- `--wide` flag and `responsiveTables: false` both mean "do not drop/truncate".
- `minColumnWidth` stays a package constant (`8`), not exposed in config.

### `cmd/*.go` migration

Hybrid: rely on the central heuristic for most of the 22 files; add explicit
`Priority` / `Shrink` to the high-traffic commands in the same PR: `agent`,
`activitylog`, `user`, `connection`, `objects`, `package`.

### Docs

- `docs/documentation/` - new page or section on responsive output, `--width`,
  `--wide`, `style.responsiveTables`.
- `README.md` global flags table.
- ADR under `docs/` (or `docs/adr/`) for the column-dropping decision.
- `make completions` after the flag additions.

## Open questions (answer before implementing)

- **Q3 - `Column` shape**: confirm the `ShrinkMode` enum + `MaxWidth` +
  `Priority` fields as proposed, or prefer a separate per-command column
  registry keyed by field name instead of struct fields.
- **Q4 - migration approach**: confirm hybrid (central heuristic + explicit
  annotation on 6 high-traffic commands). Confirm the exact heuristic signals
  so they can be documented in the ADR.
- **Q6 - config/flag surface**: confirm `style.responsiveTables` as the only
  config key, `minColumnWidth` as a constant, and the `--width` / `--wide`
  precedence chain.
- **Q7 - hint channel**: confirm the dropped-columns hint goes to stderr and
  names the hidden columns.
- **Q8 (not yet discussed)**: when cells wrap, do multi-line rows need a row
  separator for readability, and how does that interact with the existing
  themes (`default`, `minimal`, `compact`, `plain`, `markdown`, `gh`)?
- **Q9 (not yet discussed)**: is `ShrinkTruncateLeft` actually wanted for
  paths/URLs, or is right-truncation good enough everywhere?
