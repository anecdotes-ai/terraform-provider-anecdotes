// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MappingRequirementEvidenceResource{}
var _ resource.ResourceWithImportState = &MappingRequirementEvidenceResource{}

func NewMappingRequirementEvidenceResource() resource.Resource {
	return &MappingRequirementEvidenceResource{}
}

// MappingRequirementEvidenceResource defines the resource implementation for linking evidence to requirements.
type MappingRequirementEvidenceResource struct {
	client *client.AnecdotesClient
}

// MappingRequirementEvidenceResourceModel describes the resource data model.
type MappingRequirementEvidenceResourceModel struct {
	RequirementID types.String `tfsdk:"requirement_id"`
	EvidenceID    types.String `tfsdk:"evidence_id"`
}

func (r *MappingRequirementEvidenceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mapping_requirement_evidence"
}

func (r *MappingRequirementEvidenceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a link between an Anecdotes requirement and an evidence item.",
		MarkdownDescription: `
Manages a link between an Anecdotes requirement and an evidence item.

This resource creates the relationship between requirements and evidence in the
Requirements Hub. Once linked, the evidence appears in the requirement's evidence
section and is used for compliance monitoring.

> **Note:** The underlying API uses set-based replacement — each link/unlink
> operation reads the current list, modifies it, and writes back the full list.
> When multiple evidence links target the same requirement in parallel, a race
> condition may cause one link to be lost. Running ` + "`terraform apply`" + ` a second
> time will converge to the desired state. To avoid this, use ` + "`depends_on`" + `
> to sequence links to the same requirement.
`,
		Attributes: map[string]schema.Attribute{
			"requirement_id": schema.StringAttribute{
				Description: "The unique identifier of the requirement.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"evidence_id": schema.StringAttribute{
				Description: "The unique identifier of the evidence item to link.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *MappingRequirementEvidenceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.AnecdotesClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.AnecdotesClient, got: %T", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *MappingRequirementEvidenceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MappingRequirementEvidenceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requirementID := data.RequirementID.ValueString()
	evidenceID := data.EvidenceID.ValueString()

	if err := r.client.LinkEvidenceToRequirement(requirementID, evidenceID); err != nil {
		addClientError(&resp.Diagnostics, fmt.Sprintf("link evidence %s to requirement %s", evidenceID, requirementID), err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MappingRequirementEvidenceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MappingRequirementEvidenceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requirementID := data.RequirementID.ValueString()
	evidenceID := data.EvidenceID.ValueString()

	if err := r.client.GetRequirementEvidenceLink(requirementID, evidenceID); err != nil {
		// Link no longer exists — remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MappingRequirementEvidenceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Both fields are ForceNew, so Update should never be called.
	var data MappingRequirementEvidenceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MappingRequirementEvidenceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MappingRequirementEvidenceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requirementID := data.RequirementID.ValueString()
	evidenceID := data.EvidenceID.ValueString()

	if err := r.client.UnlinkEvidenceFromRequirement(requirementID, evidenceID); err != nil {
		addClientError(&resp.Diagnostics, fmt.Sprintf("unlink evidence %s from requirement %s", evidenceID, requirementID), err)
		return
	}
}

func (r *MappingRequirementEvidenceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in the format 'requirement_id:evidence_id', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("requirement_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("evidence_id"), parts[1])...)
}
