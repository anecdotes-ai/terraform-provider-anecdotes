// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The mapping resources drop state only when a link is definitively gone.
// These tests pin the contract: "not linked" and "parent missing" must satisfy
// IsNotFound, while transient server failures must NOT — otherwise a 5xx during
// refresh would silently remove managed mappings from Terraform state.

func TestGetControlRequirementLink_NotLinkedIsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/apikey/exchange"):
			_, _ = w.Write([]byte("test-token"))
		default: // POST /controls/control/read
			_, _ = w.Write([]byte(`[{"control_id":"c1","control_framework_id":"fw1","control_name":"C","control_requirement_ids":["r_other"]}]`))
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetControlRequirementLink("c1", "r1")
	if err == nil {
		t.Fatal("expected an error for an unlinked requirement")
	}
	if !IsNotFound(err) {
		t.Errorf("unlinked requirement should satisfy IsNotFound, got: %v", err)
	}
}

func TestGetControlRequirementLink_ParentMissingIsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/apikey/exchange"):
			_, _ = w.Write([]byte("test-token"))
		default: // control does not exist — empty result set
			_, _ = w.Write([]byte(`[]`))
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetControlRequirementLink("c_gone", "r1")
	if !IsNotFound(err) {
		t.Errorf("missing parent control should satisfy IsNotFound, got: %v", err)
	}
}

func TestGetControlRequirementLink_ServerErrorIsNotNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/apikey/exchange"):
			_, _ = w.Write([]byte("test-token"))
		default:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"detail":"upstream unavailable"}`))
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetControlRequirementLink("c1", "r1")
	if err == nil {
		t.Fatal("expected an error on server failure")
	}
	if IsNotFound(err) {
		t.Errorf("a 500 must not satisfy IsNotFound (would drop state on a transient failure): %v", err)
	}
}

func TestGetRequirementEvidenceLink_NotLinkedIsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/apikey/exchange"):
			_, _ = w.Write([]byte("test-token"))
		case strings.HasSuffix(r.URL.Path, "/requirement/status"):
			_, _ = w.Write([]byte(`[]`))
		default: // GET /api/v1/requirement/{id}
			_, _ = w.Write([]byte(`[{"requirement_id":"r1","requirement_name":"R","requirement_evidence_ids":["ev_other"]}]`))
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.GetRequirementEvidenceLink("r1", "ev1")
	if err == nil {
		t.Fatal("expected an error for an unlinked evidence")
	}
	if !IsNotFound(err) {
		t.Errorf("unlinked evidence should satisfy IsNotFound, got: %v", err)
	}
}

func TestGetRequirementEvidenceLink_ServerErrorIsNotNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/apikey/exchange"):
			_, _ = w.Write([]byte("test-token"))
		default:
			// 501: a server error that is never retried, keeping this test instant.
			w.WriteHeader(http.StatusNotImplemented)
			_, _ = w.Write([]byte(`{"detail":"upstream unavailable"}`))
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.GetRequirementEvidenceLink("r1", "ev1")
	if err == nil {
		t.Fatal("expected an error on server failure")
	}
	if IsNotFound(err) {
		t.Errorf("a 500 must not satisfy IsNotFound (would drop state on a transient failure): %v", err)
	}
}
