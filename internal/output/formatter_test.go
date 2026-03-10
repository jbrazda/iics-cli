package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONFormatter(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatJSON, &buf)

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
	f := New(FormatTable, &buf)

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
	f := New(FormatTable, &buf)

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
	f := New(FormatTable, &buf)

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
	f := New(FormatCSV, &buf)

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
	f := New(FormatCSV, &buf)

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
	f := New(FormatCSV, &buf)
	err := f.Format([]map[string]interface{}{{"id": "1"}}, nil)
	if err == nil {
		t.Error("expected error when no columns defined")
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
