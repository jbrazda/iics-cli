package client

import (
	"context"
	"fmt"
	"net/http"
)

// SourceControlCheckoutRequest is the request to check out an object.
type SourceControlCheckoutRequest struct {
	ObjectID string `json:"objectId"`
}

// SourceControlCheckinRequest is the request to check in an object.
type SourceControlCheckinRequest struct {
	ObjectID string `json:"objectId"`
	Comment  string `json:"comment,omitempty"`
}

// SourceControlCommitRequest is the request to commit changes.
type SourceControlCommitRequest struct {
	Comment string `json:"comment,omitempty"`
}

// SourceControlResult represents the result of a source control operation.
type SourceControlResult struct {
	ObjectID string `json:"objectId,omitempty"`
	Status   string `json:"status,omitempty"`
	Message  string `json:"message,omitempty"`
}

// Checkout checks out an object from source control.
func (c *Client) Checkout(ctx context.Context, objectID string) (*SourceControlResult, error) {
	body := SourceControlCheckoutRequest{ObjectID: objectID}
	var resp SourceControlResult
	if err := c.doJSON(ctx, http.MethodPost, "public/core/v3/sourceControl/checkout", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Checkin checks in an object to source control.
func (c *Client) Checkin(ctx context.Context, objectID, comment string) (*SourceControlResult, error) {
	body := SourceControlCheckinRequest{ObjectID: objectID, Comment: comment}
	var resp SourceControlResult
	if err := c.doJSON(ctx, http.MethodPost, "public/core/v3/sourceControl/checkin", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Pull pulls changes from source control.
func (c *Client) Pull(ctx context.Context) (*SourceControlResult, error) {
	var resp SourceControlResult
	if err := c.doJSON(ctx, http.MethodPost, "public/core/v3/sourceControl/pull", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Commit commits changes to source control.
func (c *Client) Commit(ctx context.Context, comment string) (*SourceControlResult, error) {
	body := SourceControlCommitRequest{Comment: comment}
	var resp SourceControlResult
	if err := c.doJSON(ctx, http.MethodPost, "public/core/v3/sourceControl/commit", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetSourceControlStatus retrieves the source control status of an object.
func (c *Client) GetSourceControlStatus(ctx context.Context, objectID string) (*SourceControlResult, error) {
	var resp SourceControlResult
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("public/core/v3/sourceControl/%s", objectID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
