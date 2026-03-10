package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// LoginRequest is the v3 login request body.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse is the v3 login response.
type LoginResponse struct {
	Products []Product `json:"products"`
	UserInfo UserInfo  `json:"userInfo"`
}

// Product represents a product entry in the login response.
type Product struct {
	Name       string `json:"name"`
	BaseAPIURL string `json:"baseApiUrl"`
}

// UserInfo represents the user information in the login response.
type UserInfo struct {
	SessionID   string `json:"sessionId"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	OrgID       string `json:"orgId"`
	OrgName     string `json:"orgName"`
	ParentOrgID string `json:"parentOrgId"`
	Status      string `json:"status"`
}

// Login authenticates with the IICS API and stores the session.
func (c *Client) Login(ctx context.Context) (*LoginResponse, error) {
	reqBody := LoginRequest{
		Username: c.username,
		Password: c.password,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling login request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.loginURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading login response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, newAPIError(resp, respBody)
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(respBody, &loginResp); err != nil {
		return nil, fmt.Errorf("parsing login response: %w", err)
	}

	if loginResp.UserInfo.SessionID == "" {
		return nil, fmt.Errorf("login succeeded but no session ID in response")
	}

	// Find the base API URL from products
	baseURL := ""
	for _, p := range loginResp.Products {
		if baseURL == "" {
			baseURL = p.BaseAPIURL
		}
	}

	c.mu.Lock()
	c.sessionID = loginResp.UserInfo.SessionID
	c.baseAPIURL = baseURL
	c.mu.Unlock()

	return &loginResp, nil
}

// Logout invalidates the current session.
func (c *Client) Logout(ctx context.Context) error {
	c.mu.RLock()
	if c.sessionID == "" {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	url := c.apiURL(fmt.Sprintf("%s/logout", BaseAPIPathV3))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("creating logout request: %w", err)
	}

	resp, err := c.doWithSession(ctx, req)
	if err != nil {
		return fmt.Errorf("logout request failed: %w", err)
	}
	defer resp.Body.Close()

	c.mu.Lock()
	c.sessionID = ""
	c.baseAPIURL = ""
	c.mu.Unlock()

	return nil
}
