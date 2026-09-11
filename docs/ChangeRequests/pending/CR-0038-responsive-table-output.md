# CR-0038: Responsive table output (adapt to terminal width)

## CR Type

- [x] Enhancement to existing subsystem (`internal/output/`)

## Status

Design fully settled across two grilling sessions. All open questions
resolved (see Decisions made). Ready for implementation.

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

### D3 - `Column` struct shape (confirmed)

Add `MaxWidth`, `Priority`, `Shrink ShrinkMode` directly to `output.Column`
(not a separate per-command registry). Same file the columns are already
declared in; keeps the tier visible at the definition site.

### D4 - Heuristic signals for un-annotated columns (confirmed)

For any column with `Priority == 0`:

1. `Field == "id"` or `Field` matches `/Id$/`/`/ID$/` with `Width == 24` -> P5, `ShrinkNever`
2. `Width == 0` (no floor set) -> P4, `ShrinkWrap`
3. `Width > 0 && Width <= 16` -> P2, `ShrinkNever`
4. everything else -> P3, `ShrinkTruncate`
5. the first column in the slice, if not already caught above -> P1

Document these verbatim in the ADR.

**Display-order rule:** independent of `Priority` (which drives drop
decisions) and independent of column order in `json`/`yaml`/`csv` output
(unaffected), any *kept* column with `Shrink == ShrinkWrap` is rendered in
the rightmost position among the kept columns for the `table` format only,
so its multi-line wrapping never disrupts alignment of columns to its right.
If more than one kept column wraps, order those by descending natural width
(widest last).

### D5 - Config/flag surface (confirmed, final)

`style.responsiveTables: bool` (default `true`) in config. `--width int` and
`--wide bool` persistent flags. `IICS_WIDTH` env. Precedence: `--width` >
`IICS_WIDTH` > detected TTY size > `$COLUMNS` > `80`. `--wide` or
`responsiveTables: false` => detect width for layout but skip drop / truncate
/ wrap entirely. `minColumnWidth` stays a package constant (`8`), not exposed
in config.

### D6 - Dropped-columns hint placement (confirmed)

stderr, after the table and after the existing stdout `N rows` footer (not
before). Keeps primary output uninterrupted; the hint reads as a trailing
note about what was omitted.

### D7 - Row separators for wrapped rows (confirmed)

- `default` (bordered) theme: no change needed, the border already delimits
  rows.
- `minimal` / `compact` (borderless) themes: insert one blank line between
  rows **only when** that row wrapped to more than one line; single-line
  rows stay tight, no added vertical noise in the common case.
- `plain` / `markdown` / `gh`: unaffected (existing one-record-per-line or
  bordered semantics already delimit rows).

### D8 - Left-truncation for paths/URLs (confirmed)

Build `ShrinkTruncateLeft` now (v1), not deferred. Small function, mirrors
the existing right-truncate. Used for `path`, `assetPath`, `sourcePath`,
`targetPath`, `location` and URL columns so the meaningful tail (filename,
last path segment) survives truncation instead of the prefix.

## Proposed implementation

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
  Applies the D4 heuristic to fill in `Priority`/`Shrink` when unset, then the
  D1 drop/wrap/truncate order.
- A display-order step (D4) moves any kept `ShrinkWrap` column to the end of
  the slice used for rendering only - `planColumns` returns `kept` already in
  render order; the `Priority`-driven drop decision happens before reordering.
- `ShrinkTruncateLeft` (D8): new helper alongside the existing right-truncate
  in `wrap.go`, keeps the tail (e.g. `...target/final_report.csv`).
- Row-separator handling (D7): `renderCompact`/`renderDefault` (or whichever
  borderless renderer) insert a blank line after a row only when that row's
  `splitCellLines` produced more than one line.
- `renderTable` consumes the plan; wrapped cells already supported via
  `splitCellLines` / `WrapCell`.
- Hint written to `os.Stderr` (not `f.w`) when `len(dropped) > 0`, after the
  stdout table and row-count footer (D6).

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
- ADR under `docs/` (or `docs/adr/`) for the column-dropping decision (D1),
  covering the P1-P5 tiers, the heuristic (D4), and the display-order rule.
- `make completions` after the flag additions.
