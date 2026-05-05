package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestJSONFormatter(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatJSON, &buf, TableStyle{})

	data := []map[string]interface{}{
		{"id": "1", "name": "test"},
		{"id": "2", "name": "test2"},
	}

	if err := f.Format(data, nil); err != nil {
		t.Fatalf("Format() error: %v", err)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON output is not valid: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
}

func TestTableFormatter(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatTable, &buf, TableStyle{})

	data := []map[string]interface{}{
		{"id": "abc123", "name": "My Connection", "type": "TOOLKIT"},
		{"id": "def456", "name": "Another Conn", "type": "JDBC"},
	}

	columns := []Column{
		{Header: "ID", Field: "id", Width: 10},
		{Header: "NAME", Field: "name"},
		{Header: "TYPE", Field: "type"},
	}

	if err := f.Format(data, columns); err != nil {
		t.Fatalf("Format() error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "abc123") {
		t.Errorf("table output should contain 'abc123', got:\n%s", output)
	}
	if !strings.Contains(output, "My Connection") {
		t.Errorf("table output should contain 'My Connection', got:\n%s", output)
	}
}

func TestTableFormatterEmptyData(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatTable, &buf, TableStyle{})

	data := []map[string]interface{}{}
	columns := []Column{{Header: "ID", Field: "id"}}

	if err := f.Format(data, columns); err != nil {
		t.Fatalf("Format() error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No results found") {
		t.Errorf("expected 'No results found', got:\n%s", output)
	}
}

func TestTableFormatterNestedField(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatTable, &buf, TableStyle{})

	data := []map[string]interface{}{
		{"id": "1", "status": map[string]interface{}{"state": "SUCCESSFUL"}},
	}

	columns := []Column{
		{Header: "ID", Field: "id"},
		{Header: "STATE", Field: "status.state"},
	}

	if err := f.Format(data, columns); err != nil {
		t.Fatalf("Format() error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "SUCCESSFUL") {
		t.Errorf("table output should contain 'SUCCESSFUL', got:\n%s", output)
	}
}

func TestCSVFormatter(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatCSV, &buf, TableStyle{})

	data := []map[string]interface{}{
		{"id": "abc123", "name": "My Connection", "type": "TOOLKIT"},
		{"id": "def456", "name": "Another, Conn", "type": "JDBC"},
	}

	columns := []Column{
		{Header: "ID", Field: "id"},
		{Header: "NAME", Field: "name"},
		{Header: "TYPE", Field: "type"},
	}

	if err := f.Format(data, columns); err != nil {
		t.Fatalf("Format() error: %v", err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 rows), got %d:\n%s", len(lines), out)
	}
	if lines[0] != "ID,NAME,TYPE" {
		t.Errorf("header = %q, want %q", lines[0], "ID,NAME,TYPE")
	}
	if !strings.Contains(lines[2], `"Another, Conn"`) {
		t.Errorf("line 3 should have quoted field with comma: %s", lines[2])
	}
}

func TestCSVFormatterEmpty(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatCSV, &buf, TableStyle{})

	columns := []Column{{Header: "ID", Field: "id"}}
	if err := f.Format([]map[string]interface{}{}, columns); err != nil {
		t.Fatalf("Format() error: %v", err)
	}

	out := buf.String()
	// Header only — no data rows
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 || lines[0] != "ID" {
		t.Errorf("expected header-only output, got: %q", out)
	}
}

func TestCSVFormatterNoColumns(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatCSV, &buf, TableStyle{})
	err := f.Format([]map[string]interface{}{{"id": "1"}}, nil)
	if err == nil {
		t.Error("expected error when no columns defined")
	}
}

func TestTableFormatterNoColorStripsFuncANSI(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatTable, &buf, TableStyle{NoColor: true})

	data := []map[string]interface{}{
		{"name": "alpha"},
	}
	columns := []Column{
		{
			Header: "STATUS",
			Field:  "name",
			Func: func(v interface{}) string {
				return "\x1b[32mfound\x1b[0m"
			},
		},
	}

	if err := f.Format(data, columns); err != nil {
		t.Fatalf("Format() error: %v", err)
	}
	out := buf.String()
	stripped := ansiEscape.ReplaceAllString(out, "")
	if stripped != out {
		t.Fatalf("expected no ANSI in no-color table output, got raw: %q", out)
	}
	if !strings.Contains(out, "found") {
		t.Fatalf("expected plain text cell value in output, got:\n%s", out)
	}
}

func TestCSVFormatterNoColorStripsANSI(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatCSV, &buf, TableStyle{NoColor: true})

	data := []map[string]interface{}{
		{"status": "ok"},
	}
	columns := []Column{
		{
			Header: "STATUS",
			Field:  "status",
			Func: func(v interface{}) string {
				return "\x1b[31mwarn\x1b[0m"
			},
		},
	}

	if err := f.Format(data, columns); err != nil {
		t.Fatalf("Format() error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("expected no ANSI in no-color CSV output, got:\n%s", out)
	}
	if !strings.Contains(out, "warn") {
		t.Fatalf("expected plain CSV value, got:\n%s", out)
	}
}

func TestJSONFormatterNoColorStripsANSI(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatJSON, &buf, TableStyle{NoColor: true})

	data := map[string]interface{}{
		"status": "\x1b[32mfound\x1b[0m",
		"items": []interface{}{
			map[string]interface{}{"msg": "\x1b[31mbad\x1b[0m"},
		},
	}

	if err := f.Format(data, nil); err != nil {
		t.Fatalf("Format() error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "\x1b[") || strings.Contains(out, "\\u001b[") {
		t.Fatalf("expected no ANSI in no-color JSON output, got:\n%s", out)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON output is not valid: %v", err)
	}
	if decoded["status"] != "found" {
		t.Fatalf("expected sanitized status, got %v", decoded["status"])
	}
}

func TestYAMLFormatterNoColorStripsANSI(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatYAML, &buf, TableStyle{NoColor: true})

	data := map[string]interface{}{
		"status": "\x1b[32mfound\x1b[0m",
		"meta": map[string]interface{}{
			"msg": "\x1b[31mwarn\x1b[0m",
		},
	}

	if err := f.Format(data, nil); err != nil {
		t.Fatalf("Format() error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("expected no ANSI in no-color YAML output, got:\n%s", out)
	}
	var decoded map[string]interface{}
	if err := yaml.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("YAML output is not valid: %v", err)
	}
	objects, ok := decoded["objects"].([]interface{})
	if !ok || len(objects) == 0 {
		t.Fatalf("expected objects array in YAML output, got %v", decoded["objects"])
	}
	first, ok := objects[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected first object map, got %T", objects[0])
	}
	if first["status"] != "found" {
		t.Fatalf("expected sanitized status, got %v", first["status"])
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input   string
		want    Format
		wantErr bool
	}{
		{"table", FormatTable, false},
		{"json", FormatJSON, false},
		{"csv", FormatCSV, false},
		{"yaml", FormatYAML, false},
		{"", FormatTable, false},
		{"xml", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseFormat(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
