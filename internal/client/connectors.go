package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Connector is a connector available to the organization (V2 API).
type Connector struct {
	AtType           string `json:"@type,omitempty"`
	OrgID            string `json:"orgId,omitempty"`
	Name             string `json:"name"`
	DisplayName      string `json:"displayName,omitempty"`
	Description      string `json:"description,omitempty"`
	Type             string `json:"type,omitempty"`
	Publisher        string `json:"publisher,omitempty"`
	ConnectorVersion int    `json:"connectorVersion"`
	ShortName        string `json:"shortName,omitempty"`
	IsPublic         bool   `json:"isPublic"`
	SDKType          string `json:"sdkType,omitempty"`
	PluginID         string `json:"pluginId,omitempty"`
	GUID             string `json:"guid,omitempty"`
	UpdateTime       string `json:"updateTime,omitempty"`
}

// ConnectorAttribute is a connection attribute of a connector type.
// Type is the numeric connector data type code.
type ConnectorAttribute struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	ID          string   `json:"id"`
	Value       string   `json:"value"`
	Type        int      `json:"type"`
	IsMandatory bool     `json:"isMandatory"`
	Visible     bool     `json:"visible"`
	List        []string `json:"list"`
}

// ConnectorMetadata describes the attributes of a connector type. CCI
// connectors carry their attribute definitions in CCIMeta, a JSON-encoded
// string; see CCIAttributes.
type ConnectorMetadata struct {
	AtType             string               `json:"@type,omitempty"`
	Type               string               `json:"type"`
	IsStandardConnType bool                 `json:"isStandardConnType"`
	CCIMeta            string               `json:"cciMeta,omitempty"`
	Attributes         []ConnectorAttribute `json:"attributes"`
}

// ListConnectors returns the connectors available to the organization.
func (c *Client) ListConnectors(ctx context.Context) ([]Connector, error) {
	var resp []Connector
	if err := c.doJSON(ctx, http.MethodGet, BaseAPIPathV2+"/connector", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetConnectorMetadata returns metadata for the connectors whose name starts
// with connectorName (the API matches by prefix, e.g. "Oracle" also returns
// "Oracle Autonomous Database").
func (c *Client) GetConnectorMetadata(ctx context.Context, connectorName string) ([]ConnectorMetadata, error) {
	if connectorName == "" {
		return nil, fmt.Errorf("connector name is required")
	}
	var resp []ConnectorMetadata
	if err := c.doJSONWithQuery(ctx, http.MethodGet, BaseAPIPathV2+"/connector/metadata",
		map[string]string{"connectorName": connectorName}, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ConnectorField is one connection attribute of a connector in a flat form
// that covers both standard attributes and CCI attribute definitions.
// Encrypted is reported for CCI attributes only.
type ConnectorField struct {
	Name         string `json:"name"`
	Label        string `json:"label,omitempty"`
	DataType     string `json:"dataType"`
	Mandatory    bool   `json:"mandatory"`
	Encrypted    bool   `json:"encrypted"`
	DefaultValue string `json:"defaultValue,omitempty"`
	Values       string `json:"values,omitempty"`
}

// cciMeta mirrors the parts of the cciMeta document that hold the
// connection attribute definitions.
type cciMeta struct {
	Connections []struct {
		Pages []struct {
			Groups []struct {
				Group struct {
					Controls []struct {
						AttributeType cciAttributeType `json:"attributeType"`
					} `json:"connInfoPresentationControls"`
				} `json:"group"`
			} `json:"groups"`
		} `json:"pages"`
	} `json:"connections"`
}

type cciAttributeType struct {
	AttributeName     string          `json:"attributeName"`
	AttrLocalisedName string          `json:"attrLocalisedName"`
	AttributeType     string          `json:"attributeType"`
	Mandatory         bool            `json:"mandatory"`
	Encrypted         bool            `json:"encrypted"`
	DefaultValue      json.RawMessage `json:"defaultValue"`
	ValidList         []struct {
		Name string `json:"name"`
	} `json:"validList"`
}

// Fields returns the connection attributes of the connector. For CCI
// connectors (CCIMeta set) the attributes come from the CCI definition;
// otherwise from Attributes, with DataType set to the numeric type code.
func (m ConnectorMetadata) Fields() ([]ConnectorField, error) {
	if m.CCIMeta == "" {
		out := make([]ConnectorField, 0, len(m.Attributes))
		for _, a := range m.Attributes {
			out = append(out, ConnectorField{
				Name:         a.Name,
				Label:        a.Label,
				DataType:     fmt.Sprintf("%d", a.Type),
				Mandatory:    a.IsMandatory,
				DefaultValue: a.Value,
				Values:       strings.Join(a.List, ", "),
			})
		}
		return out, nil
	}

	var meta cciMeta
	if err := json.Unmarshal([]byte(m.CCIMeta), &meta); err != nil {
		return nil, fmt.Errorf("parsing cciMeta for %s: %w", m.Type, err)
	}
	var out []ConnectorField
	seen := map[string]bool{}
	for _, conn := range meta.Connections {
		for _, page := range conn.Pages {
			for _, g := range page.Groups {
				for _, ctl := range g.Group.Controls {
					at := ctl.AttributeType
					if at.AttributeName == "" || seen[at.AttributeName] {
						continue
					}
					seen[at.AttributeName] = true
					values := make([]string, 0, len(at.ValidList))
					for _, v := range at.ValidList {
						values = append(values, v.Name)
					}
					out = append(out, ConnectorField{
						Name:         at.AttributeName,
						Label:        at.AttrLocalisedName,
						DataType:     at.AttributeType,
						Mandatory:    at.Mandatory,
						Encrypted:    at.Encrypted,
						DefaultValue: rawScalar(at.DefaultValue),
						Values:       strings.Join(values, ", "),
					})
				}
			}
		}
	}
	return out, nil
}

// rawScalar renders a JSON scalar (string, number, bool) as text; null is "".
func rawScalar(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return string(raw)
}
