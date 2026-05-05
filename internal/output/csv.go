package output

import (
	"encoding/csv"
	"fmt"
	"io"
)

type csvFormatter struct {
	w       io.Writer
	noColor bool
}

func (f *csvFormatter) Format(data interface{}, columns []Column) error {
	if len(columns) == 0 {
		return fmt.Errorf("CSV output requires column definitions")
	}

	rows, err := extractRows(data)
	if err != nil {
		return err
	}

	w := csv.NewWriter(f.w)

	// Header row
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Header
		if f.noColor {
			headers[i] = stripANSIText(headers[i])
		}
	}
	if err := w.Write(headers); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	// Data rows
	for _, row := range rows {
		record := make([]string, len(columns))
		for i, col := range columns {
			record[i] = extractField(row, col)
			if f.noColor {
				record[i] = stripANSIText(record[i])
			}
		}
		if err := w.Write(record); err != nil {
			return fmt.Errorf("writing CSV row: %w", err)
		}
	}

	w.Flush()
	return w.Error()
}
