package client

import (
	"context"
	"net/http"
	"testing"
)

func TestListConnectors(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/connector" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"@type":"connector","orgId":"0","name":"Amazon S3 v2","connectorVersion":53000414,"pluginId":"451500","shortName":"AmazonS3","isPublic":true,"sdkType":"CCI","displayName":"Amazon S3 v2"}]`))
	})

	c := newTestClient(handler)
	connectors, err := c.ListConnectors(context.Background())
	if err != nil {
		t.Fatalf("ListConnectors() error: %v", err)
	}
	if len(connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(connectors))
	}
	got := connectors[0]
	if got.Name != "Amazon S3 v2" || got.ConnectorVersion != 53000414 || !got.IsPublic || got.SDKType != "CCI" {
		t.Errorf("unexpected connector: %+v", got)
	}
}

func TestGetConnectorMetadataStandard(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/connector/metadata" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("connectorName"); got != "SqlServer" {
			t.Errorf("connectorName = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"@type":"connectorMetadata","type":"SqlServer","isStandardConnType":true,"attributes":[
			{"name":"port","label":"","id":"","value":"1433","type":2,"isMandatory":true,"visible":false,"list":[]},
			{"name":"subType","label":"","id":"","value":"","type":8,"isMandatory":true,"visible":false,"list":["SqlServer2017","SqlServer2019"]}]}]`))
	})

	c := newTestClient(handler)
	metas, err := c.GetConnectorMetadata(context.Background(), "SqlServer")
	if err != nil {
		t.Fatalf("GetConnectorMetadata() error: %v", err)
	}
	if len(metas) != 1 || !metas[0].IsStandardConnType || len(metas[0].Attributes) != 2 {
		t.Fatalf("unexpected metadata: %+v", metas)
	}
	fields, err := metas[0].Fields()
	if err != nil {
		t.Fatalf("Fields() error: %v", err)
	}
	if fields[0].DefaultValue != "1433" || fields[0].DataType != "2" || !fields[0].Mandatory {
		t.Errorf("unexpected port field: %+v", fields[0])
	}
	if fields[1].Values != "SqlServer2017, SqlServer2019" {
		t.Errorf("unexpected subType values: %q", fields[1].Values)
	}

	if _, err := c.GetConnectorMetadata(context.Background(), ""); err == nil {
		t.Error("expected error for empty connector name")
	}
}

func TestConnectorMetadataCCIFields(t *testing.T) {
	cci := `{"connections":[{"pages":[{"groups":[{"group":{"connInfoPresentationControls":[
		{"attributeType":{"attributeName":"AccessKey","attrLocalisedName":"Access Key","attributeType":"STRING","mandatory":false,"encrypted":true,"defaultValue":""}},
		{"attributeType":{"attributeName":"UseEC2Role","attrLocalisedName":"Use EC2 Role","attributeType":"BOOLEAN","mandatory":false,"encrypted":false,"defaultValue":false}},
		{"attributeType":{"attributeName":"RegionName","attrLocalisedName":"Region","attributeType":"STRING","mandatory":true,"encrypted":false,"defaultValue":"US East","validList":[{"name":"US East"},{"name":"EU"}]}}
	]}}]}]},{"pages":[{"groups":[{"group":{"connInfoPresentationControls":[
		{"attributeType":{"attributeName":"AccessKey","attributeType":"STRING"}}
	]}}]}]}]}`
	m := ConnectorMetadata{Type: "Amazon S3 v2", CCIMeta: cci, Attributes: []ConnectorAttribute{{Name: "agentGroupId"}}}

	fields, err := m.Fields()
	if err != nil {
		t.Fatalf("Fields() error: %v", err)
	}
	if len(fields) != 3 {
		t.Fatalf("expected 3 de-duplicated fields, got %d: %+v", len(fields), fields)
	}
	if !fields[0].Encrypted || fields[0].Label != "Access Key" {
		t.Errorf("unexpected AccessKey field: %+v", fields[0])
	}
	if fields[1].DefaultValue != "false" {
		t.Errorf("expected boolean default rendered as false, got %q", fields[1].DefaultValue)
	}
	if !fields[2].Mandatory || fields[2].DefaultValue != "US East" || fields[2].Values != "US East, EU" {
		t.Errorf("unexpected RegionName field: %+v", fields[2])
	}

	bad := ConnectorMetadata{Type: "X", CCIMeta: "{"}
	if _, err := bad.Fields(); err == nil {
		t.Error("expected error for invalid cciMeta")
	}
}
