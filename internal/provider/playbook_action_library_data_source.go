// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &PlaybookActionLibraryDataSource{}

func NewPlaybookActionLibraryDataSource() datasource.DataSource {
	return &PlaybookActionLibraryDataSource{}
}

type PlaybookActionLibraryDataSource struct {
	client *client.AnecdotesClient
}

type PlaybookActionLibraryDataSourceModel struct {
	ExcludeComingSoon types.Bool  `tfsdk:"exclude_coming_soon"`
	Actions           types.List  `tfsdk:"actions"`
	TotalCount        types.Int64 `tfsdk:"total_count"`
}

var playbookLibraryActionAttrTypes = map[string]attr.Type{
	"action_type":     types.StringType,
	"action_text":     types.StringType,
	"action_name":     types.StringType,
	"action_category": types.StringType,
	"description":     types.StringType,
	"coming_soon":     types.BoolType,
	"action_fields":   types.ListType{ElemType: types.ObjectType{AttrTypes: playbookActionFieldAttrTypes}},
}

var playbookActionFieldAttrTypes = map[string]attr.Type{
	"field_id":     types.StringType,
	"display_name": types.StringType,
	"type":         types.StringType,
	"is_required":  types.BoolType,
	"description":  types.StringType,
	"values":       types.ListType{ElemType: types.StringType},
}

func (d *PlaybookActionLibraryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbook_action_library"
}

func (d *PlaybookActionLibraryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the actions a playbook step can perform, with optional filtering.",
		MarkdownDescription: `
Lists the actions a playbook step can perform, so an ` + "`action_type`" + ` on
` + "`anecdotes_playbook`" + ` can be looked up rather than guessed.
`,
		Attributes: map[string]schema.Attribute{
			"exclude_coming_soon": schema.BoolAttribute{
				Description: "Omit actions that are announced but not yet available.",
				Optional:    true,
			},
			"total_count": schema.Int64Attribute{
				Description: "Total number of actions matching the filters.",
				Computed:    true,
			},
			"actions": schema.ListNestedAttribute{
				Description: "List of actions matching the filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"action_type": schema.StringAttribute{
							Description: "The value to use as a step's action_type.",
							Computed:    true,
						},
						"action_text": schema.StringAttribute{
							Description: "The human-readable name of the action.",
							Computed:    true,
						},
						"action_name": schema.StringAttribute{
							Description: "The short name of the action.",
							Computed:    true,
						},
						"action_category": schema.StringAttribute{
							Description: "The group of actions this one belongs to.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "What the action does.",
							Computed:    true,
						},
						"coming_soon": schema.BoolAttribute{
							Description: "Whether the action is announced but not yet available.",
							Computed:    true,
						},
						"action_fields": schema.ListNestedAttribute{
							Description: "The fields this action takes. A required one must appear in the payload_configuration of a step performing the action.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"field_id": schema.StringAttribute{
										Description: "The key to use in the step's payload_configuration.",
										Computed:    true,
									},
									"display_name": schema.StringAttribute{
										Description: "The human-readable name of the field.",
										Computed:    true,
									},
									"type": schema.StringAttribute{
										Description: "The kind of value the field holds.",
										Computed:    true,
									},
									"is_required": schema.BoolAttribute{
										Description: "Whether a step performing this action must supply the field.",
										Computed:    true,
									},
									"description": schema.StringAttribute{
										Description: "What the field holds.",
										Computed:    true,
									},
									"values": schema.ListAttribute{
										Description: "The values the field accepts, when it is a closed set.",
										Computed:    true,
										ElementType: types.StringType,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *PlaybookActionLibraryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PlaybookActionLibraryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PlaybookActionLibraryDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	actions, err := d.client.ListPlaybookActionLibrary(ctx)
	if err != nil {
		addClientError(&resp.Diagnostics, "read playbook actions library", err)
		return
	}

	filtered := make([]client.PlaybookLibraryAction, 0, len(actions))
	for _, a := range actions {
		if data.ExcludeComingSoon.ValueBool() && a.ComingSoon {
			continue
		}
		filtered = append(filtered, a)
	}

	items := make([]attr.Value, len(filtered))
	for i, a := range filtered {
		fields := make([]attr.Value, len(a.ActionFields))
		for j, f := range a.ActionFields {
			values, d := types.ListValueFrom(ctx, types.StringType, f.Values)
			resp.Diagnostics.Append(d...)
			fieldObj, d := types.ObjectValue(playbookActionFieldAttrTypes, map[string]attr.Value{
				"field_id":     types.StringValue(f.FieldID),
				"display_name": types.StringValue(f.DisplayName),
				"type":         types.StringValue(f.Type),
				"is_required":  types.BoolValue(f.IsRequired),
				"description":  types.StringValue(f.FieldDescription),
				"values":       values,
			})
			resp.Diagnostics.Append(d...)
			fields[j] = fieldObj
		}
		actionFields, d := types.ListValue(types.ObjectType{AttrTypes: playbookActionFieldAttrTypes}, fields)
		resp.Diagnostics.Append(d...)

		obj, diags := types.ObjectValue(playbookLibraryActionAttrTypes, map[string]attr.Value{
			"action_type":     types.StringValue(a.ActionType),
			"action_text":     types.StringValue(a.ActionText),
			"action_name":     types.StringValue(a.ActionName),
			"action_category": types.StringValue(a.ActionCategory),
			"description":     types.StringValue(a.Description),
			"coming_soon":     types.BoolValue(a.ComingSoon),
			"action_fields":   actionFields,
		})
		resp.Diagnostics.Append(diags...)
		items[i] = obj
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: playbookLibraryActionAttrTypes}, items)
	resp.Diagnostics.Append(diags...)

	data.Actions = list
	data.TotalCount = types.Int64Value(int64(len(filtered)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
