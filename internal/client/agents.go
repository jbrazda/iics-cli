package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// Agent represents an IICS Secure Agent as returned by the v2 API.
type Agent struct {
	ID               string `json:"id,omitempty"`
	OrgID            string `json:"orgId,omitempty"`
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	CreateTime       string `json:"createTime,omitempty"`
	UpdateTime       string `json:"updateTime,omitempty"`
	CreatedBy        string `json:"createdBy,omitempty"`
	UpdatedBy        string `json:"updatedBy,omitempty"`
	Active           bool   `json:"active,omitempty"`
	ReadyToRun       bool   `json:"readyToRun,omitempty"`
	Platform         string `json:"platform,omitempty"`
	AgentHost        string `json:"agentHost,omitempty"`
	ProxyHost        string `json:"proxyHost,omitempty"`
	ProxyPort        int    `json:"proxyPort,omitempty"`
	ProxyUser        string `json:"proxyUser,omitempty"`
	AgentVersion     string `json:"agentVersion,omitempty"`
	UpgradeStatus    string `json:"upgradeStatus,omitempty"`
	LastUpgraded     string `json:"lastUpgraded,omitempty"`
	LastUpgradeCheck string `json:"lastUpgradeCheck,omitempty"`
	LastStatusChange string `json:"lastStatusChange,omitempty"`
	ConfigUpdateTime string `json:"configUpdateTime,omitempty"`
	GroupID          string `json:"agentGroupId,omitempty"`
}

// AgentEngineConfig represents a single engine configuration property.
type AgentEngineConfig struct {
	Type       string `json:"type,omitempty"`
	Name       string `json:"name,omitempty"`
	Value      string `json:"value,omitempty"`
	Customized bool   `json:"customized,omitempty"`
}

// AgentEngineStatus represents the status of a service running on the agent.
type AgentEngineStatus struct {
	AppName        string `json:"appname,omitempty"`
	AppDisplayName string `json:"appDisplayName,omitempty"`
	AppVersion     string `json:"appversion,omitempty"`
	Status         string `json:"status,omitempty"`
	SubState       string `json:"subState,omitempty"`
	CreateTime     string `json:"createTime,omitempty"`
	UpdateTime     string `json:"updateTime,omitempty"`
}

// AgentDetails extends Agent with service engine status and configuration,
// as returned by GET /api/v2/agent/details/<agentID>.
type AgentDetails struct {
	Agent
	AgentEngineStatus  []AgentEngineStatus `json:"agentEngineStatus,omitempty"`
	AgentEngineConfigs []AgentEngineConfig `json:"agentEngineConfigs,omitempty"`
}

// AgentListOptions holds query parameters for listing agents.
type AgentListOptions struct {
	Limit                 int
	Skip                  int
	IncludeUnassignedOnly bool
}

// ListAgents retrieves secure agents using the v2 API.
func (c *Client) ListAgents(ctx context.Context, opts AgentListOptions) ([]Agent, error) {
	query := make(map[string]string)
	if opts.Limit > 0 {
		query["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Skip > 0 {
		query["skip"] = strconv.Itoa(opts.Skip)
	}
	if opts.IncludeUnassignedOnly {
		query["includeUnassignedOnly"] = "true"
	}

	var resp []Agent
	if err := c.doJSONWithQuery(ctx, http.MethodGet, fmt.Sprintf("%s/agent", BaseAPIPathV2), query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetAgent retrieves a single secure agent by ID using the v2 API.
func (c *Client) GetAgent(ctx context.Context, id string) (*Agent, error) {
	var resp Agent
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/agent/%s", BaseAPIPathV2, id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAgentDetails retrieves agent details including engine service status and configuration.
// Uses GET /api/v2/agent/details/<agentID>?onlyStatus=false.
func (c *Client) GetAgentDetails(ctx context.Context, id string) (*AgentDetails, error) {
	var resp AgentDetails
	query := map[string]string{"onlyStatus": "false"}
	if err := c.doJSONWithQuery(ctx, http.MethodGet, fmt.Sprintf("%s/agent/details/%s", BaseAPIPathV2, id), query, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StartAgentService starts a service on a secure agent.
func (c *Client) StartAgentService(ctx context.Context, agentID, serviceName string) error {
	body := map[string]string{"serviceName": serviceName, "action": "start"}
	return c.doJSON(ctx, http.MethodPost, fmt.Sprintf("%s/agent/%s/services", BaseAPIPathV2, agentID), body, nil)
}

// StopAgentService stops a service on a secure agent.
func (c *Client) StopAgentService(ctx context.Context, agentID, serviceName string) error {
	body := map[string]string{"serviceName": serviceName, "action": "stop"}
	return c.doJSON(ctx, http.MethodPost, fmt.Sprintf("%s/agent/%s/services", BaseAPIPathV2, agentID), body, nil)
}
