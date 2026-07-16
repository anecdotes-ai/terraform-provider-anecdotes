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

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &FrameworkDataSource{}

func NewFrameworkDataSource() datasource.DataSource {
	return &FrameworkDataSource{}
}

// FrameworkDataSource defines the data source implementation.
type FrameworkDataSource struct {
	client *client.AnecdotesClient
}

// FrameworkDataSourceModel describes the data source data model.
type FrameworkDataSourceModel struct {
	// Core identification
	FrameworkID types.String `tfsdk:"framework_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`

	// Status
	FrameworkStatus types.String `tfsdk:"framework_status"`
	IsApplicable    types.Bool   `tfsdk:"is_applicable"`

	// Auditor configuration
	FrameworkAuditable                types.Bool   `tfsdk:"framework_auditable"`
	CanAuditorDownloadEvidence        types.Bool   `tfsdk:"can_auditor_download_evidence"`
	CanAuditorViewControlAttachments  types.Bool   `tfsdk:"can_auditor_view_control_attachments"`
	CanAuditorViewControlCustomFields types.Bool   `tfsdk:"can_auditor_view_control_custom_fields"`
	CanAuditorViewSoaReport           types.Bool   `tfsdk:"can_auditor_view_soa_report"`
	CanAuditorViewTags                types.Bool   `tfsdk:"can_auditor_view_tags"`
	FrameworkAuditorControlStatus     types.Object `tfsdk:"framework_auditor_control_status"`
	FrameworkAuditorEvidenceStatus    types.Object `tfsdk:"framework_auditor_evidence_status"`

	// Structure and references
	FrameworkControlsCategories types.List   `tfsdk:"framework_controls_categories"`
	FrameworkReferenceFieldName types.String `tfsdk:"framework_reference_field_name"`
	FrameworkReferences         types.List   `tfsdk:"framework_references"`

	// Framework origin
	FrameworkDuplicatedFrom types.String `tfsdk:"framework_duplicated_from"`

	// Plugin scoping
	FrameworkExcludedPlugins types.Map `tfsdk:"framework_excluded_plugins"`

	// Customization
	FrameworkIconID types.String `tfsdk:"framework_icon_id"`

	// Ordering
	UnadoptedOrder types.Int64 `tfsdk:"unadopted_order"`

	// Detailed data (controls/categories are retrieved separately)
	Controls   types.List `tfsdk:"controls"`
	Categories types.List `tfsdk:"categories"`
}

func (d *FrameworkDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_framework"
}

func (d *FrameworkDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about an existing Anecdotes framework.",
		MarkdownDescription: `
Retrieves information about an existing Anecdotes framework, including its configuration and metadata.

This data source can be used to reference existing frameworks (both custom and out-of-the-box)
for use in other resources or to export framework configuration.
`,
		Attributes: map[string]schema.Attribute{
			// ==================== Core Identification ====================
			"framework_id": schema.StringAttribute{
				Description:         "The unique identifier of the framework to retrieve (e.g., 1234567890).",
				MarkdownDescription: "The unique identifier of the framework to retrieve (e.g., `1234567890`).",
				Required:            true,
			},

			"name": schema.StringAttribute{
				Description: "The human-readable name of the framework (e.g., 'SOC 2', 'ISO-IEC 27001 2022').",
				Computed:    true,
			},

			"description": schema.StringAttribute{
				Description: "A detailed description of the framework and its purpose.",
				Computed:    true,
			},

			// ==================== Status ====================
			"framework_status": schema.StringAttribute{
				Description: "The status of the framework. Values: 'AVAILABLE', 'ARCHIVED'.",
				Computed:    true,
			},

			"is_applicable": schema.BoolAttribute{
				Description: "Whether this framework is adopted/applicable for the organization.",
				Computed:    true,
			},

			// ==================== Auditor Configuration ====================
			"framework_auditable": schema.BoolAttribute{
				Description: "Whether auditors can access this framework.",
				Computed:    true,
			},

			"can_auditor_download_evidence": schema.BoolAttribute{
				Description: "Whether auditors can download evidence files from this framework.",
				Computed:    true,
			},

			"can_auditor_view_control_attachments": schema.BoolAttribute{
				Description: "Whether auditors can view control attachments in this framework.",
				Computed:    true,
			},

			"can_auditor_view_control_custom_fields": schema.BoolAttribute{
				Description: "Whether auditors can view custom fields on controls in this framework.",
				Computed:    true,
			},

			"can_auditor_view_soa_report": schema.BoolAttribute{
				Description: "Whether auditors can view the Statement of Applicability (SOA) report.",
				Computed:    true,
			},

			"can_auditor_view_tags": schema.BoolAttribute{
				Description: "Whether auditors can view tags on controls in this framework.",
				Computed:    true,
			},

			"framework_auditor_control_status": schema.SingleNestedAttribute{
				Description: "Configuration for which control statuses are visible to auditors.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"approved_by_auditor": schema.BoolAttribute{
						Description: "Show 'Approved by auditor' status to auditors.",
						Computed:    true,
					},
					"gap": schema.BoolAttribute{
						Description: "Show 'Gap' status to auditors.",
						Computed:    true,
					},
					"insufficient_data": schema.BoolAttribute{
						Description: "Show 'Insufficient data' status to auditors.",
						Computed:    true,
					},
					"in_progress": schema.BoolAttribute{
						Description: "Show 'In progress' status to auditors.",
						Computed:    true,
					},
					"issue": schema.BoolAttribute{
						Description: "Show 'Issue by auditor' status to auditors.",
						Computed:    true,
					},
					"monitoring": schema.BoolAttribute{
						Description: "Show 'Monitoring' status to auditors.",
						Computed:    true,
					},
					"not_applicable": schema.BoolAttribute{
						Description: "Show 'Not applicable' status to auditors.",
						Computed:    true,
					},
					"not_started": schema.BoolAttribute{
						Description: "Show 'Not started' status to auditors.",
						Computed:    true,
					},
					"ready_for_audit": schema.BoolAttribute{
						Description: "Show 'Ready for audit' status to auditors.",
						Computed:    true,
					},
					"under_review": schema.BoolAttribute{
						Description: "Show 'Under review' status to auditors.",
						Computed:    true,
					},
				},
			},

			"framework_auditor_evidence_status": schema.SingleNestedAttribute{
				Description: "Configuration for which evidence statuses are visible to auditors.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"auditable": schema.BoolAttribute{
						Description: "Show 'Auditable' evidence status to auditors.",
						Computed:    true,
					},
					"gap": schema.BoolAttribute{
						Description: "Show 'Gap' evidence status to auditors.",
						Computed:    true,
					},
					"not_set": schema.BoolAttribute{
						Description: "Show 'Not set' evidence status to auditors.",
						Computed:    true,
					},
				},
			},

			// ==================== Structure and References ====================
			"framework_controls_categories": schema.ListAttribute{
				Description: "List of control category IDs within this framework.",
				Computed:    true,
				ElementType: types.StringType,
			},

			"framework_reference_field_name": schema.StringAttribute{
				Description: "The name of the reference field for controls in this framework.",
				Computed:    true,
			},

			"framework_references": schema.ListAttribute{
				Description: "List of reference values available for controls.",
				Computed:    true,
				ElementType: types.StringType,
			},

			// ==================== Framework Origin ====================
			"framework_duplicated_from": schema.StringAttribute{
				Description: "If this framework was duplicated from another, this is the source framework ID.",
				Computed:    true,
			},

			// ==================== Plugin Scoping ====================
			"framework_excluded_plugins": schema.MapAttribute{
				Description: "Map of plugin IDs to exclude from this framework for evidence scoping.",
				Computed:    true,
				ElementType: types.StringType,
			},

			// ==================== Customization ====================
			"framework_icon_id": schema.StringAttribute{
				Description: "Custom icon ID for this framework.",
				Computed:    true,
			},

			// ==================== Ordering ====================
			"unadopted_order": schema.Int64Attribute{
				Description: "Display order for unadopted frameworks in the framework library.",
				Computed:    true,
			},

			// ==================== Controls and Categories ====================
			"controls": schema.ListNestedAttribute{
				Description: "The controls within the framework (if populated by API).",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"control_id": schema.StringAttribute{
							Description: "The unique identifier of the control.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The human-readable name of the control.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "A detailed description of the control.",
							Computed:    true,
						},
						"category": schema.StringAttribute{
							Description: "The category this control belongs to.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The current status of the control.",
							Computed:    true,
						},
						"owners": schema.ListAttribute{
							Description: "List of email addresses of users who own this control.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"tags": schema.ListAttribute{
							Description: "Tags applied to the control.",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},

			"categories": schema.ListNestedAttribute{
				Description: "The control categories within the framework (if populated by API).",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"category_id": schema.StringAttribute{
							Description: "The unique identifier of the category.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the category.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "A description of the category.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *FrameworkDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *FrameworkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FrameworkDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get framework from API
	framework, err := d.client.GetFramework(data.FrameworkID.ValueString())
	if err != nil {
		addClientError(&resp.Diagnostics, "read framework", err)
		return
	}

	// ==================== Core Identification ====================
	data.Name = types.StringValue(framework.FrameworkName)
	data.Description = types.StringValue(framework.FrameworkDescription)

	// ==================== Status ====================
	data.FrameworkStatus = types.StringValue(string(framework.FrameworkStatus))
	data.IsApplicable = types.BoolValue(framework.IsApplicable)

	// ==================== Auditor Configuration ====================
	data.FrameworkAuditable = types.BoolValue(framework.FrameworkAuditable)
	data.CanAuditorDownloadEvidence = types.BoolValue(framework.CanAuditorDownloadEvidence)
	data.CanAuditorViewControlAttachments = types.BoolValue(framework.CanAuditorViewControlAttachments)
	data.CanAuditorViewControlCustomFields = types.BoolValue(framework.CanAuditorViewControlCustomFields)
	data.CanAuditorViewSoaReport = types.BoolValue(framework.CanAuditorViewSoaReport)
	data.CanAuditorViewTags = types.BoolValue(framework.CanAuditorViewTags)

	// Auditor control status
	controlStatusAttrTypes := map[string]attr.Type{
		"approved_by_auditor": types.BoolType,
		"gap":                 types.BoolType,
		"insufficient_data":   types.BoolType,
		"in_progress":         types.BoolType,
		"issue":               types.BoolType,
		"monitoring":          types.BoolType,
		"not_applicable":      types.BoolType,
		"not_started":         types.BoolType,
		"ready_for_audit":     types.BoolType,
		"under_review":        types.BoolType,
	}
	controlStatusObj, diags := types.ObjectValue(controlStatusAttrTypes, map[string]attr.Value{
		"approved_by_auditor": types.BoolValue(framework.FrameworkAuditorControlStatus.ApprovedByAuditor),
		"gap":                 types.BoolValue(framework.FrameworkAuditorControlStatus.Gap),
		"insufficient_data":   types.BoolValue(framework.FrameworkAuditorControlStatus.InsufficientData),
		"in_progress":         types.BoolValue(framework.FrameworkAuditorControlStatus.InProgress),
		"issue":               types.BoolValue(framework.FrameworkAuditorControlStatus.Issue),
		"monitoring":          types.BoolValue(framework.FrameworkAuditorControlStatus.Monitoring),
		"not_applicable":      types.BoolValue(framework.FrameworkAuditorControlStatus.NotApplicable),
		"not_started":         types.BoolValue(framework.FrameworkAuditorControlStatus.NotStarted),
		"ready_for_audit":     types.BoolValue(framework.FrameworkAuditorControlStatus.ReadyForAudit),
		"under_review":        types.BoolValue(framework.FrameworkAuditorControlStatus.UnderReview),
	})
	resp.Diagnostics.Append(diags...)
	data.FrameworkAuditorControlStatus = controlStatusObj

	// Auditor evidence status
	evidenceStatusAttrTypes := map[string]attr.Type{
		"auditable": types.BoolType,
		"gap":       types.BoolType,
		"not_set":   types.BoolType,
	}
	evidenceStatusObj, diags := types.ObjectValue(evidenceStatusAttrTypes, map[string]attr.Value{
		"auditable": types.BoolValue(framework.FrameworkAuditorEvidenceStatus.Auditable),
		"gap":       types.BoolValue(framework.FrameworkAuditorEvidenceStatus.Gap),
		"not_set":   types.BoolValue(framework.FrameworkAuditorEvidenceStatus.NotSet),
	})
	resp.Diagnostics.Append(diags...)
	data.FrameworkAuditorEvidenceStatus = evidenceStatusObj

	// ==================== Structure and References ====================
	// Control categories
	if len(framework.FrameworkControlsCategories) > 0 {
		categoriesList, diags := types.ListValueFrom(ctx, types.StringType, framework.FrameworkControlsCategories)
		resp.Diagnostics.Append(diags...)
		data.FrameworkControlsCategories = categoriesList
	} else {
		data.FrameworkControlsCategories = types.ListNull(types.StringType)
	}

	// References
	data.FrameworkReferenceFieldName = types.StringValue(framework.FrameworkReferenceFieldName)
	if len(framework.FrameworkReferences) > 0 {
		refsList, diags := types.ListValueFrom(ctx, types.StringType, framework.FrameworkReferences)
		resp.Diagnostics.Append(diags...)
		data.FrameworkReferences = refsList
	} else {
		data.FrameworkReferences = types.ListNull(types.StringType)
	}

	// ==================== Framework Origin ====================
	if framework.FrameworkDuplicatedFrom != "" {
		data.FrameworkDuplicatedFrom = types.StringValue(framework.FrameworkDuplicatedFrom)
	} else {
		data.FrameworkDuplicatedFrom = types.StringNull()
	}

	// ==================== Plugin Scoping ====================
	if len(framework.FrameworkExcludedPlugins) > 0 {
		pluginsMap := make(map[string]attr.Value)
		for k, v := range framework.FrameworkExcludedPlugins {
			pluginsMap[k] = types.StringValue(fmt.Sprintf("%v", v))
		}
		excludedPlugins, diags := types.MapValue(types.StringType, pluginsMap)
		resp.Diagnostics.Append(diags...)
		data.FrameworkExcludedPlugins = excludedPlugins
	} else {
		data.FrameworkExcludedPlugins = types.MapNull(types.StringType)
	}

	// ==================== Customization ====================
	if framework.FrameworkIconID != "" {
		data.FrameworkIconID = types.StringValue(framework.FrameworkIconID)
	} else {
		data.FrameworkIconID = types.StringNull()
	}

	// ==================== Ordering ====================
	data.UnadoptedOrder = types.Int64Value(int64(framework.UnadoptedOrder))

	// ==================== Controls and Categories ====================
	controlsObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"control_id":  types.StringType,
			"name":        types.StringType,
			"description": types.StringType,
			"category":    types.StringType,
			"status":      types.StringType,
			"owners":      types.ListType{ElemType: types.StringType},
			"tags":        types.ListType{ElemType: types.StringType},
		},
	}

	if len(framework.Controls) > 0 {
		controls := make([]map[string]attr.Value, len(framework.Controls))
		for i, ctrl := range framework.Controls {
			// Build owners list
			var ownersList types.List
			if len(ctrl.ControlOwners) > 0 {
				ownersListValue, ownersListDiags := types.ListValueFrom(ctx, types.StringType, ctrl.ControlOwners)
				ownersList = ownersListValue
				resp.Diagnostics.Append(ownersListDiags...)
			} else {
				ownersList = types.ListNull(types.StringType)
			}

			// Build tags list
			var tagsList types.List
			if len(ctrl.ControlTags) > 0 {
				tagsListValue, tagsListDiags := types.ListValueFrom(ctx, types.StringType, ctrl.ControlTags)
				tagsList = tagsListValue
				resp.Diagnostics.Append(tagsListDiags...)
			} else {
				tagsList = types.ListNull(types.StringType)
			}

			controls[i] = map[string]attr.Value{
				"control_id":  types.StringValue(ctrl.ControlID),
				"name":        types.StringValue(ctrl.ControlName),
				"description": types.StringValue(ctrl.ControlDescription),
				"category":    types.StringValue(ctrl.ControlCategory),
				"status":      types.StringValue(ctrl.ControlStatus.Status),
				"owners":      ownersList,
				"tags":        tagsList,
			}
		}

		controlsList, diags := types.ListValueFrom(ctx, controlsObjType, controls)
		resp.Diagnostics.Append(diags...)
		data.Controls = controlsList
	} else {
		data.Controls = types.ListNull(controlsObjType)
	}

	// Convert categories
	categoriesObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"category_id": types.StringType,
			"name":        types.StringType,
			"description": types.StringType,
		},
	}

	if len(framework.Categories) > 0 {
		categories := make([]map[string]attr.Value, len(framework.Categories))
		for i, cat := range framework.Categories {
			categories[i] = map[string]attr.Value{
				"category_id": types.StringValue(cat.CategoryID),
				"name":        types.StringValue(cat.CategoryName),
				"description": types.StringValue(cat.Description),
			}
		}

		categoriesList, diags := types.ListValueFrom(ctx, categoriesObjType, categories)
		resp.Diagnostics.Append(diags...)
		data.Categories = categoriesList
	} else {
		data.Categories = types.ListNull(categoriesObjType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
