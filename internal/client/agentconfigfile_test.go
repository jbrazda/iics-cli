package client

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseAgentConfigChangesMap(t *testing.T) {
	data := `
process-engine:
  cluster:
    primary-node: true
  jvm:
    max-heap: "'4G'"
    min-heap: null
  db:
    connection-properties: |
      - key: testOnBorrow
        value: true
OpsInsightsDataCollector:
  OpsInsights:
    maxBufferSize: 1024
`
	got, err := ParseAgentConfigChanges([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	want := []AgentConfigChange{
		{Property: "process-engine:cluster:primary-node", Value: "true"},
		{Property: "process-engine:jvm:max-heap", Value: "'4G'"},
		{Property: "process-engine:jvm:min-heap", Reset: true},
		{Property: "process-engine:db:connection-properties", Value: "- key: testOnBorrow\n  value: true\n"},
		{Property: "OpsInsightsDataCollector:OpsInsights:maxBufferSize", Value: "1024"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

func TestParseAgentConfigChangesJSON(t *testing.T) {
	m, err := ParseAgentConfigChanges([]byte(`{"process-engine": {"cluster": {"primary-node": "false"}}}`))
	if err != nil || len(m) != 1 || m[0].Property != "process-engine:cluster:primary-node" || m[0].Value != "false" {
		t.Errorf("JSON map = %+v, %v", m, err)
	}
	l, err := ParseAgentConfigChanges([]byte(`[{"property": "process-engine:jvm:max-heap", "value": "'4G'"},
		{"property": "process-engine:jvm:min-heap", "reset": true}]`))
	want := []AgentConfigChange{
		{Property: "process-engine:jvm:max-heap", Value: "'4G'"},
		{Property: "process-engine:jvm:min-heap", Reset: true},
	}
	if err != nil || !reflect.DeepEqual(l, want) {
		t.Errorf("JSON list = %+v, %v", l, err)
	}
}

func TestParseAgentConfigChangesErrors(t *testing.T) {
	for name, data := range map[string]string{
		"scalar root":    `"x"`,
		"flat map":       `{"process-engine": "x"}`,
		"no names":       `{"process-engine": {"jvm": "x"}}`,
		"nested value":   `{"process-engine": {"jvm": {"max-heap": {"a": 1}}}}`,
		"unknown key":    `[{"property": "a:b:c", "val": "x"}]`,
		"no property":    `[{"value": "x"}]`,
		"invalid syntax": `{"a": [}`,
	} {
		if _, err := ParseAgentConfigChanges([]byte(data)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestAgentConfigExportRoundTrip(t *testing.T) {
	form, err := parseAgentConfigForm(agentPage("tok", "false", ""))
	if err != nil {
		t.Fatal(err)
	}
	out, err := yaml.Marshal(AgentConfigDocument(form.Properties))
	if err != nil {
		t.Fatal(err)
	}
	changes, err := ParseAgentConfigChanges(out)
	if err != nil {
		t.Fatalf("parsing export: %v\n%s", err, out)
	}
	// System properties are not exported; the three engine properties are.
	if len(changes) != 3 {
		t.Fatalf("got %d changes from export:\n%s", len(changes), out)
	}
	// Applying an unchanged export changes nothing.
	diffs, err := form.Apply(changes)
	if err != nil || len(diffs) != 0 {
		t.Errorf("applying unchanged export: %+v, %v\n%s", diffs, err, out)
	}
	m := AgentConfigMap(form.Properties)
	if m["process-engine"]["cluster"]["primary-node"] != "true" || len(m) != 2 {
		t.Errorf("AgentConfigMap = %v", m)
	}
}
