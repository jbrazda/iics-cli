# Output Formatting

Reference for `--output`, table themes, and responsive table width adaptation.
See [README.md](../../README.md#configuration) for the full config file layout.

## Formats

| Flag           | Values                          | Default |
|-----------------|----------------------------------|---------|
| `--output`, `-o` | `table`, `json`, `csv`, `yaml`  | `table` |

`json`, `csv`, and `yaml` always render every field in full - responsive width adaptation
(below) applies only to `table` output.

## Themes

Set via `--theme`, `IICS_THEME`, or config `style.theme`. See
[README.md](../../README.md#responsive-table-output) for the full theme table.

## Responsive table output

On a TTY, `table` output is fit to the terminal width. Each column carries a priority
(1 = essential, 5 = reference/opaque ID) and a shrink behavior (truncate, wrap, or never
shrink). When a table doesn't fit:

1. Priority-5 columns (opaque IDs such as `federatedId`, `agentGroupId`) are dropped first,
   widest first - a truncated UUID isn't usable for a follow-up command anyway.
2. Priority-4 columns (free text like `description`, `message`) wrap onto multiple lines.
3. Priority-3 columns (paths, secondary names) wrap or truncate, per column.
4. If priority-4 columns still don't fit after wrapping to a floor width, they are dropped too.
5. Priority-2 columns truncate as a last resort.
6. If priority-1 columns alone still don't fit, output falls back to a vertical
   `PROPERTY: VALUE` block per row instead of a side-by-side table.

Any column with a wrap shrink mode is moved to the rightmost display position so its extra
physical lines never throw off the alignment of columns to its right. Column order in
`json`/`yaml`/`csv` output is never affected.

When columns are dropped, a one-line hint is printed to **stderr** naming what was hidden:

```text
# 2 column(s) hidden (FEDERATED ID, GROUP ID) - use --wide or -o json
```

### Disabling adaptation

| Setting                          | Effect                                                        |
|-----------------------------------|----------------------------------------------------------------|
| `--wide`                         | Never drop, truncate, or wrap columns for this invocation      |
| `style.responsiveTables: false`  | Same, set permanently in config                                 |
| non-TTY output (piped/redirected) | Adaptation never applies - full, untruncated output             |
| `--output csv\|json\|yaml`       | Not affected by adaptation - these formats always render in full |
| `--theme markdown` / `--theme gh` | Always render in full, regardless of TTY                       |

### Width detection

Precedence (highest first): `--width N` flag > `IICS_WIDTH` env var > detected terminal size >
`$COLUMNS` env var > `80`.

```bash
# Force a specific width, e.g. when scripting inside a fixed-width pane
iics agent list --width 100

# Always show every column, regardless of terminal size
iics agent list --wide
```
