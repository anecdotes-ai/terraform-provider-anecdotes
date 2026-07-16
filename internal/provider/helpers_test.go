// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOptionalStringPtr(t *testing.T) {
	if optionalStringPtr(types.StringNull()) != nil {
		t.Error("null should map to nil")
	}
	if optionalStringPtr(types.StringUnknown()) != nil {
		t.Error("unknown should map to nil")
	}
	if got := optionalStringPtr(types.StringValue("hi")); got == nil || *got != "hi" {
		t.Errorf("value should map to &value, got %v", got)
	}
	// Empty string is preserved (null vs "" are distinct in Terraform).
	if got := optionalStringPtr(types.StringValue("")); got == nil || *got != "" {
		t.Errorf("empty string should map to &\"\", got %v", got)
	}
}

func TestOptionalBoolPtr(t *testing.T) {
	if optionalBoolPtr(types.BoolNull()) != nil || optionalBoolPtr(types.BoolUnknown()) != nil {
		t.Error("null/unknown should map to nil")
	}
	if got := optionalBoolPtr(types.BoolValue(true)); got == nil || *got != true {
		t.Errorf("value should map to &value, got %v", got)
	}
	if got := optionalBoolPtr(types.BoolValue(false)); got == nil || *got != false {
		t.Errorf("false should still map to &false (not nil), got %v", got)
	}
}

func TestOptionalIntPtr(t *testing.T) {
	if optionalIntPtr(types.Int64Null()) != nil || optionalIntPtr(types.Int64Unknown()) != nil {
		t.Error("null/unknown should map to nil")
	}
	if got := optionalIntPtr(types.Int64Value(5)); got == nil || *got != 5 {
		t.Errorf("value should map to &value, got %v", got)
	}
}

func TestOptionalFloat64Ptr(t *testing.T) {
	if optionalFloat64Ptr(types.Float64Null()) != nil || optionalFloat64Ptr(types.Float64Unknown()) != nil {
		t.Error("null/unknown should map to nil")
	}
	if got := optionalFloat64Ptr(types.Float64Value(1.5)); got == nil || *got != 1.5 {
		t.Errorf("value should map to &value, got %v", got)
	}
}
