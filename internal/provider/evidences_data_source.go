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

var _ datasource.DataSource = &EvidencesDataSource{}

func NewEvidencesDataSource() datasource.DataSource {
	return &EvidencesDataSource{}
}

type EvidencesDataSource struct {
	client *client.AnecdotesClient
}

type EvidencesDataSourceModel struct {
	ServiceID    types.String `tfsdk:"service_id"`
	NameContains types.String `tfsdk:"name_contains"`
	EvidenceType types.String `tfsdk:"evidence_type"`
	IsCustom     types.Bool   `tfsdk:"is_custom"`
	IsApplicable types.Bool   `tfsdk:"is_applicable"`
	IncludeViews types.Bool   `tfsdk:"include_views"`
	Evidences    types.List   `tfsdk:"evidences"`
	TotalCount   types.Int64  `tfsdk:"total_count"`
}

func (d *EvidencesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_evidences"
}

var evidenceItemAttrTypes = map[string]attr.Type{
	"evidence_id":          types.StringType,
	"evidence_instance_id": types.StringType,
	"name":                 types.StringType,
	"display_name":         types.StringType,
	"evidence_type":        types.StringType,
	"service_id":           types.StringType,
	"service_display_name": types.StringType,
	"is_applicable":        types.BoolType,
	"is_custom":            types.BoolType,
	"alert_level":          types.Int64Type,
	"items_count":          types.Int64Type,
	"processing_state":     types.StringType,
	"url":                  types.StringType,
	"creation_time":        types.StringType,
	"collection_timestamp": types.StringType,
	"entity_type":          types.StringType,
	"is_uar":               types.BoolType,
	"parent_id":            types.StringType,
}

func (d *EvidencesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all evidences in the Anecdotes account, with optional filtering.",
		MarkdownDescription: `
Lists all evidences in the Anecdotes account, with optional filtering by service, name, type, or applicability.
`,
		Attributes: map[string]schema.Attribute{
			"service_id": schema.StringAttribute{
				Description: "Filter evidences by service/plugin ID (e.g., \"aws_guard_duty\", \"github\").",
				Optional:    true,
			},
			"name_contains": schema.StringAttribute{
				Description: "Filter evidences whose name contains this substring (case-insensitive).",
				Optional:    true,
			},
			"evidence_type": schema.StringAttribute{
				Description: "Filter evidences by type. Valid values (matched case-insensitively): \"MANUAL\", \"URL\", \"LINK\", \"LIST\", \"TICKET\", \"BUILDER\", \"API\", \"MERGED\".",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(client.ValidEvidenceTypes()...),
				},
			},
			"is_custom": schema.BoolAttribute{
				Description: "Filter evidences by custom (user-created) status.",
				Optional:    true,
			},
			"is_applicable": schema.BoolAttribute{
				Description: "Filter evidences by applicability status.",
				Optional:    true,
			},
			"include_views": schema.BoolAttribute{
				Description: "Include evidence views (filtered subsets of parent evidence) in the results. By default, only parent evidences are returned. Set to true to also include views.",
				Optional:    true,
			},
			"total_count": schema.Int64Attribute{
				Description: "Total number of evidences matching the filters.",
				Computed:    true,
			},
			"evidences": schema.ListNestedAttribute{
				Description: "List of evidences matching the filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"evidence_id": schema.StringAttribute{
							Description: "The unique identifier of the evidence.",
							Computed:    true,
						},
						"evidence_instance_id": schema.StringAttribute{
							Description: "The instance identifier of the evidence.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The internal name of the evidence.",
							Computed:    true,
						},
						"display_name": schema.StringAttribute{
							Description: "The display name of the evidence.",
							Computed:    true,
						},
						"evidence_type": schema.StringAttribute{
							Description: "The type of evidence (MANUAL, URL, LINK, LIST, TICKET, BUILDER, API, MERGED).",
							Computed:    true,
						},
						"service_id": schema.StringAttribute{
							Description: "The service/plugin ID that provides this evidence.",
							Computed:    true,
						},
						"service_display_name": schema.StringAttribute{
							Description: "The display name of the service/plugin.",
							Computed:    true,
						},
						"is_applicable": schema.BoolAttribute{
							Description: "Whether the evidence is applicable.",
							Computed:    true,
						},
						"is_custom": schema.BoolAttribute{
							Description: "Whether the evidence is custom (user-created).",
							Computed:    true,
						},
						"alert_level": schema.Int64Attribute{
							Description: "The alert level of the evidence (0-5).",
							Computed:    true,
						},
						"items_count": schema.Int64Attribute{
							Description: "Number of items/rows in the evidence.",
							Computed:    true,
						},
						"processing_state": schema.StringAttribute{
							Description: "The processing state of the evidence (e.g., \"success_data_exists\").",
							Computed:    true,
						},
						"url": schema.StringAttribute{
							Description: "The URL of the evidence, if applicable.",
							Computed:    true,
						},
						"creation_time": schema.StringAttribute{
							Description: "When the evidence was created.",
							Computed:    true,
						},
						"collection_timestamp": schema.StringAttribute{
							Description: "When the evidence was last collected.",
							Computed:    true,
						},
						"entity_type": schema.StringAttribute{
							Description: "The entity type of the evidence.",
							Computed:    true,
						},
						"is_uar": schema.BoolAttribute{
							Description: "Whether the evidence is a user access review.",
							Computed:    true,
						},
						"parent_id": schema.StringAttribute{
							Description: "The parent evidence ID.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *EvidencesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *EvidencesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EvidencesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	allEvidences, err := d.client.ListEvidences(ctx)
	if err != nil {
		addClientError(&resp.Diagnostics, "list evidences", err)
		return
	}

	var filtered []client.Evidence
	for _, e := range allEvidences {
		// By default, exclude views (evidence_id != evidence_parent_id).
		// Views are filtered subsets of a parent evidence, not standalone items.
		if data.IncludeViews.IsNull() || data.IncludeViews.IsUnknown() || !data.IncludeViews.ValueBool() {
			if e.EvidenceID != e.EvidenceParentID {
				continue
			}
		}

		if !data.ServiceID.IsNull() && !data.ServiceID.IsUnknown() {
			if e.EvidenceServiceID != data.ServiceID.ValueString() {
				continue
			}
		}

		if !data.NameContains.IsNull() && !data.NameContains.IsUnknown() {
			name := e.EvidenceName
			if e.EvidenceDisplayName != nil {
				name = *e.EvidenceDisplayName
			}
			if !strings.Contains(strings.ToLower(name), strings.ToLower(data.NameContains.ValueString())) {
				continue
			}
		}

		if !data.EvidenceType.IsNull() && !data.EvidenceType.IsUnknown() {
			if !strings.EqualFold(e.EvidenceType, data.EvidenceType.ValueString()) {
				continue
			}
		}

		if !data.IsCustom.IsNull() && !data.IsCustom.IsUnknown() {
			if e.EvidenceIsCustom != data.IsCustom.ValueBool() {
				continue
			}
		}

		if !data.IsApplicable.IsNull() && !data.IsApplicable.IsUnknown() {
			if e.EvidenceIsApplicable != data.IsApplicable.ValueBool() {
				continue
			}
		}

		filtered = append(filtered, e)
	}

	evidenceItems := make([]attr.Value, len(filtered))
	for i, e := range filtered {
		displayName := e.EvidenceName
		if e.EvidenceDisplayName != nil {
			displayName = *e.EvidenceDisplayName
		}

		processingState := ""
		if e.EvidenceProcessingState != nil {
			processingState = *e.EvidenceProcessingState
		}

		url := ""
		if e.EvidenceURL != nil {
			url = *e.EvidenceURL
		}

		creationTime := ""
		if e.EvidenceCreationTime != nil {
			creationTime = *e.EvidenceCreationTime
		}

		obj, diags := types.ObjectValue(evidenceItemAttrTypes, map[string]attr.Value{
			"evidence_id":          types.StringValue(e.EvidenceID),
			"evidence_instance_id": types.StringValue(e.EvidenceInstanceID),
			"name":                 types.StringValue(e.EvidenceName),
			"display_name":         types.StringValue(displayName),
			"evidence_type":        types.StringValue(e.EvidenceType),
			"service_id":           types.StringValue(e.EvidenceServiceID),
			"service_display_name": types.StringValue(e.EvidenceServiceDisplayName),
			"is_applicable":        types.BoolValue(e.EvidenceIsApplicable),
			"is_custom":            types.BoolValue(e.EvidenceIsCustom),
			"alert_level":          types.Int64Value(int64(e.EvidenceAlertLevel)),
			"items_count":          types.Int64Value(int64(e.EvidenceItemsCount)),
			"processing_state":     types.StringValue(processingState),
			"url":                  types.StringValue(url),
			"creation_time":        types.StringValue(creationTime),
			"collection_timestamp": types.StringValue(e.EvidenceCollectionTimestamp),
			"entity_type":          types.StringValue(e.EvidenceEntityType),
			"is_uar":               types.BoolValue(e.EvidenceIsUAR),
			"parent_id":            types.StringValue(e.EvidenceParentID),
		})
		resp.Diagnostics.Append(diags...)
		evidenceItems[i] = obj
	}

	evidencesObjType := types.ObjectType{AttrTypes: evidenceItemAttrTypes}
	evidencesList, diags := types.ListValue(evidencesObjType, evidenceItems)
	resp.Diagnostics.Append(diags...)

	data.Evidences = evidencesList
	data.TotalCount = types.Int64Value(int64(len(filtered)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
