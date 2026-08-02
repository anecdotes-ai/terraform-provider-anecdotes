// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ControlCategoryResource{}
var _ resource.ResourceWithImportState = &ControlCategoryResource{}

func NewControlCategoryResource() resource.Resource {
	return &ControlCategoryResource{}
}

// ControlCategoryResource defines the resource implementation.
type ControlCategoryResource struct {
	client *client.AnecdotesClient
}

// ControlCategoryResourceModel describes the resource data model.
type ControlCategoryResourceModel struct {
	CategoryID   types.String `tfsdk:"category_id"`
	CategoryName types.String `tfsdk:"name"`
	FrameworkID  types.String `tfsdk:"framework_id"`
}

func (r *ControlCategoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_control_category"
}

func (r *ControlCategoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a control category within an Anecdotes framework.",
		MarkdownDescription: `
Manages a control category within an Anecdotes framework.

Control categories are used to organize controls within a framework. Each control must belong to a category.
`,

		Attributes: map[string]schema.Attribute{
			"category_id": schema.StringAttribute{
				Description: "The unique identifier of the control category.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"name": schema.StringAttribute{
				Description: "The name of the control category (e.g., 'Access Management', 'Security Operations').",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},

			"framework_id": schema.StringAttribute{
				Description: "The ID of the framework this category belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(), // Changing framework requires recreating
				},
			},
		},
	}
}

func (r *ControlCategoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ControlCategoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ControlCategoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build create request
	createReq := &client.ControlCategoryCreateRequest{
		CategoryName: data.CategoryName.ValueString(),
		FrameworkID:  data.FrameworkID.ValueString(),
	}

	// Call API to create category
	category, err := r.client.CreateControlCategory(createReq)
	if err != nil {
		addClientError(&resp.Diagnostics, "create control category", err)
		return
	}

	// Set computed values
	data.CategoryID = types.StringValue(category.CategoryID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ControlCategoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ControlCategoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get category from API
	category, err := r.client.GetControlCategory(data.CategoryID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addClientError(&resp.Diagnostics, "read control category", err)
		return
	}

	// Update state
	data.CategoryName = types.StringValue(category.CategoryName)
	data.FrameworkID = types.StringValue(category.FrameworkID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ControlCategoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ControlCategoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build update request
	updateReq := &client.ControlCategoryUpdateRequest{
		CategoryName: data.CategoryName.ValueString(),
	}

	// Call API to update category
	err := r.client.UpdateControlCategory(data.CategoryID.ValueString(), updateReq)
	if err != nil {
		addClientError(&resp.Diagnostics, "update control category", err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ControlCategoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ControlCategoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteControlCategory(data.CategoryID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		addClientError(&resp.Diagnostics, "delete control category", err)
		return
	}
}

func (r *ControlCategoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by category_id
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("category_id"), req.ID)...)
}
