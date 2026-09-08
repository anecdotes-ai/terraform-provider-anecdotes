// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMatchesAnalysisRuleFilters(t *testing.T) {
	rule := client.AnalysisRule{
		RuleID:     "r1",
		EvidenceID: "ev1",
		RuleOrigin: "custom",
		RuleState:  "active",
		RuleType:   "uam",
	}
	archived := rule
	archived.RuleIsArchived = true

	tests := []struct {
		name  string
		data  AnalysisRulesDataSourceModel
		rule  client.AnalysisRule
		match bool
	}{
		{"no filters", AnalysisRulesDataSourceModel{}, rule, true},
		{"archived excluded by default", AnalysisRulesDataSourceModel{}, archived, false},
		{"archived included when asked", AnalysisRulesDataSourceModel{IncludeArchived: types.BoolValue(true)}, archived, true},
		{"evidence matches", AnalysisRulesDataSourceModel{EvidenceID: types.StringValue("ev1")}, rule, true},
		{"evidence excludes", AnalysisRulesDataSourceModel{EvidenceID: types.StringValue("other")}, rule, false},
		{"origin excludes", AnalysisRulesDataSourceModel{RuleOrigin: types.StringValue("library")}, rule, false},
		{"state excludes", AnalysisRulesDataSourceModel{RuleState: types.StringValue("inactive")}, rule, false},
		{"type excludes", AnalysisRulesDataSourceModel{RuleType: types.StringValue("eid")}, rule, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchesAnalysisRuleFilters(&tc.data, tc.rule); got != tc.match {
				t.Errorf("matchesAnalysisRuleFilters = %v, want %v", got, tc.match)
			}
		})
	}
}

func TestServiceInstanceIDs(t *testing.T) {
	tests := []struct {
		name     string
		evidence client.Evidence
		want     []string
	}{
		{
			// This pins the merge, not the wire: the originating instance is
			// always emitted first, but the order of the rest is whatever the
			// platform sent. In practice an evidence usually reports only its
			// originating instance.
			name:     "originating instance only",
			evidence: client.Evidence{EvidenceOriginatedByInstanceID: "a"},
			want:     []string{"a"},
		},
		{
			name: "originating instance first, then the rest",
			evidence: client.Evidence{
				EvidenceOriginatedByInstanceID:   "a",
				EvidenceAlsoCollectedByInstances: []string{"b", "c"},
			},
			want: []string{"a", "b", "c"},
		},
		{
			name: "duplicates dropped",
			evidence: client.Evidence{
				EvidenceOriginatedByInstanceID:   "a",
				EvidenceAlsoCollectedByInstances: []string{"a", "b", "b"},
			},
			want: []string{"a", "b"},
		},
		{
			name:     "no instances",
			evidence: client.Evidence{},
			want:     []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			list := serviceInstanceIDs(context.Background(), &diags, tc.evidence)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}

			var got []string
			list.ElementsAs(context.Background(), &got, false)
			if len(got) != len(tc.want) {
				t.Fatalf("ids = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("ids = %v, want %v", got, tc.want)
					break
				}
			}
		})
	}
}

func TestStringSetFromAPI_DistinguishesEmptyFromUnset(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	// Never tracked and still empty: the attribute stays absent.
	if got := stringSetFromAPI(ctx, &diags, types.SetNull(types.StringType), nil); !got.IsNull() {
		t.Errorf("an untracked empty set became %v, want null", got)
	}

	// Tracked and now empty: the attribute is an empty set, not absent, so the
	// difference from the configured value is visible.
	tracked := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("x")})
	got := stringSetFromAPI(ctx, &diags, tracked, nil)
	if got.IsNull() {
		t.Error("a tracked set that the platform cleared became null, want an empty set")
	}
	if len(got.Elements()) != 0 {
		t.Errorf("cleared set = %v, want no elements", got)
	}

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestCollectUnsupportedAQLKeys(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{
			name:  "plain condition is accepted",
			query: `{"operator":"IsIn","left":"Policy Status","right":["Draft"]}`,
			want:  nil,
		},
		{
			// The platform keeps only operator, left and right; these two are
			// dropped, so a rule would store something other than what was written.
			name:  "derived keys are rejected",
			query: `{"operator":"IsIn","left":"a","right":["b"],"aql_column_name":"X","aql_label_name":"Y"}`,
			want:  []string{"aql_column_name", "aql_label_name"},
		},
		{
			name:  "nested conditions are walked",
			query: `{"operator":"And","left":{"operator":"Is","left":"a","right":"b","extra":1},"right":{"operator":"Is","left":"c","right":"d"}}`,
			want:  []string{"left.extra"},
		},
		{
			name:  "conditions inside arrays are walked",
			query: `{"operator":"And","left":[{"operator":"Is","left":"a","right":"b","bogus":true}],"right":"x"}`,
			want:  []string{"left[0].bogus"},
		},
		{
			name:  "scalar right-hand values are not walked",
			query: `{"operator":"IsIn","left":"a","right":["1","2","3"]}`,
			want:  nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var decoded interface{}
			if err := json.Unmarshal([]byte(tc.query), &decoded); err != nil {
				t.Fatalf("test query is not valid JSON: %v", err)
			}
			var got []string
			collectUnsupportedAQLKeys(decoded, "", &got)
			sort.Strings(got)

			if len(got) != len(tc.want) {
				t.Fatalf("unsupported keys = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("unsupported keys = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestValidateRuleQuery(t *testing.T) {
	tests := []struct {
		name      string
		queryType types.String
		query     string
		wantError bool
	}{
		{
			name:      "aql condition with only stored keys",
			queryType: types.StringValue("aql"),
			query:     `{"operator":"IsIn","left":"a","right":["b"]}`,
		},
		{
			name:      "aql condition with a discarded key",
			queryType: types.StringValue("aql"),
			query:     `{"operator":"IsIn","left":"a","right":["b"],"aql_column_name":"X"}`,
			wantError: true,
		},
		{
			// An unset type takes the schema default, so it is still checked.
			name:      "null type is treated as aql",
			queryType: types.StringNull(),
			query:     `{"operator":"IsIn","left":"a","right":["b"],"aql_column_name":"X"}`,
			wantError: true,
		},
		{
			// An aqlext query is a different structure, so its own keys stand.
			name:      "aqlext manipulations are left alone",
			queryType: types.StringValue("aqlext"),
			query:     `{"manipulations":[{"operator_name":"JoinLeftMinusRight","other_instance_id":"UAM","on_left_column":"Email"}]}`,
		},
		{
			// The condition under "filters" is an ordinary AQL condition and is
			// stored the same way, so extra keys there are discarded too.
			name:      "aqlext filters are checked as a condition",
			queryType: types.StringValue("aqlext"),
			query:     `{"manipulations":[],"filters":{"operator":"IsIn","left":"a","right":["b"],"aql_column_name":"X"}}`,
			wantError: true,
		},
		{
			// A pandas query is stored lower-cased, so anything else would store
			// a different expression than the one configured.
			name:      "pandas query in lower case",
			queryType: types.StringValue("pandas"),
			query:     `"` + "`policy status` == 'draft'" + `"`,
		},
		{
			name:      "pandas query with upper case",
			queryType: types.StringValue("pandas"),
			query:     `"` + "`Policy Status` == 'Draft'" + `"`,
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := &AnalysisRuleResourceModel{
				RuleQueryType: tc.queryType,
				RuleQuery:     jsontypes.NewNormalizedValue(tc.query),
			}
			var diags diag.Diagnostics
			validateRuleQuery(data, &diags)

			if got := diags.HasError(); got != tc.wantError {
				t.Errorf("HasError = %v, want %v (diags: %v)", got, tc.wantError, diags)
			}
		})
	}
}
