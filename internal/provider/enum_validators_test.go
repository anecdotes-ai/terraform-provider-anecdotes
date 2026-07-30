// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// These are plain unit tests (no TF_ACC / no API): they build a resource's schema,
// pull the validators off an enum attribute, and assert an off-set value is rejected
// while a valid value is accepted. stringvalidator.OneOf runs at plan time, so this
// proves invalid enum values fail before any API call.

func mustSchema(r resource.Resource) schema.Schema {
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp.Schema
}

// strValidators extracts the []validator.String from a string attribute, failing the
// test if the attribute is missing or not a StringAttribute.
func strValidators(t *testing.T, a schema.Attribute, ok bool) []validator.String {
	t.Helper()
	if !ok {
		t.Fatal("attribute not found in schema")
	}
	sa, isStr := a.(schema.StringAttribute)
	if !isStr {
		t.Fatalf("attribute is %T, not schema.StringAttribute", a)
	}
	return sa.Validators
}

// validatorsReject reports whether any of the validators rejects value.
func validatorsReject(vs []validator.String, value string) bool {
	for _, v := range vs {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("x"),
			ConfigValue: types.StringValue(value),
		}, resp)
		if resp.Diagnostics.HasError() {
			return true
		}
	}
	return false
}

func TestEnumAttributeValidators(t *testing.T) {
	cases := []struct {
		name       string
		validators []validator.String
		bad        string
		good       string
	}{
		{
			name:       "control.maturity_level",
			validators: strAttr(t, mustSchema(NewControlResource()), "maturity_level"),
			bad:        "banana", good: "DEFINED",
		},
		{
			name:       "requirement.category",
			validators: strAttr(t, mustSchema(NewRequirementResource()), "category"),
			bad:        "banana", good: "Custom Requirements",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if len(c.validators) == 0 {
				t.Fatalf("%s: expected an enum validator, found none", c.name)
			}
			if !validatorsReject(c.validators, c.bad) {
				t.Errorf("%s: value %q should be rejected at plan time, but was accepted", c.name, c.bad)
			}
			if validatorsReject(c.validators, c.good) {
				t.Errorf("%s: value %q should be accepted, but was rejected", c.name, c.good)
			}
		})
	}
}

// setValidatorsReject reports whether any of the Set validators rejects a
// single-element set containing value. Used for framework auditor_visible_*
// set attributes, whose element values are constrained via setvalidator.ValueStringsAre.
func setValidatorsReject(vs []validator.Set, value string) bool {
	cfg := types.SetValueMust(types.StringType, []attr.Value{types.StringValue(value)})
	for _, v := range vs {
		resp := &validator.SetResponse{}
		v.ValidateSet(context.Background(), validator.SetRequest{
			Path:        path.Root("x"),
			ConfigValue: cfg,
		}, resp)
		if resp.Diagnostics.HasError() {
			return true
		}
	}
	return false
}

// TestFrameworkAuditorSetEnumValidators proves the framework auditor_visible_*
// set attributes reject off-set element values at plan time and accept valid ones.
func TestFrameworkAuditorSetEnumValidators(t *testing.T) {
	fwSchema := mustSchema(NewFrameworkResource())

	cases := []struct {
		name string
		attr string
		bad  string
		good string
	}{
		{
			name: "framework.auditor_visible_control_statuses",
			attr: "auditor_visible_control_statuses",
			bad:  "banana", good: client.ValidAuditorControlStatuses()[0],
		},
		{
			name: "framework.auditor_visible_evidence_statuses",
			attr: "auditor_visible_evidence_statuses",
			bad:  "banana", good: client.ValidAuditorEvidenceStatuses()[0],
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, ok := fwSchema.Attributes[c.attr]
			if !ok {
				t.Fatalf("%s: attribute not found in schema", c.attr)
			}
			sa, isSet := a.(schema.SetAttribute)
			if !isSet {
				t.Fatalf("%s: attribute is %T, not schema.SetAttribute", c.attr, a)
			}
			if len(sa.Validators) == 0 {
				t.Fatalf("%s: expected an enum validator, found none", c.name)
			}
			if !setValidatorsReject(sa.Validators, c.bad) {
				t.Errorf("%s: value %q should be rejected at plan time, but was accepted", c.name, c.bad)
			}
			if setValidatorsReject(sa.Validators, c.good) {
				t.Errorf("%s: value %q should be accepted, but was rejected", c.name, c.good)
			}
		})
	}
}

// strAttr pulls a top-level string attribute's validators.
func strAttr(t *testing.T, s schema.Schema, name string) []validator.String {
	t.Helper()
	a, ok := s.Attributes[name]
	return strValidators(t, a, ok)
}
