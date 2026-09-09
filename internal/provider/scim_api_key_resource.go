// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ScimApiKeyResource{}
var _ resource.ResourceWithImportState = &ScimApiKeyResource{}

func NewScimApiKeyResource() resource.Resource {
	return &ScimApiKeyResource{}
}

// ScimApiKeyResource manages an API key scoped to SCIM provisioning
// (Administration > Settings > SCIM). There is no update endpoint — any change
// to api_key_name requires a new key.
type ScimApiKeyResource struct {
	client *client.AnecdotesClient
}

// ScimApiKeyResourceModel describes the resource data model.
type ScimApiKeyResourceModel struct {
	ApiKeyName   types.String `tfsdk:"api_key_name"`
	KeyID        types.String `tfsdk:"key_id"`
	Key          types.String `tfsdk:"key"`
	CreatedBy    types.String `tfsdk:"created_by"`
	CreatedAt    types.String `tfsdk:"created_at"`
	LastUsedDate types.String `tfsdk:"last_used_date"`
}

func (r *ScimApiKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scim_api_key"
}

func (r *ScimApiKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anecdotes API key scoped to SCIM provisioning (Administration > Settings > SCIM). The full key secret is only ever available immediately after creation.",
		Attributes: map[string]schema.Attribute{
			"api_key_name": schema.StringAttribute{
				Description: "A display name for the key. There is no update endpoint, so changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_id": schema.StringAttribute{
				Description: "The stable identifier of the key, used to delete it. Not the secret itself.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				Description:         "The full key secret. Only ever populated from the create response — a subsequent read from the platform returns just the last 8 characters, so this value is never refreshed after creation.",
				MarkdownDescription: "The full key secret. Only ever populated from the create response — a subsequent read from the platform returns just the last 8 characters, so this value is **never refreshed** after creation.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				Description: "The uid of the user who created the key.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the key was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_used_date": schema.StringAttribute{
				Description: "Timestamp when the key was last used, if ever.",
				Computed:    true,
			},
		},
	}
}

func (r *ScimApiKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScimApiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ScimApiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.client.CreateScimApiKey(ctx, &client.ScimApiKeyCreateRequest{
		ApiKeyName: data.ApiKeyName.ValueString(),
	})
	if err != nil {
		addClientError(&resp.Diagnostics, "create SCIM API key", err)
		return
	}

	data.ApiKeyName = types.StringValue(key.ApiKeyName)
	data.KeyID = types.StringValue(key.KeyID)
	data.Key = types.StringValue(key.Key)
	data.CreatedBy = types.StringValue(key.CreatedBy)
	data.CreatedAt = types.StringValue(key.CreatedAt)
	data.LastUsedDate = optionalStringValue(key.LastUsedDate)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScimApiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ScimApiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.client.GetScimApiKey(ctx, data.KeyID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addClientError(&resp.Diagnostics, "read SCIM API key", err)
		return
	}

	data.ApiKeyName = types.StringValue(key.ApiKeyName)
	data.KeyID = types.StringValue(key.KeyID)
	data.CreatedBy = types.StringValue(key.CreatedBy)
	data.CreatedAt = types.StringValue(key.CreatedAt)
	data.LastUsedDate = optionalStringValue(key.LastUsedDate)

	// The platform only ever returns the full secret on create; every List
	// response truncates it. Never overwrite an already-known key with that
	// truncated value — only fill it in when state doesn't have one yet (e.g.
	// right after an import).
	if data.Key.IsNull() || data.Key.IsUnknown() {
		data.Key = types.StringValue(key.Key)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScimApiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// api_key_name is the only writable attribute, and it is RequiresReplace —
	// there is no update endpoint, so this should never be invoked.
}

func (r *ScimApiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ScimApiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteScimApiKey(ctx, data.KeyID.ValueString()); err != nil {
		addClientError(&resp.Diagnostics, "delete SCIM API key", err)
		return
	}
}

func (r *ScimApiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("key_id"), req, resp)
}

// optionalStringValue converts a possibly-nil string pointer from the API into
// a Terraform string value, null when the pointer is nil.
func optionalStringValue(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}
