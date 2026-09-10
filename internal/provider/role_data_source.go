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

var _ datasource.DataSource = &RoleDataSource{}

func NewRoleDataSource() datasource.DataSource {
	return &RoleDataSource{}
}

// RoleDataSource defines the data source implementation.
type RoleDataSource struct {
	client *client.AnecdotesClient
}

// RoleDataSourceModel describes the data source data model.
type RoleDataSourceModel struct {
	RoleID               types.String `tfsdk:"role_id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Permissions          types.List   `tfsdk:"permissions"`
	Extends              types.List   `tfsdk:"extends"`
	FullAccessFrameworks types.List   `tfsdk:"full_access_frameworks"`
}

func (d *RoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *RoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing role (built-in or custom) by its `role_id` (key) — useful for referencing a base role's key in `anecdotes_role`'s `extends`, or discovering valid `permissions` values, without hardcoding them.",
		Attributes: map[string]schema.Attribute{
			"role_id": schema.StringAttribute{
				Required:    true,
				Description: "The role's key (e.g. \"viewer_role\", or a custom role's server-generated key).",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the role.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "The description of the role.",
			},
			"permissions": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "The role's live, resolved permission set.",
			},
			"extends": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Base role keys this role inherits permissions from.",
			},
			"full_access_frameworks": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Framework IDs this role has full access to, or null when unscoped.",
			},
		},
	}
}

func (d *RoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.AnecdotesClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.AnecdotesClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *RoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RoleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleID := data.RoleID.ValueString()
	role, err := d.client.GetRole(ctx, roleID)
	if err != nil {
		addClientError(&resp.Diagnostics, fmt.Sprintf("read role with ID %s", roleID), err)
		return
	}

	data.Name = types.StringValue(role.Name)
	data.Description = types.StringValue(role.Description)

	permissions, diags := types.ListValueFrom(ctx, types.StringType, role.Permissions)
	resp.Diagnostics.Append(diags...)
	data.Permissions = permissions

	extends, diags := types.ListValueFrom(ctx, types.StringType, role.Extends)
	resp.Diagnostics.Append(diags...)
	data.Extends = extends

	if len(role.FullAccessFrameworks) > 0 {
		fullAccessFrameworks, diags := types.ListValueFrom(ctx, types.StringType, role.FullAccessFrameworks)
		resp.Diagnostics.Append(diags...)
		data.FullAccessFrameworks = fullAccessFrameworks
	} else {
		data.FullAccessFrameworks = types.ListNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
