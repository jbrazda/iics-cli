package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// connectionAtType is the V2 type discriminator for connection request bodies.
const connectionAtType = "connection"

// Connection represents an IICS connection (V2 API). Connector-specific
// attributes not listed here are returned in ConnParams.
type Connection struct {
	AtType               string                 `json:"@type,omitempty"`
	ID                   string                 `json:"id,omitempty"`
	OrgID                string                 `json:"orgId,omitempty"`
	Name                 string                 `json:"name"`
	Description          string                 `json:"description,omitempty"`
	ShortDescription     string                 `json:"shortDescription,omitempty"`
	Type                 string                 `json:"type"`
	BaseType             string                 `json:"baseType,omitempty"`
	SubType              string                 `json:"subType,omitempty"`
	InstanceDisplayName  string                 `json:"instanceDisplayName,omitempty"`
	CreateTime           string                 `json:"createTime,omitempty"`
	UpdateTime           string                 `json:"updateTime,omitempty"`
	MajorUpdateTime      string                 `json:"majorUpdateTime,omitempty"`
	CreatedBy            string                 `json:"createdBy,omitempty"`
	UpdatedBy            string                 `json:"updatedBy,omitempty"`
	AgentID              string                 `json:"agentId,omitempty"`
	RuntimeEnvID         string                 `json:"runtimeEnvironmentId,omitempty"`
	FederatedID          string                 `json:"federatedId,omitempty"`
	ConnectorStatus      string                 `json:"connectorStatus,omitempty"`
	Host                 string                 `json:"host,omitempty"`
	Port                 int                    `json:"port,omitempty"`
	Domain               string                 `json:"domain,omitempty"`
	Database             string                 `json:"database,omitempty"`
	Schema               string                 `json:"schema,omitempty"`
	InstanceName         string                 `json:"instanceName,omitempty"`
	Codepage             string                 `json:"codepage,omitempty"`
	AuthenticationType   string                 `json:"authenticationType,omitempty"`
	AdjustedJdbcHostName string                 `json:"adjustedJdbcHostName,omitempty"`
	Username             string                 `json:"username,omitempty"`
	Password             string                 `json:"password,omitempty"`
	SecurityToken        string                 `json:"securityToken,omitempty"`
	ServiceURL           string                 `json:"serviceUrl,omitempty"`
	StsURL               string                 `json:"stsUrl,omitempty"`
	Timeout              int                    `json:"timeout,omitempty"`
	ConnParams           map[string]interface{} `json:"connParams,omitempty"`
}

// ConnectionListOptions holds filters for listing connections. The API
// ignores type and name, so they are applied client-side.
type ConnectionListOptions struct {
	Type  string
	Name  string
	Limit int
	Skip  int
}

// ListConnections retrieves connections with optional filtering.
// Type matches exactly (case-insensitive); Name matches a case-insensitive substring.
func (c *Client) ListConnections(ctx context.Context, opts ConnectionListOptions) ([]Connection, error) {
	query := make(map[string]string)

	if opts.Limit > 0 {
		query["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Skip > 0 {
		query["skip"] = strconv.Itoa(opts.Skip)
	}

	var resp []Connection
	if err := c.doJSONWithQuery(ctx, http.MethodGet, fmt.Sprintf("%s/connection", BaseAPIPathV2), query, nil, &resp); err != nil {
		return nil, err
	}
	if opts.Type == "" && opts.Name == "" {
		return resp, nil
	}
	name := strings.ToLower(opts.Name)
	filtered := make([]Connection, 0, len(resp))
	for _, conn := range resp {
		if opts.Type != "" && !strings.EqualFold(conn.Type, opts.Type) {
			continue
		}
		if name != "" && !strings.Contains(strings.ToLower(conn.Name), name) {
			continue
		}
		filtered = append(filtered, conn)
	}
	return filtered, nil
}

// GetConnection retrieves a single connection by ID.
func (c *Client) GetConnection(ctx context.Context, id string) (*Connection, error) {
	var resp Connection
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/connection/%s", BaseAPIPathV2, id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetConnectionByName retrieves a single connection by name.
// Spaces in the name are percent-encoded as required by the V2 Connection API:
// GET /api/v2/connection/name/<name>
func (c *Client) GetConnectionByName(ctx context.Context, name string) (*Connection, error) {
	encoded := url.PathEscape(name)
	var resp Connection
	if err := c.doJSON(ctx, http.MethodGet,
		fmt.Sprintf("%s/connection/name/%s", BaseAPIPathV2, encoded), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateConnection creates a new connection from a raw JSON definition, so
// connector-specific attributes are sent unchanged. "@type" is set to
// "connection" when missing.
func (c *Client) CreateConnection(ctx context.Context, def map[string]interface{}) (*Connection, error) {
	setConnectionAtType(def)
	var resp Connection
	if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("%s/connection", BaseAPIPathV2), def, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateConnection updates an existing connection with POST /connection/<id>.
// A full update replaces all attributes; partial sends "Update-Mode: PARTIAL"
// so only the attributes in def change.
func (c *Client) UpdateConnection(ctx context.Context, id string, def map[string]interface{}, partial bool) (*Connection, error) {
	setConnectionAtType(def)
	var headers map[string]string
	if partial {
		headers = map[string]string{"Update-Mode": "PARTIAL"}
	}
	var resp Connection
	if err := c.doJSONWithHeaders(ctx, http.MethodPost, fmt.Sprintf("%s/connection/%s", BaseAPIPathV2, id), headers, def, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func setConnectionAtType(def map[string]interface{}) {
	if _, ok := def["@type"]; !ok {
		def["@type"] = connectionAtType
	}
}

// DeleteConnection deletes a connection by ID.
func (c *Client) DeleteConnection(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("%s/connection/%s", BaseAPIPathV2, id), nil, nil)
}

// ConnectionTestResult is the success object returned by a connection test.
type ConnectionTestResult struct {
	AtType      string `json:"@type,omitempty"`
	Description string `json:"description,omitempty"`
}

// TestConnection tests a connection. A failed test is returned as an *APIError.
func (c *Client) TestConnection(ctx context.Context, id string) (*ConnectionTestResult, error) {
	var resp ConnectionTestResult
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/connection/test/%s", BaseAPIPathV2, id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ConnectionSearchOptions holds query parameters for searching connections.
// UIType is required. RuntimeEnvironmentID takes precedence over AgentID.
type ConnectionSearchOptions struct {
	UIType               string
	AgentID              string
	RuntimeEnvironmentID string
}

// SearchConnections lists the connections of a type available to a Secure
// Agent or runtime environment.
func (c *Client) SearchConnections(ctx context.Context, opts ConnectionSearchOptions) ([]Connection, error) {
	if opts.UIType == "" {
		return nil, fmt.Errorf("connection type is required")
	}
	query := map[string]string{"uiType": opts.UIType}
	if opts.AgentID != "" {
		query["agentId"] = opts.AgentID
	}
	if opts.RuntimeEnvironmentID != "" {
		query["runtimeEnvironmentId"] = opts.RuntimeEnvironmentID
	}
	var resp []Connection
	if err := c.doJSONWithQuery(ctx, http.MethodGet, fmt.Sprintf("%s/connection/search", BaseAPIPathV2), query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ConnListItem is a source or target object available through a connection.
type ConnListItem struct {
	AtType string `json:"@type,omitempty"`
	ID     string `json:"id"`
	Name   string `json:"name"`
}

// ConnectionObjectsOptions selects the connection and the objects to list.
// Exactly one of ID and Name is used (ID wins). SearchPattern is a
// case-insensitive substring filter applied by the API.
type ConnectionObjectsOptions struct {
	ID            string
	Name          string
	Target        bool
	SearchPattern string
}

// ListConnectionObjects lists the source (or target) objects of a connection.
// The API returns at most 200 objects per request.
func (c *Client) ListConnectionObjects(ctx context.Context, opts ConnectionObjectsOptions) ([]ConnListItem, error) {
	path := fmt.Sprintf("%s/connection/%s/", BaseAPIPathV2, sourceOrTarget(opts.Target))
	switch {
	case opts.ID != "":
		path += opts.ID
	case opts.Name != "":
		path += "name/" + url.PathEscape(opts.Name)
	default:
		return nil, fmt.Errorf("connection ID or name is required")
	}
	query := map[string]string{}
	if opts.SearchPattern != "" {
		query["searchPattern"] = opts.SearchPattern
	}
	var resp []ConnListItem
	if err := c.doJSONWithQuery(ctx, http.MethodGet, path, query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// RuntimeAttribute is a source or target runtime attribute of a connection.
type RuntimeAttribute struct {
	AtType            string   `json:"@type,omitempty"`
	Name              string   `json:"name"`
	UniqueName        string   `json:"uniqueName,omitempty"`
	DataType          string   `json:"dataType"`
	DefaultValue      string   `json:"defaultValue"`
	Label             string   `json:"label"`
	Mandatory         bool     `json:"mandatory"`
	MaxLength         int      `json:"maxLength"`
	SessionVarAllowed bool     `json:"sessionVarAllowed"`
	PossibleValues    []string `json:"possibleValues"`
}

// GetConnectionMetadata returns the source (or target) runtime attributes of
// a connection. Connectors without runtime attributes return an APIError
// (APP_13405 "The connection has no object.").
func (c *Client) GetConnectionMetadata(ctx context.Context, id string, target bool) ([]RuntimeAttribute, error) {
	var resp []RuntimeAttribute
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/connection/%s/%s/metadata", BaseAPIPathV2, sourceOrTarget(target), id), nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func sourceOrTarget(target bool) string {
	if target {
		return "target"
	}
	return "source"
}
