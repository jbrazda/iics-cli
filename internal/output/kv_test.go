package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestKVRendersVertically(t *testing.T) {
	var buf bytes.Buffer
	f := New(FormatTable, &buf, TableStyle{NoColor: true})
	rows := []KVRow{
		KV("id", "abc"),
		KV("name", "My Thing"),
	}
	if err := f.Format(rows, KVCols); err != nil {
		t.Fatalf("Format: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"PROPERTY", "VALUE", "id", "abc", "name", "My Thing"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}
