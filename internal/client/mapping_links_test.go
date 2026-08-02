// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

func TestGetRequirementEvidenceLink_ParentMissingIsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/apikey/exchange"):
			_, _ = w.Write([]byte("test-token"))
		default: // requirement does not exist — empty result set
			_, _ = w.Write([]byte(`[]`))
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.GetRequirementEvidenceLink("r_gone", "ev1")
	if !IsNotFound(err) {
		t.Errorf("missing parent requirement should satisfy IsNotFound, got: %v", err)
	}
}

// Link lists are updated by reading the parent, changing the list and writing
// it back. Without serialization, two links applied concurrently both read the
// original list and the second write drops the first. These tests run the
// operations in parallel against a server that models the list, and assert every
// link survives.

func TestLinkRequirementToControl_ConcurrentLinksAllSurvive(t *testing.T) {
	var mu sync.Mutex
	linked := []string{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/apikey/exchange"):
			_, _ = w.Write([]byte("test-token"))

		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/controls/control/read"):
			mu.Lock()
			ids, _ := json.Marshal(linked)
			mu.Unlock()
			_, _ = fmt.Fprintf(w, `[{"control_id":"c1","control_framework_id":"fw1","control_name":"C","control_requirement_ids":%s}]`, ids)

		case r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/controls/controls"):
			body, _ := io.ReadAll(r.Body)
			var payload []struct {
				ControlRelatedRequirements []string `json:"control_related_requirements"`
			}
			if err := json.Unmarshal(body, &payload); err != nil || len(payload) == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			mu.Lock()
			linked = payload[0].ControlRelatedRequirements
			mu.Unlock()
			_, _ = w.Write([]byte(`{}`))

		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	requirements := []string{"r1", "r2", "r3", "r4", "r5"}
	var wg sync.WaitGroup
	for _, id := range requirements {
		wg.Add(1)
		go func(requirementID string) {
			defer wg.Done()
			if _, err := c.LinkRequirementToControl("c1", requirementID); err != nil {
				t.Errorf("link %s failed: %v", requirementID, err)
			}
		}(id)
	}
	wg.Wait()

	mu.Lock()
	got := append([]string{}, linked...)
	mu.Unlock()

	if len(got) != len(requirements) {
		t.Fatalf("expected all %d requirements linked, got %d: %v", len(requirements), len(got), got)
	}
	for _, want := range requirements {
		found := false
		for _, g := range got {
			if g == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("requirement %s was lost", want)
		}
	}
}

func TestLinkEvidenceToRequirement_ConcurrentLinksAllSurvive(t *testing.T) {
	var mu sync.Mutex
	linked := []string{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/apikey/exchange"):
			_, _ = w.Write([]byte("test-token"))

		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/requirement/"):
			mu.Lock()
			ids, _ := json.Marshal(linked)
			mu.Unlock()
			_, _ = fmt.Fprintf(w, `[{"requirement_id":"req1","requirement_name":"R","requirement_evidence_ids":%s}]`, ids)

		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/requirement/"):
			body, _ := io.ReadAll(r.Body)
			var payload struct {
				Requirement struct {
					RelatedEvidences []string `json:"requirement_related_evidences"`
				} `json:"requirement"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			mu.Lock()
			linked = payload.Requirement.RelatedEvidences
			mu.Unlock()
			_, _ = w.Write([]byte(`{}`))

		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	evidences := []string{"e1", "e2", "e3", "e4", "e5"}
	var wg sync.WaitGroup
	for _, id := range evidences {
		wg.Add(1)
		go func(evidenceID string) {
			defer wg.Done()
			if err := c.LinkEvidenceToRequirement("req1", evidenceID); err != nil {
				t.Errorf("link %s failed: %v", evidenceID, err)
			}
		}(id)
	}
	wg.Wait()

	mu.Lock()
	got := append([]string{}, linked...)
	mu.Unlock()

	if len(got) != len(evidences) {
		t.Fatalf("expected all %d evidences linked, got %d: %v", len(evidences), len(got), got)
	}
}
