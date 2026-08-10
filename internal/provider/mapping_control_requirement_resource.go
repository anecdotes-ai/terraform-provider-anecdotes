// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MappingControlRequirementResource{}
var _ resource.ResourceWithImportState = &MappingControlRequirementResource{}

func NewMappingControlRequirementResource() resource.Resource {
	return &MappingControlRequirementResource{}
}

// MappingControlRequirementResource defines the resource implementation.
type MappingControlRequirementResource struct {
	client *client.AnecdotesClient
}

// MappingControlRequirementResourceModel describes the resource data model.
type MappingControlRequirementResourceModel struct {
	ControlID     types.String `tfsdk:"control_id"`
	RequirementID types.String `tfsdk:"requirement_id"`
	FrameworkID   types.String `tfsdk:"framework_id"`
}

func (r *MappingControlRequirementResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mapping_control_requirement"
}

func (r *MappingControlRequirementResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a link between an Anecdotes control and a requirement.",
		MarkdownDescription: `
Manages a link between an Anecdotes control and a requirement.

This resource creates the M:N relationship between controls and requirements, enabling
cross-framework requirement mapping. A single requirement can be linked to multiple
controls across different frameworks.
`,

		Attributes: map[string]schema.Attribute{
			"control_id": schema.StringAttribute{
				Description: "The unique identifier of the control to link.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"requirement_id": schema.StringAttribute{
				Description: "The unique identifier of the requirement to link.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"framework_id": schema.StringAttribute{
				Description: "The framework ID associated with the control-requirement link (computed).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *MappingControlRequirementResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.AnecdotesClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.AnecdotesClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *MappingControlRequirementResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MappingControlRequirementResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	controlID := data.ControlID.ValueString()
	requirementID := data.RequirementID.ValueString()

	link, err := r.client.LinkRequirementToControl(ctx, controlID, requirementID)
	if err != nil {
		addClientError(&resp.Diagnostics, "link requirement to control", err)
		return
	}

	data.ControlID = types.StringValue(link.ControlID)
	data.RequirementID = types.StringValue(link.RequirementID)
	data.FrameworkID = types.StringValue(link.FrameworkID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MappingControlRequirementResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MappingControlRequirementResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	controlID := data.ControlID.ValueString()
	requirementID := data.RequirementID.ValueString()

	link, err := r.client.GetControlRequirementLink(ctx, controlID, requirementID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addClientError(&resp.Diagnostics, "read control-requirement mapping", err)
		return
	}

	data.ControlID = types.StringValue(link.ControlID)
	data.RequirementID = types.StringValue(link.RequirementID)
	data.FrameworkID = types.StringValue(link.FrameworkID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MappingControlRequirementResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MappingControlRequirementResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MappingControlRequirementResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MappingControlRequirementResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	controlID := data.ControlID.ValueString()
	requirementID := data.RequirementID.ValueString()

	err := r.client.UnlinkRequirementFromControl(ctx, controlID, requirementID)
	if err != nil {
		addClientError(&resp.Diagnostics, "unlink requirement from control", err)
		return
	}
}

func (r *MappingControlRequirementResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: control_id/requirement_id
	parts := splitImportID(req.ID, 2)
	if parts == nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in the format 'control_id/requirement_id', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("control_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("requirement_id"), parts[1])...)
}
