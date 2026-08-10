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

var _ datasource.DataSource = &RequirementsDataSource{}

func NewRequirementsDataSource() datasource.DataSource {
	return &RequirementsDataSource{}
}

type RequirementsDataSource struct {
	client *client.AnecdotesClient
}

type RequirementsDataSourceModel struct {
	NameContains    types.String `tfsdk:"name_contains"`
	Category        types.String `tfsdk:"category"`
	Status          types.String `tfsdk:"status"`
	IsCustom        types.Bool   `tfsdk:"is_custom"`
	IncludeUnlinked types.Bool   `tfsdk:"include_unlinked"`
	Requirements    types.List   `tfsdk:"requirements"`
	TotalCount      types.Int64  `tfsdk:"total_count"`
}

func (d *RequirementsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_requirements"
}

var requirementItemAttrTypes = map[string]attr.Type{
	"requirement_id": types.StringType,
	"name":           types.StringType,
	"description":    types.StringType,
	"category":       types.StringType,
	"status":         types.StringType,
	"status_name":    types.StringType,
	"is_custom":      types.BoolType,
}

func (d *RequirementsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all requirements in the Requirements Hub, with optional filtering.",
		MarkdownDescription: `
Lists all requirements in the Anecdotes Requirements Hub, with optional filtering.

> **Note:** To read requirements created in the same configuration, add
> ` + "`depends_on`" + ` to this data source so Terraform reads it only after those
> requirements exist.
`,
		Attributes: map[string]schema.Attribute{
			"name_contains": schema.StringAttribute{
				Description: "Filter requirements whose name contains this substring (case-insensitive).",
				Optional:    true,
			},
			"category": schema.StringAttribute{
				Description: "Filter requirements by category (exact match).",
				Optional:    true,
			},
			"status": schema.StringAttribute{
				Description: "Filter requirements by status. Statuses are tenant-defined (up to 17 per tenant); the defaults are \"To do\", \"In Progress\", and \"Completed\". No fixed value set is enforced.",
				Optional:    true,
			},
			"is_custom": schema.BoolAttribute{
				Description: "Filter by custom requirements only (true) or built-in only (false).",
				Optional:    true,
			},
			"include_unlinked": schema.BoolAttribute{
				Description: "Include requirements with no linked controls. Defaults to false — only requirements linked to at least one control are returned.",
				Optional:    true,
			},
			"total_count": schema.Int64Attribute{
				Description: "Total number of requirements matching the filters.",
				Computed:    true,
			},
			"requirements": schema.ListNestedAttribute{
				Description: "List of requirements matching the filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"requirement_id": schema.StringAttribute{
							Description: "The unique identifier of the requirement.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the requirement.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the requirement.",
							Computed:    true,
						},
						"category": schema.StringAttribute{
							Description: "The requirement category.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The requirement status.",
							Computed:    true,
						},
						"status_name": schema.StringAttribute{
							Description: "The human-readable status name.",
							Computed:    true,
						},
						"is_custom": schema.BoolAttribute{
							Description: "Whether this is a custom requirement.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *RequirementsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RequirementsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RequirementsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	allRequirements, err := d.client.ListRequirements(ctx)
	if err != nil {
		addClientError(&resp.Diagnostics, "list requirements", err)
		return
	}

	var filtered []client.Requirement
	for _, r := range allRequirements {
		if !data.NameContains.IsNull() && !data.NameContains.IsUnknown() {
			if !strings.Contains(strings.ToLower(r.RequirementName), strings.ToLower(data.NameContains.ValueString())) {
				continue
			}
		}

		if !data.Category.IsNull() && !data.Category.IsUnknown() {
			if r.RequirementCategory != data.Category.ValueString() {
				continue
			}
		}

		if !data.Status.IsNull() && !data.Status.IsUnknown() {
			if r.RequirementStatus != data.Status.ValueString() {
				continue
			}
		}

		if !data.IsCustom.IsNull() && !data.IsCustom.IsUnknown() {
			if r.RequirementIsCustom != data.IsCustom.ValueBool() {
				continue
			}
		}

		// Filter out unlinked requirements by default
		if len(r.RequirementRelatedControls) == 0 {
			if data.IncludeUnlinked.IsNull() || !data.IncludeUnlinked.ValueBool() {
				continue
			}
		}

		filtered = append(filtered, r)
	}

	items := make([]attr.Value, len(filtered))
	for i, r := range filtered {
		obj, diags := types.ObjectValue(requirementItemAttrTypes, map[string]attr.Value{
			"requirement_id": types.StringValue(r.RequirementID),
			"name":           types.StringValue(r.RequirementName),
			"description":    types.StringValue(r.RequirementDescription),
			"category":       types.StringValue(r.RequirementCategory),
			"status":         types.StringValue(r.RequirementStatus),
			"status_name":    types.StringValue(r.RequirementStatusName),
			"is_custom":      types.BoolValue(r.RequirementIsCustom),
		})
		resp.Diagnostics.Append(diags...)
		items[i] = obj
	}

	objType := types.ObjectType{AttrTypes: requirementItemAttrTypes}
	list, diags := types.ListValue(objType, items)
	resp.Diagnostics.Append(diags...)

	data.Requirements = list
	data.TotalCount = types.Int64Value(int64(len(filtered)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
