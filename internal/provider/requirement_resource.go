// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
var _ resource.Resource = &RequirementResource{}
var _ resource.ResourceWithImportState = &RequirementResource{}

func NewRequirementResource() resource.Resource {
	return &RequirementResource{}
}

// RequirementResource defines the resource implementation.
type RequirementResource struct {
	client *client.AnecdotesClient
}

// RequirementResourceModel describes the resource data model.
type RequirementResourceModel struct {
	// Core identification
	RequirementID types.String `tfsdk:"requirement_id"`
	Name          types.String `tfsdk:"name"`

	// Content
	Description types.String `tfsdk:"description"`
	Category    types.String `tfsdk:"category"`

	// Ownership
	Owners types.Set `tfsdk:"owners"`
}

func (r *RequirementResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_requirement"
}

func (r *RequirementResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anecdotes requirement in the Requirements Hub.",
		MarkdownDescription: `
Manages an Anecdotes requirement in the Requirements Hub.

A **Requirement** represents an operational action that needs to happen to ensure a control is being enforced. 
Requirements are standalone entities that can be linked to multiple controls across multiple frameworks.

## Relationships

` + "```" + `
Requirement (this resource)
    │
    ├── linked to Control A (Framework 1) via anecdotes_mapping_control_requirement
    ├── linked to Control B (Framework 1) via anecdotes_mapping_control_requirement
    └── linked to Control C (Framework 2) via anecdotes_mapping_control_requirement  ← Cross-framework!
` + "```" + `

## Key Concept: Shared Requirements

Requirements are **shared entities**. A single requirement like "MFA is enabled for all users" can satisfy 
controls across SOC2, ISO 27001, HIPAA, and custom frameworks simultaneously. This is the core of 
Anecdotes' cross-mapping architecture.

## Requirement Status Values

Status values can be customized per organization, but common defaults include:
- ` + "`Not started`" + ` - Requirement has not been addressed
- ` + "`In progress`" + ` - Requirement is being worked on
- ` + "`Completed`" + ` - Requirement has been fulfilled
`,

		Attributes: map[string]schema.Attribute{
			// Core identification
			"requirement_id": schema.StringAttribute{
				Description:         "The unique identifier of the requirement in Anecdotes (e.g., requirement_9012).",
				MarkdownDescription: "The unique identifier of the requirement in Anecdotes (e.g., `requirement_9012`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"name": schema.StringAttribute{
				Description: "The human-readable name/description of the requirement. This is displayed as the requirement title.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 500),
				},
			},

			// Content
			"description": schema.StringAttribute{
				Description: "A detailed description/help text explaining what this requirement entails. Supports HTML.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},

			"category": schema.StringAttribute{
				Description:         "The category this requirement belongs to. Must be one of the categories Anecdotes defines. Defaults to 'Custom Requirements'.",
				MarkdownDescription: "The category this requirement belongs to. One of the categories Anecdotes defines; requirements that do not fit one of them belong under `Custom Requirements`, which is also the default. Removing the attribute restores the default.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("Custom Requirements"),
				Validators: []validator.String{
					stringvalidator.OneOf(client.ValidRequirementCategories()...),
				},
			},

			// Ownership
			"owners": schema.SetAttribute{
				Description: "Email addresses of users responsible for this requirement. Order does not matter. Terraform owns this attribute: removing it clears the owners.",
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

func (r *RequirementResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RequirementResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RequirementResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planned := data

	// Build create request
	// API mapping: name → requirement_description, description → requirement_help
	createReq := &client.RequirementCreateRequest{
		RequirementDescription:       data.Name.ValueString(),        // Title/name goes to requirement_description
		RequirementHelp:              data.Description.ValueString(), // Description goes to requirement_help
		RequirementCategory:          data.Category.ValueString(),
		RequirementRelatedFrameworks: []string{}, // Required field, can be empty
	}

	// Handle owners list
	if !data.Owners.IsNull() && !data.Owners.IsUnknown() {
		var owners []string
		resp.Diagnostics.Append(data.Owners.ElementsAs(ctx, &owners, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.RequirementOwners = owners
	}

	// Call API
	requirement, recovered, err := r.client.CreateRequirement(createReq)
	if err != nil {
		addClientError(&resp.Diagnostics, "create requirement", err)
		return
	}

	// Set state from response
	r.setRequirementState(ctx, &data, requirement, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// A requirement recovered by name after an ambiguous error may differ from
	// what was planned. Keep the planned values so the resource stays tracked;
	// the next refresh reports the real values.
	if recovered {
		data.Name = planned.Name
		data.Description = planned.Description
		data.Category = planned.Category
		data.Owners = planned.Owners
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RequirementResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RequirementResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get requirement from API
	requirement, err := r.client.GetRequirement(data.RequirementID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addClientError(&resp.Diagnostics, "read requirement", err)
		return
	}

	// Set state from response
	r.setRequirementState(ctx, &data, requirement, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RequirementResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RequirementResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build update request
	// API mapping: name → requirement_description, description → requirement_help
	help := data.Description.ValueString()
	updateReq := &client.RequirementUpdateRequest{
		RequirementDescription: data.Name.ValueString(), // Title/name goes to requirement_description
		RequirementHelp:        &help,                   // Description goes to requirement_help
		RequirementCategory:    data.Category.ValueString(),
	}

	// Terraform owns the attribute: absent means no owners.
	owners := []string{}
	if !data.Owners.IsNull() && !data.Owners.IsUnknown() {
		resp.Diagnostics.Append(data.Owners.ElementsAs(ctx, &owners, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	updateReq.RequirementOwners = &owners

	// Call API
	requirement, err := r.client.UpdateRequirement(data.RequirementID.ValueString(), updateReq)
	if err != nil {
		addClientError(&resp.Diagnostics, "update requirement", err)
		return
	}

	// Set state from response
	r.setRequirementState(ctx, &data, requirement, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RequirementResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RequirementResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteRequirement(data.RequirementID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		addClientError(&resp.Diagnostics, "delete requirement", err)
		return
	}
}

func (r *RequirementResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("requirement_id"), req, resp)
}

// setRequirementState sets the Terraform state from an API Requirement response
func (r *RequirementResource) setRequirementState(ctx context.Context, data *RequirementResourceModel, requirement *client.Requirement, diags *diag.Diagnostics) {
	// Set IDs
	data.RequirementID = types.StringValue(requirement.RequirementID)

	// Set name (API returns it in requirement_name, which is derived from requirement_description)
	if requirement.RequirementName != "" {
		data.Name = types.StringValue(requirement.RequirementName)
	} else {
		data.Name = types.StringValue(requirement.RequirementDescription)
	}

	// Set content (requirement_help → description, requirement_category → category)
	data.Description = types.StringValue(requirement.RequirementHelp)
	data.Category = types.StringValue(requirement.RequirementCategory)

	// An empty set and an unset attribute are distinct.
	if len(requirement.RequirementOwners) > 0 {
		ownersSet, d := types.SetValueFrom(ctx, types.StringType, requirement.RequirementOwners)
		diags.Append(d...)
		data.Owners = ownersSet
	} else if !data.Owners.IsNull() {
		data.Owners = types.SetValueMust(types.StringType, []attr.Value{})
	} else {
		data.Owners = types.SetNull(types.StringType)
	}
}
