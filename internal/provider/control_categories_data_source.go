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

var _ datasource.DataSource = &ControlCategoriesDataSource{}

func NewControlCategoriesDataSource() datasource.DataSource {
	return &ControlCategoriesDataSource{}
}

type ControlCategoriesDataSource struct {
	client *client.AnecdotesClient
}

type ControlCategoriesDataSourceModel struct {
	FrameworkID  types.String `tfsdk:"framework_id"`
	NameContains types.String `tfsdk:"name_contains"`
	Categories   types.List   `tfsdk:"categories"`
	TotalCount   types.Int64  `tfsdk:"total_count"`
}

func (d *ControlCategoriesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_control_categories"
}

var controlCategoryItemAttrTypes = map[string]attr.Type{
	"category_id":   types.StringType,
	"category_name": types.StringType,
	"framework_id":  types.StringType,
}

func (d *ControlCategoriesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all control categories, with optional filtering by framework.",
		MarkdownDescription: `
Lists all control categories in the Anecdotes account, with optional filtering by framework or name.
`,
		Attributes: map[string]schema.Attribute{
			"framework_id": schema.StringAttribute{
				Description: "Filter categories by framework ID.",
				Optional:    true,
			},
			"name_contains": schema.StringAttribute{
				Description: "Filter categories whose name contains this substring (case-insensitive).",
				Optional:    true,
			},
			"total_count": schema.Int64Attribute{
				Description: "Total number of control categories matching the filters.",
				Computed:    true,
			},
			"categories": schema.ListNestedAttribute{
				Description: "List of control categories.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"category_id": schema.StringAttribute{
							Description: "The unique identifier of the category.",
							Computed:    true,
						},
						"category_name": schema.StringAttribute{
							Description: "The name of the category.",
							Computed:    true,
						},
						"framework_id": schema.StringAttribute{
							Description: "The framework this category belongs to.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ControlCategoriesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ControlCategoriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ControlCategoriesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	allCategories, err := d.client.ListControlCategories()
	if err != nil {
		addClientError(&resp.Diagnostics, "list control categories", err)
		return
	}

	var filtered []client.ControlCategory
	for _, cat := range allCategories {
		if !data.FrameworkID.IsNull() && !data.FrameworkID.IsUnknown() {
			if cat.FrameworkID != data.FrameworkID.ValueString() {
				continue
			}
		}

		if !data.NameContains.IsNull() && !data.NameContains.IsUnknown() {
			if !strings.Contains(strings.ToLower(cat.CategoryName), strings.ToLower(data.NameContains.ValueString())) {
				continue
			}
		}

		filtered = append(filtered, cat)
	}

	items := make([]attr.Value, len(filtered))
	for i, cat := range filtered {
		obj, diags := types.ObjectValue(controlCategoryItemAttrTypes, map[string]attr.Value{
			"category_id":   types.StringValue(cat.CategoryID),
			"category_name": types.StringValue(cat.CategoryName),
			"framework_id":  types.StringValue(cat.FrameworkID),
		})
		resp.Diagnostics.Append(diags...)
		items[i] = obj
	}

	objType := types.ObjectType{AttrTypes: controlCategoryItemAttrTypes}
	list, diags := types.ListValue(objType, items)
	resp.Diagnostics.Append(diags...)

	data.Categories = list
	data.TotalCount = types.Int64Value(int64(len(filtered)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
