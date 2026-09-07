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
		obj, diags := types.ObjectValue(playbookLibraryActionAttrTypes, map[string]attr.Value{
			"action_type":     types.StringValue(a.ActionType),
			"action_text":     types.StringValue(a.ActionText),
			"action_name":     types.StringValue(a.ActionName),
			"action_category": types.StringValue(a.ActionCategory),
			"description":     types.StringValue(a.Description),
			"coming_soon":     types.BoolValue(a.ComingSoon),
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
