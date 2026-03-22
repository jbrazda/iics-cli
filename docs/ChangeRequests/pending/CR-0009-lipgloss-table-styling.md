# CR-0009: Lip Gloss Table Styling and Configurable Themes

## CR Type

- [x] **Enhancement** - replace monochrome table output with styled, configurable rendering

---

## Problem

1. **No color or styling in table output.** `tablewriter` renders plain ASCII boxes with
   no color support. The `gh` CLI (which users are familiar with) renders tables with
   colored headers, clean unicode borders, and compact alignment.

2. **`--no-color` flag is a no-op.** The flag is declared in `cmd/root.go` but is never
   read by the output layer, so CI scripts receive styled output even when explicitly
   disabled.

3. **`NO_COLOR` env var is not respected.** The POSIX `NO_COLOR` convention
   (https://no-color.org) is not checked anywhere in the CLI.

4. **No user-configurable themes.** There is no way to choose between compact, minimal,
   or plain output styles without editing source code.

---

## Solution: `charmbracelet/lipgloss`

`github.com/charmbracelet/lipgloss` is the same library used by the `gh` CLI for terminal
styling. It is pure Go, composable, terminal-width aware, and respects `NO_COLOR`
automatically via `go-isatty` (already an indirect dependency).

| Choice | Reason |
| ------ | ------ |
| `lipgloss` | Used by gh CLI; rich border/color API; respects NO_COLOR; widely adopted |
| `tablewriter` (keep) | No color/style support in v1.x API; would require an external wrapper |
| `go-pretty` | Extra dep; heavier than needed for styled tables |

---

## Desired Change

### 1. Add `StyleConfig` to `Config` in `internal/config/config.go`

Style is global (not per-profile) because it is a user interface preference, not an
org-specific connection setting.

```go
// StyleConfig holds user preferences for table output styling.
type StyleConfig struct {
    Theme   string `yaml:"theme,omitempty"   mapstructure:"theme"`
    NoColor bool   `yaml:"noColor,omitempty" mapstructure:"noColor"`
}

type Config struct {
    DefaultProfile string              `yaml:"defaultProfile" mapstructure:"defaultProfile"`
    Profiles       map[string]*Profile `yaml:"profiles"       mapstructure:"profiles"`
    Style          StyleConfig         `yaml:"style,omitempty" mapstructure:"style"`
}
```

### 2. Add `TableStyle` type and update `output.New()` in `internal/output/formatter.go`

```go
// TableStyle carries resolved styling options for the table formatter.
type TableStyle struct {
    Theme   string // "default" | "minimal" | "compact" | "plain"
    NoColor bool
}

// New returns a Formatter for the given format.
func New(format Format, w io.Writer, style TableStyle) Formatter {
    ...
    default:
        return newTableFormatter(w, style)
    }
}
```

Only one call site exists: `getFormatter()` in `cmd/root.go`, updated in step 4.

### 3. Replace `tablewriter` with lipgloss renderer in `internal/output/table.go`

Remove `github.com/olekukonko/tablewriter`. Implement a custom lipgloss-based table
renderer. The existing `extractRows()` and `extractField()` helpers are pure
data-extraction logic with no styling dependency and are unchanged.

**Theme definitions:**

| Theme | Borders | Header style | Use case |
| ----- | ------- | ------------ | -------- |
| `default` | Unicode rounded | Bold + cyan/blue fg | Interactive terminal (gh-like) |
| `minimal` | None; header underline only | Bold + fg color | Narrow/compact output |
| `compact` | None | Bold | Dense listings |
| `plain` | ASCII `+`, `-`, `\|` | UPPERCASE | Scripts, non-TTY, `--no-color` |

Auto-detect non-TTY: when stdout is not a terminal (checked via `go-isatty`), the
renderer automatically downgrades to `plain` regardless of the configured theme.

**Renderer algorithm:**

1. Compute column widths: `max(header_len, max_cell_len)`, capped by `Column.Width`
   if set, and constrained to available terminal width.
2. Apply `lipgloss.Style` to header cells based on theme.
3. Render optional top border, header row, separator.
4. Render each data row with optional alternating row styling.
5. Render optional bottom border.

### 4. Wire style into `getFormatter()` in `cmd/root.go`

```go
// resolveTableStyle returns the effective TableStyle for the current invocation.
// Precedence (highest first): --no-color flag / NO_COLOR env > config file.
func resolveTableStyle(cfg *config.Config) output.TableStyle {
    style := output.TableStyle{Theme: "default"}
    if cfg != nil && cfg.Style.Theme != "" {
        style.Theme = cfg.Style.Theme
    }
    if noColor || (cfg != nil && cfg.Style.NoColor) || os.Getenv("NO_COLOR") != "" {
        style.NoColor = true
        style.Theme = "plain"
    }
    return style
}

func getFormatter() (output.Formatter, error) {
    f, err := output.ParseFormat(outputFmt)
    if err != nil {
        return nil, err
    }
    cfg, _ := loadConfig() // best-effort; nil cfg uses defaults
    return output.New(f, os.Stdout, resolveTableStyle(cfg)), nil
}
```

### 5. Update go.mod

Add direct dependency:

```
github.com/charmbracelet/lipgloss v1.x
```

Remove direct dependency (no other usages remain):

```
github.com/olekukonko/tablewriter
```

---

## Config Schema After Change

```yaml
defaultProfile: prod
style:
  theme: default     # default | minimal | compact | plain
  noColor: false     # true = persistent no-color (same as --no-color flag)
profiles:
  prod:
    region: us1
    username: ...
    password: ...
```

---

## Scope

### Files to MODIFY

```text
go.mod                              # add lipgloss, remove tablewriter
internal/config/config.go           # add StyleConfig struct and Style field to Config
internal/output/formatter.go        # add TableStyle type, update New() signature
internal/output/table.go            # replace tablewriter with lipgloss renderer
internal/output/formatter_test.go   # update New() call sites in tests
cmd/root.go                         # add resolveTableStyle(), update getFormatter()
```

### Files to READ (context only)

```text
internal/output/formatter.go        # Formatter interface, New() factory, Column type
internal/output/table.go            # tableFormatter, extractRows(), extractField()
internal/config/config.go           # Config and Profile structs, Save/Load
cmd/root.go                         # noColor var, getFormatter(), loadConfig()
```

---

## Verification

```bash
/opt/local/bin/go get github.com/charmbracelet/lipgloss
/opt/local/bin/go build ./...
/opt/local/bin/go vet ./...
/opt/local/bin/go test ./internal/output/...

# Default theme (colored headers, unicode borders)
iics objects list --limit 5

# Plain via flag
iics objects list --limit 5 --no-color

# Plain via env (POSIX NO_COLOR)
NO_COLOR=1 iics objects list --limit 5

# Minimal theme (set style.theme: minimal in ~/.iics/config.yaml)
iics objects list --limit 5

# Non-TTY auto-plain
iics objects list --limit 5 | cat
```
