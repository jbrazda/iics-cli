package client

import (
	"context"
	"fmt"
	"net/http"
)

// TagAssignment represents a tag assignment request.
type TagAssignment struct {
	Tags []string `json:"tags"`
}

// AssignTags assigns tags to an object.
func (c *Client) AssignTags(ctx context.Context, objectID string, tags []string) error {
	body := TagAssignment{Tags: tags}
	return c.doJSON(ctx, http.MethodPost, fmt.Sprintf("objects/%s/tags", objectID), body, nil)
}

// RemoveTags removes tags from an object.
func (c *Client) RemoveTags(ctx context.Context, objectID string, tags []string) error {
	body := TagAssignment{Tags: tags}
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("objects/%s/tags", objectID), body, nil)
}
