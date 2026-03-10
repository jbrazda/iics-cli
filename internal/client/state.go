package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// FetchStateRequest is the request body for fetchState.
type FetchStateRequest struct {
	ObjectID string `json:"objectId"`
}

// LoadStateRequest is the request body for loadState.
type LoadStateRequest struct {
	ObjectID string `json:"objectId"`
}

// StateResult represents the result of a state operation.
type StateResult struct {
	ObjectID string `json:"objectId,omitempty"`
	Status   string `json:"status,omitempty"`
	Message  string `json:"message,omitempty"`
}

// FetchState fetches the state of an object and writes it to dest.
func (c *Client) FetchState(ctx context.Context, objectID string, dest io.Writer) error {
	path := fmt.Sprintf("%s/fetchState/%s", BaseAPIPathV3, objectID)
	body, err := c.doRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer body.Close()

	if _, err := io.Copy(dest, body); err != nil {
		return fmt.Errorf("downloading state: %w", err)
	}
	return nil
}

// LoadState loads state data for an object.
func (c *Client) LoadState(ctx context.Context, objectID string, data io.Reader) (*StateResult, error) {
	url := c.apiURL(fmt.Sprintf("%s/loadState/%s", BaseAPIPathV3, objectID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, data)
	if err != nil {
		return nil, fmt.Errorf("creating loadState request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading loadState response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newAPIError(resp, respData)
	}

	var result StateResult
	if len(respData) > 0 {
		if err := parseJSON(respData, &result); err != nil {
			return nil, fmt.Errorf("parsing loadState response: %w", err)
		}
	}
	return &result, nil
}
