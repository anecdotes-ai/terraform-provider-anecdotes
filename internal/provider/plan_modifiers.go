// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// An Optional+Computed attribute keeps its prior state value when it is removed
// from configuration, because the framework cannot tell "the user deleted this"
// apart from "the user never set it and the server computed it". On a resource
// whose Update is a full-object replace that is actively wrong: the stale value
// is re-sent on the next PUT, so deleting the attribute from config reports "No
// changes" and silently preserves what the user just removed.
//
// These modifiers restore the expected behavior by planning the attribute as
// unknown once it is absent from config but present in state. The update then
// sends no value for it, the platform re-applies its own default, and the
// refreshed result is what lands in state.
//
// Use them only where the platform genuinely has a default to fall back to.

type resetOnConfigRemovalSet struct{}

// ResetOnConfigRemovalSet returns a plan modifier that re-derives a set
// attribute from the platform once it is removed from configuration.
func ResetOnConfigRemovalSet() planmodifier.Set { return resetOnConfigRemovalSet{} }

func (m resetOnConfigRemovalSet) Description(ctx context.Context) string {
	return "Re-derives the value from the platform when the attribute is removed from configuration."
}

func (m resetOnConfigRemovalSet) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m resetOnConfigRemovalSet) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	// Create (no prior state) and destroy (no plan) are not removals.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	// Still configured, or never in state — nothing to reset.
	if !req.ConfigValue.IsNull() || req.StateValue.IsNull() {
		return
	}
	resp.PlanValue = types.SetUnknown(req.StateValue.ElementType(ctx))
}

type resetOnConfigRemovalList struct{}

// ResetOnConfigRemovalList returns a plan modifier that re-derives a list
// attribute from the platform once it is removed from configuration.
func ResetOnConfigRemovalList() planmodifier.List { return resetOnConfigRemovalList{} }

func (m resetOnConfigRemovalList) Description(ctx context.Context) string {
	return "Re-derives the value from the platform when the attribute is removed from configuration."
}

func (m resetOnConfigRemovalList) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m resetOnConfigRemovalList) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	if !req.ConfigValue.IsNull() || req.StateValue.IsNull() {
		return
	}
	resp.PlanValue = types.ListUnknown(req.StateValue.ElementType(ctx))
}

type resetOnConfigRemovalString struct{}

// ResetOnConfigRemovalString returns a plan modifier that re-derives a string
// attribute from the platform once it is removed from configuration.
func ResetOnConfigRemovalString() planmodifier.String { return resetOnConfigRemovalString{} }

func (m resetOnConfigRemovalString) Description(ctx context.Context) string {
	return "Re-derives the value from the platform when the attribute is removed from configuration."
}

func (m resetOnConfigRemovalString) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m resetOnConfigRemovalString) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	if !req.ConfigValue.IsNull() || req.StateValue.IsNull() {
		return
	}
	resp.PlanValue = types.StringUnknown()
}
