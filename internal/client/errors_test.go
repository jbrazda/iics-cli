package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIErrorError(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		expected string
	}{
		{
			name:     "with code",
			err:      &APIError{StatusCode: 403, Code: "AUTH_FAILED", Message: "Invalid credentials"},
			expected: "IICS API error AUTH_FAILED (HTTP 403): Invalid credentials",
		},
		{
			name:     "without code",
			err:      &APIError{StatusCode: 500, Message: "Internal server error"},
			expected: "IICS API error (HTTP 500): Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAPIErrorVerbose(t *testing.T) {
	body := json.RawMessage(`{"code":"NOT_FOUND","message":"Object not found","requestId":"req-123"}`)
	headers := http.Header{
		"Content-Type":   {"application/json"},
		"X-Request-Id":   {"req-456"},
		"X-Infa-Org-Id":  {"org-789"},
	}

	apiErr := &APIError{
		StatusCode:      404,
		Code:            "NOT_FOUND",
		Message:         "Object not found",
		RequestID:       "req-123",
		ResponseHeaders: headers,
		ResponseBody:    body,
	}

	verbose := apiErr.Verbose()

	// Should contain HTTP status line
	if !strings.Contains(verbose, "HTTP 404 Not Found") {
		t.Errorf("Verbose() missing status line, got:\n%s", verbose)
	}

	// Should contain response headers
	if !strings.Contains(verbose, "Content-Type: application/json") {
		t.Errorf("Verbose() missing Content-Type header, got:\n%s", verbose)
	}
	if !strings.Contains(verbose, "X-Request-Id: req-456") {
		t.Errorf("Verbose() missing X-Request-Id header, got:\n%s", verbose)
	}

	// Should contain formatted JSON body
	if !strings.Contains(verbose, `"code"`) {
		t.Errorf("Verbose() missing JSON body, got:\n%s", verbose)
	}
	if !strings.Contains(verbose, `"NOT_FOUND"`) {
		t.Errorf("Verbose() missing error code in body, got:\n%s", verbose)
	}
}

func TestAPIErrorVerboseNoHeaders(t *testing.T) {
	apiErr := &APIError{
		StatusCode:   500,
		Message:      "something failed",
		ResponseBody: json.RawMessage(`{"error":"bad"}`),
	}

	verbose := apiErr.Verbose()
	if !strings.Contains(verbose, "HTTP 500") {
		t.Errorf("Verbose() missing status, got:\n%s", verbose)
	}
	if strings.Contains(verbose, "Response Headers:") {
		t.Errorf("Verbose() should not show headers section when empty, got:\n%s", verbose)
	}
}

func TestAPIErrorVerboseNonJSONBody(t *testing.T) {
	apiErr := &APIError{
		StatusCode:   502,
		Message:      "Bad Gateway",
		ResponseBody: json.RawMessage(`<html>Bad Gateway</html>`),
	}

	verbose := apiErr.Verbose()
	if !strings.Contains(verbose, "<html>Bad Gateway</html>") {
		t.Errorf("Verbose() should print non-JSON body as-is, got:\n%s", verbose)
	}
}

func TestNewAPIError(t *testing.T) {
	// Simulate an HTTP response
	recorder := httptest.NewRecorder()
	recorder.Header().Set("Content-Type", "application/json")
	recorder.Header().Set("X-Request-Id", "test-req-id")
	recorder.WriteHeader(http.StatusNotFound)

	body := []byte(`{"code":"NOT_FOUND","message":"Connection not found","requestId":"req-abc"}`)

	resp := recorder.Result()
	apiErr := newAPIError(resp, body)

	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if apiErr.Code != "NOT_FOUND" {
		t.Errorf("Code = %q, want NOT_FOUND", apiErr.Code)
	}
	if apiErr.Message != "Connection not found" {
		t.Errorf("Message = %q, want 'Connection not found'", apiErr.Message)
	}
	if apiErr.RequestID != "req-abc" {
		t.Errorf("RequestID = %q, want 'req-abc'", apiErr.RequestID)
	}
	if apiErr.ResponseHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("ResponseHeaders missing Content-Type")
	}
	if apiErr.ResponseHeaders.Get("X-Request-Id") != "test-req-id" {
		t.Errorf("ResponseHeaders missing X-Request-Id")
	}
	if string(apiErr.ResponseBody) != string(body) {
		t.Errorf("ResponseBody = %q, want %q", string(apiErr.ResponseBody), string(body))
	}
}

func TestNewAPIErrorNonJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	recorder.Header().Set("Content-Type", "text/html")
	recorder.WriteHeader(http.StatusBadGateway)

	body := []byte("<html>Bad Gateway</html>")
	resp := recorder.Result()
	apiErr := newAPIError(resp, body)

	if apiErr.StatusCode != 502 {
		t.Errorf("StatusCode = %d, want 502", apiErr.StatusCode)
	}
	// Non-JSON body should be used as the message
	if apiErr.Message != "<html>Bad Gateway</html>" {
		t.Errorf("Message = %q, want raw body", apiErr.Message)
	}
}

func TestAPIErrorFromHTTPClient(t *testing.T) {
	// Verify that APIError returned from doJSON contains headers and body
	errorBody := map[string]string{
		"code":      "VALIDATION_ERROR",
		"message":   "Name is required",
		"requestId": "req-xyz",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Infa-Request-Id", "server-req-id")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorBody)
	})

	c := newTestClient(handler)
	var result map[string]interface{}
	err := c.doJSON(context.Background(), http.MethodPost, "connections", nil, &result)
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}

	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
	if apiErr.Code != "VALIDATION_ERROR" {
		t.Errorf("Code = %q, want VALIDATION_ERROR", apiErr.Code)
	}
	if apiErr.ResponseHeaders == nil {
		t.Fatal("ResponseHeaders should not be nil")
	}
	if apiErr.ResponseHeaders.Get("X-Infa-Request-Id") != "server-req-id" {
		t.Errorf("ResponseHeaders missing X-Infa-Request-Id")
	}
	if len(apiErr.ResponseBody) == 0 {
		t.Error("ResponseBody should not be empty")
	}
}

func TestAPIErrorHelpers(t *testing.T) {
	tests := []struct {
		status    int
		notFound  bool
		unauth    bool
		tooMany   bool
	}{
		{404, true, false, false},
		{401, false, true, false},
		{429, false, false, true},
		{500, false, false, false},
	}

	for _, tt := range tests {
		apiErr := &APIError{StatusCode: tt.status}
		if apiErr.IsNotFound() != tt.notFound {
			t.Errorf("status %d: IsNotFound() = %v, want %v", tt.status, apiErr.IsNotFound(), tt.notFound)
		}
		if apiErr.IsUnauthorized() != tt.unauth {
			t.Errorf("status %d: IsUnauthorized() = %v, want %v", tt.status, apiErr.IsUnauthorized(), tt.unauth)
		}
		if apiErr.IsTooManyRequests() != tt.tooMany {
			t.Errorf("status %d: IsTooManyRequests() = %v, want %v", tt.status, apiErr.IsTooManyRequests(), tt.tooMany)
		}
	}
}

func TestSessionExpiredError(t *testing.T) {
	inner := &APIError{StatusCode: 401, Message: "unauthorized"}
	err := &SessionExpiredError{Wrapped: inner}

	if !strings.Contains(err.Error(), "session expired") {
		t.Errorf("Error() = %q, should contain 'session expired'", err.Error())
	}
	if err.Unwrap() != inner {
		t.Error("Unwrap() should return the wrapped error")
	}
}

func TestExitCodes(t *testing.T) {
	if ExitOK != 0 {
		t.Errorf("ExitOK = %d, want 0", ExitOK)
	}
	if ExitError != 1 {
		t.Errorf("ExitError = %d, want 1", ExitError)
	}
	if ExitUsageError != 2 {
		t.Errorf("ExitUsageError = %d, want 2", ExitUsageError)
	}
}
