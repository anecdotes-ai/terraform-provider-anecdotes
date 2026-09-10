// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// These helpers convert an optional Terraform attribute into a pointer for an API
// request struct (pointer + `omitempty`), so a field the user did not configure is
// omitted from the request rather than sent as a zero value.
//
// A null or unknown value returns nil (the field is omitted). For strings, an empty
// string is preserved as a non-nil pointer to "" — Terraform distinguishes null
// ("not set") from "" ("set to empty"), so a user can clear a field by setting "".
// Use a site-specific guard instead where the API rejects empty values.

func optionalStringPtr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

func optionalBoolPtr(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

// optionalIntPtr returns *int (the element type used by several request structs),
// or nil when the value is null/unknown.
func optionalIntPtr(v types.Int64) *int {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	n := int(v.ValueInt64())
	return &n
}

func optionalFloat64Ptr(v types.Float64) *float64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	f := v.ValueFloat64()
	return &f
}

// optionalStringValue converts a possibly-nil string pointer from the API into
// a Terraform string value, null when the pointer is nil.
func optionalStringValue(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

// stringsFromList converts an optional/required list-of-string attribute into a
// []string, or nil when the value is null/unknown (so it marshals to JSON null,
// matching the platform's canonical "no value" representation for these fields).
func stringsFromList(ctx context.Context, list types.List, diags *diag.Diagnostics) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var values []string
	diags.Append(list.ElementsAs(ctx, &values, false)...)
	return values
}

// stringsFromSet converts an optional/required set-of-string attribute into a
// []string, or nil when the value is null/unknown (so it marshals to JSON null,
// matching the platform's canonical "no value" representation for these fields).
func stringsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var values []string
	diags.Append(set.ElementsAs(ctx, &values, false)...)
	return values
}

// isCustomRole reports whether a role is a tenant-scoped custom role (as
// opposed to a built-in global role) based on its attributes map.
func isCustomRole(role client.Role) bool {
	_, ok := role.Attributes["tenant"]
	return ok
}
