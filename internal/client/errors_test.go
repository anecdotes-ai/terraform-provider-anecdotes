// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseAPIError_Shapes(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		wantClass   ErrorClass
		wantMsg     string // exact, when non-empty
		wantContain string // substring, when wantMsg empty
		wantFields  int
		mustNotHave []string // strings that must never leak into the message
	}{
		{
			name:       "pydantic v1 array",
			status:     422,
			body:       `{"detail":[{"loc":["body","name"],"msg":"field required","type":"value_error.missing"}]}`,
			wantClass:  ClassValidation,
			wantMsg:    "name: field required",
			wantFields: 1,
		},
		{
			name:       "pydantic v2 array",
			status:     422,
			body:       `{"detail":[{"type":"missing","loc":["body","name"],"msg":"Field required","input":{}}]}`,
			wantClass:  ClassValidation,
			wantMsg:    "name: Field required",
			wantFields: 1,
		},
		{
			name:      "pydantic string 405",
			status:    405,
			body:      `{"detail":"Method Not Allowed"}`,
			wantClass: ClassUnsupported,
			wantMsg:   "Method Not Allowed",
		},
		{
			name:      "pydantic string 404",
			status:    404,
			body:      `{"detail":"Framework does not exists"}`,
			wantClass: ClassNotFound,
			wantMsg:   "Framework does not exists",
		},
		{
			name:      "402 feature gate",
			status:    402,
			body:      `{"detail":"Your tenant does not enable ['Premium'] feature/s"}`,
			wantClass: ClassFeatureGate,
			wantMsg:   "Your tenant does not enable ['Premium'] feature/s",
		},
		{
			name:      "framework envelope title only",
			status:    400,
			body:      `{"error_title":"Framework name cannot be empty","error_detail":null}`,
			wantClass: ClassValidation,
			wantMsg:   "Framework name cannot be empty",
		},
		{
			name:        "framework envelope with request_id redacted",
			status:      400,
			body:        `{"error_title":"Bad Request","error_detail":"'user' is a required property","request_id":140257178975552}`,
			wantClass:   ClassValidation,
			wantMsg:     "Bad Request: 'user' is a required property",
			mustNotHave: []string{"140257178975552", "request_id"},
		},
		{
			name:      "framework envelope numeric error_detail ignored",
			status:    404,
			body:      `{"error_title":"failed to get evidence, evidence does not exist","error_detail":404}`,
			wantClass: ClassNotFound,
			wantMsg:   "failed to get evidence, evidence does not exist",
		},
		{
			name:      "bare json string",
			status:    404,
			body:      `"Control not found"`,
			wantClass: ClassNotFound,
			wantMsg:   "Control not found",
		},
		{
			name:        "html 500 redacted",
			status:      500,
			body:        `<!doctype html><html lang=en><title>500 Internal Server Error</title><h1>Internal Server Error</h1>`,
			wantClass:   ClassServer,
			wantContain: "internal error",
			mustNotHave: []string{"doctype", "<html", "<title"},
		},
		{
			name:        "server taskgroup leak redacted",
			status:      500,
			body:        `{"error_title":"unhandled errors in a TaskGroup (1 sub-exception)"}`,
			wantClass:   ClassServer,
			wantContain: "internal error",
			mustNotHave: []string{"TaskGroup", "sub-exception"},
		},
		{
			name:        "empty body 401",
			status:      401,
			body:        ``,
			wantClass:   ClassPermission,
			wantContain: "credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := parseAPIError("POST", "/x", tt.status, []byte(tt.body))
			if e.Class != tt.wantClass {
				t.Errorf("class = %q, want %q", e.Class, tt.wantClass)
			}
			if tt.wantMsg != "" && e.Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", e.Message, tt.wantMsg)
			}
			if tt.wantContain != "" && !strings.Contains(strings.ToLower(e.Message), strings.ToLower(tt.wantContain)) {
				t.Errorf("message = %q, want to contain %q", e.Message, tt.wantContain)
			}
			if tt.wantFields > 0 && len(e.FieldErrors) != tt.wantFields {
				t.Errorf("fieldErrors = %d, want %d", len(e.FieldErrors), tt.wantFields)
			}
			for _, bad := range tt.mustNotHave {
				if strings.Contains(e.Message, bad) {
					t.Errorf("message %q leaked forbidden substring %q", e.Message, bad)
				}
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	if !IsNotFound(&APIError{StatusCode: 404, Class: ClassNotFound}) {
		t.Error("404 APIError should be IsNotFound")
	}
	if IsNotFound(&APIError{StatusCode: 422, Class: ClassValidation}) {
		t.Error("422 APIError should not be IsNotFound")
	}
	if IsNotFound(fmt.Errorf("wrapped: %w", &APIError{StatusCode: 404, Class: ClassNotFound})) == false {
		t.Error("wrapped 404 APIError should be IsNotFound via errors.As")
	}
	if IsNotFound(errors.New("plain")) {
		t.Error("non-APIError should not be IsNotFound")
	}
	// list-then-filter lookups surface a missing resource via ErrNotFound, not a 404.
	if !IsNotFound(ErrNotFound) {
		t.Error("ErrNotFound should be IsNotFound")
	}
	if !IsNotFound(fmt.Errorf("tag not found: %s: %w", "tag_1", ErrNotFound)) {
		t.Error("error wrapping ErrNotFound should be IsNotFound")
	}
}

func TestErrorHelpers_ConflictServerStatus(t *testing.T) {
	conflict := &APIError{StatusCode: 409, Class: ClassConflict}
	server := &APIError{StatusCode: 500, Class: ClassServer}
	validation := &APIError{StatusCode: 400, Class: ClassValidation}

	if !IsConflict(conflict) || IsConflict(server) {
		t.Error("IsConflict should be true only for 409")
	}
	if !IsServerError(server) || IsServerError(conflict) {
		t.Error("IsServerError should be true only for 5xx")
	}
	if StatusCode(validation) != 400 {
		t.Errorf("StatusCode = %d, want 400", StatusCode(validation))
	}
	if StatusCode(errors.New("plain")) != 0 {
		t.Error("StatusCode of non-APIError should be 0")
	}
	// classifiers must see through a wrapped error (errors.As)
	if !IsServerError(fmt.Errorf("create failed: %w", server)) {
		t.Error("IsServerError should unwrap wrapped APIError")
	}
}

func TestAPIError_Classifiers(t *testing.T) {
	e := &APIError{StatusCode: 422, Method: "POST", Class: ClassValidation}
	if !e.IsValidation() || e.IsNotFound() || e.IsRetryable() {
		t.Errorf("validation classifiers wrong: %+v", e)
	}
	if !(&APIError{StatusCode: 503, Method: "GET", Class: ClassServer}).IsRetryable() {
		t.Error("GET 503 should be retryable")
	}
	if (&APIError{StatusCode: 500, Method: "POST", Class: ClassServer}).IsRetryable() {
		t.Error("POST 500 must not be retryable — re-sending could execute the create twice")
	}
	if !(&APIError{StatusCode: 429, Method: "POST", Class: ClassUnknown}).IsRetryable() {
		t.Error("429 should be retryable for any method")
	}
	if (&APIError{StatusCode: 404, Method: "GET", Class: ClassNotFound}).IsRetryable() {
		t.Error("404 should not be retryable")
	}
}

// newTestClient builds a client pointed at a test server, satisfying the initial
// token exchange.
func newTestClient(t *testing.T, srv *httptest.Server) *AnecdotesClient {
	t.Helper()
	c, err := NewAnecdotesClient("test-key", srv.URL)
	if err != nil {
		t.Fatalf("NewAnecdotesClient: %v", err)
	}
	return c
}

func TestDoRequest_ReturnsClassifiedAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/apikey/exchange") {
			_, _ = w.Write([]byte("test-token"))
			return
		}
		w.WriteHeader(422)
		_, _ = w.Write([]byte(`{"detail":[{"loc":["body","name"],"msg":"field required","type":"value_error.missing"}]}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.doRequest("POST", "/framework/v1/framework", map[string]string{})
	apiErr, ok := AsAPIError(err)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if !apiErr.IsValidation() {
		t.Errorf("class = %q, want validation", apiErr.Class)
	}
	if apiErr.Message != "name: field required" {
		t.Errorf("message = %q", apiErr.Message)
	}
}

func TestDoRequest_401RefreshesAndRetries(t *testing.T) {
	var exchanges, things int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/apikey/exchange") {
			atomic.AddInt32(&exchanges, 1)
			_, _ = w.Write([]byte("test-token"))
			return
		}
		// First call → 401 (forces a refresh), subsequent → 200.
		if atomic.AddInt32(&things, 1) == 1 {
			w.WriteHeader(401)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if _, err := c.doRequest("GET", "/api/v1/framework", nil); err != nil {
		t.Fatalf("expected success after 401 refresh, got %v", err)
	}
	if got := atomic.LoadInt32(&exchanges); got < 2 {
		t.Errorf("token exchanges = %d, want >= 2 (initial + 401 refresh)", got)
	}
	if got := atomic.LoadInt32(&things); got != 2 {
		t.Errorf("resource calls = %d, want 2 (401 then success)", got)
	}
}

func TestDoRequest_PersistentNon2xxIsNotInfiniteLoop(t *testing.T) {
	var things int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/apikey/exchange") {
			_, _ = w.Write([]byte("test-token"))
			return
		}
		atomic.AddInt32(&things, 1)
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"detail":"not found"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.doRequest("GET", "/api/v1/framework/x", nil)
	apiErr, ok := AsAPIError(err)
	if !ok || !apiErr.IsNotFound() {
		t.Fatalf("expected not-found APIError, got %T: %v", err, err)
	}
	// 404 is not retryable: exactly one resource call.
	if got := atomic.LoadInt32(&things); got != 1 {
		t.Errorf("resource calls = %d, want 1 (404 not retried)", got)
	}
}

// A classified error from the identity exchange must never leak API error
// classification into callers: a 404/500 on a mid-session token refresh would
// otherwise satisfy IsNotFound/IsServerError and trigger state-drop or create
// recovery for a request that was never sent.
func TestTokenRefreshErrors_DoNotClassify(t *testing.T) {
	// A 500 from the identity endpoint is now retried, so keep the waits short.
	shortenRetryBackoff(t)
	for _, status := range []int{404, 500} {
		var exchangeFails atomic.Bool
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/apikey/exchange") {
				if exchangeFails.Load() {
					w.WriteHeader(status)
					_, _ = w.Write([]byte(`{"detail":"identity error"}`))
					return
				}
				_, _ = w.Write([]byte("test-token"))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		}))

		c := newTestClient(t, srv)
		// Expire the token so the next request forces a refresh, then fail it.
		exchangeFails.Store(true)
		c.mu.Lock()
		c.tokenExp = time.Time{}
		c.mu.Unlock()

		_, err := c.ListFrameworks()
		if err == nil {
			t.Fatalf("status %d: expected an error from the failed refresh", status)
		}
		if IsNotFound(err) {
			t.Errorf("status %d: auth failure must not satisfy IsNotFound (would drop state)", status)
		}
		if IsServerError(err) {
			t.Errorf("status %d: auth failure must not satisfy IsServerError (would trigger create recovery)", status)
		}
		srv.Close()
	}
}
