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

var _ datasource.DataSource = &PlaybookLibraryDataSource{}

func NewPlaybookLibraryDataSource() datasource.DataSource {
	return &PlaybookLibraryDataSource{}
}

type PlaybookLibraryDataSource struct {
	client *client.AnecdotesClient
}

type PlaybookLibraryDataSourceModel struct {
	Category      types.String `tfsdk:"category"`
	AvailableOnly types.Bool   `tfsdk:"available_only"`
	Events        types.List   `tfsdk:"events"`
	TotalCount    types.Int64  `tfsdk:"total_count"`
}

var playbookLibraryEventAttrTypes = map[string]attr.Type{
	"event_type":             types.StringType,
	"trigger_key":            types.StringType,
	"trigger_value":          types.StringType,
	"event_text":             types.StringType,
	"category":               types.StringType,
	"description":            types.StringType,
	"is_available":           types.BoolType,
	"supported_actions":      types.ListType{ElemType: types.StringType},
	"coming_soon_actions":    types.ListType{ElemType: types.StringType},
	"required_changed_field": types.StringType,
}

func (d *PlaybookLibraryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbook_library"
}

func (d *PlaybookLibraryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the trigger events a playbook step can subscribe to, with optional filtering.",
		MarkdownDescription: `
Lists the trigger events a playbook step can subscribe to, so a ` + "`trigger_event`" + ` on
` + "`anecdotes_playbook`" + ` can be looked up rather than guessed.

Use ` + "`trigger_value`" + ` as the ` + "`trigger_event`" + ` of a step: it is the event's
` + "`trigger_key`" + ` when the event declares one, and its ` + "`event_type`" + ` otherwise.
`,
		Attributes: map[string]schema.Attribute{
			"category": schema.StringAttribute{
				Description: "Filter events by category (case-insensitive), for example \"control\" or \"risk\".",
				Optional:    true,
			},
			"available_only": schema.BoolAttribute{
				Description: "Return only the events available to this account.",
				Optional:    true,
			},
			"total_count": schema.Int64Attribute{
				Description: "Total number of trigger events matching the filters.",
				Computed:    true,
			},
			"events": schema.ListNestedAttribute{
				Description: "List of trigger events matching the filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"event_type": schema.StringAttribute{
							Description: "The platform event that fires the step.",
							Computed:    true,
						},
						"trigger_key": schema.StringAttribute{
							Description: "The qualified form of the event when it distinguishes a changed field, for example FindingUpdated:severity. Empty when the event does not declare one.",
							Computed:    true,
						},
						"trigger_value": schema.StringAttribute{
							Description: "The value to use as a step's trigger_event: trigger_key when set, event_type otherwise.",
							Computed:    true,
						},
						"event_text": schema.StringAttribute{
							Description: "The human-readable name of the event.",
							Computed:    true,
						},
						"category": schema.StringAttribute{
							Description: "The part of the platform the event belongs to.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "What the event describes.",
							Computed:    true,
						},
						"is_available": schema.BoolAttribute{
							Description: "Whether the event is available to this account.",
							Computed:    true,
						},
						"supported_actions": schema.ListAttribute{
							Description: "The action types a step subscribing to this event can perform.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"coming_soon_actions": schema.ListAttribute{
							Description: "The action types announced for this event but not yet available.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"required_changed_field": schema.StringAttribute{
							Description: "The field whose change the event reports, when it reports one.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *PlaybookLibraryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PlaybookLibraryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PlaybookLibraryDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	events, err := d.client.ListPlaybookLibrary(ctx)
	if err != nil {
		addClientError(&resp.Diagnostics, "read playbook library", err)
		return
	}

	filtered := make([]client.PlaybookLibraryEvent, 0, len(events))
	for _, e := range events {
		if !data.Category.IsNull() && !data.Category.IsUnknown() {
			if !strings.EqualFold(e.Category, data.Category.ValueString()) {
				continue
			}
		}
		if data.AvailableOnly.ValueBool() && !e.IsAvailable {
			continue
		}
		filtered = append(filtered, e)
	}

	items := make([]attr.Value, len(filtered))
	for i, e := range filtered {
		supported, diags := types.ListValueFrom(ctx, types.StringType, e.SupportedActions)
		resp.Diagnostics.Append(diags...)
		comingSoon, diags := types.ListValueFrom(ctx, types.StringType, e.ComingSoonActions)
		resp.Diagnostics.Append(diags...)

		triggerValue := e.TriggerKey
		if triggerValue == "" {
			triggerValue = e.EventType
		}

		obj, diags := types.ObjectValue(playbookLibraryEventAttrTypes, map[string]attr.Value{
			"event_type":             types.StringValue(e.EventType),
			"trigger_key":            types.StringValue(e.TriggerKey),
			"trigger_value":          types.StringValue(triggerValue),
			"event_text":             types.StringValue(e.EventText),
			"category":               types.StringValue(e.Category),
			"description":            types.StringValue(e.Description),
			"is_available":           types.BoolValue(e.IsAvailable),
			"supported_actions":      supported,
			"coming_soon_actions":    comingSoon,
			"required_changed_field": types.StringValue(e.RequiredChangedField),
		})
		resp.Diagnostics.Append(diags...)
		items[i] = obj
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: playbookLibraryEventAttrTypes}, items)
	resp.Diagnostics.Append(diags...)

	data.Events = list
	data.TotalCount = types.Int64Value(int64(len(filtered)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
