// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Retry-After is server-controlled input. An uncapped value pauses the provider
// for as long as the server asks, which is indistinguishable from a hang.
func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		value string
		want  time.Duration
		ok    bool
	}{
		{"seconds", "30", 30 * time.Second, true},
		{"seconds padded", "  30  ", 30 * time.Second, true},
		{"seconds capped", "3600", maxRetryAfter, true},
		{"seconds at the cap", "60", 60 * time.Second, true},
		{"zero means retry now", "0", 0, false},
		{"negative is ignored", "-5", 0, false},

		// The HTTP-date form is legal and was previously ignored, silently
		// falling back to the shorter default backoff.
		{"http date", "Thu, 06 Aug 2026 12:00:45 GMT", 45 * time.Second, true},
		{"http date capped", "Thu, 06 Aug 2026 13:00:00 GMT", maxRetryAfter, true},
		{"http date in the past", "Thu, 06 Aug 2026 11:59:00 GMT", 0, false},

		{"empty", "", 0, false},
		{"garbage", "soon", 0, false},
		// Fractional seconds are not delay-seconds. The previous implementation
		// used ParseDuration(value+"s"), which accepted this.
		{"fractional is not a valid delay", "1.5", 0, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := parseRetryAfter(c.value, now)
			if ok != c.ok {
				t.Fatalf("ok = %v, want %v (value %q)", ok, c.ok, c.value)
			}
			if ok && got != c.want {
				t.Errorf("got %v, want %v (value %q)", got, c.want, c.value)
			}
		})
	}
}

// The cap has to hold end to end, not just in the helper: a 429 asking for an
// hour must not stall the request for an hour.
func TestDoRequest_RetryAfterIsCapped(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/identity/v1/apikey/exchange" {
			_, _ = w.Write([]byte("test-token"))
			return
		}
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "3600")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	// Shorten the cap so the test proves the wiring without sleeping for it.
	original := maxRetryAfter
	maxRetryAfter = 100 * time.Millisecond
	t.Cleanup(func() { maxRetryAfter = original })

	c := newTestClient(t, srv)

	start := time.Now()
	if _, err := c.doRequest("GET", "/api/v1/framework", nil); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	elapsed := time.Since(start)

	if attempts != 2 {
		t.Fatalf("expected one retry, got %d attempts", attempts)
	}
	// The server asked for an hour. Anything near the default 2s backoff would
	// also pass a naive check, so assert against the cap that is actually in force.
	if elapsed > time.Second {
		t.Errorf("waited %v; the Retry-After of 3600s was not capped to %v", elapsed, maxRetryAfter)
	}
}
