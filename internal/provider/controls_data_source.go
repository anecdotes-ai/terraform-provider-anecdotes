// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ControlsDataSource{}

func NewControlsDataSource() datasource.DataSource {
	return &ControlsDataSource{}
}

type ControlsDataSource struct {
	client *client.AnecdotesClient
}

type ControlsDataSourceModel struct {
	FrameworkID  types.String `tfsdk:"framework_id"`
	CategoryID   types.String `tfsdk:"category_id"`
	NameContains types.String `tfsdk:"name_contains"`
	Status       types.String `tfsdk:"status"`
	Controls     types.List   `tfsdk:"controls"`
	TotalCount   types.Int64  `tfsdk:"total_count"`
}

func (d *ControlsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_controls"
}

var controlItemAttrTypes = map[string]attr.Type{
	"control_id":   types.StringType,
	"framework_id": types.StringType,
	"name":         types.StringType,
	"description":  types.StringType,
	"category":     types.StringType,
	"category_id":  types.StringType,
	"status":       types.StringType,
	"owners":       types.ListType{ElemType: types.StringType},
	"tags":         types.ListType{ElemType: types.StringType},
}

func (d *ControlsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all controls for a given framework, with optional filtering.",
		MarkdownDescription: `
Lists all controls for a given framework, with optional filtering by category, name, or status.

> **Note:** To read controls created in the same configuration, add ` + "`depends_on`" + `
> to this data source so Terraform reads it only after those controls exist.
`,
		Attributes: map[string]schema.Attribute{
			"framework_id": schema.StringAttribute{
				Description: "The framework ID to list controls for. Required.",
				Required:    true,
			},
			"category_id": schema.StringAttribute{
				Description: "Filter controls by category ID.",
				Optional:    true,
			},
			"name_contains": schema.StringAttribute{
				Description: "Filter controls whose name contains this substring (case-insensitive).",
				Optional:    true,
			},
			"status": schema.StringAttribute{
				Description: "Filter controls by status. Valid values: \"NOT_STARTED\", \"IN_PROGRESS\", \"READY_FOR_AUDIT\", \"GAP\", \"ISSUE\", \"APPROVED_BY_AUDITOR\", \"MONITORING\", \"NOT_APPLICABLE\", \"NOT_READY_FOR_AUDIT\", \"INSUFFICIENT_DATA\", \"UNDER_REVIEW\".",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.ValidControlStatuses()...),
				},
			},
			"total_count": schema.Int64Attribute{
				Description: "Total number of controls matching the filters.",
				Computed:    true,
			},
			"controls": schema.ListNestedAttribute{
				Description: "List of controls matching the filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"control_id": schema.StringAttribute{
							Description: "The unique identifier of the control.",
							Computed:    true,
						},
						"framework_id": schema.StringAttribute{
							Description: "The framework this control belongs to.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the control.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the control.",
							Computed:    true,
						},
						"category": schema.StringAttribute{
							Description: "The control category name.",
							Computed:    true,
						},
						"category_id": schema.StringAttribute{
							Description: "The control category ID.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The control status.",
							Computed:    true,
						},
						"owners": schema.ListAttribute{
							Description: "List of control owners.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"tags": schema.ListAttribute{
							Description: "List of tags applied to this control.",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *ControlsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ControlsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ControlsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	frameworkID := data.FrameworkID.ValueString()

	allControls, err := d.client.ListControls(frameworkID)
	if err != nil {
		addClientError(&resp.Diagnostics, "list controls", err)
		return
	}

	var filtered []client.Control
	for _, c := range allControls {
		if !data.CategoryID.IsNull() && !data.CategoryID.IsUnknown() {
			if c.ControlFrameworkCategoryID != data.CategoryID.ValueString() {
				continue
			}
		}

		if !data.NameContains.IsNull() && !data.NameContains.IsUnknown() {
			if !strings.Contains(strings.ToLower(c.ControlName), strings.ToLower(data.NameContains.ValueString())) {
				continue
			}
		}

		if !data.Status.IsNull() && !data.Status.IsUnknown() {
			if c.ControlStatus.Status != data.Status.ValueString() {
				continue
			}
		}

		filtered = append(filtered, c)
	}

	items := make([]attr.Value, len(filtered))
	for i, c := range filtered {
		var ownersList types.List
		if len(c.ControlOwners) > 0 {
			ownersList, _ = types.ListValueFrom(ctx, types.StringType, c.ControlOwners)
		} else {
			ownersList = types.ListValueMust(types.StringType, []attr.Value{})
		}

		var tagsList types.List
		if len(c.ControlTags) > 0 {
			tagsList, _ = types.ListValueFrom(ctx, types.StringType, c.ControlTags)
		} else {
			tagsList = types.ListValueMust(types.StringType, []attr.Value{})
		}

		obj, diags := types.ObjectValue(controlItemAttrTypes, map[string]attr.Value{
			"control_id":   types.StringValue(c.ControlID),
			"framework_id": types.StringValue(c.FrameworkID),
			"name":         types.StringValue(c.ControlName),
			"description":  types.StringValue(c.ControlDescription),
			"category":     types.StringValue(c.ControlFrameworkCategory),
			"category_id":  types.StringValue(c.ControlFrameworkCategoryID),
			"status":       types.StringValue(c.ControlStatus.Status),
			"owners":       ownersList,
			"tags":         tagsList,
		})
		resp.Diagnostics.Append(diags...)
		items[i] = obj
	}

	objType := types.ObjectType{AttrTypes: controlItemAttrTypes}
	list, diags := types.ListValue(objType, items)
	resp.Diagnostics.Append(diags...)

	data.Controls = list
	data.TotalCount = types.Int64Value(int64(len(filtered)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
