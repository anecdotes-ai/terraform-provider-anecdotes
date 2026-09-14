// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strings"
	"testing"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestConfigureClient(t *testing.T) {
	configured := &client.AnecdotesClient{}

	tests := []struct {
		name         string
		providerData any
		kind         string
		wantClient   *client.AnecdotesClient
		wantSummary  string
	}{
		{
			// The provider's own configuration is not resolved yet. There is no
			// client to hold, and that is not something to report.
			name:         "not configured yet",
			providerData: nil,
			kind:         "Data Source",
			wantClient:   nil,
		},
		{
			name:         "configured",
			providerData: configured,
			kind:         "Resource",
			wantClient:   configured,
		},
		{
			name:         "wrong type names which side it happened on",
			providerData: "not a client",
			kind:         "Data Source",
			wantClient:   nil,
			wantSummary:  "Unexpected Data Source Configure Type",
		},
		{
			name:         "wrong type on a resource",
			providerData: 42,
			kind:         "Resource",
			wantClient:   nil,
			wantSummary:  "Unexpected Resource Configure Type",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			got := configureClient(tc.providerData, tc.kind, &diags)

			if got != tc.wantClient {
				t.Errorf("client = %v, want %v", got, tc.wantClient)
			}

			if tc.wantSummary == "" {
				if diags.HasError() {
					t.Errorf("expected no diagnostic, got %v", diags.Errors())
				}
				return
			}

			if !diags.HasError() {
				t.Fatalf("expected the diagnostic %q, got none", tc.wantSummary)
			}
			if summary := diags.Errors()[0].Summary(); summary != tc.wantSummary {
				t.Errorf("summary = %q, want %q", summary, tc.wantSummary)
			}
			// The detail names the type that arrived, which is what makes the
			// report actionable.
			if detail := diags.Errors()[0].Detail(); !strings.Contains(detail, "*client.AnecdotesClient") {
				t.Errorf("detail should name the expected type, got %q", detail)
			}
		})
	}
}
