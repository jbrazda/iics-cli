package client

import (
	"context"
	"fmt"
	"net/http"
)

// Folder represents an IICS folder.
type Folder struct {
	ID          string `json:"id,omitempty"`
	OrgID       string `json:"orgId,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ParentID    string `json:"parentId,omitempty"`
	CreateTime  string `json:"createTime,omitempty"`
	UpdateTime  string `json:"updateTime,omitempty"`
	CreatedBy   string `json:"createdBy,omitempty"`
	UpdatedBy   string `json:"updatedBy,omitempty"`
}

// CreateFolder creates a new folder.
func (c *Client) CreateFolder(ctx context.Context, folder *Folder) (*Folder, error) {
	var resp Folder
	if err := c.doJSON(ctx, http.MethodPost, "folders", folder, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateFolder updates an existing folder.
func (c *Client) UpdateFolder(ctx context.Context, id string, folder *Folder) (*Folder, error) {
	var resp Folder
	if err := c.doJSON(ctx, http.MethodPut, fmt.Sprintf("folders/%s", id), folder, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteFolder deletes a folder by ID.
func (c *Client) DeleteFolder(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("folders/%s", id), nil, nil)
}
