// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// shortenRetryBackoff keeps retry tests fast. Without it the 2s/4s/6s production
// backoff dominates the unit suite.
func shortenRetryBackoff(t *testing.T) {
	t.Helper()
	original := retryBackoffBase
	retryBackoffBase = time.Millisecond
	t.Cleanup(func() { retryBackoffBase = original })
}

// Terraform runs ten resource operations at once by default. Every one of them
// needs a token, so an expired cache used to mean ten simultaneous exchanges.
func TestGetToken_RefreshesOnceUnderConcurrency(t *testing.T) {
	var exchanges int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/apikey/exchange") {
			atomic.AddInt64(&exchanges, 1)
			// Long enough that every goroutine reaches the refresh together.
			time.Sleep(50 * time.Millisecond)
			_, _ = w.Write([]byte("test-token"))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	// Construction performs one exchange; expire it so the next call refreshes.
	atomic.StoreInt64(&exchanges, 0)
	c.mu.Lock()
	c.tokenExp = time.Now().Add(-time.Minute)
	c.mu.Unlock()

	const goroutines = 10
	var wg sync.WaitGroup
	tokens := make([]string, goroutines)
	errs := make([]error, goroutines)
	start := make(chan struct{})

	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			tokens[i], errs[i] = c.getToken(context.Background())
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
		if tokens[i] != "test-token" {
			t.Errorf("goroutine %d got %q", i, tokens[i])
		}
	}

	if got := atomic.LoadInt64(&exchanges); got != 1 {
		t.Errorf("expected exactly 1 token exchange for %d concurrent callers, got %d", goroutines, got)
	}
}

// A rate-limited identity endpoint is transient. Reporting it as an
// authentication failure sends users to check a key that is fine.
func TestRefreshToken_RetriesTransientFailures(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		wantRetried bool
	}{
		{"rate limited", http.StatusTooManyRequests, true},
		{"server error", http.StatusInternalServerError, true},
		{"bad gateway", http.StatusBadGateway, true},
		// A rejected key stays rejected; retrying only delays the error.
		{"unauthorized", http.StatusUnauthorized, false},
		{"forbidden", http.StatusForbidden, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			shortenRetryBackoff(t)
			var calls int64
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := atomic.AddInt64(&calls, 1)
				if n == 1 {
					w.WriteHeader(c.status)
					return
				}
				_, _ = w.Write([]byte("test-token"))
			}))
			defer srv.Close()

			client, err := NewAnecdotesClient(context.Background(), "test-key", srv.URL, "test-agent")

			if c.wantRetried {
				if err != nil {
					t.Fatalf("a transient %d should have been retried, got %v", c.status, err)
				}
				if got := atomic.LoadInt64(&calls); got != 2 {
					t.Errorf("expected 2 calls (fail then succeed), got %d", got)
				}
				if client == nil {
					t.Error("expected a client")
				}
				return
			}

			if err == nil {
				t.Fatalf("a %d must not be retried into success", c.status)
			}
			if got := atomic.LoadInt64(&calls); got != 1 {
				t.Errorf("a %d must not be retried, got %d calls", c.status, got)
			}
			// The plain error must not carry API classification, or callers
			// treating it as a 5xx would trigger create recovery.
			if IsServerError(err) || IsNotFound(err) {
				t.Errorf("auth failure must not be classified: %v", err)
			}
		})
	}
}

// The exchange is given up on rather than retried forever.
func TestRefreshToken_GivesUpAfterMaxRetries(t *testing.T) {
	shortenRetryBackoff(t)
	var calls int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	if _, err := NewAnecdotesClient(context.Background(), "test-key", srv.URL, "test-agent"); err == nil {
		t.Fatal("expected an error when the identity endpoint never recovers")
	}
	if got := atomic.LoadInt64(&calls); got != maxRetries+1 {
		t.Errorf("expected %d attempts, got %d", maxRetries+1, got)
	}
}
