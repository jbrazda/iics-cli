package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// RuntimeEnvironmentAgent represents an agent embedded in a runtime environment response.
type RuntimeEnvironmentAgent struct {
	ID               string `json:"id,omitempty"`
	OrgID            string `json:"orgId,omitempty"`
	Name             string `json:"name,omitempty"`
	CreateTime       string `json:"createTime,omitempty"`
	UpdateTime       string `json:"updateTime,omitempty"`
	CreatedBy        string `json:"createdBy,omitempty"`
	UpdatedBy        string `json:"updatedBy,omitempty"`
	Active           bool   `json:"active,omitempty"`
	ReadyToRun       bool   `json:"readyToRun,omitempty"`
	Platform         string `json:"platform,omitempty"`
	AgentHost        string `json:"agentHost,omitempty"`
	ServerURL        string `json:"serverUrl,omitempty"`
	ProxyPort        int    `json:"proxyPort,omitempty"`
	AgentVersion     string `json:"agentVersion,omitempty"`
	UpgradeStatus    string `json:"upgradeStatus,omitempty"`
	SpiURL           string `json:"spiUrl,omitempty"`
	FederatedID      string `json:"federatedId,omitempty"`
	LastUpgraded     string `json:"lastUpgraded,omitempty"`
	LastUpgradeCheck string `json:"lastUpgradeCheck,omitempty"`
	LastStatusChange string `json:"lastStatusChange,omitempty"`
	ConfigUpdateTime string `json:"configUpdateTime,omitempty"`
	CreateTimeUTC    string `json:"createTimeUTC,omitempty"`
	UpdateTimeUTC    string `json:"updateTimeUTC,omitempty"`
	GroupID          string `json:"agentGroupId,omitempty"`
}

// CloudProviderConfig holds cloud-provider-specific configuration for serverless environments.
type CloudProviderConfig struct {
	CloudConfig []map[string]interface{} `json:"cloudConfig,omitempty"`
}

// ServerlessConfig holds serverless runtime environment configuration.
type ServerlessConfig struct {
	Platform            string              `json:"platform,omitempty"`
	ApplicationType     string              `json:"applicationType,omitempty"`
	Status              string              `json:"status,omitempty"`
	StatusMessage       string              `json:"statusMessage,omitempty"`
	MaxComputeUnits     int                 `json:"maxComputeUnits,omitempty"`
	ExecutionTimeout    int                 `json:"executionTimeout,omitempty"`
	CloudProviderConfig CloudProviderConfig `json:"cloudProviderConfig,omitempty"`
}

// RuntimeEnvironment represents an IICS runtime environment (V2 API).
type RuntimeEnvironment struct {
	ID               string                    `json:"id,omitempty"`
	OrgID            string                    `json:"orgId,omitempty"`
	Name             string                    `json:"name"`
	CreateTime       string                    `json:"createTime,omitempty"`
	UpdateTime       string                    `json:"updateTime,omitempty"`
	CreatedBy        string                    `json:"createdBy,omitempty"`
	UpdatedBy        string                    `json:"updatedBy,omitempty"`
	Agents           []RuntimeEnvironmentAgent `json:"agents,omitempty"`
	IsShared         bool                      `json:"isShared,omitempty"`
	FederatedID      string                    `json:"federatedId,omitempty"`
	CreateTimeUTC    string                    `json:"createTimeUTC,omitempty"`
	UpdateTimeUTC    string                    `json:"updateTimeUTC,omitempty"`
	ServerlessConfig *ServerlessConfig         `json:"serverlessConfig,omitempty"`
}

// RuntimeListOptions holds query parameters for listing runtimes.
type RuntimeListOptions struct {
	Limit int
	Skip  int
}

// ListRuntimeEnvironments retrieves runtime environments.
func (c *Client) ListRuntimeEnvironments(ctx context.Context, opts RuntimeListOptions) ([]RuntimeEnvironment, error) {
	query := make(map[string]string)
	if opts.Limit > 0 {
		query["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Skip > 0 {
		query["skip"] = strconv.Itoa(opts.Skip)
	}

	var resp []RuntimeEnvironment
	if err := c.doJSONWithQuery(ctx, http.MethodGet, BaseAPIPathV2+"/runtimeEnvironment", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetRuntimeEnvironment retrieves a single runtime environment by ID.
func (c *Client) GetRuntimeEnvironment(ctx context.Context, id string) (*RuntimeEnvironment, error) {
	var resp RuntimeEnvironment
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/runtimeEnvironment/%s", BaseAPIPathV2, id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetRuntimeEnvironmentByName retrieves a single runtime environment by name.
func (c *Client) GetRuntimeEnvironmentByName(ctx context.Context, name string) (*RuntimeEnvironment, error) {
	var resp RuntimeEnvironment
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/runtimeEnvironment/name/%s", BaseAPIPathV2, name), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateRuntimeEnvironment creates a new runtime environment.
func (c *Client) CreateRuntimeEnvironment(ctx context.Context, rt *RuntimeEnvironment) (*RuntimeEnvironment, error) {
	var resp RuntimeEnvironment
	if err := c.doJSON(ctx, http.MethodPost, BaseAPIPathV2+"/runtimeEnvironment", rt, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateRuntimeEnvironment updates an existing runtime environment.
func (c *Client) UpdateRuntimeEnvironment(ctx context.Context, id string, rt *RuntimeEnvironment) (*RuntimeEnvironment, error) {
	var resp RuntimeEnvironment
	if err := c.doJSON(ctx, http.MethodPut, fmt.Sprintf("%s/runtimeEnvironment/%s", BaseAPIPathV2, id), rt, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
