package client

import (
	"context"
	"net/http"
)

// Privilege represents an IICS privilege.
type Privilege struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Service     string `json:"service"`
	Status      string `json:"status"`
}

// ListPrivileges retrieves all available privileges.
func (c *Client) ListPrivileges(ctx context.Context) ([]Privilege, error) {
	var resp []Privilege
	if err := c.doJSON(ctx, http.MethodGet, "privileges", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
