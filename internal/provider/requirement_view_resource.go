// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RequirementViewResource{}
var _ resource.ResourceWithImportState = &RequirementViewResource{}

func NewRequirementViewResource() resource.Resource {
	return &RequirementViewResource{}
}

// RequirementViewResource defines the resource implementation.
type RequirementViewResource struct {
	client *client.AnecdotesClient
}

// RequirementViewResourceModel describes the resource data model.
type RequirementViewResourceModel struct {
	RequirementID types.String `tfsdk:"requirement_id"`
	ParentID      types.String `tfsdk:"parent_id"`
	ViewName      types.String `tfsdk:"view_name"`
	Category      types.String `tfsdk:"category"`
	Owners        types.Set    `tfsdk:"owners"`
}

func (r *RequirementViewResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_requirement_view"
}

func (r *RequirementViewResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anecdotes Requirement View.",
		MarkdownDescription: `
Manages an Anecdotes Requirement View: a requirement scoped beneath another ("parent")
requirement, letting the same underlying requirement content apply per control or
framework without duplicating it.

## Relationships

` + "```" + `
Requirement (anecdotes_requirement, the parent)
    │
    └── Requirement View (this resource)
            │
            └── linked to a control via anecdotes_mapping_control_requirement
` + "```" + `

## Key Concept: Parent-Inherited Fields

On creation, a view inherits its description, related evidences/policies, and scoping
overrides from its parent — those are not settable on the view itself. ` + "`category`" + `
and ` + "`owners`" + ` are independent of the parent, exactly like on ` + "`anecdotes_requirement`" + `.

## Immutability

` + "`parent_id`" + ` cannot change after creation; changing it in configuration replaces
the resource.
`,

		Attributes: map[string]schema.Attribute{
			"requirement_id": schema.StringAttribute{
				Description:         "The unique identifier of the requirement view in Anecdotes (e.g., requirement_9012).",
				MarkdownDescription: "The unique identifier of the requirement view in Anecdotes (e.g., `requirement_9012`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"parent_id": schema.StringAttribute{
				Description: "The ID of the parent requirement this view is scoped beneath. Immutable after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"view_name": schema.StringAttribute{
				Description: "The human-readable name of the requirement view. This is displayed as the view's title.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 500),
				},
			},

			"category": schema.StringAttribute{
				Description:         "The category this requirement view belongs to. Must be one of the categories Anecdotes defines. Defaults to 'Custom Requirements'.",
				MarkdownDescription: "The category this requirement view belongs to. One of the categories Anecdotes defines; views that do not fit one of them belong under `Custom Requirements`, which is also the default. Removing the attribute restores the default.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("Custom Requirements"),
				Validators: []validator.String{
					stringvalidator.OneOf(client.ValidRequirementCategories()...),
				},
			},

			"owners": schema.SetAttribute{
				Description: "Email addresses of users responsible for this requirement view. Order does not matter. Terraform owns this attribute: removing it clears the owners.",
				Optional:    true,
				ElementType: types.StringType,
				Validators:  []validator.Set{ownersEmailValidator},
			},
		},
	}
}

func (r *RequirementViewResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RequirementViewResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RequirementViewResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.RequirementViewCreateRequest{
		RequirementParentID: data.ParentID.ValueString(),
		ViewName:            data.ViewName.ValueString(),
		RequirementCategory: data.Category.ValueString(),
	}

	if !data.Owners.IsNull() && !data.Owners.IsUnknown() {
		var owners []string
		resp.Diagnostics.Append(data.Owners.ElementsAs(ctx, &owners, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.RequirementOwners = owners
	}

	view, err := r.client.CreateRequirementView(ctx, createReq)
	if err != nil {
		addClientError(&resp.Diagnostics, "create requirement view", err)
		return
	}

	r.setRequirementViewState(ctx, &data, view, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RequirementViewResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RequirementViewResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	view, err := r.client.GetRequirement(ctx, data.RequirementID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addClientError(&resp.Diagnostics, "read requirement view", err)
		return
	}

	// A requirement id that isn't a view (no parent) was not created by this
	// resource type; importing one here would silently misrepresent it.
	if view.RequirementParentID == "" {
		resp.Diagnostics.AddError(
			"Not a Requirement View",
			fmt.Sprintf("Requirement %s has no parent requirement, so it is not a Requirement View. "+
				"Manage it with anecdotes_requirement instead.", data.RequirementID.ValueString()),
		)
		return
	}

	r.setRequirementViewState(ctx, &data, view, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RequirementViewResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RequirementViewResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	viewName := data.ViewName.ValueString()
	updateReq := &client.RequirementViewUpdateRequest{
		ViewName:            &viewName,
		RequirementCategory: data.Category.ValueString(),
	}

	// Terraform owns the attribute: absent means no owners.
	owners := []string{}
	if !data.Owners.IsNull() && !data.Owners.IsUnknown() {
		resp.Diagnostics.Append(data.Owners.ElementsAs(ctx, &owners, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	updateReq.RequirementOwners = &owners

	view, err := r.client.UpdateRequirementView(ctx, data.RequirementID.ValueString(), updateReq)
	if err != nil {
		addClientError(&resp.Diagnostics, "update requirement view", err)
		return
	}

	r.setRequirementViewState(ctx, &data, view, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RequirementViewResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RequirementViewResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteRequirement(ctx, data.RequirementID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		addClientError(&resp.Diagnostics, "delete requirement view", err)
		return
	}
}

func (r *RequirementViewResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("requirement_id"), req, resp)
}

// setRequirementViewState sets the Terraform state from an API Requirement response.
func (r *RequirementViewResource) setRequirementViewState(ctx context.Context, data *RequirementViewResourceModel, view *client.Requirement, diags *diag.Diagnostics) {
	data.RequirementID = types.StringValue(view.RequirementID)
	data.ParentID = types.StringValue(view.RequirementParentID)
	// requirement_name resolves to the parent's name for a view, not the view's
	// own name — view_name must be read directly.
	data.ViewName = types.StringValue(view.ViewName)
	data.Category = types.StringValue(view.RequirementCategory)
	data.Owners = ownersFromAPI(ctx, diags, data.Owners, view.RequirementOwners)
}
