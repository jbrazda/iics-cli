package client

import (
	"context"
	"fmt"
	"net/http"
)

// ObjectPermission represents permissions on an IICS object.
type ObjectPermission struct {
	ObjectID    string       `json:"objectId"`
	ObjectType  string       `json:"objectType,omitempty"`
	Permissions []Permission `json:"permissions,omitempty"`
}

// Permission represents a single permission entry.
type Permission struct {
	PrincipalID   string `json:"principalId"`
	PrincipalType string `json:"principalType"`
	PrincipalName string `json:"principalName,omitempty"`
	Permission    string `json:"permission"`
}

// GetObjectPermissions retrieves permissions for an object.
func (c *Client) GetObjectPermissions(ctx context.Context, objectID string) (*ObjectPermission, error) {
	var resp ObjectPermission
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("public/core/v3/objects/%s/permissions", objectID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetObjectPermissions sets permissions on an object.
func (c *Client) SetObjectPermissions(ctx context.Context, objectID string, perms *ObjectPermission) (*ObjectPermission, error) {
	var resp ObjectPermission
	if err := c.doJSON(ctx, http.MethodPut, fmt.Sprintf("public/core/v3/objects/%s/permissions", objectID), perms, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteObjectPermissions deletes permissions from an object.
func (c *Client) DeleteObjectPermissions(ctx context.Context, objectID string) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("public/core/v3/objects/%s/permissions", objectID), nil, nil)
}
