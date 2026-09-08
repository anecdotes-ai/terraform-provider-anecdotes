// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RolesDataSource{}

func NewRolesDataSource() datasource.DataSource {
	return &RolesDataSource{}
}

type RolesDataSource struct {
	client *client.AnecdotesClient
}

type RolesDataSourceModel struct {
	NameContains types.String `tfsdk:"name_contains"`
	IsCustom     types.Bool   `tfsdk:"is_custom"`
	Roles        types.List   `tfsdk:"roles"`
	TotalCount   types.Int64  `tfsdk:"total_count"`
}

func (d *RolesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

var roleItemAttrTypes = map[string]attr.Type{
	"role_id":                types.StringType,
	"name":                   types.StringType,
	"description":            types.StringType,
	"permissions":            types.ListType{ElemType: types.StringType},
	"extends":                types.ListType{ElemType: types.StringType},
	"full_access_frameworks": types.ListType{ElemType: types.StringType},
	"is_custom":              types.BoolType,
}

func (d *RolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Lists every role visible to the tenant — built-in global roles plus tenant-specific custom roles.",
		MarkdownDescription: "Lists every role visible to the tenant — built-in global roles plus tenant-specific custom roles. Useful for discovering valid `extends`/`permissions` values for `anecdotes_role` without hardcoding them.",
		Attributes: map[string]schema.Attribute{
			"name_contains": schema.StringAttribute{
				Description: "Filter roles whose name contains this substring (case-insensitive).",
				Optional:    true,
			},
			"is_custom": schema.BoolAttribute{
				Description: "Filter by tenant-scoped custom roles only (true) or built-in global roles only (false).",
				Optional:    true,
			},
			"total_count": schema.Int64Attribute{
				Description: "Total number of roles matching the filters.",
				Computed:    true,
			},
			"roles": schema.ListNestedAttribute{
				Description: "List of roles matching the filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role_id": schema.StringAttribute{
							Description: "The role's key.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the role.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the role.",
							Computed:    true,
						},
						"permissions": schema.ListAttribute{
							Description: "The role's live, resolved permission set.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"extends": schema.ListAttribute{
							Description: "Base role keys this role inherits permissions from.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"full_access_frameworks": schema.ListAttribute{
							Description: "Framework IDs this role has full access to, or null when unscoped.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"is_custom": schema.BoolAttribute{
							Description: "Whether this is a tenant-scoped custom role (as opposed to a built-in global role).",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *RolesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.AnecdotesClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.AnecdotesClient, got: %T.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *RolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RolesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	allRoles, err := d.client.ListRoles(ctx)
	if err != nil {
		addClientError(&resp.Diagnostics, "list roles", err)
		return
	}

	var filtered []client.Role
	for _, r := range allRoles {
		if !data.NameContains.IsNull() && !data.NameContains.IsUnknown() {
			if !strings.Contains(strings.ToLower(r.Name), strings.ToLower(data.NameContains.ValueString())) {
				continue
			}
		}

		if !data.IsCustom.IsNull() && !data.IsCustom.IsUnknown() {
			if isCustomRole(r) != data.IsCustom.ValueBool() {
				continue
			}
		}

		filtered = append(filtered, r)
	}

	items := make([]attr.Value, len(filtered))
	for i, r := range filtered {
		permissions, diags := types.ListValueFrom(ctx, types.StringType, r.Permissions)
		resp.Diagnostics.Append(diags...)

		extends, diags := types.ListValueFrom(ctx, types.StringType, r.Extends)
		resp.Diagnostics.Append(diags...)

		var fullAccessFrameworks types.List
		if len(r.FullAccessFrameworks) > 0 {
			fullAccessFrameworks, diags = types.ListValueFrom(ctx, types.StringType, r.FullAccessFrameworks)
			resp.Diagnostics.Append(diags...)
		} else {
			fullAccessFrameworks = types.ListNull(types.StringType)
		}

		obj, diags := types.ObjectValue(roleItemAttrTypes, map[string]attr.Value{
			"role_id":                types.StringValue(r.Key),
			"name":                   types.StringValue(r.Name),
			"description":            types.StringValue(r.Description),
			"permissions":            permissions,
			"extends":                extends,
			"full_access_frameworks": fullAccessFrameworks,
			"is_custom":              types.BoolValue(isCustomRole(r)),
		})
		resp.Diagnostics.Append(diags...)
		items[i] = obj
	}

	objType := types.ObjectType{AttrTypes: roleItemAttrTypes}
	list, diags := types.ListValue(objType, items)
	resp.Diagnostics.Append(diags...)

	data.Roles = list
	data.TotalCount = types.Int64Value(int64(len(filtered)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
