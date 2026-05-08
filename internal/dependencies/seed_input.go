package dependencies

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseSeedIDsFromInput parses object seed IDs from JSON, CSV, or YAML input.
// Supported shapes are objects-list style rows that contain an ID column/field.
func ParseSeedIDsFromInput(data []byte) ([]string, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, fmt.Errorf("stdin is empty")
	}

	switch detectInputFormat(trimmed) {
	case "json":
		return parseSeedIDsJSON([]byte(trimmed))
	case "csv":
		return parseSeedIDsCSV([]byte(trimmed))
	default:
		return parseSeedIDsYAML([]byte(trimmed))
	}
}

func detectInputFormat(trimmed string) string {
	if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
		return "json"
	}
	firstLine := trimmed
	if idx := strings.IndexByte(trimmed, '\n'); idx >= 0 {
		firstLine = strings.TrimSpace(trimmed[:idx])
	}
	if strings.Contains(firstLine, ",") ||
		strings.EqualFold(firstLine, "id") ||
		strings.EqualFold(firstLine, "path") ||
		strings.EqualFold(firstLine, "location") {
		return "csv"
	}
	return "yaml"
}

func parseSeedIDsJSON(data []byte) ([]string, error) {
	var objs []map[string]interface{}
	if err := json.Unmarshal(data, &objs); err != nil {
		return nil, fmt.Errorf("parsing stdin as JSON array: %w", err)
	}
	ids := extractIDsFromMaps(objs)
	if len(ids) == 0 {
		return nil, fmt.Errorf("no IDs found in JSON input")
	}
	return ids, nil
}

func parseSeedIDsCSV(data []byte) ([]string, error) {
	r := csv.NewReader(strings.NewReader(string(data)))
	headers, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV headers: %w", err)
	}
	idIdx := -1
	for i, h := range headers {
		if strings.EqualFold(strings.TrimSpace(h), "id") {
			idIdx = i
			break
		}
	}
	if idIdx < 0 {
		return nil, fmt.Errorf("CSV must include an ID column")
	}
	ids := make([]string, 0, 64)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading CSV row: %w", err)
		}
		if idIdx >= len(row) {
			continue
		}
		id := strings.TrimSpace(row[idIdx])
		if id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no IDs found in CSV input")
	}
	return ids, nil
}

func parseSeedIDsYAML(data []byte) ([]string, error) {
	var objs []map[string]interface{}
	if err := yaml.Unmarshal(data, &objs); err != nil {
		return nil, fmt.Errorf("parsing stdin as YAML list: %w", err)
	}
	ids := extractIDsFromMaps(objs)
	if len(ids) == 0 {
		return nil, fmt.Errorf("no IDs found in YAML input")
	}
	return ids, nil
}

func extractIDsFromMaps(rows []map[string]interface{}) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		id := ""
		for k, v := range row {
			if strings.EqualFold(k, "id") {
				id = strings.TrimSpace(fmt.Sprintf("%v", v))
				break
			}
		}
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
