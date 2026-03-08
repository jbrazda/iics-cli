package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// Agent represents an IICS Secure Agent.
type Agent struct {
	ID          string `json:"id,omitempty"`
	OrgID       string `json:"orgId,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CreateTime  string `json:"createTime,omitempty"`
	UpdateTime  string `json:"updateTime,omitempty"`
	Host        string `json:"host,omitempty"`
	Status      string `json:"status,omitempty"`
	Platform    string `json:"platform,omitempty"`
	GroupID     string `json:"agentGroupId,omitempty"`
	GroupName   string `json:"agentGroupName,omitempty"`
}

// AgentListOptions holds query parameters for listing agents.
type AgentListOptions struct {
	Limit int
	Skip  int
}

// ListAgents retrieves secure agents.
func (c *Client) ListAgents(ctx context.Context, opts AgentListOptions) ([]Agent, error) {
	query := make(map[string]string)
	if opts.Limit > 0 {
		query["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Skip > 0 {
		query["skip"] = strconv.Itoa(opts.Skip)
	}

	var resp []Agent
	if err := c.doJSONWithQuery(ctx, http.MethodGet, "public/core/v3/agents", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// StartAgentService starts a service on a secure agent.
func (c *Client) StartAgentService(ctx context.Context, agentID, serviceName string) error {
	body := map[string]string{"serviceName": serviceName, "action": "start"}
	return c.doJSON(ctx, http.MethodPost, fmt.Sprintf("public/core/v3/agents/%s/services", agentID), body, nil)
}

// StopAgentService stops a service on a secure agent.
func (c *Client) StopAgentService(ctx context.Context, agentID, serviceName string) error {
	body := map[string]string{"serviceName": serviceName, "action": "stop"}
	return c.doJSON(ctx, http.MethodPost, fmt.Sprintf("public/core/v3/agents/%s/services", agentID), body, nil)
}
