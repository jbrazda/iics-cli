# ADR-0001: Drop columns, not just truncate, when table output doesn't fit

## Status

Accepted (CR-0038).

## Context

`internal/output/table.go` sized every table column to `max(header, widest cell,
Column.Width)` with no ceiling. On a narrow terminal, wide tables (many of our
command outputs have 5+ columns including 24-character UUID columns and
unbounded free-text columns like `description`/`message`) hard-wrap into
unreadable output.

A column survey across all `cmd/*.go` files found two dominant, and
irreconcilable, offenders:

- **Opaque GUID columns** (`id`, `federatedId`, `agentGroupId`, `orgId`,
  `objectId`, `agentId`, `runtimeEnvironmentId`, `contextExternalId`, ~40
  uses): always 24 characters. A partial UUID is useless - the IICS API does
  not accept ID prefixes - so these columns cannot be truncated or wrapped
  without losing all utility.
- **Unbounded free text** (`description`, `message`, `errorMsg`, `detail`,
  config `value`, ~30 uses): wraps or truncates fine, but is frequently the
  least essential column in a list view.

Because GUID columns cannot be shrunk at all, any strategy limited to
truncating and wrapping cannot make a table with both an ID column and a wide
terminal-unfriendly description column fit a narrow terminal - something has
to be dropped entirely.

## Decision

Every table column is assigned a priority tier (1 = essential, 5 =
reference) and a shrink mode (truncate, left-truncate, wrap, or never-shrink).
When a table is wider than the detected terminal, the response is a fixed
sequence, in this order:

1. Drop priority-5 columns (opaque GUIDs not the row's primary key), widest
   first.
2. Wrap priority-4 columns (free text) to the remaining space.
3. Wrap or truncate priority-3 columns.
4. If priority-4 columns still don't fit after wrapping to a floor width
   (8 columns), drop them too.
5. Truncate priority-2 columns as a last resort.
6. If priority-1 columns alone still overflow, fall back to a vertical
   `PROPERTY: VALUE` block per row instead of a side-by-side table.

A column left unannotated gets a priority/shrink mode inferred from its
`Field`/`Width` (see `resolveDefaults` in `internal/output/table.go`); the
six highest-traffic commands (`agent`, `activitylog`, `user`, `connection`,
`objects`, `package`) annotate columns explicitly.

When columns are dropped, a one-line hint is printed to stderr naming them
(`# N column(s) hidden (...) - use --wide or -o json`), so the information
isn't silently lost - it's one flag or `--output json` away.

This never applies to non-TTY output, `--output csv|json|yaml`, or the
`markdown`/`gh` themes (always rendered in full), and can be disabled
per-invocation (`--wide`) or permanently (config `style.responsiveTables:
false`).

## Alternatives considered

- **Always fall back to a vertical layout when the table doesn't fit.**
  Rejected: a 20-row list becomes 20 stacked blocks, which is a lot of
  scrolling for what is usually still a reasonably narrow overflow.
- **Truncate everything, including GUID columns, to fit.** Rejected: a
  truncated UUID can't be pasted into a follow-up command (`agent start
  <id>`), so it's actively misleading rather than merely less useful.
- **No dropping at all; only wrap/truncate shrinkable columns.** Rejected for
  the reason in Context: it cannot make room when the overflow is caused by
  an unshrinkable GUID column, which is the common case in this codebase.

## Consequences

- Every `cmd/*.go` file that defines table columns is a candidate to touch
  (most keep working via the inferred defaults; the six high-traffic ones
  were annotated explicitly in the same change).
- A user relying on a column that's dropped on their terminal width sees a
  stderr hint and can add `--wide` or switch to `-o json` - this is a
  behavior change from prior versions, which always showed every column.
- The inferred-default heuristic is necessarily approximate for the ~16
  files that don't have explicit annotations; it can be revisited per
  command if it turns out wrong in practice.
