package output

import (
	"encoding/json"
	"fmt"
	"io"
)

type jsonFormatter struct {
	w       io.Writer
	noColor bool
}

func (f *jsonFormatter) Format(data interface{}, columns []Column) error {
	if f.noColor {
		data = sanitizeANSIData(data)
	}
	enc := json.NewEncoder(f.w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encoding JSON output: %w", err)
	}
	return nil
}
