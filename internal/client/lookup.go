package client

import (
	"context"
	"net/http"
)

// LookupObject specifies an object to look up by ID or path+type.
type LookupObject struct {
	ID   string `json:"id,omitempty"`
	Path string `json:"path,omitempty"`
	Type string `json:"type,omitempty"`
}

// LookupRequest is the request body for the lookup API.
type LookupRequest struct {
	Objects []LookupObject `json:"objects"`
}

// LookupResult represents a resolved object from the lookup API.
type LookupResult struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Description string `json:"description"`
	UpdatedBy   string `json:"updatedBy"`
	UpdateTime  string `json:"updateTime"`
}

// LookupResponse is the response from the lookup API.
type LookupResponse struct {
	Objects []LookupResult `json:"objects"`
}

// Lookup resolves one or more objects by ID, path+type, or a mix.
func (c *Client) Lookup(ctx context.Context, objects []LookupObject) (*LookupResponse, error) {
	req := LookupRequest{Objects: objects}
	var resp LookupResponse
	if err := c.doJSON(ctx, http.MethodPost, "lookup", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
