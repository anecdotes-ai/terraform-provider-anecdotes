// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// A rejected category is a 400: the request is wrong and will stay wrong.
// Sending it again only delays the error the user needs to see.
func TestAddControl_DoesNotRetryARejectedCategory(t *testing.T) {
	var attempts int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/apikey/exchange") {
			_, _ = w.Write([]byte("test-token"))
			return
		}
		atomic.AddInt64(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error_title":"Cannot add control with a nonexistent category - category_1.","error_detail":null}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	start := time.Now()
	_, err := c.AddControl("fw1", &ControlCreateRequest{
		ControlName:                "c",
		ControlFrameworkCategory:   "cat",
		ControlFrameworkCategoryID: "category_1",
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected the rejection to surface")
	}
	if got := atomic.LoadInt64(&attempts); got != 1 {
		t.Errorf("expected a single attempt, got %d", got)
	}
	// The previous implementation slept 3s, 6s and 9s before giving up.
	if elapsed > 2*time.Second {
		t.Errorf("took %v; a rejected category must fail immediately", elapsed)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Class != ClassValidation {
		t.Errorf("a 400 should classify as validation, got %v", err)
	}
}

// Errors the shared transport does retry still reach the caller unchanged.
func TestAddControl_SurfacesServerErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/apikey/exchange") {
			_, _ = w.Write([]byte("test-token"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).AddControl("fw1", &ControlCreateRequest{ControlName: "c"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !IsServerError(err) {
		t.Errorf("expected a server error, got %v", err)
	}
}
