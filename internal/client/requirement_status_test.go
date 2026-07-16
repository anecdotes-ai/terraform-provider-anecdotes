// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The requirement read endpoints return the status only as `requirement_status_id`.
// GetRequirement must resolve that ID to the human-readable name via the statuses
// endpoint and populate both RequirementStatusName and RequirementStatus.
func TestGetRequirement_ResolvesStatusName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/apikey/exchange"):
			_, _ = w.Write([]byte("test-token"))
		case strings.HasSuffix(r.URL.Path, "/requirement/status"):
			_, _ = w.Write([]byte(`[{"name":"In progress","status_id":"st_123"},{"name":"Done","status_id":"st_999"}]`))
		default: // GET /api/v1/requirement/{id}
			_, _ = w.Write([]byte(`[{"requirement_id":"r1","requirement_name":"R","requirement_status_id":"st_123"}]`))
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	req, err := c.GetRequirement("r1")
	if err != nil {
		t.Fatalf("GetRequirement: %v", err)
	}
	if req.RequirementStatusID != "st_123" {
		t.Errorf("status id = %q, want st_123", req.RequirementStatusID)
	}
	if req.RequirementStatusName != "In progress" {
		t.Errorf("status name = %q, want \"In progress\" (resolved from id)", req.RequirementStatusName)
	}
	if req.RequirementStatus != "In progress" {
		t.Errorf("status = %q, want \"In progress\" (backs data-source/filter)", req.RequirementStatus)
	}
}

// When the API already returns a status name (or there is no status id), no extra
// statuses lookup should change the value.
func TestGetRequirement_NoStatusIDLeavesEmpty(t *testing.T) {
	var statusCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/apikey/exchange"):
			_, _ = w.Write([]byte("test-token"))
		case strings.HasSuffix(r.URL.Path, "/requirement/status"):
			statusCalls++
			_, _ = w.Write([]byte(`[]`))
		default:
			_, _ = w.Write([]byte(`[{"requirement_id":"r1","requirement_name":"R"}]`))
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	req, err := c.GetRequirement("r1")
	if err != nil {
		t.Fatalf("GetRequirement: %v", err)
	}
	if req.RequirementStatusName != "" {
		t.Errorf("status name = %q, want empty", req.RequirementStatusName)
	}
	if statusCalls != 0 {
		t.Errorf("statuses endpoint called %d times; should be skipped when there is no status id", statusCalls)
	}
}
