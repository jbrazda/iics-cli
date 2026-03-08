package client

import (
	"context"
	"fmt"
	"net/http"
)

// Folder represents an IICS folder as returned by the v3 API.
type Folder struct {
	ID          string `json:"id,omitempty"`
	OrgID       string `json:"orgId,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	UpdatedBy   string `json:"updatedBy,omitempty"`
	UpdateTime  string `json:"updateTime,omitempty"`
	CreateTime  string `json:"createTime,omitempty"`
	CreatedBy   string `json:"createdBy,omitempty"`
}

// folderRequest is the body sent to create or update a folder.
// Only name and description are accepted by the v3 API.
type folderRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// CreateFolder creates a new folder.
//   - projectID and projectName are mutually exclusive; if neither is set the
//     folder is created in the Default project.
func (c *Client) CreateFolder(ctx context.Context, name, description, projectID, projectName string) (*Folder, error) {
	path, err := folderBasePath(projectID, projectName)
	if err != nil {
		return nil, err
	}

	body := folderRequest{Name: name, Description: description}
	var resp Folder
	if err := c.doJSON(ctx, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateFolder updates name and/or description of an existing folder using PATCH.
//   - When projectName and folderName are both set, uses the name-based URL path.
//   - When projectID is set, uses project ID + folder ID path.
//   - Otherwise uses the default /folders/<folderID> path.
func (c *Client) UpdateFolder(ctx context.Context, folderID, name, description, projectID, projectName, folderName string) (*Folder, error) {
	var path string
	switch {
	case projectName != "" && folderName != "":
		path = fmt.Sprintf("%s/projects/name/%s/folders/name/%s", BaseAPIPathV3, projectName, folderName)
	case projectID != "":
		if folderID == "" {
			return nil, fmt.Errorf("--id is required when --project-id is set")
		}
		path = fmt.Sprintf("%s/projects/%s/folders/%s", BaseAPIPathV3, projectID, folderID)
	default:
		if folderID == "" {
			return nil, fmt.Errorf("--id is required")
		}
		path = fmt.Sprintf("%s/folders/%s", BaseAPIPathV3, folderID)
	}

	body := folderRequest{Name: name, Description: description}
	var resp Folder
	if err := c.doJSON(ctx, http.MethodPatch, path, body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteFolder deletes a folder by ID.
func (c *Client) DeleteFolder(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("%s/folders/%s", BaseAPIPathV3, id), nil, nil)
}

// folderBasePath returns the POST base path for folder creation.
func folderBasePath(projectID, projectName string) (string, error) {
	if projectID != "" && projectName != "" {
		return "", fmt.Errorf("--project-id and --project-name are mutually exclusive")
	}
	if projectID != "" {
		return fmt.Sprintf("%s/projects/%s/folders", BaseAPIPathV3, projectID), nil
	}
	if projectName != "" {
		return fmt.Sprintf("%s/projects/name/%s/folders", BaseAPIPathV3, projectName), nil
	}
	return fmt.Sprintf("%s/folders", BaseAPIPathV3), nil
}
