// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &LoginSettingsResource{}

func NewLoginSettingsResource() resource.Resource {
	return &LoginSettingsResource{}
}

// LoginSettingsResource manages the tenant's Login Methods settings. This is a
// singleton resource: there is one login configuration per tenant, and it always
// exists on the platform, so Create and Update both apply the desired settings,
// and Delete only removes the resource from state.
type LoginSettingsResource struct {
	client *client.AnecdotesClient
}

// LoginSettingsResourceModel describes the resource data model.
type LoginSettingsResourceModel struct {
	InternalUsersLogin        types.Bool `tfsdk:"internal_users_login"`
	AuditorsLogin             types.Bool `tfsdk:"auditors_login"`
	ExternalStakeholdersLogin types.Bool `tfsdk:"external_stakeholders_login"`
	GoogleIdpEnabled          types.Bool `tfsdk:"google_idp_enabled"`
	MicrosoftIdpEnabled       types.Bool `tfsdk:"microsoft_idp_enabled"`
	SupportTeamLogin          types.Bool `tfsdk:"support_team_login"`
}

func (r *LoginSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_login_settings"
}

func (r *LoginSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the Anecdotes tenant's Login Methods settings (Administration > Settings). This is a singleton resource: there is exactly one login configuration per tenant.",
		Attributes: map[string]schema.Attribute{
			"internal_users_login": schema.BoolAttribute{
				Description: "Whether email/password login is permitted for internal users.",
				Optional:    true,
				Computed:    true,
			},
			"auditors_login": schema.BoolAttribute{
				Description: "Whether email/password login is permitted for auditors.",
				Optional:    true,
				Computed:    true,
			},
			"external_stakeholders_login": schema.BoolAttribute{
				Description: "Whether email/password login is permitted for external stakeholders.",
				Optional:    true,
				Computed:    true,
			},
			"google_idp_enabled": schema.BoolAttribute{
				Description: "Whether login via Google is enabled.",
				Optional:    true,
				Computed:    true,
			},
			"microsoft_idp_enabled": schema.BoolAttribute{
				Description: "Whether login via Microsoft is enabled.",
				Optional:    true,
				Computed:    true,
			},
			"support_team_login": schema.BoolAttribute{
				Description: "Whether the Anecdotes support team may log in to this tenant.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *LoginSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.AnecdotesClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.AnecdotesClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *LoginSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LoginSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated := r.applyLoginSettings(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	mapLoginSettingsToState(updated, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LoginSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LoginSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := r.client.GetLoginSettings(ctx)
	if err != nil {
		addClientError(&resp.Diagnostics, "read login settings", err)
		return
	}

	mapLoginSettingsToState(settings, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LoginSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data LoginSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated := r.applyLoginSettings(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	mapLoginSettingsToState(updated, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LoginSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Settings is a singleton — "delete" just removes from state. Settings persist on the platform.
}

// applyLoginSettings reads the tenant's current login settings, overlays the
// fields present in data, and PUTs the merged object back. PUT is a full-object
// replace, so any field the user did not configure must be carried through
// unchanged rather than sent as a zero value — this also preserves seamless_login,
// which is not exposed as an attribute on this resource.
func (r *LoginSettingsResource) applyLoginSettings(ctx context.Context, data *LoginSettingsResourceModel, diags *diag.Diagnostics) *client.LoginSettings {
	current, err := r.client.GetLoginSettings(ctx)
	if err != nil {
		addClientError(diags, "read current login settings", err)
		return nil
	}

	desired := *current

	if !data.InternalUsersLogin.IsNull() && !data.InternalUsersLogin.IsUnknown() {
		desired.EmailLogin.InternalUsers = data.InternalUsersLogin.ValueBool()
	}
	if !data.AuditorsLogin.IsNull() && !data.AuditorsLogin.IsUnknown() {
		desired.EmailLogin.Auditors = data.AuditorsLogin.ValueBool()
	}
	if !data.ExternalStakeholdersLogin.IsNull() && !data.ExternalStakeholdersLogin.IsUnknown() {
		desired.EmailLogin.ExternalStakeholders = data.ExternalStakeholdersLogin.ValueBool()
	}
	if !data.GoogleIdpEnabled.IsNull() && !data.GoogleIdpEnabled.IsUnknown() {
		desired.Idps.Google = data.GoogleIdpEnabled.ValueBool()
	}
	if !data.MicrosoftIdpEnabled.IsNull() && !data.MicrosoftIdpEnabled.IsUnknown() {
		desired.Idps.Microsoft = data.MicrosoftIdpEnabled.ValueBool()
	}
	if !data.SupportTeamLogin.IsNull() && !data.SupportTeamLogin.IsUnknown() {
		desired.SupportTeamLogin = data.SupportTeamLogin.ValueBool()
	}

	updated, err := r.client.UpdateLoginSettings(ctx, &desired)
	if err != nil {
		addClientError(diags, "update login settings", err)
		return nil
	}

	return updated
}

// mapLoginSettingsToState sets the Terraform state from the API login settings response.
func mapLoginSettingsToState(settings *client.LoginSettings, data *LoginSettingsResourceModel) {
	data.InternalUsersLogin = types.BoolValue(settings.EmailLogin.InternalUsers)
	data.AuditorsLogin = types.BoolValue(settings.EmailLogin.Auditors)
	data.ExternalStakeholdersLogin = types.BoolValue(settings.EmailLogin.ExternalStakeholders)
	data.GoogleIdpEnabled = types.BoolValue(settings.Idps.Google)
	data.MicrosoftIdpEnabled = types.BoolValue(settings.Idps.Microsoft)
	data.SupportTeamLogin = types.BoolValue(settings.SupportTeamLogin)
}
