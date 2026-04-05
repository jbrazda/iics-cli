# CR-0016: Table Output Enhancements

## Type

- [ ] New resource
- [ ] New subcommand
- [x] Enhancement - change behaviour of an existing command
- [x] Output change - add/remove/rename columns, change default format, fix display
- [x] Flag / config change - add/rename/remove a CLI flag or config field

---

## Problem

The table output system has several gaps affecting interactive and CI use:

1. There is no way to override the theme from the command line. The only path is editing
   `~/.iics/config.yaml` manually. CI pipelines have no env-var equivalent.
2. There is no `markdown` theme for pipelines that render GitHub-flavored markdown logs.
3. There is no `gh` theme matching the GitHub CLI (`gh`) table style: plain space-padded
   columns, no borders, no header decoration, always colorless - useful for scripts and
   pipelines that process the output further.
4. Table output never shows a row count, making it hard to verify completeness at a glance.
5. The header color is hardcoded to cyan; it cannot be configured.
6. The `compact` theme uses the same 2-space column gap as `minimal` and its headers are
   bold-only with no color, making it hard to distinguish from data rows.
7. Users setting up a profile for the first time have no way to preview or select a theme
   interactively - they must know the theme names and edit the config file manually.

---

## Desired Change

Implement all six items as a single coherent CR:

1. **`--theme` global flag** - persistent flag on `rootCmd`; overrides config file theme.
   Valid values: `default`, `minimal`, `compact`, `plain`, `markdown`, `gh`.
2. **`IICS_THEME` env var** - for CI pipelines; same effect as `--theme`; flag takes
   precedence over env var.
3. **`markdown` theme** - GitHub-flavored markdown table format; never uses ANSI color;
   renders regardless of TTY status.
4. **`gh` theme** - mimics the GitHub CLI (`gh`) output style: no borders, no separator
   line, no header color, plain bold headers, 2-space column gap. Like `compact` without
   any color. Renders the same in TTY and non-TTY (no color to downgrade).
5. **Row count footer** - printed after every non-empty table on a separate line.
   Format: `N rows` (or `1 row`) for all TTY themes and `plain`; `<!-- N rows -->` for
   `markdown` (renders invisibly in GitHub markdown, preserving the document).
6. **Configurable header color** - new `style.headerColor` field in `config.yaml`.
   Value is a lipgloss color string: `"6"` (cyan), `"244"` (gray), `"#FF0000"` (hex),
   or any 256-color index. Applied to `default` and `minimal` themes. `compact` uses
   `"244"` (gray) as its built-in default when `headerColor` is not set.
7. **Compact theme changes** - column gap reduced from 2 spaces to 1; header rendered in
   gray by default instead of bold-only.
8. **Interactive theme selection** - `profile add` and `profile edit` show a live sample
   table for each of the 5 themes, then present a numbered menu. The selected theme is
   saved to the global `style.theme` in `~/.iics/config.yaml`.

---

## Precedence Table (theme resolution in `resolveTableStyle`)

Highest to lowest:

| Rank | Source | Mechanism |
| ---- | ------ | --------- |
| 1 | `--no-color` flag or `NO_COLOR` env | Forces `plain`, overrides everything |
| 2 | `--theme` flag | Package-level `themeFlag` string in `cmd/root.go` |
| 3 | `IICS_THEME` env var | `os.Getenv("IICS_THEME")` |
| 4 | `style.noColor: true` in config | Forces `plain` |
| 5 | `style.theme` in config file | User preference |
| 6 | `"default"` | Built-in fallback |

The `markdown` and `gh` themes bypass the TTY-downgrade-to-plain rule in `effectiveTheme` -
they always render as-is regardless of whether stdout is a terminal (both are already colorless).

---

## Scope

### Files to modify

```text
internal/config/config.go      - StyleConfig: add HeaderColor field; Save() guard update
internal/output/formatter.go   - TableStyle: add HeaderColor field
internal/output/table.go       - markdown theme, row footer, compact changes, header color
cmd/root.go                    - --theme flag, IICS_THEME env var, resolveTableStyle() update
cmd/profile.go                 - promptThemeSelection() helper, call in add and edit commands
```

### Files to create / update (tests)

```text
internal/output/table_test.go  - new tests for markdown theme, row footer, compact gap, header color
```

### Documentation to update

```text
docs/documentation/profile.md  - document theme selection step in profile add and edit
README.md                      - update theme table, add --theme flag to global flags table,
                                  add IICS_THEME to env vars table, add headerColor to config example
```

### Do NOT touch

```text
internal/client/    - no API changes
cmd/*.go            - all files except root.go and profile.go
```

---

## Detailed Design

### 1. `internal/config/config.go`

Add `HeaderColor` to `StyleConfig`:

```go
type StyleConfig struct {
    Theme       string `yaml:"theme,omitempty"       mapstructure:"theme"`
    NoColor     bool   `yaml:"noColor,omitempty"     mapstructure:"noColor"`
    HeaderColor string `yaml:"headerColor,omitempty" mapstructure:"headerColor"`
}
```

Update `Save()` guard condition:

```go
// before
if c.Style.Theme != "" || c.Style.NoColor {

// after
if c.Style.Theme != "" || c.Style.NoColor || c.Style.HeaderColor != "" {
```

---

### 2. `internal/output/formatter.go`

Add `HeaderColor` to `TableStyle`:

```go
// TableStyle carries resolved styling options for the table formatter.
// Theme selects the visual style: "default", "minimal", "compact", "plain", or "markdown".
// NoColor disables all ANSI color and forces the "plain" theme.
// HeaderColor is a lipgloss color string (e.g. "6", "244", "#FF0000").
// Empty string means use the theme built-in default.
type TableStyle struct {
    Theme       string
    NoColor     bool
    HeaderColor string
}
```

No changes to `ParseFormat` - `markdown` is a theme, not an output format.

---

### 3. `cmd/root.go`

**New package-level variable:**

```go
themeFlag string
```

**New persistent flag in `init()`** (add after the `--no-color` line):

```go
rootCmd.PersistentFlags().StringVar(&themeFlag, "theme", "",
    "table theme: default|minimal|compact|plain|markdown|gh (overrides config)")
```

**Updated `resolveTableStyle()`** with full precedence chain:

```go
func resolveTableStyle(cfg *config.Config) output.TableStyle {
    style := output.TableStyle{Theme: "default"}

    // Config-level preferences (lowest named precedence)
    if cfg != nil {
        if cfg.Style.Theme != "" {
            style.Theme = cfg.Style.Theme
        }
        if cfg.Style.HeaderColor != "" {
            style.HeaderColor = cfg.Style.HeaderColor
        }
    }

    // IICS_THEME env var (overrides config)
    if v := os.Getenv("IICS_THEME"); v != "" {
        style.Theme = v
    }

    // --theme flag (overrides env var)
    if themeFlag != "" {
        style.Theme = themeFlag
    }

    // config.noColor (overrides theme, but below --no-color flag)
    if cfg != nil && cfg.Style.NoColor {
        style.NoColor = true
        style.Theme = "plain"
    }

    // --no-color flag / NO_COLOR env (highest precedence)
    if noColor || os.Getenv("NO_COLOR") != "" {
        style.NoColor = true
        style.Theme = "plain"
    }

    return style
}
```

**Updated `getFormatter()`:** no change needed; it already calls `resolveTableStyle(cfg)`.

**Updated `init()` global flags documentation comment** in README - see Documentation section.

---

### 4. `internal/output/table.go`

#### 4a - Constants

Add `compactColGap` alongside the existing `colGap`:

```go
const (
    cellPad       = 1 // padding on each side inside a bordered cell
    colGap        = 2 // spaces between columns in minimal theme
    compactColGap = 1 // spaces between columns in compact theme
)
```

#### 4b - Header style helpers

Remove the two package-level vars and three static functions:

```go
// REMOVE these:
var (
    colorHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
    boldHeaderStyle  = lipgloss.NewStyle().Bold(true)
)
func colorHeader(s string) string { return colorHeaderStyle.Render(s) }
func boldHeader(s string) string  { return boldHeaderStyle.Render(s) }
```

Replace with a single constructor:

```go
// makeHeaderStyle returns a lipgloss.Style for header cells.
// color is a lipgloss color string. If empty, only Bold is applied (no color).
func makeHeaderStyle(color string) lipgloss.Style {
    s := lipgloss.NewStyle().Bold(true)
    if color != "" {
        s = s.Foreground(lipgloss.Color(color))
    }
    return s
}
```

#### 4c - `effectiveTheme` - bypass TTY check for markdown and gh

```go
func effectiveTheme(w io.Writer, style TableStyle) string {
    // markdown and gh are always colorless - no TTY downgrade needed
    if style.Theme == "markdown" || style.Theme == "gh" {
        return style.Theme
    }
    if style.NoColor || !isTerminal(w) {
        return "plain"
    }
    if style.Theme == "" {
        return "default"
    }
    return style.Theme
}
```

#### 4d - `renderTable` - add markdown case, pass style through

```go
func renderTable(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int, theme string, style TableStyle) {
    switch theme {
    case "minimal":
        renderMinimal(w, rows, columns, widths, style)
    case "compact":
        renderCompact(w, rows, columns, widths, style)
    case "plain":
        renderPlain(w, rows, columns, widths)
    case "markdown":
        renderMarkdown(w, rows, columns, widths)
    case "gh":
        renderGH(w, rows, columns, widths)
    default:
        renderDefault(w, rows, columns, widths, style)
    }
}
```

#### 4e - Row count footer in `Format()`

After `renderTable` returns:

```go
func (f *tableFormatter) Format(data interface{}, columns []Column) error {
    // ... existing early-return for len(columns)==0 and len(rows)==0 ...

    theme := effectiveTheme(f.w, f.style)
    widths := computeColWidths(rows, columns)
    renderTable(f.w, rows, columns, widths, theme, f.style)

    // Row count footer
    n := len(rows)
    if theme == "markdown" {
        if n == 1 {
            _, _ = fmt.Fprintln(f.w, "<!-- 1 row -->")
        } else {
            _, _ = fmt.Fprintf(f.w, "<!-- %d rows -->\n", n)
        }
    } else {
        if n == 1 {
            _, _ = fmt.Fprintln(f.w, "1 row")
        } else {
            _, _ = fmt.Fprintf(f.w, "%d rows\n", n)
        }
    }

    return nil
}
```

The footer is skipped when `len(rows) == 0` because `Format()` already returns early with
`"No results found."` in that case.

#### 4f - `renderDefault` - header color plumbing

```go
func renderDefault(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int, style TableStyle) {
    b := unicodeBorders
    color := style.HeaderColor
    if color == "" {
        color = "6" // built-in default: cyan
    }
    hStyle := makeHeaderStyle(color)
    headerFn := func(s string) string { return hStyle.Render(s) }

    headers := headerStrings(columns)
    _, _ = fmt.Fprintln(w, hSep(b, widths, b.topLeft, b.topMid, b.topRight))
    _, _ = fmt.Fprintln(w, borderedRow(b, headers, widths, headerFn))
    _, _ = fmt.Fprintln(w, hSep(b, widths, b.midLeft, b.midMid, b.midRight))
    for _, row := range rows {
        _, _ = fmt.Fprintln(w, borderedRow(b, dataCells(row, columns), widths, noStyle))
    }
    _, _ = fmt.Fprintln(w, hSep(b, widths, b.botLeft, b.botMid, b.botRight))
}
```

#### 4g - `renderMinimal` - header color plumbing

```go
func renderMinimal(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int, style TableStyle) {
    color := style.HeaderColor
    if color == "" {
        color = "6" // built-in default: cyan
    }
    hStyle := makeHeaderStyle(color)
    headerFn := func(s string) string { return hStyle.Render(s) }

    headers := headerStrings(columns)
    var hdr strings.Builder
    for i, h := range headers {
        hdr.WriteString(headerFn(padRight(h, widths[i])))
        if i < len(widths)-1 {
            hdr.WriteString(strings.Repeat(" ", colGap))
        }
    }
    _, _ = fmt.Fprintln(w, hdr.String())

    var ul strings.Builder
    for i, wd := range widths {
        ul.WriteString(strings.Repeat("─", wd))
        if i < len(widths)-1 {
            ul.WriteString(strings.Repeat(" ", colGap))
        }
    }
    _, _ = fmt.Fprintln(w, ul.String())

    for _, row := range rows {
        cells := dataCells(row, columns)
        var rb strings.Builder
        for i, cell := range cells {
            rb.WriteString(padRight(cell, widths[i]))
            if i < len(widths)-1 {
                rb.WriteString(strings.Repeat(" ", colGap))
            }
        }
        _, _ = fmt.Fprintln(w, rb.String())
    }
}
```

#### 4h - `renderCompact` - gray header, 1-space gap

```go
func renderCompact(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int, style TableStyle) {
    color := style.HeaderColor
    if color == "" {
        color = "244" // built-in default for compact: gray
    }
    hStyle := makeHeaderStyle(color)
    headerFn := func(s string) string { return hStyle.Render(s) }

    headers := headerStrings(columns)
    var hdr strings.Builder
    for i, h := range headers {
        hdr.WriteString(headerFn(padRight(h, widths[i])))
        if i < len(widths)-1 {
            hdr.WriteString(strings.Repeat(" ", compactColGap))
        }
    }
    _, _ = fmt.Fprintln(w, hdr.String())

    for _, row := range rows {
        cells := dataCells(row, columns)
        var rb strings.Builder
        for i, cell := range cells {
            rb.WriteString(padRight(cell, widths[i]))
            if i < len(widths)-1 {
                rb.WriteString(strings.Repeat(" ", compactColGap))
            }
        }
        _, _ = fmt.Fprintln(w, rb.String())
    }
}
```

#### 4i - New `renderMarkdown`

```go
// renderMarkdown renders a GitHub-Flavored Markdown table.
// No ANSI color is used regardless of TTY or style settings.
// Column widths are padded for alignment in raw source.
func renderMarkdown(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int) {
    headers := headerStrings(columns)

    // Header row
    var hdr strings.Builder
    hdr.WriteString("| ")
    for i, h := range headers {
        hdr.WriteString(padRight(h, widths[i]))
        if i < len(widths)-1 {
            hdr.WriteString(" | ")
        }
    }
    hdr.WriteString(" |")
    _, _ = fmt.Fprintln(w, hdr.String())

    // Separator row
    var sep strings.Builder
    sep.WriteString("| ")
    for i, wd := range widths {
        sep.WriteString(strings.Repeat("-", wd))
        if i < len(widths)-1 {
            sep.WriteString(" | ")
        }
    }
    sep.WriteString(" |")
    _, _ = fmt.Fprintln(w, sep.String())

    // Data rows
    for _, row := range rows {
        cells := dataCells(row, columns)
        var rb strings.Builder
        rb.WriteString("| ")
        for i, cell := range cells {
            rb.WriteString(padRight(cell, widths[i]))
            if i < len(widths)-1 {
                rb.WriteString(" | ")
            }
        }
        rb.WriteString(" |")
        _, _ = fmt.Fprintln(w, rb.String())
    }
}
```

Sample output for a 3-column table:

```text
| ID   | NAME     | STATUS |
| ---- | -------- | ------ |
| abc1 | dev-org  | active |
| abc2 | prod-org | active |
<!-- 2 rows -->
```

#### 4j - New `renderGH`

Mimics the GitHub CLI (`gh`) output style: no borders, no separator line, plain
space-padded columns, headers in plain text (no color, no bold). Identical rendering
in TTY and non-TTY - no ANSI codes at all.

```go
// renderGH renders a plain space-padded table with no borders or decorations,
// matching the GitHub CLI (gh) table output style.
// No ANSI color is used regardless of TTY or style settings.
func renderGH(w io.Writer, rows []map[string]interface{}, columns []Column, widths []int) {
    headers := headerStrings(columns)

    var hdr strings.Builder
    for i, h := range headers {
        hdr.WriteString(padRight(h, widths[i]))
        if i < len(widths)-1 {
            hdr.WriteString(strings.Repeat(" ", colGap))
        }
    }
    _, _ = fmt.Fprintln(w, hdr.String())

    for _, row := range rows {
        cells := dataCells(row, columns)
        var rb strings.Builder
        for i, cell := range cells {
            rb.WriteString(padRight(cell, widths[i]))
            if i < len(widths)-1 {
                rb.WriteString(strings.Repeat(" ", colGap))
            }
        }
        _, _ = fmt.Fprintln(w, rb.String())
    }
}
```

Sample output:

```text
NAME          REGION  STATUS
dev-org       USW3    active
prod-org      EMEA    active
2 rows
```

The `gh` theme differs from `compact` in two ways: no color on headers at all (not even
gray), and uses 2-space gaps (`colGap`) instead of 1-space `compactColGap`.

---

### 5. `cmd/profile.go`

Add a package-private helper that renders the theme preview and prompts for selection.
This helper lives in `profile.go` (not `internal/config/prompt.go`) to avoid introducing
a new import dependency between `internal/config` and `internal/output`.

```go
// sampleThemeData and sampleThemeCols are used for the interactive theme preview.
var sampleThemeCols = []output.Column{
    {Header: "NAME",   Field: "name",   Width: 12},
    {Header: "REGION", Field: "region", Width: 6},
    {Header: "STATUS", Field: "status", Width: 8},
}

var sampleThemeData = []map[string]interface{}{
    {"name": "dev-org",  "region": "USW3", "status": "active"},
    {"name": "prod-org", "region": "EMEA", "status": "active"},
}

// promptThemeSelection shows a live sample for each theme and returns the
// theme name the user selected. currentTheme is shown as the default.
// Writes prompts and previews to stderr so stdout piping is not contaminated.
// Returns "" and no error when the user presses Enter to keep the current theme.
func promptThemeSelection(currentTheme string) (string, error) {
    themes := []string{"default", "minimal", "compact", "plain", "markdown", "gh"}

    _, _ = fmt.Fprintln(os.Stderr, "\nTable theme selection - preview of each theme:\n")

    for i, theme := range themes {
        _, _ = fmt.Fprintf(os.Stderr, "  [%d] %s\n", i+1, theme)
        f := output.New(output.FormatTable, os.Stderr, output.TableStyle{Theme: theme})
        _ = f.Format(sampleThemeData, sampleThemeCols)
        _, _ = fmt.Fprintln(os.Stderr)
    }

    currentIdx := 0
    for i, t := range themes {
        if t == currentTheme {
            currentIdx = i + 1
            break
        }
    }

    r := bufio.NewReader(os.Stdin)
    for {
        if currentIdx > 0 {
            _, _ = fmt.Fprintf(os.Stderr,
                "Select theme [1-%d, current: %d (%s), Enter to keep]: ",
                len(themes), currentIdx, currentTheme)
        } else {
            _, _ = fmt.Fprintf(os.Stderr, "Select theme [1-%d]: ", len(themes))
        }

        line, err := r.ReadString('\n')
        if err != nil {
            return "", fmt.Errorf("reading theme selection: %w", err)
        }
        line = strings.TrimSpace(line)

        if line == "" {
            return currentTheme, nil // keep existing
        }

        var n int
        if _, scanErr := fmt.Sscanf(line, "%d", &n); scanErr == nil && n >= 1 && n <= len(themes) {
            return themes[n-1], nil
        }
        _, _ = fmt.Fprintf(os.Stderr,
            "Invalid selection %q. Enter a number from 1 to %d.\n", line, len(themes))
    }
}
```

**Required new import in `profile.go`:** `"bufio"` and `"strings"` (check which are already present).

**`newProfileAddCmd` change** - insert after `if makeDefault { ... }` block and before `cfg.Save(...)`:

```go
// Interactive theme selection (terminal only)
if config.IsTerminal() {
    selected, styleErr := promptThemeSelection(cfg.Style.Theme)
    if styleErr != nil {
        _, _ = fmt.Fprintf(cmd.ErrOrStderr(),
            "Warning: theme selection skipped: %v\n", styleErr)
    } else if selected != "" {
        cfg.Style.Theme = selected
    }
}

if err := cfg.Save(cfgFile); err != nil {
    return fmt.Errorf("saving config: %w", err)
}
```

**`newProfileEditCmd` change** - insert after the `cfg.Profiles[name] = p` / `if makeDefault` block
and before the existing `cfg.Save(...)` call (around line 187):

```go
// Interactive theme selection (terminal only)
if config.IsTerminal() {
    selected, styleErr := promptThemeSelection(cfg.Style.Theme)
    if styleErr != nil {
        _, _ = fmt.Fprintf(cmd.ErrOrStderr(),
            "Warning: theme selection skipped: %v\n", styleErr)
    } else if selected != "" {
        cfg.Style.Theme = selected
    }
}

if err := cfg.Save(cfgFile); err != nil {
    return fmt.Errorf("saving config: %w", err)
}
```

---

### 6. Modified function signatures in `internal/output/table.go`

| Old | New |
| --- | --- |
| `renderTable(w, rows, columns, widths, theme string)` | `renderTable(w, rows, columns, widths, theme string, style TableStyle)` |
| `renderDefault(w, rows, columns, widths)` | `renderDefault(w, rows, columns, widths, style TableStyle)` |
| `renderMinimal(w, rows, columns, widths)` | `renderMinimal(w, rows, columns, widths, style TableStyle)` |
| `renderCompact(w, rows, columns, widths)` | `renderCompact(w, rows, columns, widths, style TableStyle)` |
| `renderPlain(w, rows, columns, widths)` | no change |

---

## Config File After CR

```yaml
defaultProfile: dev
style:
  theme: minimal        # default | minimal | compact | plain | markdown
  headerColor: "244"    # lipgloss color: "6"=cyan, "244"=gray, "#FF0000"=hex
  noColor: false
profiles:
  dev:
    username: dev@company.com
    password: ""
    region: USW3
```

---

## Documentation Updates

### `README.md`

Update the **Themes** table:

| Theme | Description |
| ----- | ----------- |
| `default` | Unicode rounded borders, cyan bold headers (TTY only) |
| `minimal` | No borders, colored bold headers with unicode underline |
| `compact` | No borders, gray bold headers, dense layout (1-space column gap) |
| `plain` | ASCII borders, no color - used automatically for non-TTY output |
| `markdown` | GitHub-flavored markdown table, no color, always rendered regardless of TTY |
| `gh` | GitHub CLI-style: no borders, no separator, plain headers, no color, always rendered regardless of TTY |

Add `--theme` to the **Global flags** table.

Add `IICS_THEME` to the **Environment variable overrides** table.

Add `headerColor` to the config example under `style:`.

### `docs/documentation/profile.md`

Add a section documenting the theme selection step in `profile add` and `profile edit`.

---

## Testing Instructions

### New unit tests in `internal/output/table_test.go`

Use `bytes.Buffer` as the writer (non-TTY); set `TableStyle.Theme` explicitly to test non-plain themes.

| Test | What to assert |
| ---- | -------------- |
| `TestMarkdownTheme` | Output contains `\|`, `---` separator row, `<!-- 2 rows -->` footer |
| `TestMarkdownThemeNonTTY` | `effectiveTheme` returns `"markdown"` even with a non-TTY writer |
| `TestGHTheme` | Output has no `\|`, no separator row, no ANSI codes, `2 rows` footer |
| `TestGHThemeNonTTY` | `effectiveTheme` returns `"gh"` even with a non-TTY writer |
| `TestRowFooterPlural` | `plain` theme with 3 rows appends `"3 rows\n"` |
| `TestRowFooterSingular` | `plain` theme with 1 row appends `"1 row\n"` |
| `TestRowFooterMarkdown` | `markdown` theme appends `"<!-- 3 rows -->\n"` |
| `TestCompactGap` | `compact` theme output between columns contains exactly 1 space, not 2 |
| `TestHeaderColorPropagation` | `TableStyle{Theme:"default", HeaderColor:"244"}` produces output containing the header text without panic |

Existing tests that `strings.Contains` the header text or data cells remain valid - the footer is new text appended after.

### Manual verification

```bash
# Build first
/opt/local/bin/go build ./...

# --theme flag overrides config
iics --theme minimal objects list

# IICS_THEME env var for CI
IICS_THEME=compact iics objects list

# --no-color still overrides --theme
iics --theme minimal --no-color objects list

# markdown theme (always renders, even when piped)
iics --theme markdown objects list | cat

# gh theme - plain output like github cli
iics --theme gh objects list

# row count visible after table
iics --theme plain objects list

# Interactive theme selection during profile setup
iics profile add test-theme-cr

# Run tests
/opt/local/bin/go test ./...
/opt/local/bin/go vet ./...
```

---

## Acceptance Criteria

- [ ] `iics --theme minimal objects list` renders with minimal theme
- [ ] `IICS_THEME=compact iics objects list` renders with compact theme
- [ ] `--theme` flag overrides `IICS_THEME` env var
- [ ] `--no-color` overrides `--theme`
- [ ] `--theme markdown` renders GFM table with `<!-- N rows -->` footer
- [ ] `--theme markdown` renders without ANSI color even when stdout is a TTY
- [ ] `--theme markdown` renders even when stdout is not a TTY (no downgrade to `plain`)
- [ ] `--theme gh` renders plain space-padded columns with no borders, no decorations, no ANSI codes
- [ ] `--theme gh` renders identically in TTY and non-TTY contexts
- [ ] All non-empty TTY table outputs append `N rows` (or `1 row`) on a new line after the table
- [ ] `"No results found."` is shown for empty data with no footer line
- [ ] `style.headerColor: "244"` in config renders gray headers in `default` and `minimal` themes
- [ ] `compact` theme uses 1-space column gaps
- [ ] `compact` theme uses gray headers when `headerColor` is not set in config
- [ ] `iics profile add` shows theme preview and numbered menu after credential prompts
- [ ] `iics profile edit` shows theme preview and numbered menu after credential prompts
- [ ] Theme selection saves to `style.theme` in `~/.iics/config.yaml`
- [ ] Profile credentials are also saved when theme selection runs
- [ ] Theme selection is skipped with a warning when stdin is not a terminal
- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] All existing tests continue to pass

---

## Do NOT

- Refactor or add comments to code outside the scope listed above
- Add a `--header-color` flag - header color is config-file-only
- Add theme name validation in `resolveTableStyle` - unknown theme names fall through to `renderDefault`
- Add per-profile style settings - `style` is global
- Import `internal/output` from `internal/config` - the theme selection helper belongs in `cmd/profile.go`
- Change `ParseFormat` to accept `"markdown"` as an output format - it is a theme, not a format
- Add `Co-Authored-By` trailers to commit messages
- Call `os.Exit()` - return errors from `RunE`
