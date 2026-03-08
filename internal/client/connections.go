package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// Connection represents an IICS connection.
type Connection struct {
	ID           string                 `json:"id,omitempty"`
	OrgID        string                 `json:"orgId,omitempty"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	Type         string                 `json:"type"`
	CreateTime   string                 `json:"createTime,omitempty"`
	UpdateTime   string                 `json:"updateTime,omitempty"`
	CreatedBy    string                 `json:"createdBy,omitempty"`
	UpdatedBy    string                 `json:"updatedBy,omitempty"`
	AgentID      string                 `json:"agentId,omitempty"`
	RuntimeEnvID string                 `json:"runtimeEnvironmentId,omitempty"`
	ConnParams   map[string]interface{} `json:"connParams,omitempty"`
}

// ConnectionListOptions holds query parameters for listing connections.
type ConnectionListOptions struct {
	Type  string
	Name  string
	Limit int
	Skip  int
}

// ListConnections retrieves connections with optional filtering.
func (c *Client) ListConnections(ctx context.Context, opts ConnectionListOptions) ([]Connection, error) {
	query := make(map[string]string)

	if opts.Type != "" {
		query["type"] = opts.Type
	}
	if opts.Name != "" {
		query["name"] = opts.Name
	}
	if opts.Limit > 0 {
		query["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Skip > 0 {
		query["skip"] = strconv.Itoa(opts.Skip)
	}

	var resp []Connection
	if err := c.doJSONWithQuery(ctx, http.MethodGet, "api/v2/connection", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetConnection retrieves a single connection by ID.
func (c *Client) GetConnection(ctx context.Context, id string) (*Connection, error) {
	var resp Connection
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("api/v2/connection/%s", id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateConnection creates a new connection.
func (c *Client) CreateConnection(ctx context.Context, conn *Connection) (*Connection, error) {
	var resp Connection
	if err := c.doJSON(ctx, http.MethodPost, "api/v2/connection", conn, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateConnection updates an existing connection.
func (c *Client) UpdateConnection(ctx context.Context, id string, conn *Connection) (*Connection, error) {
	var resp Connection
	if err := c.doJSON(ctx, http.MethodPut, fmt.Sprintf("api/v2/connection/%s", id), conn, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteConnection deletes a connection by ID.
func (c *Client) DeleteConnection(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("api/v2/connection/%s", id), nil, nil)
}
