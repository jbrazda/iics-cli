package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Agent represents an IICS Secure Agent as returned by the v2 API.
type Agent struct {
	Type             string `json:"@type,omitempty"`
	ID               string `json:"id,omitempty"`
	OrgID            string `json:"orgId,omitempty"`
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	CreateTime       string `json:"createTime,omitempty"`
	UpdateTime       string `json:"updateTime,omitempty"`
	CreateTimeUTC    string `json:"createTimeUTC,omitempty"`
	UpdateTimeUTC    string `json:"updateTimeUTC,omitempty"`
	CreatedBy        string `json:"createdBy,omitempty"`
	UpdatedBy        string `json:"updatedBy,omitempty"`
	Active           bool   `json:"active,omitempty"`
	ReadyToRun       bool   `json:"readyToRun,omitempty"`
	Platform         string `json:"platform,omitempty"`
	AgentHost        string `json:"agentHost,omitempty"`
	ServerURL        string `json:"serverUrl,omitempty"`
	SpiURL           string `json:"spiUrl,omitempty"`
	FederatedID      string `json:"federatedId,omitempty"`
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

// AgentEngineConfig represents a single engine (or agent) configuration property
// as returned inside GET /api/v2/agent/details/<agentID>?onlyStatus=false.
type AgentEngineConfig struct {
	Type         string `json:"type,omitempty"`
	Name         string `json:"name,omitempty"`
	Value        string `json:"value,omitempty"`
	Platform     string `json:"platform,omitempty"`
	DefaultValue string `json:"defaultValue,omitempty"`
	Customized   bool   `json:"customized,omitempty"`
}

// AgentEngineStatus represents the status of a service (engine) running on the agent.
type AgentEngineStatus struct {
	Type           string `json:"@type,omitempty"`
	AppName        string `json:"appname,omitempty"`
	AppDisplayName string `json:"appDisplayName,omitempty"`
	AppVersion     string `json:"appversion,omitempty"`
	ReplacePolicy  string `json:"replacePolicy,omitempty"`
	Status         string `json:"status,omitempty"`
	DesiredStatus  string `json:"desiredStatus,omitempty"`
	SubState       string `json:"subState,omitempty"`
	CreateTime     string `json:"createTime,omitempty"`
	UpdateTime     string `json:"updateTime,omitempty"`
}

// AgentEngine is one service engine entry in an agent details response.
type AgentEngine struct {
	Type               string              `json:"@type,omitempty"`
	AgentEngineStatus  AgentEngineStatus   `json:"agentEngineStatus,omitempty"`
	AgentEngineConfigs []AgentEngineConfig `json:"agentEngineConfigs,omitempty"`
}

// AgentDetails extends Agent with per-engine service status and configuration,
// as returned by GET /api/v2/agent/details/<agentID>.
type AgentDetails struct {
	Agent
	PlatformAgent bool                     `json:"platformAgent,omitempty"`
	Packages      []map[string]interface{} `json:"packages,omitempty"`
	AgentConfigs  []AgentEngineConfig      `json:"agentConfigs,omitempty"`
	AgentEngines  []AgentEngine            `json:"agentEngines,omitempty"`
}

// AgentListOptions holds query parameters for listing agents.
type AgentListOptions struct {
	Limit                 int
	Skip                  int
	IncludeUnassignedOnly bool
	BasicInfo             bool
}

// AgentSelector identifies a single agent by exactly one attribute.
type AgentSelector struct {
	ID          string
	Name        string
	Hostname    string
	FederatedID string
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
	if opts.BasicInfo {
		query["basicInfo"] = "true"
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

// GetAgentDetails retrieves agent details including per-engine service status.
// When full is true it requests onlyStatus=false, which also includes the
// agent-level and per-engine configuration properties.
func (c *Client) GetAgentDetails(ctx context.Context, id string, full bool) (*AgentDetails, error) {
	var resp AgentDetails
	var query map[string]string
	if full {
		query = map[string]string{"onlyStatus": "false"}
	}
	if err := c.doJSONWithQuery(ctx, http.MethodGet, fmt.Sprintf("%s/agent/details/%s", BaseAPIPathV2, id), query, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAgentByName retrieves a single secure agent by name using the v2 API.
func (c *Client) GetAgentByName(ctx context.Context, name string) (*Agent, error) {
	var resp Agent
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/agent/name/%s", BaseAPIPathV2, url.PathEscape(name)), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteAgent deletes a secure agent by ID using the v2 API.
func (c *Client) DeleteAgent(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("%s/agent/%s", BaseAPIPathV2, id), nil, nil)
}

// FindAgent resolves an agent from a selector. Name lookups use the dedicated
// by-name endpoint; hostname and federatedId lookups scan the agent list (which
// carries both fields). As a fallback, a federatedId that is not in the list is
// looked up via runtime environment membership.
func (c *Client) FindAgent(ctx context.Context, sel AgentSelector) (*Agent, error) {
	switch {
	case sel.ID != "":
		return c.GetAgent(ctx, sel.ID)
	case sel.Name != "":
		return c.GetAgentByName(ctx, sel.Name)
	case sel.Hostname != "":
		agents, err := c.ListAgents(ctx, AgentListOptions{})
		if err != nil {
			return nil, err
		}
		for i := range agents {
			if strings.EqualFold(agents[i].AgentHost, sel.Hostname) {
				return &agents[i], nil
			}
		}
		return nil, fmt.Errorf("no agent found with hostname %q", sel.Hostname)
	case sel.FederatedID != "":
		agents, err := c.ListAgents(ctx, AgentListOptions{})
		if err != nil {
			return nil, err
		}
		for i := range agents {
			if agents[i].FederatedID == sel.FederatedID {
				return &agents[i], nil
			}
		}
		// Fallback: resolve via runtime environment membership.
		envs, err := c.ListRuntimeEnvironments(ctx, RuntimeListOptions{})
		if err != nil {
			return nil, err
		}
		for _, env := range envs {
			for _, a := range env.Agents {
				if a.FederatedID == sel.FederatedID {
					return c.GetAgent(ctx, a.ID)
				}
			}
		}
		return nil, fmt.Errorf("no agent found with federatedId %q", sel.FederatedID)
	default:
		return nil, fmt.Errorf("no agent selector provided")
	}
}

// AgentInstallerInfo holds the Secure Agent installer download details returned by
// GET /api/v2/agent/installerInfo/<platform>.
type AgentInstallerInfo struct {
	Type                string `json:"@type,omitempty"`
	DownloadURL         string `json:"downloadUrl,omitempty"`
	InstallToken        string `json:"installToken,omitempty"`
	ChecksumDownloadURL string `json:"checksumDownloadUrl,omitempty"`
}

// GetAgentInstallerInfo retrieves the Secure Agent installer information for a
// platform. Valid platform values are "win64" and "linux64".
func (c *Client) GetAgentInstallerInfo(ctx context.Context, platform string) (*AgentInstallerInfo, error) {
	var resp AgentInstallerInfo
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/agent/installerInfo/%s", BaseAPIPathV2, platform), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DownloadFile streams an absolute URL to dest and returns the number of bytes written.
// The installer binary and checksum files are served from a public CDN and require
// no session header.
func (c *Client) DownloadFile(ctx context.Context, url string, dest io.Writer) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("creating request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("downloading %s: unexpected status %s", url, resp.Status)
	}
	n, err := io.Copy(dest, resp.Body)
	if err != nil {
		return n, fmt.Errorf("downloading %s: %w", url, err)
	}
	return n, nil
}

// FetchText fetches an absolute URL and returns the response body as a string.
func (c *Client) FetchText(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", url, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetching %s: unexpected status %s", url, resp.Status)
	}
	return string(body), nil
}

// AgentServiceAction is the operation to perform on an agent service.
type AgentServiceAction string

const (
	AgentServiceStart AgentServiceAction = "start"
	AgentServiceStop  AgentServiceAction = "stop"
)

// SetAgentServiceState starts or stops a service on a secure agent using the v3
// API: POST public/core/v3/agent/service with
// {"agentId","serviceName","serviceAction"}.
func (c *Client) SetAgentServiceState(ctx context.Context, agentID, serviceName string, action AgentServiceAction) error {
	body := map[string]string{
		"agentId":       agentID,
		"serviceName":   serviceName,
		"serviceAction": string(action),
	}
	return c.doJSON(ctx, http.MethodPost, fmt.Sprintf("%s/agent/service", BaseAPIPathV3), body, nil)
}
