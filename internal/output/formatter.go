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

// Column defines a table column for human-readable output.
type Column struct {
	Header string
	Field  string
	Width  int
	Func   func(v interface{}) string
}

// TableStyle carries resolved styling options for the table formatter.
// Theme selects the visual style: "default", "minimal", "compact", or "plain".
// NoColor disables all ANSI color and forces the "plain" theme.
type TableStyle struct {
	Theme   string
	NoColor bool
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
		return &jsonFormatter{w: w}
	case FormatCSV:
		return &csvFormatter{w: w}
	case FormatYAML:
		return &yamlFormatter{w: w}
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
