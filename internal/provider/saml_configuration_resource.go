// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &SamlConfigurationResource{}
var _ resource.ResourceWithImportState = &SamlConfigurationResource{}

func NewSamlConfigurationResource() resource.Resource {
	return &SamlConfigurationResource{}
}

// SamlConfigurationResource manages a SAML 2.0 identity provider configuration
// (Settings > Login Methods > SAML 2.0).
type SamlConfigurationResource struct {
	client *client.AnecdotesClient
}

// SamlConfigurationResourceModel describes the resource data model.
type SamlConfigurationResourceModel struct {
	ProviderID       types.String `tfsdk:"provider_id"`
	DisplayName      types.String `tfsdk:"display_name"`
	IdpEntityID      types.String `tfsdk:"idp_entity_id"`
	RpEntityID       types.String `tfsdk:"rp_entity_id"`
	SsoURL           types.String `tfsdk:"sso_url"`
	X509Certificates types.List   `tfsdk:"x509_certificates"`
	IdpType          types.String `tfsdk:"idp_type"`
}

var samlDisplayNamePattern = regexp.MustCompile(`^\w+$`)
var samlURLPattern = regexp.MustCompile(`^https?://`)

func (r *SamlConfigurationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saml_configuration"
}

func (r *SamlConfigurationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SAML 2.0 identity provider configuration (Settings > Login Methods > SAML 2.0).",
		Attributes: map[string]schema.Attribute{
			"provider_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The configuration's identifier, derived from `display_name` at creation and stable across updates — including a `display_name` change. Because it is a one-time derivation, not a live mapping, `provider_id` no longer corresponds to the current `display_name` after a rename: it keeps reflecting whatever `display_name` was set at creation. If state is lost after a rename, `provider_id` cannot be recomputed from the current `display_name` — look it up in the platform UI (or via the identity API's list endpoint) before importing.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "A display name for the configuration. Renameable in place — doing so does not replace the resource or affect `provider_id` — but see `provider_id`'s description for the recovery caveat that follows from that.",
				Validators: []validator.String{
					stringvalidator.LengthAtMost(30),
					stringvalidator.RegexMatches(samlDisplayNamePattern, "must contain only word characters (letters, digits, underscore)"),
				},
			},
			"idp_entity_id": schema.StringAttribute{
				Required:    true,
				Description: "The identity provider's entity ID.",
				Validators: []validator.String{
					stringvalidator.LengthAtMost(256),
				},
			},
			"rp_entity_id": schema.StringAttribute{
				Required:    true,
				Description: "The relying party (this platform's) entity ID.",
			},
			"sso_url": schema.StringAttribute{
				Required:    true,
				Description: "The identity provider's SSO URL.",
				Validators: []validator.String{
					stringvalidator.LengthAtMost(256),
					stringvalidator.RegexMatches(samlURLPattern, "must be an http(s) URL"),
				},
			},
			"x509_certificates": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "One or more PEM-encoded signing certificates.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"idp_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Server-assigned, e.g. `okta`, `azuread`, or `custom`.",
			},
		},
	}
}

func (r *SamlConfigurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SamlConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SamlConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	certs := stringsFromList(ctx, plan.X509Certificates, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.CreateSamlConfiguration(ctx, &client.SamlCreateRequest{
		DisplayName:      plan.DisplayName.ValueString(),
		IdpEntityID:      plan.IdpEntityID.ValueString(),
		RpEntityID:       plan.RpEntityID.ValueString(),
		SsoURL:           plan.SsoURL.ValueString(),
		X509Certificates: certs,
	})
	if err != nil {
		if client.IsConflict(err) {
			resp.Diagnostics.AddError(
				"SAML Configuration Already Exists",
				fmt.Sprintf("A SAML configuration named %q already exists. display_name must be unique per tenant — provider_id is derived from it.", plan.DisplayName.ValueString()),
			)
			return
		}
		addClientError(&resp.Diagnostics, "create SAML configuration", err)
		return
	}

	mapSamlConfigurationToState(ctx, cfg, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SamlConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SamlConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetSamlConfiguration(ctx, state.ProviderID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addClientError(&resp.Diagnostics, "read SAML configuration", err)
		return
	}

	mapSamlConfigurationToState(ctx, cfg, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SamlConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state SamlConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	certs := stringsFromList(ctx, plan.X509Certificates, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.UpdateSamlConfiguration(ctx, &client.SamlUpdateRequest{
		ProviderID:       state.ProviderID.ValueString(),
		DisplayName:      plan.DisplayName.ValueString(),
		IdpEntityID:      plan.IdpEntityID.ValueString(),
		RpEntityID:       plan.RpEntityID.ValueString(),
		SsoURL:           plan.SsoURL.ValueString(),
		X509Certificates: certs,
	})
	if err != nil {
		addClientError(&resp.Diagnostics, "update SAML configuration", err)
		return
	}

	mapSamlConfigurationToState(ctx, cfg, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes a SAML configuration. The platform has no reliable not-found
// signal for this endpoint — an unknown provider_id and a genuine upstream
// failure both surface as a plain-text 502. A 502 is therefore not enough on
// its own to treat as success: a follow-up GetSamlConfiguration confirms the
// configuration is actually gone (ErrNotFound) before this reports success —
// a 502 from a real transient failure, with the configuration still present,
// still surfaces as an error rather than silently dropping it from state.
func (r *SamlConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SamlConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSamlConfiguration(ctx, state.ProviderID.ValueString())
	if err == nil {
		return
	}

	if client.StatusCode(err) == 502 {
		if _, getErr := r.client.GetSamlConfiguration(ctx, state.ProviderID.ValueString()); client.IsNotFound(getErr) {
			tflog.Warn(ctx, "SAML configuration delete returned 502, but a follow-up read confirms the configuration is gone; treating as success", map[string]interface{}{
				"provider_id": state.ProviderID.ValueString(),
			})
			return
		}
	}

	addClientError(&resp.Diagnostics, "delete SAML configuration", err)
}

func (r *SamlConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("provider_id"), req, resp)
}

// mapSamlConfigurationToState sets the Terraform state from the API response.
func mapSamlConfigurationToState(ctx context.Context, cfg *client.SamlConfiguration, data *SamlConfigurationResourceModel, diags *diag.Diagnostics) {
	data.ProviderID = types.StringValue(cfg.ProviderID)
	data.DisplayName = types.StringValue(cfg.DisplayName)
	data.IdpEntityID = types.StringValue(cfg.IdpEntityID)
	data.RpEntityID = types.StringValue(cfg.RpEntityID)
	data.SsoURL = types.StringValue(cfg.SsoURL)
	data.IdpType = types.StringValue(cfg.IdpType)

	certs, listDiags := types.ListValueFrom(ctx, types.StringType, cfg.X509Certificates)
	diags.Append(listDiags...)
	data.X509Certificates = certs
}
