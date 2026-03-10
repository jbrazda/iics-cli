package output

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/olekukonko/tablewriter"
)

type tableFormatter struct {
	w io.Writer
}

func (f *tableFormatter) Format(data interface{}, columns []Column) error {
	if len(columns) == 0 {
		// Fall back to JSON if no columns defined
		enc := json.NewEncoder(f.w)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}

	rows, err := extractRows(data)
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		_, _ = fmt.Fprintln(f.w, "No results found.") //nolint:errcheck
		return nil
	}

	table := tablewriter.NewTable(f.w)

	// Set headers
	headers := make([]interface{}, len(columns))
	for i, col := range columns {
		headers[i] = col.Header
	}
	table.Header(headers...)

	// Add rows
	for _, row := range rows {
		record := make([]interface{}, len(columns))
		for i, col := range columns {
			record[i] = extractField(row, col)
		}
		if err := table.Append(record); err != nil {
			return err
		}
	}

	return table.Render()
}

// extractRows converts the data to a slice of maps for processing.
func extractRows(data interface{}) ([]map[string]interface{}, error) {
	// If data is already []map[string]interface{}, use directly
	if rows, ok := data.([]map[string]interface{}); ok {
		return rows, nil
	}

	// Marshal and unmarshal through JSON for a uniform representation
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshaling data: %w", err)
	}

	// Try as array first
	var rows []map[string]interface{}
	if err := json.Unmarshal(jsonData, &rows); err == nil {
		return rows, nil
	}

	// Try as single object
	var single map[string]interface{}
	if err := json.Unmarshal(jsonData, &single); err == nil {
		return []map[string]interface{}{single}, nil
	}

	// Handle nil/empty
	v := reflect.ValueOf(data)
	if !v.IsValid() || (v.Kind() == reflect.Ptr && v.IsNil()) {
		return nil, nil
	}
	if v.Kind() == reflect.Slice && v.Len() == 0 {
		return nil, nil
	}

	return nil, fmt.Errorf("unsupported data type for table output: %T", data)
}

// extractField gets a field value from a row map using a Column definition.
func extractField(row map[string]interface{}, col Column) string {
	if col.Func != nil {
		return col.Func(row)
	}

	// Support nested fields with dot notation
	parts := strings.Split(col.Field, ".")
	var val interface{} = row

	for _, part := range parts {
		m, ok := val.(map[string]interface{})
		if !ok {
			return ""
		}
		val, ok = m[part]
		if !ok {
			return ""
		}
	}

	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%.2f", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}
