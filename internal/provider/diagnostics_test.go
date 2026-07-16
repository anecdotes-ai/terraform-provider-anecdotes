// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestDescribeClientError_Classified(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantSummary string
		wantDetail  string
	}{
		{
			name:        "validation",
			err:         &client.APIError{StatusCode: 422, Class: client.ClassValidation, Message: "name: field required"},
			wantSummary: "Invalid Configuration",
			wantDetail:  "Unable to create framework: name: field required",
		},
		{
			name:        "permission",
			err:         &client.APIError{StatusCode: 401, Class: client.ClassPermission, Message: "creds rejected"},
			wantSummary: "Authentication Failed",
		},
		{
			name:        "feature gate",
			err:         &client.APIError{StatusCode: 402, Class: client.ClassFeatureGate, Message: "AI Consent"},
			wantSummary: "Feature Not Enabled",
		},
		{
			name:        "server",
			err:         &client.APIError{StatusCode: 500, Class: client.ClassServer, Message: "internal"},
			wantSummary: "Anecdotes API Error (HTTP 500)",
		},
		{
			name:        "non-api error falls back",
			err:         errors.New("dial tcp: connection refused"),
			wantSummary: "Anecdotes API Error",
			wantDetail:  "Unable to create framework: dial tcp: connection refused",
		},
		{
			name:        "wrapped ErrNotFound (list-then-filter lookup)",
			err:         fmt.Errorf("tag not found: %s: %w", "tag_1", client.ErrNotFound),
			wantSummary: "Resource Not Found",
			wantDetail:  "Unable to create framework: tag not found: tag_1: resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, detail := describeClientError("create framework", tt.err)
			if summary != tt.wantSummary {
				t.Errorf("summary = %q, want %q", summary, tt.wantSummary)
			}
			if tt.wantDetail != "" && detail != tt.wantDetail {
				t.Errorf("detail = %q, want %q", detail, tt.wantDetail)
			}
			if !strings.HasPrefix(detail, "Unable to create framework:") {
				t.Errorf("detail = %q, want it to start with the action phrase", detail)
			}
		})
	}
}

func TestAddClientError_RecordsDiagnostic(t *testing.T) {
	var diags diag.Diagnostics
	addClientError(&diags, "delete framework", &client.APIError{StatusCode: 404, Class: client.ClassNotFound, Message: "Framework does not exists"})
	if !diags.HasError() {
		t.Fatal("expected a diagnostic to be recorded")
	}
	d := diags.Errors()[0]
	if d.Summary() != "Resource Not Found" {
		t.Errorf("summary = %q", d.Summary())
	}
	if d.Detail() != "Unable to delete framework: Framework does not exists" {
		t.Errorf("detail = %q", d.Detail())
	}
}
