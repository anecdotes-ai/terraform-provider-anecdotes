// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RoleResourceModel describes the resource data model.
//
// Permissions is deliberately NOT sourced from the API on Read/Create/Update:
// per the platform's own contract, a role's submitted permissions are echoed
// back but never persisted or enforced — the role's real, effective permission
// set is always the inheritance-expanded resolution of Extends, which List/Get
// return. Treating that resolved set as this attribute's value made every
// apply of a role with a non-empty Extends fail ("provider produced
// inconsistent result after apply"), since Permissions is Required and the
// resolved set almost never equals what was configured. Permissions therefore
// always holds exactly what the user configured; the resolved set is exposed
// separately as EffectivePermissions.
type RoleResourceModel struct {
	RoleID               types.String `tfsdk:"role_id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Permissions          types.Set    `tfsdk:"permissions"`
	EffectivePermissions types.Set    `tfsdk:"effective_permissions"`
	Extends              types.Set    `tfsdk:"extends"`
	FullAccessFrameworks types.List   `tfsdk:"full_access_frameworks"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

var _ resource.Resource = &RoleResource{}
var _ resource.ResourceWithImportState = &RoleResource{}

func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

// RoleResource manages a tenant-scoped custom RBAC role.
type RoleResource struct {
	client *client.AnecdotesClient
}

func (r *RoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *RoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a tenant-scoped custom RBAC role.",
		Attributes: map[string]schema.Attribute{
			"role_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The role's key, server-generated once from `name` at creation. It never changes afterward, including on rename.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the role.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The description of the role. If omitted, the platform auto-generates one.",
			},
			"permissions": schema.SetAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Permissions to submit for this role (e.g. `control:read`, `evidence:read`). **The platform accepts and echoes this list but does not persist or enforce it** — a role's real, effective permissions are always the inheritance-expanded resolution of `extends`. See `effective_permissions` for that resolved set. This attribute exists to match the API's create/update contract; it never reflects drift, since the platform has no way to report a change to it.",
			},
			"effective_permissions": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "The role's live, effective permission set — the inheritance-expanded resolution of `extends`, as returned by the platform. This is what actually governs access; `permissions` does not.",
			},
			"extends": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Base role keys this role inherits permissions from. Defaults to `[\"basic_role\"]` if omitted.",
			},
			"full_access_frameworks": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Framework IDs this role has full access to. Omit for an unscoped role.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the role was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the role was last updated.",
			},
		},
	}
}

func (r *RoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissions := stringsFromSet(ctx, plan.Permissions, &resp.Diagnostics)
	extends := stringsFromSet(ctx, plan.Extends, &resp.Diagnostics)
	fullAccessFrameworks := stringsFromList(ctx, plan.FullAccessFrameworks, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.CreateRole(ctx, &client.RoleCreateRequest{
		Name:                 plan.Name.ValueString(),
		Description:          plan.Description.ValueString(),
		Extends:              extends,
		Permissions:          permissions,
		FullAccessFrameworks: fullAccessFrameworks,
	})
	if err != nil {
		addClientError(&resp.Diagnostics, "create role", err)
		return
	}

	mapRoleToState(ctx, role, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.GetRole(ctx, state.RoleID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addClientError(&resp.Diagnostics, "read role ID "+state.RoleID.ValueString(), err)
		return
	}

	mapRoleToState(ctx, role, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update sends the plan's fields as the complete desired object (PUT is a
// full-object replace). Any attribute left unset in config carries forward its
// prior known value automatically — standard behavior for an Optional+Computed
// attribute across an Update — so the plan is already the right full object,
// with no separate read-then-merge step needed.
func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state RoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissions := stringsFromSet(ctx, plan.Permissions, &resp.Diagnostics)
	extends := stringsFromSet(ctx, plan.Extends, &resp.Diagnostics)
	fullAccessFrameworks := stringsFromList(ctx, plan.FullAccessFrameworks, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.UpdateRole(ctx, &client.RoleUpdateRequest{
		Key:                  state.RoleID.ValueString(),
		Name:                 plan.Name.ValueString(),
		Description:          plan.Description.ValueString(),
		Extends:              extends,
		Permissions:          permissions,
		FullAccessFrameworks: fullAccessFrameworks,
	})
	if err != nil {
		addClientError(&resp.Diagnostics, "update role", err)
		return
	}

	mapRoleToState(ctx, role, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteRole(ctx, state.RoleID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		addClientError(&resp.Diagnostics, "delete role", err)
		return
	}
}

func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("role_id"), req, resp)
}

// mapRoleToState sets the Terraform state from the API role response.
// Permissions is deliberately left untouched here — see RoleResourceModel's
// doc comment for why the resolved set must never be written to it.
func mapRoleToState(ctx context.Context, role *client.Role, data *RoleResourceModel, diags *diag.Diagnostics) {
	data.RoleID = types.StringValue(role.Key)
	data.Name = types.StringValue(role.Name)
	data.Description = types.StringValue(role.Description)
	data.CreatedAt = types.StringValue(role.CreatedAt)
	data.UpdatedAt = types.StringValue(role.UpdatedAt)

	effectivePermissions, setDiags := types.SetValueFrom(ctx, types.StringType, role.Permissions)
	diags.Append(setDiags...)
	data.EffectivePermissions = effectivePermissions

	extends, setDiags := types.SetValueFrom(ctx, types.StringType, role.Extends)
	diags.Append(setDiags...)
	data.Extends = extends

	if len(role.FullAccessFrameworks) > 0 {
		fullAccessFrameworks, listDiags := types.ListValueFrom(ctx, types.StringType, role.FullAccessFrameworks)
		diags.Append(listDiags...)
		data.FullAccessFrameworks = fullAccessFrameworks
	} else {
		data.FullAccessFrameworks = types.ListNull(types.StringType)
	}
}
