// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MappingRequirementEvidenceScopeResource{}
var _ resource.ResourceWithImportState = &MappingRequirementEvidenceScopeResource{}
var _ resource.ResourceWithValidateConfig = &MappingRequirementEvidenceScopeResource{}

func NewMappingRequirementEvidenceScopeResource() resource.Resource {
	return &MappingRequirementEvidenceScopeResource{}
}

// MappingRequirementEvidenceScopeResource defines the resource implementation for scoping evidence on requirements.
type MappingRequirementEvidenceScopeResource struct {
	client *client.AnecdotesClient
}

// MappingRequirementEvidenceScopeResourceModel describes the resource data model.
type MappingRequirementEvidenceScopeResourceModel struct {
	RequirementID types.String `tfsdk:"requirement_id"`
	EvidenceID    types.String `tfsdk:"evidence_id"`
	ScopeID       types.String `tfsdk:"scope_id"`
	ScopeType     types.String `tfsdk:"scope_type"`
	Excluded      types.Bool   `tfsdk:"excluded"`
}

func (r *MappingRequirementEvidenceScopeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mapping_requirement_evidence_scope"
}

func (r *MappingRequirementEvidenceScopeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Scopes evidence on a requirement to one framework or one control.",
		MarkdownDescription: `
Scopes evidence on a requirement to one framework or one control.

` + "`anecdotes_mapping_requirement_evidence`" + ` links evidence to a requirement everywhere the
requirement is used. This resource adds the evidence within one framework or control only,
or with ` + "`excluded = true`" + ` removes it there.

An entry that already exists must be imported. A framework scope is stored only for a
custom framework or one that is active in your account.
`,
		Attributes: map[string]schema.Attribute{
			"requirement_id": schema.StringAttribute{
				Description: "The unique identifier of the requirement.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"evidence_id": schema.StringAttribute{
				Description: "The unique identifier of the evidence item to scope.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scope_id": schema.StringAttribute{
				Description: "The unique identifier of the framework or control the evidence is scoped to.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scope_type": schema.StringAttribute{
				Description:         "The kind of object scope_id refers to: framework or control.",
				MarkdownDescription: "The kind of object `scope_id` refers to: `framework` or `control`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.EvidenceScopeFramework, client.EvidenceScopeControl),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"excluded": schema.BoolAttribute{
				Description:         "When true, the evidence is removed from the scope instead of added to it. Defaults to false.",
				MarkdownDescription: "When `true`, the evidence is removed from the scope instead of added to it. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

// ValidateConfig rejects a prefixed scope_id whose prefix names the other scope type.
func (r *MappingRequirementEvidenceScopeResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data MappingRequirementEvidenceScopeResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.ScopeID.IsNull() || data.ScopeID.IsUnknown() || data.ScopeType.IsNull() || data.ScopeType.IsUnknown() {
		return
	}

	scopeID, scopeType := data.ScopeID.ValueString(), data.ScopeType.ValueString()
	for _, prefixed := range []string{client.EvidenceScopeFramework, client.EvidenceScopeControl} {
		if strings.HasPrefix(scopeID, prefixed+"_") && scopeType != prefixed {
			resp.Diagnostics.AddAttributeError(
				path.Root("scope_type"),
				"Scope Type Does Not Match Scope ID",
				fmt.Sprintf("scope_id %s is a %s ID, so scope_type must be %q.", scopeID, prefixed, prefixed),
			)
		}
	}
}

func (r *MappingRequirementEvidenceScopeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, "Resource", &resp.Diagnostics)
}

func (r *MappingRequirementEvidenceScopeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MappingRequirementEvidenceScopeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.CreateRequirementEvidenceScope(ctx,
		data.RequirementID.ValueString(), data.ScopeID.ValueString(), data.ScopeType.ValueString(),
		data.EvidenceID.ValueString(), !data.Excluded.ValueBool())
	if errors.Is(err, client.ErrEvidenceScopeExists) {
		resp.Diagnostics.AddError(
			"Evidence Already Scoped",
			fmt.Sprintf(
				"This evidence already has an entry in the scope. To manage it with Terraform, import it first:\n\n"+
					"  terraform import anecdotes_mapping_requirement_evidence_scope.<name> %s",
				evidenceScopeImportID(data),
			),
		)
		return
	}
	if err != nil {
		r.addWriteError(&resp.Diagnostics, "add evidence scope", err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MappingRequirementEvidenceScopeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MappingRequirementEvidenceScopeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scope, err := r.client.GetRequirementEvidenceScope(ctx,
		data.RequirementID.ValueString(), data.ScopeID.ValueString(), data.EvidenceID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addClientError(&resp.Diagnostics, "read evidence scope", err)
		return
	}

	data.ScopeType = types.StringValue(scope.ScopeType)
	data.Excluded = types.BoolValue(!scope.Included)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MappingRequirementEvidenceScopeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MappingRequirementEvidenceScopeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.SetRequirementEvidenceScope(ctx,
		data.RequirementID.ValueString(), data.ScopeID.ValueString(), data.ScopeType.ValueString(),
		data.EvidenceID.ValueString(), !data.Excluded.ValueBool())
	if err != nil {
		r.addWriteError(&resp.Diagnostics, "update evidence scope", err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MappingRequirementEvidenceScopeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MappingRequirementEvidenceScopeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RemoveRequirementEvidenceScope(ctx,
		data.RequirementID.ValueString(), data.ScopeID.ValueString(), data.EvidenceID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		r.addWriteError(&resp.Diagnostics, "remove evidence scope", err)
		return
	}
}

func (r *MappingRequirementEvidenceScopeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: requirement_id/scope_type/scope_id/evidence_id
	parts := splitImportID(req.ID, 4)
	if parts == nil || (parts[1] != client.EvidenceScopeFramework && parts[1] != client.EvidenceScopeControl) {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in the format 'requirement_id/scope_type/scope_id/evidence_id', where scope_type is framework or control, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("requirement_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("scope_type"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("scope_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("evidence_id"), parts[3])...)
}

func evidenceScopeImportID(data MappingRequirementEvidenceScopeResourceModel) string {
	return strings.Join([]string{
		data.RequirementID.ValueString(), data.ScopeType.ValueString(),
		data.ScopeID.ValueString(), data.EvidenceID.ValueString(),
	}, "/")
}

// addWriteError reports a failed scope write.
func (r *MappingRequirementEvidenceScopeResource) addWriteError(diags *diag.Diagnostics, action string, err error) {
	if errors.Is(err, client.ErrEvidenceScopeNotStored) {
		diags.AddError(
			"Evidence Scope Not Stored",
			"The platform accepted the request but did not store the change. A framework scope is stored only "+
				"when every framework scoped on the requirement is custom or active in your account.",
		)
		return
	}
	addClientError(diags, action, err)
}
