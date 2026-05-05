package output

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

type yamlFormatter struct {
	w       io.Writer
	noColor bool
}

func (f *yamlFormatter) Format(data interface{}, columns []Column) error {
	// Extract rows for uniform handling; wrap slices in {"objects": [...]} envelope.
	rows, err := extractRows(data)
	if err != nil {
		return fmt.Errorf("extracting rows for YAML: %w", err)
	}

	var out interface{}
	if rows != nil {
		out = map[string]interface{}{"objects": rows}
	} else {
		out = data
	}
	if f.noColor {
		out = sanitizeANSIData(out)
	}

	enc := yaml.NewEncoder(f.w)
	enc.SetIndent(2)
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("encoding YAML: %w", err)
	}
	return enc.Close()
}
