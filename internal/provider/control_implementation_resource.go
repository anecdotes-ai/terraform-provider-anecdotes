// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ControlImplementationResource{}
var _ resource.ResourceWithImportState = &ControlImplementationResource{}

func NewControlImplementationResource() resource.Resource {
	return &ControlImplementationResource{}
}

// ControlImplementationResource manages the implementation text of an existing
// control. It never creates or deletes the control itself.
type ControlImplementationResource struct {
	client *client.AnecdotesClient
}

// ControlImplementationResourceModel describes the resource data model.
type ControlImplementationResourceModel struct {
	ControlID             types.String `tfsdk:"control_id"`
	ControlImplementation types.String `tfsdk:"control_implementation"`
}

func (r *ControlImplementationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_control_implementation"
}

func (r *ControlImplementationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the implementation text of an Anecdotes control: how the control is implemented.",
		MarkdownDescription: `
Manages the implementation text of an Anecdotes control: how the control is implemented.

Works on any control, whether it was created in your account or provided with a framework.
The resource manages only the implementation text. It never creates or deletes the control.

Creating the resource sets the text on a control that has none. A control that already
has implementation text must be imported first, so existing text is never replaced without
an explicit ` + "`terraform import`" + `. Destroying the resource clears the text and leaves
the control and its other fields unchanged.

Use one ` + "`anecdotes_control_implementation`" + ` per control, across all configurations.
Two resources on the same control overwrite each other's text on every apply.

The text is stored as written. Line breaks are kept. Text formatted in the Anecdotes
application is stored as HTML, so an imported control's text can contain HTML tags.

## Relationships

` + "```" + `
Control (anecdotes_control, or provided with a framework)
    └── Implementation text (this resource)
` + "```" + `
`,

		Attributes: map[string]schema.Attribute{
			"control_id": schema.StringAttribute{
				Description:         "The ID of the control whose implementation text this resource manages (e.g., control_55312). Changing it replaces the resource.",
				MarkdownDescription: "The ID of the control whose implementation text this resource manages (e.g., `control_55312`). Changing it replaces the resource.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"control_implementation": schema.StringAttribute{
				Description:         "How the control is implemented. Line breaks are kept. To clear the text, remove the resource.",
				MarkdownDescription: "How the control is implemented. Line breaks are kept. To clear the text, remove the resource.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *ControlImplementationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, "Resource", &resp.Diagnostics)
}

func (r *ControlImplementationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ControlImplementationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	controlID := data.ControlID.ValueString()

	err := r.client.SetControlImplementationIfEmpty(ctx, controlID, data.ControlImplementation.ValueString())
	switch {
	case err == nil:
	case client.IsNotFound(err):
		resp.Diagnostics.AddError(
			"Control Not Found",
			fmt.Sprintf("No control with ID %s exists in this account.", controlID),
		)
		return
	case errors.Is(err, client.ErrImplementationAlreadySet):
		resp.Diagnostics.AddError(
			"Control Already Has Implementation Text",
			fmt.Sprintf(
				"Control %s already has implementation text. To manage it with Terraform, import it first:\n\n"+
					"  terraform import anecdotes_control_implementation.<name> %s\n\n"+
					"Then set control_implementation to the text you want.",
				controlID, controlID,
			),
		)
		return
	default:
		addClientError(&resp.Diagnostics, "set control implementation", err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ControlImplementationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ControlImplementationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	control, err := r.client.GetControl(ctx, "", data.ControlID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addClientError(&resp.Diagnostics, "read control implementation", err)
		return
	}

	// Stored exactly as read, including an empty value, so text cleared or
	// changed outside Terraform shows as a difference on the next plan.
	data.ControlImplementation = types.StringValue(control.ControlImplementation)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ControlImplementationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ControlImplementationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.SetControlImplementation(ctx, data.ControlID.ValueString(), data.ControlImplementation.ValueString()); err != nil {
		addClientError(&resp.Diagnostics, "set control implementation", err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ControlImplementationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ControlImplementationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.SetControlImplementation(ctx, data.ControlID.ValueString(), "")
	if err != nil && !client.IsNotFound(err) {
		addClientError(&resp.Diagnostics, "clear control implementation", err)
		return
	}
}

func (r *ControlImplementationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: control_id
	// Example: control_55312
	resource.ImportStatePassthroughID(ctx, path.Root("control_id"), req, resp)
}
