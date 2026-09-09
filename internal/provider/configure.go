// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// configureClient resolves the API client the provider hands to a resource or a
// data source. kind names which of the two is being configured, so a mismatch
// says where it happened.
//
// providerData is nil while the provider's own configuration is still unknown.
// That is not an error: the framework configures again once it is known, and
// until then there is no client to hold.
func configureClient(providerData any, kind string, diags *diag.Diagnostics) *client.AnecdotesClient {
	if providerData == nil {
		return nil
	}

	c, ok := providerData.(*client.AnecdotesClient)
	if !ok {
		diags.AddError(
			fmt.Sprintf("Unexpected %s Configure Type", kind),
			fmt.Sprintf("Expected *client.AnecdotesClient, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}

	return c
}
