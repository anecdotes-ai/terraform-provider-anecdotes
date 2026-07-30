// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ControlResource{}
var _ resource.ResourceWithImportState = &ControlResource{}

func NewControlResource() resource.Resource {
	return &ControlResource{}
}

// ControlResource defines the resource implementation.
type ControlResource struct {
	client *client.AnecdotesClient
}

// ControlResourceModel describes the resource data model.
type ControlResourceModel struct {
	// Core identification
	ControlID   types.String `tfsdk:"control_id"`
	FrameworkID types.String `tfsdk:"framework_id"`
	Name        types.String `tfsdk:"name"`

	// Control content
	Description types.String `tfsdk:"description"`
	CategoryID  types.String `tfsdk:"category_id"`

	// Maturity
	MaturityLevel types.String `tfsdk:"maturity_level"`

	// Ownership and assignment
	Owners types.Set `tfsdk:"owners"`
}

func (r *ControlResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_control"
}

func (r *ControlResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anecdotes control within a framework.",
		MarkdownDescription: `
Manages an Anecdotes control within a framework.

A **Control** is a prescriptive statement relating to a mechanism that should be implemented. 
Controls belong to exactly one framework and can have multiple requirements linked to them.

## Relationships

` + "```" + `
Framework (anecdotes_framework)
    └── Control (this resource)
            └── Requirements linked via (anecdotes_mapping_control_requirement)
` + "```" + `

> **Note:** Control status is not managed by this resource — it is computed by the
> platform. Inspect it with the ` + "`anecdotes_control`" + ` or ` + "`anecdotes_controls`" + `
> data sources.
`,

		Attributes: map[string]schema.Attribute{
			// Core identification
			"control_id": schema.StringAttribute{
				Description:         "The unique identifier of the control in Anecdotes (e.g., control_55312).",
				MarkdownDescription: "The unique identifier of the control in Anecdotes (e.g., `control_55312`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"framework_id": schema.StringAttribute{
				Description:         "The ID of the framework this control belongs to. A control can only belong to one framework.",
				MarkdownDescription: "The ID of the framework this control belongs to. A control can only belong to **one** framework.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(), // Changing framework requires recreating the control
				},
			},

			"name": schema.StringAttribute{
				Description: "The human-readable name of the control (e.g., 'CC6.1 - Logical access controls').",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 500),
				},
			},

			// Control content
			"description": schema.StringAttribute{
				Description: "A detailed description of the control and its implementation requirements.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},

			"category_id": schema.StringAttribute{
				Description:         "The ID of the control category this control belongs to. Use anecdotes_control_category to create categories.",
				MarkdownDescription: "The ID of the control category this control belongs to. Use `anecdotes_control_category` to create categories.",
				Required:            true,
			},

			// Maturity
			"maturity_level": schema.StringAttribute{
				Description:         "The maturity level of the control. One of: INITIAL, REPEATABLE, DEFINED, MANAGED, OPTIMIZING. A control has no maturity level until one is set; removing the attribute clears it.",
				MarkdownDescription: "The maturity level of the control. One of: `INITIAL`, `REPEATABLE`, `DEFINED`, `MANAGED`, `OPTIMIZING`. A control has no maturity level until one is set; removing the attribute clears it.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.ValidMaturityLevels()...),
				},
			},

			// Ownership and assignment
			"owners": schema.SetAttribute{
				Description: "Email addresses of users who own this control. Owners are responsible for maintaining and updating the control. Order does not matter. Terraform owns this attribute: removing it clears the owners.",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
							"must be a valid email address",
						),
					),
				},
			},
		},
	}
}

func (r *ControlResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ControlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ControlResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get category details to get the name
	category, err := r.client.GetControlCategory(data.CategoryID.ValueString())
	if err != nil {
		addClientError(&resp.Diagnostics, "get control category", err)
		return
	}

	// Build create request
	createReq := &client.ControlCreateRequest{
		ControlName:                data.Name.ValueString(),
		ControlDescription:         data.Description.ValueString(),
		ControlFrameworkCategory:   category.CategoryName,
		ControlFrameworkCategoryID: data.CategoryID.ValueString(),
	}

	// Handle owners
	if !data.Owners.IsNull() && !data.Owners.IsUnknown() {
		var owners []string
		resp.Diagnostics.Append(data.Owners.ElementsAs(ctx, &owners, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.ControlOwners = owners
	}

	// Call API
	control, err := r.client.AddControl(data.FrameworkID.ValueString(), createReq)
	if err != nil {
		addClientError(&resp.Diagnostics, "create control", err)
		return
	}

	data.ControlID = types.StringValue(control.ControlID)

	// Set maturity level if user specified it (separate API call)
	if !data.MaturityLevel.IsNull() && !data.MaturityLevel.IsUnknown() {
		if err := r.client.SetControlMaturityLevel(control.ControlID, data.MaturityLevel.ValueString()); err != nil {
			// The control exists on the platform — persist it in state (as
			// tainted) before erroring, so the next apply does not create a
			// duplicate.
			data.MaturityLevel = types.StringNull()
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			addClientError(&resp.Diagnostics, "set control maturity level", err)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ControlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ControlResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get control from API
	control, err := r.client.GetControl(data.FrameworkID.ValueString(), data.ControlID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addClientError(&resp.Diagnostics, "read control", err)
		return
	}

	if control.ControlIsCustom != nil && !*control.ControlIsCustom {
		resp.Diagnostics.AddError(
			"Control cannot be managed by Terraform",
			fmt.Sprintf(
				"Control %s is provided by the Anecdotes platform. Only custom controls can be updated or deleted, "+
					"so Terraform cannot manage it.\n\n"+
					"Read it with the anecdotes_control or anecdotes_controls data source instead. If it is already "+
					"in state, remove it with `terraform state rm` before applying again.",
				data.ControlID.ValueString(),
			),
		)
		return
	}

	// Update state with API response
	data.Name = types.StringValue(control.ControlName)
	data.Description = types.StringValue(control.ControlDescription)
	if control.ControlFrameworkCategoryID != "" {
		data.CategoryID = types.StringValue(control.ControlFrameworkCategoryID)
	}

	// An empty set and an unset attribute are distinct.
	if len(control.ControlOwners) > 0 {
		ownersSet, diags := types.SetValueFrom(ctx, types.StringType, control.ControlOwners)
		resp.Diagnostics.Append(diags...)
		data.Owners = ownersSet
	} else if !data.Owners.IsNull() {
		data.Owners = types.SetValueMust(types.StringType, []attr.Value{})
	} else {
		data.Owners = types.SetNull(types.StringType)
	}

	if level, err := r.client.GetControlMaturityLevel(data.ControlID.ValueString()); err == nil {
		if level == "" {
			data.MaturityLevel = types.StringNull()
		} else {
			data.MaturityLevel = types.StringValue(level)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ControlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ControlResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get category details to get the name
	category, err := r.client.GetControlCategory(data.CategoryID.ValueString())
	if err != nil {
		addClientError(&resp.Diagnostics, "get control category", err)
		return
	}

	// Build update request
	updateReq := &client.ControlUpdateRequest{
		ControlName:                data.Name.ValueString(),
		ControlDescription:         data.Description.ValueString(),
		ControlFrameworkCategory:   category.CategoryName,
		ControlFrameworkCategoryID: data.CategoryID.ValueString(),
	}

	// Terraform owns the attribute: absent means no owners.
	owners := []string{}
	if !data.Owners.IsNull() && !data.Owners.IsUnknown() {
		resp.Diagnostics.Append(data.Owners.ElementsAs(ctx, &owners, false)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Call API
	if _, err := r.client.UpdateControl(data.FrameworkID.ValueString(), data.ControlID.ValueString(), updateReq); err != nil {
		addClientError(&resp.Diagnostics, "update control", err)
		return
	}

	if err := r.client.SetControlOwners(data.ControlID.ValueString(), owners); err != nil {
		addClientError(&resp.Diagnostics, "set control owners", err)
		return
	}

	// An absent attribute clears the maturity level.
	if err := r.client.SetControlMaturityLevel(data.ControlID.ValueString(), data.MaturityLevel.ValueString()); err != nil {
		addClientError(&resp.Diagnostics, "set control maturity level", err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ControlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ControlResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteControl(data.FrameworkID.ValueString(), data.ControlID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		addClientError(&resp.Diagnostics, "delete control", err)
		return
	}
}

func (r *ControlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: framework_id/control_id
	// Example: framework_14421/control_55312
	idParts := splitImportID(req.ID, 2)
	if idParts == nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'framework_id/control_id', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("framework_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("control_id"), idParts[1])...)
}

// splitImportID splits an import ID by "/" and returns exactly expectedParts
// non-empty parts, or nil for a malformed ID. Empty segments are rejected so
// typos like "a//b" or a trailing slash fail loudly instead of importing.
func splitImportID(id string, expectedParts int) []string {
	parts := strings.Split(id, "/")
	if len(parts) != expectedParts {
		return nil
	}
	for _, p := range parts {
		if p == "" {
			return nil
		}
	}
	return parts
}
