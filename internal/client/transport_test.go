// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The API key is sent in a custom header, and Go forwards custom headers across
// hosts even though it strips Authorization. A redirect must therefore fail
// rather than carry the credential to wherever it points.
func TestClient_RefusesRedirects(t *testing.T) {
	var keyReachedRedirectTarget bool

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-anecdotes-api-key") != "" || r.Header.Get("Authorization") != "" {
			keyReachedRedirectTarget = true
		}
		_, _ = w.Write([]byte("leaked-token"))
	}))
	defer target.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+r.URL.Path, http.StatusMovedPermanently)
	}))
	defer origin.Close()

	_, err := NewAnecdotesClient(context.Background(), "test-key", origin.URL, "test-agent")
	if err == nil {
		t.Fatal("expected the client to refuse the redirect, got no error")
	}
	if !strings.Contains(err.Error(), "refusing to follow a redirect") {
		t.Errorf("expected a redirect refusal, got: %v", err)
	}
	if keyReachedRedirectTarget {
		t.Error("the credential was sent to the redirect target")
	}
}

// Redirects are refused on ordinary requests too, not only the token exchange.
func TestClient_RefusesRedirectsOnAPICalls(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer target.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/apikey/exchange") {
			_, _ = w.Write([]byte("test-token"))
			return
		}
		http.Redirect(w, r, target.URL+r.URL.Path, http.StatusTemporaryRedirect)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if _, err := c.ListFrameworks(context.Background()); err == nil {
		t.Fatal("expected the redirect to fail the call, got no error")
	}
}
