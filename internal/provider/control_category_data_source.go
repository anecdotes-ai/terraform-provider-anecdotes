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
var _ datasource.DataSource = &ControlCategoryDataSource{}

func NewControlCategoryDataSource() datasource.DataSource {
	return &ControlCategoryDataSource{}
}

// ControlCategoryDataSource defines the data source implementation.
type ControlCategoryDataSource struct {
	client *client.AnecdotesClient
}

// ControlCategoryDataSourceModel describes the data source data model.
type ControlCategoryDataSourceModel struct {
	CategoryID   types.String `tfsdk:"category_id"`
	CategoryName types.String `tfsdk:"category_name"`
	FrameworkID  types.String `tfsdk:"framework_id"`
}

func (d *ControlCategoryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_control_category"
}

func (d *ControlCategoryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to look up an existing control category by category ID.",

		Attributes: map[string]schema.Attribute{
			"category_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the control category to look up.",
			},
			"category_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The name of the control category.",
			},
			"framework_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the framework this category belongs to.",
			},
		},
	}
}

func (d *ControlCategoryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ControlCategoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ControlCategoryDataSourceModel

	// Read Terraform configuration data into the model
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	categoryID := data.CategoryID.ValueString()

	// Get control category from API
	category, err := d.client.GetControlCategory(ctx, categoryID)
	if err != nil {
		addClientError(&resp.Diagnostics, fmt.Sprintf("read control category with ID %s", categoryID), err)
		return
	}

	// Set state
	data.CategoryID = types.StringValue(category.CategoryID)
	data.CategoryName = types.StringValue(category.CategoryName)
	data.FrameworkID = types.StringValue(category.FrameworkID)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
