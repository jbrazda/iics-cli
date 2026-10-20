package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
//   - Otherwise uses /folders/<folderID>. Folder IDs are unique in the org, so
//     this works for folders in any project. projectID is accepted for
//     compatibility but not used: the documented
//     /projects/<projectID>/folders/<folderID> path returns 404 (verified
//     2026-10-19).
//
// IDs and names cannot be mixed in one URI. The API may return an empty body
// on success, in which case the returned Folder is empty.
func (c *Client) UpdateFolder(ctx context.Context, folderID, name, description, projectID, projectName, folderName string) (*Folder, error) {
	byName := projectName != "" || folderName != ""
	if byName && (folderID != "" || projectID != "") {
		return nil, fmt.Errorf("use either --id/--project-id or --project-name/--folder-name, not both")
	}
	if name == "" && description == "" {
		return nil, fmt.Errorf("nothing to update: provide --name and/or --description")
	}

	var path string
	switch {
	case byName:
		if projectName == "" || folderName == "" {
			return nil, fmt.Errorf("--project-name and --folder-name must be used together")
		}
		path = fmt.Sprintf("%s/projects/name/%s/folders/name/%s", BaseAPIPathV3, url.PathEscape(projectName), url.PathEscape(folderName))
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
		return fmt.Sprintf("%s/projects/name/%s/folders", BaseAPIPathV3, url.PathEscape(projectName)), nil
	}
	return fmt.Sprintf("%s/folders", BaseAPIPathV3), nil
}
