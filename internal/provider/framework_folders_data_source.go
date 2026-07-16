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

var _ datasource.DataSource = &FrameworkFoldersDataSource{}

func NewFrameworkFoldersDataSource() datasource.DataSource {
	return &FrameworkFoldersDataSource{}
}

type FrameworkFoldersDataSource struct {
	client *client.AnecdotesClient
}

type FrameworkFoldersDataSourceModel struct {
	NameContains types.String `tfsdk:"name_contains"`
	Folders      types.List   `tfsdk:"folders"`
	TotalCount   types.Int64  `tfsdk:"total_count"`
}

func (d *FrameworkFoldersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_framework_folders"
}

var folderItemAttrTypes = map[string]attr.Type{
	"folder_id":       types.StringType,
	"name":            types.StringType,
	"frameworks_list": types.ListType{ElemType: types.StringType},
}

func (d *FrameworkFoldersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all framework folders in the Anecdotes account.",
		MarkdownDescription: `
Lists all framework folders in the Anecdotes account, with optional name filtering.
`,
		Attributes: map[string]schema.Attribute{
			"name_contains": schema.StringAttribute{
				Description: "Filter folders whose name contains this substring (case-insensitive).",
				Optional:    true,
			},
			"total_count": schema.Int64Attribute{
				Description: "Total number of folders matching the filters.",
				Computed:    true,
			},
			"folders": schema.ListNestedAttribute{
				Description: "List of framework folders.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"folder_id": schema.StringAttribute{
							Description: "The unique identifier of the folder.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the folder.",
							Computed:    true,
						},
						"frameworks_list": schema.ListAttribute{
							Description: "List of framework IDs in this folder.",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *FrameworkFoldersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *FrameworkFoldersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FrameworkFoldersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	allFolders, err := d.client.ListFolders()
	if err != nil {
		addClientError(&resp.Diagnostics, "list folders", err)
		return
	}

	var filtered []client.Folder
	for _, f := range allFolders {
		if !data.NameContains.IsNull() && !data.NameContains.IsUnknown() {
			if !strings.Contains(strings.ToLower(f.Name), strings.ToLower(data.NameContains.ValueString())) {
				continue
			}
		}
		filtered = append(filtered, f)
	}

	items := make([]attr.Value, len(filtered))
	for i, f := range filtered {
		var frameworksList types.List
		if len(f.FrameworksList) > 0 {
			frameworksList, _ = types.ListValueFrom(ctx, types.StringType, f.FrameworksList)
		} else {
			frameworksList = types.ListValueMust(types.StringType, []attr.Value{})
		}

		obj, diags := types.ObjectValue(folderItemAttrTypes, map[string]attr.Value{
			"folder_id":       types.StringValue(f.ID),
			"name":            types.StringValue(f.Name),
			"frameworks_list": frameworksList,
		})
		resp.Diagnostics.Append(diags...)
		items[i] = obj
	}

	objType := types.ObjectType{AttrTypes: folderItemAttrTypes}
	list, diags := types.ListValue(objType, items)
	resp.Diagnostics.Append(diags...)

	data.Folders = list
	data.TotalCount = types.Int64Value(int64(len(filtered)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
