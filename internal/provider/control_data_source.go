// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ControlDataSource{}

func NewControlDataSource() datasource.DataSource {
	return &ControlDataSource{}
}

// ControlDataSource defines the data source implementation.
type ControlDataSource struct {
	client *client.AnecdotesClient
}

// ControlDataSourceModel describes the data source data model.
type ControlDataSourceModel struct {
	ControlID   types.String `tfsdk:"control_id"`
	FrameworkID types.String `tfsdk:"framework_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	CategoryID  types.String `tfsdk:"category_id"`
	Category    types.String `tfsdk:"category"`
	Status      types.String `tfsdk:"status"`
	Owners      types.List   `tfsdk:"owners"`
	Tags        types.List   `tfsdk:"tags"`
}

func (d *ControlDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_control"
}

func (d *ControlDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to look up an existing Anecdotes control by its control ID.",

		Attributes: map[string]schema.Attribute{
			"control_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The unique identifier of the control to look up.",
			},
			"framework_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The framework ID that the control belongs to.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The human-readable name of the control.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "A detailed description of the control.",
			},
			"category_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The framework category ID this control belongs to.",
			},
			"category": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The category name this control belongs to.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The current status of the control.",
			},
			"owners": schema.ListAttribute{
				Computed:            true,
				MarkdownDescription: "List of email addresses of users who own this control.",
				ElementType:         types.StringType,
			},
			"tags": schema.ListAttribute{
				Computed:            true,
				MarkdownDescription: "Tags applied to the control.",
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *ControlDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.AnecdotesClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.AnecdotesClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *ControlDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ControlDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get control from API (GetControl takes frameworkID and controlID)
	control, err := d.client.GetControl(ctx, data.FrameworkID.ValueString(), data.ControlID.ValueString())
	if err != nil {
		addClientError(&resp.Diagnostics, "read control", err)
		return
	}

	// Set state from the API response
	data.ControlID = types.StringValue(control.ControlID)
	data.FrameworkID = types.StringValue(control.FrameworkID)
	data.Name = types.StringValue(control.ControlName)
	data.Description = types.StringValue(control.ControlDescription)
	data.CategoryID = types.StringValue(control.ControlFrameworkCategoryID)
	data.Category = types.StringValue(control.ControlCategory)
	data.Status = types.StringValue(control.ControlStatus.Status)

	// Build owners list
	if len(control.ControlOwners) > 0 {
		ownersList, diags := types.ListValueFrom(ctx, types.StringType, control.ControlOwners)
		resp.Diagnostics.Append(diags...)
		data.Owners = ownersList
	} else {
		data.Owners = types.ListNull(types.StringType)
	}

	// Build tags list
	if len(control.ControlTags) > 0 {
		tagsList, diags := types.ListValueFrom(ctx, types.StringType, control.ControlTags)
		resp.Diagnostics.Append(diags...)
		data.Tags = tagsList
	} else {
		data.Tags = types.ListNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
