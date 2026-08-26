// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A wrong field name or a dropped empty value fails silently in both
// directions: the write is accepted and the value simply never changes. These
// tests pin the request bodies of the writes where that matters.

// captureRequest runs fn against a server that records the body of the first
// non-authentication request it receives.
func captureRequest(t *testing.T, fn func(*AnecdotesClient) error) []byte {
	t.Helper()

	var captured []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/apikey/exchange") {
			_, _ = w.Write([]byte("test-token"))
			return
		}
		if captured == nil {
			captured, _ = io.ReadAll(r.Body)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	if err := fn(newTestClient(t, srv)); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if captured == nil {
		t.Fatal("no request was captured")
	}
	return captured
}

func TestSetControlOwners_SendsSingularFieldAndClears(t *testing.T) {
	body := captureRequest(t, func(c *AnecdotesClient) error {
		return c.SetControlOwners(context.Background(), "c1", []string{"a@example.com"})
	})

	var payload []map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal: %v (body %s)", err, body)
	}
	if len(payload) != 1 {
		t.Fatalf("expected a single entry, got %d", len(payload))
	}
	owners, ok := payload[0]["control_owner"]
	if !ok {
		t.Fatalf("owners must be sent as control_owner, got keys %v", keysOf(payload[0]))
	}
	if got := owners.([]interface{}); len(got) != 1 || got[0] != "a@example.com" {
		t.Errorf("unexpected owners: %v", got)
	}

	// Clearing must send an empty list, not null: a null value leaves the
	// owners in place.
	body = captureRequest(t, func(c *AnecdotesClient) error {
		return c.SetControlOwners(context.Background(), "c1", nil)
	})
	if !strings.Contains(string(body), `"control_owner":[]`) {
		t.Errorf("clearing owners must send an empty list, got %s", body)
	}
}

// Owners are sent on create as well as update, under the same field name.
func TestControlCreateRequest_SendsOwnersUnderTheSameField(t *testing.T) {
	encoded, err := json.Marshal(&ControlCreateRequest{
		ControlName:   "c",
		ControlOwners: []string{"a@example.com", "b@example.com"},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	owners, ok := decoded["control_owner"]
	if !ok {
		t.Fatalf("owners must be sent as control_owner on create, got keys %v", keysOf(decoded))
	}
	if got := owners.([]interface{}); len(got) != 2 {
		t.Errorf("expected both owners, got %v", got)
	}
}

func TestSetControlMaturityLevel_ClearsWithNull(t *testing.T) {
	body := captureRequest(t, func(c *AnecdotesClient) error {
		return c.SetControlMaturityLevel(context.Background(), "c1", "")
	})
	if !strings.Contains(string(body), `"maturity_level":null`) {
		t.Errorf("an empty level must be sent as null, got %s", body)
	}

	body = captureRequest(t, func(c *AnecdotesClient) error {
		return c.SetControlMaturityLevel(context.Background(), "c1", "DEFINED")
	})
	if !strings.Contains(string(body), `"maturity_level":"DEFINED"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

// Empty strings must survive serialization, otherwise a field can be set but
// never emptied.
func TestUpdateRequests_KeepEmptyStrings(t *testing.T) {
	cases := []struct {
		name    string
		request interface{}
		field   string
	}{
		{"framework description", &FrameworkUpdateRequest{FrameworkName: "f"}, "framework_description"},
		{"framework create description", &FrameworkCreateRequest{FrameworkName: "f"}, "framework_description"},
		{"control description", &ControlUpdateRequest{ControlName: "c"}, "control_description"},
		{"requirement help", &RequirementUpdateRequest{RequirementDescription: "r", RequirementHelp: stringPtr("")}, "requirement_help"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			encoded, err := json.Marshal(c.request)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var decoded map[string]interface{}
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			value, ok := decoded[c.field]
			if !ok {
				t.Fatalf("%s must be sent even when empty, got keys %v", c.field, keysOf(decoded))
			}
			if value != "" {
				t.Errorf("expected an empty string, got %v", value)
			}
		})
	}
}

func TestUnlinkRequirementFromControl_SendsEmptyListNotNull(t *testing.T) {
	var patched []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/apikey/exchange"):
			_, _ = w.Write([]byte("test-token"))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/controls/control/read"):
			_, _ = w.Write([]byte(`[{"control_id":"c1","control_framework_id":"fw1","control_name":"C","control_requirement_ids":["r1"]}]`))
		case r.Method == http.MethodPatch:
			patched, _ = io.ReadAll(r.Body)
			_, _ = w.Write([]byte(`{}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()

	if err := newTestClient(t, srv).UnlinkRequirementFromControl(context.Background(), "c1", "r1"); err != nil {
		t.Fatalf("unlink failed: %v", err)
	}
	if patched == nil {
		t.Fatal("no PATCH was sent")
	}
	if !strings.Contains(string(patched), `"control_related_requirements":[]`) {
		t.Errorf("removing the last requirement must send an empty list, got %s", patched)
	}
}

// A partial update — changing only the evidence links — must not carry the
// other fields, or it silently clears them.
func TestLinkEvidenceToRequirement_DoesNotSendOtherFields(t *testing.T) {
	var patched []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/apikey/exchange"):
			_, _ = w.Write([]byte("test-token"))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/requirement/"):
			_, _ = w.Write([]byte(`[{"requirement_id":"req1","requirement_name":"R","requirement_help":"keep me","requirement_evidence_ids":[]}]`))
		case r.Method == http.MethodPatch:
			patched, _ = io.ReadAll(r.Body)
			_, _ = w.Write([]byte(`{}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()

	if err := newTestClient(t, srv).LinkEvidenceToRequirement(context.Background(), "req1", "e1"); err != nil {
		t.Fatalf("link failed: %v", err)
	}
	if patched == nil {
		t.Fatal("no PATCH was sent")
	}
	var payload struct {
		Requirement map[string]interface{} `json:"requirement"`
	}
	if err := json.Unmarshal(patched, &payload); err != nil {
		t.Fatalf("unmarshal: %v (body %s)", err, patched)
	}
	for _, field := range []string{"requirement_help", "requirement_description", "requirement_category", "requirement_owners"} {
		if _, ok := payload.Requirement[field]; ok {
			t.Errorf("%s must not be sent by an evidence-only update, got %s", field, patched)
		}
	}
	if _, ok := payload.Requirement["requirement_related_evidences"]; !ok {
		t.Errorf("the evidence list must be sent, got %s", patched)
	}
}

// A view's create payload must never carry requirement_description — the API
// rejects a view that does, since view_name is what it uses instead.
func TestRequirementViewCreateRequest_NeverSendsDescription(t *testing.T) {
	encoded, err := json.Marshal(&RequirementViewCreateRequest{
		RequirementParentID: "req_parent",
		ViewName:            "My View",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := decoded["requirement_description"]; ok {
		t.Errorf("a view create request must never carry requirement_description, got %s", encoded)
	}
	if decoded["requirement_parent_id"] != "req_parent" {
		t.Errorf("expected requirement_parent_id to be sent, got keys %v", keysOf(decoded))
	}
	if decoded["view_name"] != "My View" {
		t.Errorf("expected view_name to be sent, got keys %v", keysOf(decoded))
	}
}

// A view's update payload must never carry requirement_parent_id — the API
// 400s on any attempt to change it after creation, so the struct must not
// even have a field a caller could accidentally populate.
func TestRequirementViewUpdateRequest_HasNoParentIDField(t *testing.T) {
	encoded, err := json.Marshal(&RequirementViewUpdateRequest{ViewName: stringPtr("Renamed")})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := decoded["requirement_parent_id"]; ok {
		t.Errorf("a view update request must never carry requirement_parent_id, got %s", encoded)
	}
	if _, ok := decoded["requirement_description"]; ok {
		t.Errorf("a view update request must never carry requirement_description, got %s", encoded)
	}
}

// CreateRequirementView must post to the shared requirement endpoint with the
// parent id + view_name, and must not fall back to a by-name lookup on a
// server error the way CreateRequirement does — view names aren't unique, so
// that fallback could silently adopt the wrong view.
func TestCreateRequirementView_SendsParentAndViewName(t *testing.T) {
	body := captureRequest(t, func(c *AnecdotesClient) error {
		_, err := c.CreateRequirementView(context.Background(), &RequirementViewCreateRequest{
			RequirementParentID: "req_parent",
			ViewName:            "My View",
		})
		return err
	})
	var decoded map[string]interface{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal: %v (body %s)", err, body)
	}
	if decoded["requirement_parent_id"] != "req_parent" || decoded["view_name"] != "My View" {
		t.Errorf("unexpected create payload: %s", body)
	}
}

func TestCreateRequirementView_ServerErrorDoesNotFallBackToNameLookup(t *testing.T) {
	listCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/apikey/exchange"):
			_, _ = w.Write([]byte("test-token"))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/api/v1/requirement"):
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"detail":"boom"}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/api/v1/requirement"):
			listCalled = true
			_, _ = w.Write([]byte(`[]`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).CreateRequirementView(context.Background(), &RequirementViewCreateRequest{
		RequirementParentID: "req_parent",
		ViewName:            "My View",
	})
	if err == nil {
		t.Fatal("expected the server error to surface, not be swallowed by a name-lookup fallback")
	}
	if listCalled {
		t.Error("CreateRequirementView must not fall back to a by-name list lookup: view names aren't unique")
	}
}

// A delete that removed nothing must not be reported as success — but a
// requirement that was already gone still counts as deleted.
func TestDeleteRequirement_VerifiesTheOutcome(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		stillExists bool
		wantErr     bool
	}{
		{"removed", `{"deleted_count":1}`, false, false},
		{"already gone", `{"deleted_count":0}`, false, false},
		{"refused — requirement still there", `{"deleted_count":0}`, true, true},
		// An unreadable response is not evidence of success either: the outcome
		// is decided by reading the requirement back.
		{"unparseable body, requirement gone", `not json`, false, false},
		{"unparseable body, requirement still there", `not json`, true, true},
		{"empty body, requirement still there", ``, true, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasSuffix(r.URL.Path, "/apikey/exchange"):
					_, _ = w.Write([]byte("test-token"))
				case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/requirement/delete"):
					_, _ = w.Write([]byte(c.body))
				case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/requirement/"):
					if c.stillExists {
						_, _ = w.Write([]byte(`[{"requirement_id":"r1","requirement_name":"R"}]`))
						return
					}
					w.WriteHeader(http.StatusNotFound)
				default:
					_, _ = w.Write([]byte(`{}`))
				}
			}))
			defer srv.Close()

			err := newTestClient(t, srv).DeleteRequirement(context.Background(), "r1")
			if c.wantErr && err == nil {
				t.Error("expected an error when the requirement survived the delete, got none")
			}
			if !c.wantErr && err != nil {
				t.Errorf("expected success, got %v", err)
			}
		})
	}
}

func stringPtr(s string) *string { return &s }

func keysOf(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
