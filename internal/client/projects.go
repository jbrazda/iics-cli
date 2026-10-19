package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
	if err := c.doJSON(ctx, http.MethodPost, BaseAPIPathV3+"/projects", project, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// projectRequest is the body sent to update a project.
type projectRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// UpdateProject changes the name and/or description of a project using PATCH.
// The project is identified by projectID or by its current projectName
// (mutually exclusive). The API may return an empty body on success, in which
// case the returned Project is empty.
func (c *Client) UpdateProject(ctx context.Context, projectID, projectName, name, description string) (*Project, error) {
	var path string
	switch {
	case projectID != "" && projectName != "":
		return nil, fmt.Errorf("--id and --project-name are mutually exclusive")
	case projectID != "":
		path = fmt.Sprintf("%s/projects/%s", BaseAPIPathV3, projectID)
	case projectName != "":
		path = fmt.Sprintf("%s/projects/name/%s", BaseAPIPathV3, url.PathEscape(projectName))
	default:
		return nil, fmt.Errorf("--id or --project-name is required")
	}
	if name == "" && description == "" {
		return nil, fmt.Errorf("nothing to update: provide --name and/or --description")
	}

	body := projectRequest{Name: name, Description: description}
	var resp Project
	if err := c.doJSON(ctx, http.MethodPatch, path, body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteProject deletes a project by ID.
func (c *Client) DeleteProject(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("%s/projects/%s", BaseAPIPathV3, id), nil, nil)
}
