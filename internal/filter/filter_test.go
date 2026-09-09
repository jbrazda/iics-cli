package filter

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		in           string
		key, op, val string
		wantErr      bool
	}{
		{in: "agentHost==host01", key: "agentHost", op: "==", val: "host01"},
		{in: " active != true ", key: "active", op: "!=", val: "true"},
		{in: "status.subState==0", key: "status.subState", op: "==", val: "0"},
		{in: "name==", key: "name", op: "==", val: ""},
		{in: "noOperator", wantErr: true},
		{in: "==value", wantErr: true},
	}
	for _, tt := range tests {
		p, err := Parse(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("Parse(%q): expected error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("Parse(%q): %v", tt.in, err)
		}
		if p.Key != tt.key || p.Op != tt.op || p.Value != tt.val {
			t.Errorf("Parse(%q) = %+v, want {%s %s %s}", tt.in, p, tt.key, tt.op, tt.val)
		}
	}
}

func TestMatch(t *testing.T) {
	row := map[string]interface{}{
		"agentHost": "DevInfaCld01",
		"active":    true,
		"proxyPort": float64(80),
		"status":    map[string]interface{}{"subState": "0"},
	}
	cases := []struct {
		expr string
		want bool
	}{
		{"agentHost==devinfacld01", true}, // case-insensitive
		{"agentHost!=other", true},
		{"active==true", true},
		{"active!=true", false},
		{"proxyPort==80", true},
		{"status.subState==0", true},
		{"missing==x", false},
		{"missing!=x", true},
	}
	for _, c := range cases {
		p, err := Parse(c.expr)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.expr, err)
		}
		if got := Match(row, []Predicate{p}); got != c.want {
			t.Errorf("Match(%q) = %v, want %v", c.expr, got, c.want)
		}
	}
}

func TestApply(t *testing.T) {
	type agent struct {
		Name string `json:"name"`
		Host string `json:"agentHost"`
		OK   bool   `json:"active"`
	}
	data := []agent{
		{"a1", "host01", true},
		{"a2", "host02", false},
		{"a3", "host01", false},
	}
	preds, err := ParseAll([]string{"agentHost==host01", "active!=true"})
	if err != nil {
		t.Fatalf("ParseAll: %v", err)
	}
	rows, err := Apply(data, preds)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(rows) != 1 || rows[0]["name"] != "a3" {
		t.Fatalf("Apply returned %d rows: %+v", len(rows), rows)
	}
}
