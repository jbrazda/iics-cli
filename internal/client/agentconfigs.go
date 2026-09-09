package client

import (
	"context"
	"fmt"
	"net/http"
)

// AgentGroupServiceConfig holds Secure Agent service property overrides for a
// Secure Agent group (runtime environment), keyed by service name. Each value is
// the list of property-setting objects for that service. The setting object
// shape is defined by the API and passed through unchanged.
type AgentGroupServiceConfig map[string][]map[string]interface{}

// GetAgentGroupConfigs retrieves the service property overrides for a Secure
// Agent group. GET /api/v2/runtimeEnvironment/<groupID>/configs. An empty result
// means the group has no overrides.
func (c *Client) GetAgentGroupConfigs(ctx context.Context, groupID string) (AgentGroupServiceConfig, error) {
	var resp AgentGroupServiceConfig
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/runtimeEnvironment/%s/configs", BaseAPIPathV2, groupID), nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// UpdateAgentGroupConfigs replaces the service property overrides for a Secure
// Agent group. PUT /api/v2/runtimeEnvironment/<groupID>/configs.
func (c *Client) UpdateAgentGroupConfigs(ctx context.Context, groupID string, cfg AgentGroupServiceConfig) (AgentGroupServiceConfig, error) {
	var resp AgentGroupServiceConfig
	if err := c.doJSON(ctx, http.MethodPut, fmt.Sprintf("%s/runtimeEnvironment/%s/configs", BaseAPIPathV2, groupID), cfg, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
