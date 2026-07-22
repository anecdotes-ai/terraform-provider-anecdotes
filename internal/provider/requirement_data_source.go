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
var _ datasource.DataSource = &RequirementDataSource{}

func NewRequirementDataSource() datasource.DataSource {
	return &RequirementDataSource{}
}

// RequirementDataSource defines the data source implementation.
type RequirementDataSource struct {
	client *client.AnecdotesClient
}

// RequirementDataSourceModel describes the data source data model.
type RequirementDataSourceModel struct {
	RequirementID types.String `tfsdk:"requirement_id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	Category      types.String `tfsdk:"category"`
	Status        types.String `tfsdk:"status"`
	StatusName    types.String `tfsdk:"status_name"`
	IsCustom      types.Bool   `tfsdk:"is_custom"`
}

func (d *RequirementDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_requirement"
}

func (d *RequirementDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to look up an existing requirement by requirement ID.",

		Attributes: map[string]schema.Attribute{
			"requirement_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the requirement to look up.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The name of the requirement.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The description/help text of the requirement.",
			},
			"category": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The category of the requirement.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the requirement.",
			},
			"status_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The human-readable status name of the requirement.",
			},
			"is_custom": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this is a custom requirement.",
			},
		},
	}
}

func (d *RequirementDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RequirementDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RequirementDataSourceModel

	// Read Terraform configuration data into the model
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	requirementID := data.RequirementID.ValueString()

	// Get requirement from API
	requirement, err := d.client.GetRequirement(requirementID)
	if err != nil {
		addClientError(&resp.Diagnostics, fmt.Sprintf("read requirement with ID %s", requirementID), err)
		return
	}

	// Set state
	data.RequirementID = types.StringValue(requirement.RequirementID)
	data.Name = types.StringValue(requirement.RequirementName)
	data.Description = types.StringValue(requirement.RequirementHelp)
	data.Category = types.StringValue(requirement.RequirementCategory)
	data.Status = types.StringValue(requirement.RequirementStatus)
	data.StatusName = types.StringValue(requirement.RequirementStatusName)
	data.IsCustom = types.BoolValue(requirement.RequirementIsCustom)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
