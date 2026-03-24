package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	// sessionHeader is the header used for session authentication in API requests.
	// Note: IICS v3 uses "INFA-SESSION-ID" while v2 used "icSessionId".
	sessionHeaderV3 = "INFA-SESSION-ID"
	sessionHeaderV2 = "icSessionId"
	// baseAPIPathV2 is the base path for v2 API endpoints.
	BaseAPIPathV2 = "api/v2"
	// baseAPIPathV3 is the base path for v3 API endpoints.
	BaseAPIPathV3 = "public/core/v3"
)

// Client is the IICS v3 API client with automatic session management.
type Client struct {
	httpClient     *http.Client
	loginURL       string
	baseAPIURL     string
	caiURL         string
	sessionID      string
	username       string
	password       string
	verbose        bool
	debug          bool
	onLoginSuccess func(*LoginResponse)
	mu             sync.RWMutex
}

// ClientOption is a functional option for Client construction.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithVerbose enables verbose logging.
func WithVerbose(v bool) ClientOption {
	return func(c *Client) {
		c.verbose = v
	}
}

// WithDebug enables debug mode, which prints the request body to stderr on API errors.
func WithDebug(v bool) ClientOption {
	return func(c *Client) {
		c.debug = v
	}
}

// WithCAIURL sets the CAI-specific base URL (overrides auto-detection from login response).
func WithCAIURL(url string) ClientOption {
	return func(c *Client) { c.caiURL = url }
}

// NewClient creates a new IICS API client.
func NewClient(loginURL, username, password string, opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		loginURL:   loginURL,
		username:   username,
		password:   password,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// SetSession sets the session ID and base API URL directly (e.g., from cache).
func (c *Client) SetSession(sessionID, baseAPIURL string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionID = sessionID
	c.baseAPIURL = baseAPIURL
}

// SessionID returns the current session ID.
func (c *Client) SessionID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessionID
}

// BaseAPIURL returns the current base API URL.
func (c *Client) BaseAPIURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.baseAPIURL
}

// CAIURL returns the CAI-specific base URL.
func (c *Client) CAIURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.caiURL
}

// SetCAIURL sets the CAI-specific base URL (e.g., restored from session cache).
func (c *Client) SetCAIURL(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.caiURL = url
}

// SetOnLoginSuccess registers a callback that is invoked after every successful
// login (initial auto-login or 401 renewal). Use this to persist the session.
func (c *Client) SetOnLoginSuccess(fn func(*LoginResponse)) {
	c.onLoginSuccess = fn
}

// apiURL constructs a full API URL from a resource path.
func (c *Client) apiURL(resourcePath string) string {
	c.mu.RLock()
	base := c.baseAPIURL
	c.mu.RUnlock()

	base = strings.TrimRight(base, "/")
	return fmt.Sprintf("%s/%s", base, resourcePath)
}

// doWithSession executes an HTTP request with the session header injected.
func (c *Client) doWithSession(ctx context.Context, req *http.Request) (*http.Response, error) {
	c.mu.RLock()
	sessionID := c.sessionID
	c.mu.RUnlock()
	if strings.Contains(req.URL.RequestURI(), "/v2/") {
		req.Header.Set(sessionHeaderV2, sessionID)
	} else {
		req.Header.Set(sessionHeaderV3, sessionID)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	if req.Header.Get("Content-Type") == "" && req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// do executes an HTTP request with session header injection.
// On 401 response, it re-authenticates and retries once.
func (c *Client) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Ensure we have a session
	c.mu.RLock()
	hasSession := c.sessionID != ""
	c.mu.RUnlock()

	if !hasSession {
		if _, err := c.Login(ctx); err != nil {
			return nil, fmt.Errorf("auto-login failed: %w", err)
		}
	}

	// Save request body for potential retry
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("reading request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	resp, err := c.doWithSession(ctx, req)
	if err != nil {
		return nil, err
	}

	// On 401, re-authenticate and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		_ = resp.Body.Close()

		if _, err := c.Login(ctx); err != nil {
			return nil, &SessionExpiredError{Wrapped: fmt.Errorf("%w; run 'iics login' to re-authenticate", err)}
		}

		// Rebuild the request with fresh body
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
		return c.doWithSession(ctx, req)
	}

	return resp, nil
}

// ensureSession guarantees the client has a valid session before a URL is built.
// It must be called before c.apiURL() so that baseAPIURL is populated.
func (c *Client) ensureSession(ctx context.Context) error {
	c.mu.RLock()
	hasSession := c.sessionID != ""
	c.mu.RUnlock()
	if !hasSession {
		if _, err := c.Login(ctx); err != nil {
			return fmt.Errorf("auto-login failed: %w", err)
		}
	}
	return nil
}

// prettyJSONString pretty-prints JSON bytes; returns the raw string on failure.
func prettyJSONString(data []byte) string {
	var buf bytes.Buffer
	if json.Indent(&buf, data, "", "  ") == nil {
		return buf.String()
	}
	return string(data)
}

// debugPrintHTTP writes a human-readable request/response trace to stderr.
// It is level-gated via slog: output is suppressed unless the default logger
// is configured at DEBUG level (i.e. --debug flag is set).
// Session header values are masked. JSON bodies are pretty-printed.
func debugPrintHTTP(req *http.Request, reqData []byte, resp *http.Response, respData []byte) {
	if !slog.Default().Enabled(req.Context(), slog.LevelDebug) {
		return
	}
	w := os.Stderr
	_, _ = fmt.Fprintf(w, "DEBUG > %s %s\n", req.Method, req.URL)
	_, _ = fmt.Fprintln(w, "Request Headers:")
	for k, vs := range req.Header {
		for _, v := range vs {
			if strings.EqualFold(k, sessionHeaderV3) || strings.EqualFold(k, sessionHeaderV2) {
				_, _ = fmt.Fprintf(w, "  %s: ***\n", k)
			} else {
				_, _ = fmt.Fprintf(w, "  %s: %s\n", k, v)
			}
		}
	}
	if len(reqData) > 0 {
		_, _ = fmt.Fprintln(w, "Request Body:")
		_, _ = fmt.Fprintf(w, "  %s\n", prettyJSONString(reqData))
	}
	_, _ = fmt.Fprintf(w, "DEBUG < %s\n", resp.Status)
	_, _ = fmt.Fprintln(w, "Response Headers:")
	for k, vs := range resp.Header {
		for _, v := range vs {
			_, _ = fmt.Fprintf(w, "  %s: %s\n", k, v)
		}
	}
	if len(respData) > 0 {
		_, _ = fmt.Fprintln(w, "Response Body:")
		_, _ = fmt.Fprintf(w, "  %s\n", prettyJSONString(respData))
	}
	_, _ = fmt.Fprintln(w)
}

// doJSON is a convenience method that sends a JSON request and decodes the response.
func (c *Client) doJSON(ctx context.Context, method, path string, reqBody, respBody interface{}) error {
	if err := c.ensureSession(ctx); err != nil {
		return err
	}

	var body io.Reader
	var reqData []byte
	if reqBody != nil {
		var err error
		reqData, err = json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request: %w", err)
		}
		body = bytes.NewReader(reqData)
	}

	url := c.apiURL(path)
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.do(ctx, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if c.debug {
		debugPrintHTTP(req, reqData, resp, respData)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newAPIError(resp, respData)
	}

	if respBody != nil && len(respData) > 0 {
		if err := json.Unmarshal(respData, respBody); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
	}

	return nil
}

// doJSONWithQuery is like doJSON but with query parameters.
func (c *Client) doJSONWithQuery(ctx context.Context, method, path string, query map[string]string, reqBody, respBody interface{}) error {
	if err := c.ensureSession(ctx); err != nil {
		return err
	}

	var body io.Reader
	var reqData []byte
	if reqBody != nil {
		var err error
		reqData, err = json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request: %w", err)
		}
		body = bytes.NewReader(reqData)
	}

	url := c.apiURL(path)
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if len(query) > 0 {
		q := req.URL.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.do(ctx, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if c.debug {
		debugPrintHTTP(req, reqData, resp, respData)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newAPIError(resp, respData)
	}

	if respBody != nil && len(respData) > 0 {
		if err := json.Unmarshal(respData, respBody); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
	}

	return nil
}

// doCAIJSON sends a JSON:API request to an absolute CAI URL and decodes the response.
// It sets application/vnd.api+json Content-Type and bypasses c.apiURL().
func (c *Client) doCAIJSON(ctx context.Context, method, absoluteURL string, reqBody, respBody interface{}) error {
	if err := c.ensureSession(ctx); err != nil {
		return err
	}
	var body io.Reader
	var reqData []byte
	if reqBody != nil {
		var err error
		reqData, err = json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request: %w", err)
		}
		body = bytes.NewReader(reqData)
	}
	req, err := http.NewRequestWithContext(ctx, method, absoluteURL, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Accept", "application/vnd.api+json")
	resp, err := c.do(ctx, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if c.debug {
		debugPrintHTTP(req, reqData, resp, respData)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newAPIError(resp, respData)
	}
	if respBody != nil && len(respData) > 0 {
		if err := json.Unmarshal(respData, respBody); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
	}
	return nil
}

// doRaw performs an HTTP request and returns the raw response body reader.
// The caller is responsible for closing the reader.
func (c *Client) doRaw(ctx context.Context, method, path string, query map[string]string) (io.ReadCloser, error) {
	if err := c.ensureSession(ctx); err != nil {
		return nil, err
	}

	url := c.apiURL(path)
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if len(query) > 0 {
		q := req.URL.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer func() { _ = resp.Body.Close() }()
		respData, _ := io.ReadAll(resp.Body)
		return nil, newAPIError(resp, respData)
	}

	return resp.Body, nil
}
