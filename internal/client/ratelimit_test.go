package client

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

// noSleep replaces sleepCtx for the test and records the requested waits.
func noSleep(t *testing.T) *[]time.Duration {
	t.Helper()
	var waits []time.Duration
	orig := sleepCtx
	sleepCtx = func(_ context.Context, d time.Duration) error {
		waits = append(waits, d)
		return nil
	}
	t.Cleanup(func() { sleepCtx = orig })
	return &waits
}

func TestRateLimitWait(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		headers map[string]string
		want    time.Duration
		ok      bool
	}{
		{"no header", nil, 0, false},
		{"retry-after seconds", map[string]string{"Retry-After": "5"}, 5 * time.Second, true},
		{"retry-after date", map[string]string{"Retry-After": now.Add(7 * time.Second).Format(http.TimeFormat)}, 7 * time.Second, true},
		{"retry-after past date", map[string]string{"Retry-After": now.Add(-time.Minute).Format(http.TimeFormat)}, 0, true},
		{"retry-after capped", map[string]string{"Retry-After": "3600"}, maxRateLimitWait, true},
		{"retry-after invalid", map[string]string{"Retry-After": "soon"}, 0, false},
		{"ratelimit-reset", map[string]string{"ratelimit-reset": "3"}, 3 * time.Second, true},
		{"ratelimit-reset invalid", map[string]string{"ratelimit-reset": "-1"}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.Header{}
			for k, v := range tt.headers {
				h.Set(k, v)
			}
			got, ok := rateLimitWait(h, now)
			if got != tt.want || ok != tt.ok {
				t.Errorf("rateLimitWait() = (%v, %v), want (%v, %v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestRateLimitRetriesWithRetryAfter(t *testing.T) {
	waits := noSleep(t)
	var calls atomic.Int32
	c := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) <= 2 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"objects":[{"id":"1","path":"P/A","type":"TASKFLOW"}]}`))
	}))
	resp, err := c.Lookup(context.Background(), []LookupObject{{Path: "P/A", Type: "TASKFLOW"}})
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if len(resp.Objects) != 1 {
		t.Errorf("objects = %d, want 1", len(resp.Objects))
	}
	if n := calls.Load(); n != 3 {
		t.Errorf("calls = %d, want 3", n)
	}
	if len(*waits) != 2 || (*waits)[0] != 2*time.Second {
		t.Errorf("waits = %v, want [2s 2s]", *waits)
	}
}

func TestRateLimitBackoffWithoutHeader(t *testing.T) {
	waits := noSleep(t)
	var calls atomic.Int32
	c := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) <= 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"code":"429","message":"too many requests"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"objects":[{"id":"1","path":"P/A","type":"TASKFLOW"}]}`))
	}))
	if _, err := c.Lookup(context.Background(), []LookupObject{{Path: "P/A", Type: "TASKFLOW"}}); err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if n := calls.Load(); n != 4 {
		t.Errorf("calls = %d, want 4", n)
	}
	// 2s, 4s, 8s, each plus up to 25% jitter.
	if len(*waits) != 3 {
		t.Fatalf("waits = %v, want 3", *waits)
	}
	for i, base := range []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second} {
		if w := (*waits)[i]; w < base || w > base+base/4 {
			t.Errorf("wait[%d] = %v, want %v..%v", i, w, base, base+base/4)
		}
	}
}

func TestRateLimitGivesUpAfterMaxRetries(t *testing.T) {
	noSleep(t)
	var calls atomic.Int32
	c := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	_, err := c.Lookup(context.Background(), []LookupObject{{Path: "P/A", Type: "TASKFLOW"}})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !apiErr.IsTooManyRequests() {
		t.Fatalf("err = %v, want 429 APIError", err)
	}
	if n := calls.Load(); n != maxRateLimitRetries+1 {
		t.Errorf("calls = %d, want %d", n, maxRateLimitRetries+1)
	}
}
