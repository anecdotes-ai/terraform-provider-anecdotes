// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

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
