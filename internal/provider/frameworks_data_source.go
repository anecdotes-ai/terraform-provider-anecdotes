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

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &FrameworksDataSource{}

func NewFrameworksDataSource() datasource.DataSource {
	return &FrameworksDataSource{}
}

// FrameworksDataSource defines the data source implementation.
type FrameworksDataSource struct {
	client *client.AnecdotesClient
}

// FrameworksDataSourceModel describes the data source data model.
type FrameworksDataSourceModel struct {
	IsApplicable types.Bool   `tfsdk:"is_applicable"`
	NameContains types.String `tfsdk:"name_contains"`
	Frameworks   types.List   `tfsdk:"frameworks"`
	TotalCount   types.Int64  `tfsdk:"total_count"`
}

func (d *FrameworksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_frameworks"
}

// frameworkItemAttrTypes defines the attribute types for a single framework in the list.
var frameworkItemAttrTypes = map[string]attr.Type{
	"framework_id":                  types.StringType,
	"name":                          types.StringType,
	"description":                   types.StringType,
	"framework_status":              types.StringType,
	"is_applicable":                 types.BoolType,
	"framework_auditable":           types.BoolType,
	"categories_count":              types.Int64Type,
	"references_count":              types.Int64Type,
	"framework_controls_categories": types.ListType{ElemType: types.StringType},
}

func (d *FrameworksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all frameworks in the Anecdotes account, with optional filtering.",
		MarkdownDescription: `
Lists all frameworks in the Anecdotes account, with optional filtering by adoption status or name.

Use this data source to discover frameworks and their IDs. For full framework details
(auditor configuration, control references, etc.), use the singular ` + "`anecdotes_framework`" + ` data source.
`,
		Attributes: map[string]schema.Attribute{

			// ==================== Filters ====================
			"is_applicable": schema.BoolAttribute{
				Description:         "Filter by adoption status. Defaults to true — only adopted (active) frameworks are returned. Set to false to include unadopted system frameworks.",
				MarkdownDescription: "Filter by adoption status. Defaults to `true` — only adopted (active) frameworks are returned. Set to `false` to include unadopted system frameworks.",
				Optional:            true,
			},

			"name_contains": schema.StringAttribute{
				Description: "Filter frameworks whose name contains this substring (case-insensitive).",
				Optional:    true,
			},

			// ==================== Results ====================
			"total_count": schema.Int64Attribute{
				Description: "Total number of frameworks matching the filters.",
				Computed:    true,
			},

			"frameworks": schema.ListNestedAttribute{
				Description: "List of frameworks matching the filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"framework_id": schema.StringAttribute{
							Description: "The unique identifier of the framework.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The human-readable name of the framework.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "A detailed description of the framework.",
							Computed:    true,
						},
						"framework_status": schema.StringAttribute{
							Description: "The catalog status of the framework (e.g., AVAILABLE, ARCHIVED).",
							Computed:    true,
						},
						"is_applicable": schema.BoolAttribute{
							Description: "Whether this framework is adopted for the organization.",
							Computed:    true,
						},
						"framework_auditable": schema.BoolAttribute{
							Description: "Whether auditors can access this framework.",
							Computed:    true,
						},
						"categories_count": schema.Int64Attribute{
							Description: "Number of control categories in this framework.",
							Computed:    true,
						},
						"references_count": schema.Int64Attribute{
							Description: "Number of control references in this framework.",
							Computed:    true,
						},
						"framework_controls_categories": schema.ListAttribute{
							Description: "List of control category IDs within this framework.",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *FrameworksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *FrameworksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FrameworksDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch all frameworks from API
	allFrameworks, err := d.client.ListFrameworks(ctx)
	if err != nil {
		addClientError(&resp.Diagnostics, "list frameworks", err)
		return
	}

	// Apply filters
	var filtered []client.Framework
	for _, fw := range allFrameworks {
		// Filter by is_applicable — defaults to true (only adopted frameworks)
		if data.IsApplicable.IsNull() || data.IsApplicable.IsUnknown() {
			if !fw.IsApplicable {
				continue
			}
		} else {
			if fw.IsApplicable != data.IsApplicable.ValueBool() {
				continue
			}
		}

		// Filter by name_contains (case-insensitive)
		if !data.NameContains.IsNull() && !data.NameContains.IsUnknown() {
			if !strings.Contains(
				strings.ToLower(fw.FrameworkName),
				strings.ToLower(data.NameContains.ValueString()),
			) {
				continue
			}
		}

		filtered = append(filtered, fw)
	}

	// Build the frameworks list
	frameworkItems := make([]attr.Value, len(filtered))
	for i, fw := range filtered {
		// Build categories list
		var categoriesList types.List
		if len(fw.FrameworkControlsCategories) > 0 {
			categoriesListValue, diags := types.ListValueFrom(ctx, types.StringType, fw.FrameworkControlsCategories)
			categoriesList = categoriesListValue
			resp.Diagnostics.Append(diags...)
		} else {
			categoriesList = types.ListValueMust(types.StringType, []attr.Value{})
		}

		obj, diags := types.ObjectValue(frameworkItemAttrTypes, map[string]attr.Value{
			"framework_id":                  types.StringValue(fw.FrameworkID),
			"name":                          types.StringValue(fw.FrameworkName),
			"description":                   types.StringValue(fw.FrameworkDescription),
			"framework_status":              types.StringValue(string(fw.FrameworkStatus)),
			"is_applicable":                 types.BoolValue(fw.IsApplicable),
			"framework_auditable":           types.BoolValue(fw.FrameworkAuditable),
			"categories_count":              types.Int64Value(int64(len(fw.FrameworkControlsCategories))),
			"references_count":              types.Int64Value(int64(len(fw.FrameworkReferences))),
			"framework_controls_categories": categoriesList,
		})
		resp.Diagnostics.Append(diags...)
		frameworkItems[i] = obj
	}

	frameworksObjType := types.ObjectType{AttrTypes: frameworkItemAttrTypes}
	frameworksList, diags := types.ListValue(frameworksObjType, frameworkItems)
	resp.Diagnostics.Append(diags...)

	data.Frameworks = frameworksList
	data.TotalCount = types.Int64Value(int64(len(filtered)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
