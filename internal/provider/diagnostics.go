// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"errors"
	"fmt"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// addClientError renders a client/API error into a Terraform diagnostic with a
// classification-appropriate summary and a clean, redacted detail message.
//
// `action` is a short phrase describing what failed, e.g. "create framework";
// the rendered detail reads "Unable to <action>: <message>". When the error is a
// *client.APIError its decoded message is used (raw bodies, HTML, request IDs and
// credentials are never surfaced); otherwise the error string is used as-is.
func addClientError(diags *diag.Diagnostics, action string, err error) {
	summary, detail := describeClientError(action, err)
	diags.AddError(summary, detail)
}

// describeClientError computes the (summary, detail) pair for a client error.
func describeClientError(action string, err error) (summary, detail string) {
	apiErr, ok := client.AsAPIError(err)
	if !ok {
		// List-then-filter lookups report a missing resource via ErrNotFound rather
		// than an HTTP 404. Classify it the same as a 404 so data sources (which have
		// no drop-from-state) still surface a clear "Resource Not Found".
		if errors.Is(err, client.ErrNotFound) {
			return "Resource Not Found", fmt.Sprintf("Unable to %s: %s", action, err)
		}
		return "Anecdotes API Error", fmt.Sprintf("Unable to %s: %s", action, err)
	}

	switch apiErr.Class {
	case client.ClassValidation:
		summary = "Invalid Configuration"
	case client.ClassPermission:
		summary = "Authentication Failed"
	case client.ClassFeatureGate:
		summary = "Feature Not Enabled"
	case client.ClassNotFound:
		summary = "Resource Not Found"
	case client.ClassConflict:
		summary = "Resource Conflict"
	case client.ClassUnsupported:
		summary = "Operation Not Supported"
	case client.ClassServer:
		summary = fmt.Sprintf("Anecdotes API Error (HTTP %d)", apiErr.StatusCode)
	default:
		summary = "Anecdotes API Error"
	}

	return summary, fmt.Sprintf("Unable to %s: %s", action, apiErr.Message)
}
