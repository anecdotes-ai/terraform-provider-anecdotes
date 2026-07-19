// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &FrameworkResource{}
var _ resource.ResourceWithImportState = &FrameworkResource{}

func NewFrameworkResource() resource.Resource {
	return &FrameworkResource{}
}

// FrameworkResource defines the resource implementation.
type FrameworkResource struct {
	client *client.AnecdotesClient
}

// FrameworkResourceModel describes the resource data model.
type FrameworkResourceModel struct {
	// Core identification
	FrameworkID types.String `tfsdk:"framework_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	FolderID    types.String `tfsdk:"folder_id"`

	// Auditor configuration
	FrameworkAuditable                types.Bool `tfsdk:"framework_auditable"`
	CanAuditorDownloadEvidence        types.Bool `tfsdk:"can_auditor_download_evidence"`
	CanAuditorViewControlAttachments  types.Bool `tfsdk:"can_auditor_view_control_attachments"`
	CanAuditorViewControlCustomFields types.Bool `tfsdk:"can_auditor_view_control_custom_fields"`
	CanAuditorViewSoaReport           types.Bool `tfsdk:"can_auditor_view_soa_report"`
	CanAuditorViewTags                types.Bool `tfsdk:"can_auditor_view_tags"`
	AuditorVisibleControlStatuses     types.Set  `tfsdk:"auditor_visible_control_statuses"`
	AuditorVisibleEvidenceStatuses    types.Set  `tfsdk:"auditor_visible_evidence_statuses"`
}

func (r *FrameworkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_framework"
}

func (r *FrameworkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anecdotes compliance framework container.",
		MarkdownDescription: `
Manages an Anecdotes compliance framework container.

A **Framework** is a top-level container representing a compliance standard or regulation (e.g., SOC2, ISO 27001, HIPAA). 
Controls are added to frameworks using the separate ` + "`anecdotes_control`" + ` resource.

## Framework Hierarchy

` + "```" + `
Framework (this resource)
    └── Controls (anecdotes_control)
            └── Requirements (anecdotes_requirement)
                    linked via (anecdotes_mapping_control_requirement)
` + "```" + `
`,

		Attributes: map[string]schema.Attribute{
			// ==================== Core Identification ====================
			"framework_id": schema.StringAttribute{
				Description:         "The unique identifier of the framework in Anecdotes (e.g., 1234567890).",
				MarkdownDescription: "The unique identifier of the framework in Anecdotes (e.g., `1234567890`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"name": schema.StringAttribute{
				Description: "The human-readable name of the framework (e.g., 'SOC 2', 'ISO-IEC 27001 2022').",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},

			"description": schema.StringAttribute{
				Description: "A detailed description of the framework and its purpose.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},

			"folder_id": schema.StringAttribute{
				Description:         "The ID of the folder where this framework will be placed. Optional for import.",
				MarkdownDescription: "The ID of the folder where this framework will be placed. Optional for import.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(), // Changing folder requires recreating the framework
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// ==================== Auditor Configuration ====================
			"framework_auditable": schema.BoolAttribute{
				Description: "Whether the framework is auditable. Managed by the platform's audit lifecycle and cannot be set through this provider; read-only.",
				Computed:    true,
			},

			"can_auditor_download_evidence": schema.BoolAttribute{
				Description: "Whether auditors can download evidence files from this framework.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},

			"can_auditor_view_control_attachments": schema.BoolAttribute{
				Description: "Whether auditors can view control attachments in this framework.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},

			"can_auditor_view_control_custom_fields": schema.BoolAttribute{
				Description: "Whether auditors can view custom fields on controls in this framework.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},

			"can_auditor_view_soa_report": schema.BoolAttribute{
				Description: "Whether auditors can view the Statement of Applicability (SOA) report for this framework.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},

			"can_auditor_view_tags": schema.BoolAttribute{
				Description: "Whether auditors can view tags on controls in this framework.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},

			"auditor_visible_control_statuses": schema.SetAttribute{
				Description:         "The set of control statuses visible to auditors. A status is visible when present in the set.",
				MarkdownDescription: "The set of control statuses visible to auditors. A status is visible when present in the set. Valid members: `approved_by_auditor`, `gap`, `in_progress`, `insufficient_data`, `issue`, `monitoring`, `not_applicable`, `not_started`, `ready_for_audit`, `under_review`.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.OneOf(client.ValidAuditorControlStatuses()...),
					),
				},
			},

			"auditor_visible_evidence_statuses": schema.SetAttribute{
				Description:         "The set of evidence statuses visible to auditors. A status is visible when present in the set.",
				MarkdownDescription: "The set of evidence statuses visible to auditors. A status is visible when present in the set. Valid members: `auditable`, `gap`, `not_set`.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.OneOf(client.ValidAuditorEvidenceStatuses()...),
					),
				},
			},
		},
	}
}

func (r *FrameworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.AnecdotesClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.AnecdotesClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *FrameworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FrameworkResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create ignores auditor configuration; it is applied afterward.
	framework, err := r.client.CreateFramework(&client.FrameworkCreateRequest{
		FrameworkName:        data.Name.ValueString(),
		FrameworkDescription: data.Description.ValueString(),
		FolderID:             data.FolderID.ValueString(),
	})
	if err != nil {
		addClientError(&resp.Diagnostics, "create framework", err)
		return
	}

	r.configureFrameworkAuditing(ctx, framework.FrameworkID, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		// The framework exists — persist state so the next apply does not duplicate it.
		var d diag.Diagnostics
		r.setFrameworkState(ctx, &data, framework, &d)
		resp.Diagnostics.Append(d...)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	framework, err = r.client.GetFramework(framework.FrameworkID)
	if err != nil {
		addClientError(&resp.Diagnostics, "read framework after create", err)
		return
	}

	var diags diag.Diagnostics
	r.setFrameworkState(ctx, &data, framework, &diags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FrameworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FrameworkResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get framework from API
	framework, err := r.client.GetFramework(data.FrameworkID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addClientError(&resp.Diagnostics, "read framework", err)
		return
	}

	// Update state with API response
	var diags diag.Diagnostics
	r.setFrameworkState(ctx, &data, framework, &diags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FrameworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FrameworkResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.configureFrameworkAuditing(ctx, data.FrameworkID.ValueString(), &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	framework, err := r.client.GetFramework(data.FrameworkID.ValueString())
	if err != nil {
		addClientError(&resp.Diagnostics, "read framework after update", err)
		return
	}

	var diags diag.Diagnostics
	r.setFrameworkState(ctx, &data, framework, &diags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// configureFrameworkAuditing applies the base fields and auditor booleans via
// PATCH and the auditor visibility status sets via their dedicated endpoints —
// the configuration the create endpoint does not accept. Shared by Create and Update.
func (r *FrameworkResource) configureFrameworkAuditing(ctx context.Context, frameworkID string, data *FrameworkResourceModel, diags *diag.Diagnostics) {
	patchReq := &client.FrameworkUpdateRequest{
		FrameworkName:                     data.Name.ValueString(),
		FrameworkDescription:              data.Description.ValueString(),
		CanAuditorDownloadEvidence:        optionalBoolPtr(data.CanAuditorDownloadEvidence),
		CanAuditorViewControlAttachments:  optionalBoolPtr(data.CanAuditorViewControlAttachments),
		CanAuditorViewControlCustomFields: optionalBoolPtr(data.CanAuditorViewControlCustomFields),
		CanAuditorViewSoaReport:           optionalBoolPtr(data.CanAuditorViewSoaReport),
		CanAuditorViewTags:                optionalBoolPtr(data.CanAuditorViewTags),
	}
	if _, err := r.client.UpdateFramework(frameworkID, patchReq); err != nil {
		addClientError(diags, "configure framework", err)
		return
	}

	if !data.AuditorVisibleControlStatuses.IsNull() && !data.AuditorVisibleControlStatuses.IsUnknown() {
		status := auditorControlStatusFromSet(ctx, data.AuditorVisibleControlStatuses, diags)
		if diags.HasError() {
			return
		}
		if err := r.client.SetFrameworkAuditorControlStatus(frameworkID, status); err != nil {
			addClientError(diags, "set framework auditor control statuses", err)
			return
		}
	}

	if !data.AuditorVisibleEvidenceStatuses.IsNull() && !data.AuditorVisibleEvidenceStatuses.IsUnknown() {
		status := auditorEvidenceStatusFromSet(ctx, data.AuditorVisibleEvidenceStatuses, diags)
		if diags.HasError() {
			return
		}
		if err := r.client.SetFrameworkAuditorEvidenceStatus(frameworkID, status); err != nil {
			addClientError(diags, "set framework auditor evidence statuses", err)
			return
		}
	}
}

func (r *FrameworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FrameworkResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteFramework(data.FrameworkID.ValueString())
	if err != nil {
		addClientError(&resp.Diagnostics, "delete framework", err)
		return
	}
}

func (r *FrameworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("framework_id"), req, resp)
}

// setFrameworkState sets the Terraform state from the API framework response
func (r *FrameworkResource) setFrameworkState(ctx context.Context, data *FrameworkResourceModel, framework *client.Framework, d *diag.Diagnostics) {
	data.FrameworkID = types.StringValue(framework.FrameworkID)
	data.Name = types.StringValue(framework.FrameworkName)
	data.Description = types.StringValue(framework.FrameworkDescription)

	// folder_id: API GET does not return this field reliably.
	// Preserve the plan/state value when the API response is empty.
	if framework.FolderID != "" {
		data.FolderID = types.StringValue(framework.FolderID)
	} else if data.FolderID.IsUnknown() {
		// Create without a configured folder and no API echo: record "no
		// folder" — a Computed attribute must not remain unknown after apply.
		data.FolderID = types.StringNull()
	}

	// framework_auditable is platform-managed (read-only) — always from the API.
	data.FrameworkAuditable = types.BoolValue(framework.FrameworkAuditable)
	data.CanAuditorDownloadEvidence = types.BoolValue(framework.CanAuditorDownloadEvidence)
	data.CanAuditorViewControlAttachments = types.BoolValue(framework.CanAuditorViewControlAttachments)
	data.CanAuditorViewControlCustomFields = types.BoolValue(framework.CanAuditorViewControlCustomFields)
	data.CanAuditorViewSoaReport = types.BoolValue(framework.CanAuditorViewSoaReport)
	data.CanAuditorViewTags = types.BoolValue(framework.CanAuditorViewTags)

	// Auditor control status (bool object → set membership)
	controlStatusMembers := make([]string, 0, 10)
	cs := framework.FrameworkAuditorControlStatus
	if cs.ApprovedByAuditor {
		controlStatusMembers = append(controlStatusMembers, "approved_by_auditor")
	}
	if cs.Gap {
		controlStatusMembers = append(controlStatusMembers, "gap")
	}
	if cs.InProgress {
		controlStatusMembers = append(controlStatusMembers, "in_progress")
	}
	if cs.InsufficientData {
		controlStatusMembers = append(controlStatusMembers, "insufficient_data")
	}
	if cs.Issue {
		controlStatusMembers = append(controlStatusMembers, "issue")
	}
	if cs.Monitoring {
		controlStatusMembers = append(controlStatusMembers, "monitoring")
	}
	if cs.NotApplicable {
		controlStatusMembers = append(controlStatusMembers, "not_applicable")
	}
	if cs.NotStarted {
		controlStatusMembers = append(controlStatusMembers, "not_started")
	}
	if cs.ReadyForAudit {
		controlStatusMembers = append(controlStatusMembers, "ready_for_audit")
	}
	if cs.UnderReview {
		controlStatusMembers = append(controlStatusMembers, "under_review")
	}
	controlStatusSet, diagsControl := types.SetValueFrom(ctx, types.StringType, controlStatusMembers)
	d.Append(diagsControl...)
	data.AuditorVisibleControlStatuses = controlStatusSet

	// Auditor evidence status (bool object → set membership)
	evidenceStatusMembers := make([]string, 0, 3)
	es := framework.FrameworkAuditorEvidenceStatus
	if es.Auditable {
		evidenceStatusMembers = append(evidenceStatusMembers, "auditable")
	}
	if es.Gap {
		evidenceStatusMembers = append(evidenceStatusMembers, "gap")
	}
	if es.NotSet {
		evidenceStatusMembers = append(evidenceStatusMembers, "not_set")
	}
	evidenceStatusSet, diagsEvidence := types.SetValueFrom(ctx, types.StringType, evidenceStatusMembers)
	d.Append(diagsEvidence...)
	data.AuditorVisibleEvidenceStatuses = evidenceStatusSet
}

// auditorControlStatusFromSet maps a set of visible-status member strings to the
// bool-object API contract (a member present in the set → that bool = true).
func auditorControlStatusFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) *client.FrameworkAuditorControlStatus {
	var members []string
	diags.Append(set.ElementsAs(ctx, &members, false)...)
	m := make(map[string]bool, len(members))
	for _, s := range members {
		m[s] = true
	}
	return &client.FrameworkAuditorControlStatus{
		ApprovedByAuditor: m["approved_by_auditor"],
		Gap:               m["gap"],
		InsufficientData:  m["insufficient_data"],
		InProgress:        m["in_progress"],
		Issue:             m["issue"],
		Monitoring:        m["monitoring"],
		NotApplicable:     m["not_applicable"],
		NotStarted:        m["not_started"],
		ReadyForAudit:     m["ready_for_audit"],
		UnderReview:       m["under_review"],
	}
}

// auditorEvidenceStatusFromSet maps a set of visible-status member strings to the
// bool-object API contract (a member present in the set → that bool = true).
func auditorEvidenceStatusFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) *client.FrameworkAuditorEvidenceStatus {
	var members []string
	diags.Append(set.ElementsAs(ctx, &members, false)...)
	m := make(map[string]bool, len(members))
	for _, s := range members {
		m[s] = true
	}
	return &client.FrameworkAuditorEvidenceStatus{
		Auditable: m["auditable"],
		Gap:       m["gap"],
		NotSet:    m["not_set"],
	}
}
