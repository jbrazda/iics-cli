package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// Exit codes following POSIX conventions.
const (
	ExitOK         = 0 // Successful execution
	ExitError      = 1 // Runtime / API errors
	ExitUsageError = 2 // Invalid command usage, missing flags, etc.
)

// APIError represents a structured error from the IICS API.
// It captures the full HTTP response details for diagnostic output.
type APIError struct {
	StatusCode      int             `json:"-"`
	Code            string          `json:"code"`
	Message         string          `json:"message"`
	RequestID       string          `json:"requestId,omitempty"`
	ResponseHeaders http.Header     `json:"-"`
	ResponseBody    json.RawMessage `json:"-"`
}

// Error returns a short, single-line error summary.
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("IICS API error %s (HTTP %d): %s", e.Code, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("IICS API error (HTTP %d): %s", e.StatusCode, e.Message)
}

// Verbose returns a detailed, multi-line error report including
// HTTP status, response headers, and the formatted JSON response body.
func (e *APIError) Verbose() string {
	var b strings.Builder

	// Status line
	b.WriteString(fmt.Sprintf("HTTP %d %s\n", e.StatusCode, http.StatusText(e.StatusCode)))

	// Response headers (sorted for deterministic output)
	if len(e.ResponseHeaders) > 0 {
		b.WriteString("\nResponse Headers:\n")
		keys := make([]string, 0, len(e.ResponseHeaders))
		for k := range e.ResponseHeaders {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			for _, v := range e.ResponseHeaders[k] {
				b.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
			}
		}
	}

	// Response body (pretty-printed JSON if possible)
	if len(e.ResponseBody) > 0 {
		b.WriteString("\nResponse Body:\n")
		var pretty bytes.Buffer
		if json.Indent(&pretty, e.ResponseBody, "  ", "  ") == nil {
			b.WriteString("  ")
			b.Write(pretty.Bytes())
			b.WriteString("\n")
		} else {
			// Not valid JSON; print as-is
			b.WriteString("  ")
			b.Write(e.ResponseBody)
			b.WriteString("\n")
		}
	}

	return b.String()
}

// IsNotFound returns true for 404 responses.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

// IsUnauthorized returns true for 401 responses.
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

// IsTooManyRequests returns true for 429 responses.
func (e *APIError) IsTooManyRequests() bool {
	return e.StatusCode == 429
}

// newAPIError constructs an APIError from an HTTP response and its body bytes.
func newAPIError(resp *http.Response, body []byte) *APIError {
	apiErr := &APIError{
		StatusCode:      resp.StatusCode,
		ResponseHeaders: resp.Header.Clone(),
		ResponseBody:    json.RawMessage(body),
	}

	// Try to parse structured error fields from the body
	if json.Unmarshal(body, apiErr) != nil {
		// Not a structured IICS error; use raw body as message
		apiErr.Message = string(body)
	}

	return apiErr
}

// SessionExpiredError signals a 401 that triggers re-login.
type SessionExpiredError struct {
	Wrapped error
}

func (e *SessionExpiredError) Error() string {
	return fmt.Sprintf("session expired: %v", e.Wrapped)
}

func (e *SessionExpiredError) Unwrap() error {
	return e.Wrapped
}
