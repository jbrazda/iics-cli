package client

import (
	"context"
	"fmt"
	"net/http"
)

// Project represents an IICS project.
type Project struct {
	ID          string `json:"id,omitempty"`
	OrgID       string `json:"orgId,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CreateTime  string `json:"createTime,omitempty"`
	UpdateTime  string `json:"updateTime,omitempty"`
	CreatedBy   string `json:"createdBy,omitempty"`
	UpdatedBy   string `json:"updatedBy,omitempty"`
}

// CreateProject creates a new project.
func (c *Client) CreateProject(ctx context.Context, project *Project) (*Project, error) {
	var resp Project
	if err := c.doJSON(ctx, http.MethodPost, "projects", project, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateProject updates an existing project.
func (c *Client) UpdateProject(ctx context.Context, id string, project *Project) (*Project, error) {
	var resp Project
	if err := c.doJSON(ctx, http.MethodPut, fmt.Sprintf("projects/%s", id), project, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteProject deletes a project by ID.
func (c *Client) DeleteProject(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("projects/%s", id), nil, nil)
}
