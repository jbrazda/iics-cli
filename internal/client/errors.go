package client

import "fmt"

// APIError represents a structured error from the IICS API.
type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	RequestID  string `json:"requestId,omitempty"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("IICS API error %d (%s): %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("IICS API error %d: %s", e.StatusCode, e.Message)
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
