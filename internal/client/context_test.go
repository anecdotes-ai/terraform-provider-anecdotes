// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Cancellation has to reach a request that is already in flight, or Ctrl-C does
// nothing until the 120s client timeout expires.
func TestDoRequest_CancelledMidFlight(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/apikey/exchange") {
			_, _ = w.Write([]byte("test-token"))
			return
		}
		<-release // hold the request open
	}))
	defer srv.Close()
	defer close(release)

	c := newTestClient(t, srv)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := c.doRequest(ctx, "GET", "/api/v1/framework", nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected the cancelled request to fail")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	// The HTTP client timeout is 120s; anything near that means cancellation
	// never reached the request.
	if elapsed > 5*time.Second {
		t.Errorf("took %v to notice cancellation", elapsed)
	}
}

// A retry sleep must be interruptible too, otherwise the wait outlives the
// cancellation that should have ended it.
func TestDoRequest_CancelledDuringRetryBackoff(t *testing.T) {
	var attempts int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/apikey/exchange") {
			_, _ = w.Write([]byte("test-token"))
			return
		}
		atomic.AddInt64(&attempts, 1)
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := c.doRequest(ctx, "GET", "/api/v1/framework", nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if elapsed > 5*time.Second {
		t.Errorf("waited %v; the 30s Retry-After sleep ignored cancellation", elapsed)
	}
	if got := atomic.LoadInt64(&attempts); got != 1 {
		t.Errorf("expected to stop after the first attempt, got %d", got)
	}
}

// The token exchange runs before any request, so it needs the same treatment.
func TestRefreshToken_RespectsCancellation(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer srv.Close()
	defer close(release)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := NewAnecdotesClient(ctx, "test-key", srv.URL, "test-agent")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected the cancelled exchange to fail")
	}
	if elapsed > 5*time.Second {
		t.Errorf("took %v to notice cancellation", elapsed)
	}
}
