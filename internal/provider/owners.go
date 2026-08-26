// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ownersEmailValidator rejects any owners element that is not a valid email
// address. Shared by every resource with a Terraform-owned owners attribute
// (anecdotes_control, anecdotes_requirement, anecdotes_requirement_view), so
// a format change only needs to happen in one place.
var ownersEmailValidator = setvalidator.ValueStringsAre(
	stringvalidator.RegexMatches(
		regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
		"must be a valid email address",
	),
)

// ownersFromAPI computes the owners attribute value from the platform's
// response. An empty set and an unset attribute are distinct: the platform
// reporting no owners only clears state that was already tracked (Terraform
// owns this attribute), so `current` — the value already in state — decides
// between an empty set and null when apiOwners is empty. Checking length
// rather than whether apiOwners is nil also avoids depending on whether the
// platform's JSON encoder omits the key or sends `[]` for "no owners".
func ownersFromAPI(ctx context.Context, diags *diag.Diagnostics, current types.Set, apiOwners []string) types.Set {
	if len(apiOwners) > 0 {
		ownersSet, d := types.SetValueFrom(ctx, types.StringType, apiOwners)
		diags.Append(d...)
		return ownersSet
	}
	if !current.IsNull() {
		return types.SetValueMust(types.StringType, []attr.Value{})
	}
	return types.SetNull(types.StringType)
}
