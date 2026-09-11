package output

import (
	"fmt"
	"io"
	"os"
)

// Format is the output format type.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatCSV   Format = "csv"
	FormatYAML  Format = "yaml"
)

// ShrinkMode controls how a column behaves when the table is wider than the
// terminal and must be narrowed to fit.
type ShrinkMode int

const (
	// ShrinkTruncate right-truncates the cell content with an ellipsis. Default.
	ShrinkTruncate ShrinkMode = iota
	// ShrinkTruncateLeft truncates from the left, keeping the tail (paths, URLs).
	ShrinkTruncateLeft
	// ShrinkWrap hard-wraps the cell content onto multiple physical lines.
	ShrinkWrap
	// ShrinkNever never squeezes the column; it is dropped instead when it
	// doesn't fit.
	ShrinkNever
)

// Column defines a table column for human-readable output.
type Column struct {
	Header string
	Field  string
	Width  int // minimum/floor width
	// MaxWidth caps the column's natural width even when the terminal is wide
	// enough to show it in full. 0 means unbounded.
	MaxWidth int
	// Priority ranks the column for width-adaptation purposes: 1 (essential,
	// never dropped or shrunk) through 5 (reference, dropped first). 0 means
	// unset - a default priority and shrink mode are inferred from Field and
	// Width.
	Priority int
	// Shrink controls how the column narrows under width pressure. Only
	// applied when Priority is explicitly set; an unset Priority infers both
	// Priority and Shrink together.
	Shrink ShrinkMode
	Func   func(v interface{}) string
}

// TableStyle carries resolved styling options for the table formatter.
// Theme selects the visual style: "default", "minimal", "compact", "plain", "markdown", or "gh".
// NoColor disables all ANSI color and forces the "plain" theme.
// HeaderColor is a lipgloss color string (e.g. "6", "244", "#FF0000").
// Empty string means use the theme built-in default.
type TableStyle struct {
	Theme       string
	NoColor     bool
	HeaderColor string
	// Width is an explicit terminal-width override for responsive tables
	// (from --width or IICS_WIDTH). 0 means auto-detect.
	Width int
	// SkipAdapt disables width adaptation (dropping/truncating/wrapping
	// columns) entirely - set by --wide or config style.responsiveTables:
	// false. Detected/overridden width, if any, is still used for layout
	// info but never causes a column to be dropped, truncated, or wrapped.
	SkipAdapt bool
}

// Formatter is the interface for rendering API results.
type Formatter interface {
	Format(data interface{}, columns []Column) error
}

// New returns a Formatter for the given format and optional table style.
func New(format Format, w io.Writer, style TableStyle) Formatter {
	if w == nil {
		w = os.Stdout
	}
	switch format {
	case FormatJSON:
		return &jsonFormatter{w: w, noColor: style.NoColor}
	case FormatCSV:
		return &csvFormatter{w: w, noColor: style.NoColor}
	case FormatYAML:
		return &yamlFormatter{w: w, noColor: style.NoColor}
	default:
		return newTableFormatter(w, style)
	}
}

// ParseFormat parses a format string into a Format type.
func ParseFormat(s string) (Format, error) {
	switch s {
	case "table", "":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "csv":
		return FormatCSV, nil
	case "yaml":
		return FormatYAML, nil
	default:
		return "", fmt.Errorf("unknown output format %q; valid formats: table, json, csv, yaml", s)
	}
}
