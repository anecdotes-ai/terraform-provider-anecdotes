// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// analysisRuleServer routes the analysis-rules endpoints to handlers keyed by
// "METHOD path-suffix", recording every path it is asked for.
func analysisRuleServer(t *testing.T, routes map[string]func(w http.ResponseWriter, r *http.Request)) (*AnecdotesClient, *[]string) {
	t.Helper()

	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/apikey/exchange") {
			_, _ = w.Write([]byte("test-token"))
			return
		}
		key := r.Method + " " + strings.TrimPrefix(r.URL.Path, analysisRulesPath)
		seen = append(seen, key)
		if h, ok := routes[key]; ok {
			h(w, r)
			return
		}
		http.Error(w, `{"error_title":"unrouted"}`, http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	return newTestClient(t, srv), &seen
}

func TestCreateAnalysisRule_SendsQueryVerbatim(t *testing.T) {
	query := json.RawMessage(`{"left":"Min Password Length","operator":"IsIn","right":["1","2"]}`)

	body := captureRequest(t, func(c *AnecdotesClient) error {
		_, err := c.CreateAnalysisRule(context.Background(), AnalysisRuleCreateRequest{
			EvidenceID:         "ev1",
			RuleName:           "n",
			AlertLevel:         50,
			RuleQueryType:      "aql",
			RuleQuery:          query,
			AccountScopingType: "included_accounts",
			AccountScopingList: []string{"inst-1"},
		})
		return err
	})

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal: %v (body %s)", err, body)
	}

	for _, field := range []string{"evidence_id", "rule_name", "alert_level", "rule_query_type", "rule_query", "account_scoping_type", "account_scoping_list"} {
		if _, ok := payload[field]; !ok {
			t.Errorf("create payload is missing %q (body %s)", field, body)
		}
	}

	// The query is the rule; a re-serialization that drops or renames a key
	// would produce a rule that stores something the practitioner never wrote.
	gotQuery, err := json.Marshal(payload["rule_query"])
	if err != nil {
		t.Fatalf("marshal rule_query: %v", err)
	}
	var want, got interface{}
	_ = json.Unmarshal(query, &want)
	_ = json.Unmarshal(gotQuery, &got)
	if !jsonDeepEqual(want, got) {
		t.Errorf("rule_query changed in transit:\n sent %s\n got  %s", query, gotQuery)
	}
}

func TestUpdateAnalysisRule_RefusesEmptyQuery(t *testing.T) {
	c, seen := analysisRuleServer(t, nil)

	_, err := c.UpdateAnalysisRule(context.Background(), "r1", AnalysisRuleUpdateRequest{RuleName: "renamed"})
	if err == nil {
		t.Fatal("expected an error when updating without a rule query")
	}
	// The API assigns the stored query from the request on every call, so the
	// request must never leave the client at all.
	for _, s := range *seen {
		if strings.HasPrefix(s, "PATCH") {
			t.Errorf("a PATCH was sent despite the missing query: %s", s)
		}
	}
}

func TestUpdateAnalysisRule_AlwaysSendsQueryAndType(t *testing.T) {
	body := captureRequest(t, func(c *AnecdotesClient) error {
		_, err := c.UpdateAnalysisRule(context.Background(), "r1", AnalysisRuleUpdateRequest{
			RuleName:      "renamed",
			RuleQueryType: "aqlext",
			RuleQuery:     json.RawMessage(`{"manipulations":[]}`),
		})
		return err
	})

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal: %v (body %s)", err, body)
	}
	// The query is read according to rule_query_type. Sending the query without
	// it has the server fall back to its default language, which rejects or
	// silently rewrites a query written in any other one.
	if _, ok := payload["rule_query"]; !ok {
		t.Errorf("update payload dropped rule_query (body %s)", body)
	}
	if got := payload["rule_query_type"]; got != "aqlext" {
		t.Errorf("update payload rule_query_type = %v, want aqlext (body %s)", got, body)
	}
}

func TestUpdateAnalysisRule_RefusesMissingQueryType(t *testing.T) {
	c, seen := analysisRuleServer(t, nil)

	_, err := c.UpdateAnalysisRule(context.Background(), "r1", AnalysisRuleUpdateRequest{
		RuleName:  "renamed",
		RuleQuery: json.RawMessage(`{"manipulations":[]}`),
	})
	if err == nil {
		t.Fatal("expected an error when updating without a rule query type")
	}
	for _, s := range *seen {
		if strings.HasPrefix(s, "PATCH") {
			t.Errorf("a PATCH was sent despite the missing query type: %s", s)
		}
	}
}

func TestGetAnalysisRule_PrefersEvidenceScopedRead(t *testing.T) {
	c, seen := analysisRuleServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /ev1": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`[{"rule_id":"r1","evidence_id":"ev1","rule_name":"found"}]`))
		},
	})

	rule, err := c.GetAnalysisRule(context.Background(), "ev1", "r1")
	if err != nil {
		t.Fatalf("GetAnalysisRule: %v", err)
	}
	if rule.RuleName != "found" {
		t.Errorf("rule name = %q, want %q", rule.RuleName, "found")
	}
	// The full list merges the platform's rule library and is orders of
	// magnitude larger; it must not be fetched when the scoped read answers.
	for _, s := range *seen {
		if s == "GET " {
			t.Error("the full rule list was fetched even though the scoped read succeeded")
		}
	}
}

func TestGetAnalysisRule_FallsBackToFullScan(t *testing.T) {
	// A rule created against an evidence view is stored under the view's id but
	// the scoped read resolves to the parent, so it is absent there.
	c, _ := analysisRuleServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /view1": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`[]`))
		},
		"GET ": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`[{"rule_id":"r1","evidence_id":"view1","rule_name":"via scan"}]`))
		},
	})

	rule, err := c.GetAnalysisRule(context.Background(), "view1", "r1")
	if err != nil {
		t.Fatalf("GetAnalysisRule: %v", err)
	}
	if rule.RuleName != "via scan" {
		t.Errorf("rule name = %q, want %q", rule.RuleName, "via scan")
	}
}

func TestGetAnalysisRule_ArchivedIsNotFound(t *testing.T) {
	// Deleting a rule archives it, and both read paths keep returning it.
	// Reporting it as present would leave a destroyed rule in state forever.
	c, _ := analysisRuleServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /ev1": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`[{"rule_id":"r1","evidence_id":"ev1","rule_is_archived":true}]`))
		},
		"GET ": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`[{"rule_id":"r1","evidence_id":"ev1","rule_is_archived":true}]`))
		},
	})

	_, err := c.GetAnalysisRule(context.Background(), "ev1", "r1")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestSetAnalysisRuleState_UsesStatePathAndIgnoresBody(t *testing.T) {
	// The endpoint answers 200 with an empty list both for a no-op and for an
	// unknown rule id, and returns objects shaped unlike the rest of the
	// service, so the body is never parsed.
	c, seen := analysisRuleServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"PATCH /r1/state/inactive": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`[{"model_id":"r1","_rule_is_archived":false}]`))
		},
	})

	if err := c.SetAnalysisRuleState(context.Background(), "r1", "inactive"); err != nil {
		t.Fatalf("SetAnalysisRuleState: %v", err)
	}
	if len(*seen) != 1 || (*seen)[0] != "PATCH /r1/state/inactive" {
		t.Errorf("requests = %v, want a single PATCH to /r1/state/inactive", *seen)
	}
}

func TestDeleteAnalysisRule_UsesRuleIDPath(t *testing.T) {
	c, seen := analysisRuleServer(t, map[string]func(http.ResponseWriter, *http.Request){
		"DELETE /r1": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`true`))
		},
	})

	if err := c.DeleteAnalysisRule(context.Background(), "r1"); err != nil {
		t.Fatalf("DeleteAnalysisRule: %v", err)
	}
	if len(*seen) != 1 || (*seen)[0] != "DELETE /r1" {
		t.Errorf("requests = %v, want a single DELETE to /r1", *seen)
	}
}

// jsonDeepEqual compares two decoded JSON values structurally.
func jsonDeepEqual(a, b interface{}) bool {
	ab, err1 := json.Marshal(a)
	bb, err2 := json.Marshal(b)
	return err1 == nil && err2 == nil && string(ab) == string(bb)
}
