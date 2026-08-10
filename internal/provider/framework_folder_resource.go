// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &FrameworkFolderResource{}
var _ resource.ResourceWithImportState = &FrameworkFolderResource{}

func NewFrameworkFolderResource() resource.Resource {
	return &FrameworkFolderResource{}
}

// FrameworkFolderResource defines the resource implementation.
type FrameworkFolderResource struct {
	client *client.AnecdotesClient
}

// FrameworkFolderResourceModel describes the resource data model.
type FrameworkFolderResourceModel struct {
	FolderID types.String `tfsdk:"folder_id"`
	Name     types.String `tfsdk:"name"`
}

func (r *FrameworkFolderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_framework_folder"
}

func (r *FrameworkFolderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Anecdotes Folder Resource - Used to organize frameworks into folders.",

		Attributes: map[string]schema.Attribute{
			"folder_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier for the folder (UUID format).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the folder.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
		},
	}
}

func (r *FrameworkFolderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *FrameworkFolderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FrameworkFolderResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate a UUID for the folder ID
	folderID, err := uuid.GenerateUUID()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error generating folder ID",
			"Could not generate UUID for folder: "+err.Error(),
		)
		return
	}

	// Create Folder
	folderReq := &client.FolderCreateRequest{
		ID:             folderID,
		Name:           plan.Name.ValueString(),
		FrameworksList: []string{},
	}

	folder, err := r.client.CreateFolder(ctx, folderReq)
	if err != nil {
		addClientError(&resp.Diagnostics, "create folder", err)
		return
	}

	// Update the model with values from the API
	plan.FolderID = types.StringValue(folder.ID)
	plan.Name = types.StringValue(folder.Name)

	// Write logs using the tflog package
	tflog.Trace(ctx, "created a folder", map[string]interface{}{
		"folder_id":   folder.ID,
		"folder_name": folder.Name,
	})

	// Save data into Terraform state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *FrameworkFolderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FrameworkFolderResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	folder, err := r.client.GetFolder(ctx, state.FolderID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addClientError(&resp.Diagnostics, "read folder ID "+state.FolderID.ValueString(), err)
		return
	}

	state.Name = types.StringValue(folder.Name)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *FrameworkFolderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FrameworkFolderResourceModel
	var state FrameworkFolderResourceModel

	// Read plan and state
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the folder
	folderReq := &client.FolderUpdateRequest{
		Name: plan.Name.ValueString(),
	}

	folder, err := r.client.UpdateFolder(ctx, state.FolderID.ValueString(), folderReq)
	if err != nil {
		addClientError(&resp.Diagnostics, "update folder", err)
		return
	}

	// Update state with values from API
	plan.FolderID = state.FolderID
	plan.Name = types.StringValue(folder.Name)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *FrameworkFolderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FrameworkFolderResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteFolder(ctx, state.FolderID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		addClientError(&resp.Diagnostics, "delete folder", err)
		return
	}
}

func (r *FrameworkFolderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("folder_id"), req, resp)
}
