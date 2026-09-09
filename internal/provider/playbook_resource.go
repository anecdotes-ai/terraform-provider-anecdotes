// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PlaybookResource{}
var _ resource.ResourceWithImportState = &PlaybookResource{}
var _ resource.ResourceWithValidateConfig = &PlaybookResource{}

func NewPlaybookResource() resource.Resource {
	return &PlaybookResource{}
}

// PlaybookResource defines the resource implementation.
type PlaybookResource struct {
	client *client.AnecdotesClient
}

// PlaybookResourceModel describes the resource data model.
type PlaybookResourceModel struct {
	PlaybookID          types.String `tfsdk:"playbook_id"`
	Title               types.String `tfsdk:"title"`
	Description         types.String `tfsdk:"description"`
	Active              types.Bool   `tfsdk:"active"`
	Type                types.String `tfsdk:"type"`
	Status              types.String `tfsdk:"status"`
	RestrictedFeatures  types.List   `tfsdk:"restricted_features"`
	CreatedBy           types.String `tfsdk:"created_by"`
	CreationTimestamp   types.String `tfsdk:"creation_timestamp"`
	LastUpdatedBy       types.String `tfsdk:"last_updated_by"`
	LastUpdateTimestamp types.String `tfsdk:"last_update_timestamp"`
	ScheduleConfig      types.Object `tfsdk:"schedule_config"`
	Steps               types.List   `tfsdk:"steps"`
}

// PlaybookScheduleModel describes the schedule_config nested attribute.
type PlaybookScheduleModel struct {
	Period    types.String      `tfsdk:"period"`
	Time      types.String      `tfsdk:"time"`
	Timezone  types.String      `tfsdk:"timezone"`
	StartDate timetypes.RFC3339 `tfsdk:"start_date"`
	EndsIn    types.String      `tfsdk:"ends_in"`
	EndDate   timetypes.RFC3339 `tfsdk:"end_date"`
}

// PlaybookStepModel describes one element of the steps nested attribute.
type PlaybookStepModel struct {
	StepID               types.String         `tfsdk:"step_id"`
	Title                types.String         `tfsdk:"title"`
	TriggerEvent         types.String         `tfsdk:"trigger_event"`
	ActionType           types.String         `tfsdk:"action_type"`
	URLToTrigger         types.String         `tfsdk:"url_to_trigger"`
	InternalAction       types.Bool           `tfsdk:"internal_action"`
	FilterConfiguration  jsontypes.Normalized `tfsdk:"filter_configuration"`
	PayloadConfiguration jsontypes.Normalized `tfsdk:"payload_configuration"`
	HeadersConfiguration jsontypes.Normalized `tfsdk:"headers_configuration"`
	LastRunStatus        types.String         `tfsdk:"last_run_status"`
	LastRunTimestamp     types.String         `tfsdk:"last_run_timestamp"`
}

func scheduleAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"period":     types.StringType,
		"time":       types.StringType,
		"timezone":   types.StringType,
		"start_date": timetypes.RFC3339Type{},
		"ends_in":    types.StringType,
		"end_date":   timetypes.RFC3339Type{},
	}
}

func stepAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"step_id":               types.StringType,
		"title":                 types.StringType,
		"trigger_event":         types.StringType,
		"action_type":           types.StringType,
		"url_to_trigger":        types.StringType,
		"internal_action":       types.BoolType,
		"filter_configuration":  jsontypes.NormalizedType{},
		"payload_configuration": jsontypes.NormalizedType{},
		"headers_configuration": jsontypes.NormalizedType{},
		"last_run_status":       types.StringType,
		"last_run_timestamp":    types.StringType,
	}
}

func (r *PlaybookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbook"
}

func (r *PlaybookResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anecdotes Playbook: an automation that runs one or more steps when a platform event fires or on a schedule.",
		MarkdownDescription: `
Manages an Anecdotes Playbook: an automation that runs one or more steps when a platform
event fires or on a schedule.

## Relationships

` + "```" + `
Playbook (this resource)
    │
    └── Step (one or more, nested in this resource)
            ├── trigger_event  — a platform event, or another step's step_id to chain
            └── action_type    — an internal action, or an outbound webhook
` + "```" + `

Use the ` + "`anecdotes_playbook_library`" + ` and ` + "`anecdotes_playbook_action_library`" + `
data sources to discover the valid ` + "`trigger_event`" + ` and ` + "`action_type`" + ` values.

## Key Concept: Step Membership Is Fixed

Steps can be edited in place, but they cannot be added to or removed from a playbook
after it is created. Changing how many steps a playbook has, or which ` + "`step_id`" + `s
it carries, replaces the playbook.

## Key Concept: Chaining Steps

A step runs either when a platform event fires or when another step completes. To chain,
set the second step's ` + "`trigger_event`" + ` to the first step's ` + "`step_id`" + `,
which means assigning ` + "`step_id`" + ` explicitly rather than letting it be generated.

## Key Concept: Scheduled Playbooks

A playbook runs on a schedule when its first step's ` + "`trigger_event`" + ` is
` + "`ScheduledPlaybookTriggered`" + ` and ` + "`schedule_config`" + ` is set. The
platform owns that step's ` + "`filter_configuration`" + `, so it cannot be configured.
A schedule cannot be removed once set; removing ` + "`schedule_config`" + ` replaces the
playbook.
`,

		Attributes: map[string]schema.Attribute{
			"playbook_id": schema.StringAttribute{
				Description: "The unique identifier of the playbook in Anecdotes.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"title": schema.StringAttribute{
				Description: "The human-readable name of the playbook.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"description": schema.StringAttribute{
				Description: "The description of the playbook. Cannot be empty.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"active": schema.BoolAttribute{
				Description: "Whether the playbook runs when its trigger fires. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},

			"type": schema.StringAttribute{
				Description: "The type of the automation. Always 'playbook' for this resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"status": schema.StringAttribute{
				Description: "The publication status of the playbook.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"restricted_features": schema.ListAttribute{
				Description: "Operations the Anecdotes platform restricts on this playbook.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},

			"created_by": schema.StringAttribute{
				Description: "The email of the user who created the playbook.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"creation_timestamp": schema.StringAttribute{
				Description: "When the playbook was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"last_updated_by": schema.StringAttribute{
				Description: "The email of the user who last updated the playbook.",
				Computed:    true,
			},

			"last_update_timestamp": schema.StringAttribute{
				Description: "When the playbook was last updated.",
				Computed:    true,
			},

			"schedule_config": schema.SingleNestedAttribute{
				Description: "The recurring schedule of the playbook. Only valid when the first step's trigger_event is ScheduledPlaybookTriggered. Removing it replaces the playbook.",
				Optional:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplaceIf(scheduleRemovedRequiresReplace,
						"A schedule cannot be removed from an existing playbook.",
						"A schedule cannot be removed from an existing playbook."),
				},
				Attributes: map[string]schema.Attribute{
					"period": schema.StringAttribute{
						Description: "How often the playbook runs.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf(client.ValidPlaybookSchedulePeriods()...),
						},
					},
					"time": schema.StringAttribute{
						Description: "The time of day the playbook runs, as HH:MM in 24-hour format.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.RegexMatches(regexp.MustCompile(`^\d{2}:\d{2}$`),
								"must be HH:MM in 24-hour format, for example 09:30"),
						},
					},
					"timezone": schema.StringAttribute{
						Description:         "The IANA timezone the schedule runs in, for example America/New_York. Defaults to UTC.",
						MarkdownDescription: "The IANA timezone the schedule runs in, for example `America/New_York`. Defaults to `UTC`.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("UTC"),
						Validators:          []validator.String{currentTimezoneValidator{}},
					},
					"start_date": schema.StringAttribute{
						Description: "The RFC 3339 timestamp from which the schedule is active, in UTC. The recurring day and month are derived from it.",
						Required:    true,
						CustomType:  timetypes.RFC3339Type{},
						Validators:  []validator.String{utcTimestampValidator{}},
					},
					"ends_in": schema.StringAttribute{
						Description: "How long after start_date the schedule expires. Conflicts with end_date.",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf(client.ValidPlaybookScheduleEndsIn()...),
							stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("end_date")),
						},
					},
					"end_date": schema.StringAttribute{
						Description: "The RFC 3339 timestamp at which the schedule expires, in UTC. Derived from start_date and ends_in when only ends_in is set.",
						Optional:    true,
						Computed:    true,
						CustomType:  timetypes.RFC3339Type{},
						Validators:  []validator.String{utcTimestampValidator{}},
					},
				},
			},

			"steps": schema.ListNestedAttribute{
				Description: "The steps of the playbook, in order. At least one is required. Steps cannot be added or removed after creation.",
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplaceIf(stepMembershipRequiresReplace,
						"Steps cannot be added to or removed from an existing playbook.",
						"Steps cannot be added to or removed from an existing playbook."),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"step_id": schema.StringAttribute{
							Description: "The unique identifier of the step, as a version 4 UUID. Generated when not set. Set it explicitly to chain another step to this one.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
							Validators: []validator.String{
								stringvalidator.RegexMatches(uuidV4Pattern, "must be a version 4 UUID"),
							},
						},
						"title": schema.StringAttribute{
							Description: "The human-readable name of the step.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
						"trigger_event": schema.StringAttribute{
							Description: "What starts this step: a platform event from the anecdotes_playbook_library data source, or the step_id of another step in this playbook.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
						"action_type": schema.StringAttribute{
							Description: "What this step does. Defaults to webhook.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(webhookActionType),
							Validators: []validator.String{
								stringvalidator.OneOf(client.ValidPlaybookStepActionTypes()...),
							},
						},
						"url_to_trigger": schema.StringAttribute{
							Description: "The URL a webhook step posts to. Must be publicly reachable over https. Leave unset for an internal action.",
							Optional:    true,
						},
						"internal_action": schema.BoolAttribute{
							Description: "Whether the step runs inside the Anecdotes platform rather than posting to an external URL. Derived from url_to_trigger.",
							Computed:    true,
							PlanModifiers: []planmodifier.Bool{
								internalActionFromURL{},
							},
						},
						"filter_configuration": schema.StringAttribute{
							Description: "A JSON object restricting which events run this step. Owned by the platform on a scheduled playbook's first step.",
							Optional:    true,
							Computed:    true,
							CustomType:  jsontypes.NormalizedType{},
							Validators:  []validator.String{jsonObjectValidator{}},
						},
						"payload_configuration": schema.StringAttribute{
							Description: "A JSON object describing the payload the step sends.",
							Optional:    true,
							Computed:    true,
							CustomType:  jsontypes.NormalizedType{},
							Validators:  []validator.String{playbookConfigurationValidator{}},
						},
						"headers_configuration": schema.StringAttribute{
							Description: "A JSON object describing the headers a webhook step sends.",
							Optional:    true,
							Computed:    true,
							CustomType:  jsontypes.NormalizedType{},
							Validators:  []validator.String{playbookConfigurationValidator{}},
						},
						"last_run_status": schema.StringAttribute{
							Description: "The outcome of the step's most recent run.",
							Computed:    true,
						},
						"last_run_timestamp": schema.StringAttribute{
							Description: "When the step last ran.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *PlaybookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, "Resource", &resp.Diagnostics)
}

func (r *PlaybookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PlaybookResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	steps := stepsFromModel(ctx, data.Steps, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.PlaybookCreateRequest{
		PlaybookTitle:       data.Title.ValueString(),
		PlaybookDescription: data.Description.ValueString(),
		ScheduleConfig:      scheduleFromModel(ctx, data.ScheduleConfig, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Every step is given an id up front so the configured order can be
	// restored from the created playbook by those ids.
	order := make([]string, 0, len(steps))
	for _, step := range steps {
		stepID := step.StepID.ValueString()
		if stepID == "" {
			generated, err := uuid.NewRandom()
			if err != nil {
				resp.Diagnostics.AddError("Configuration Error",
					fmt.Sprintf("Unable to generate a step identifier: %s", err))
				return
			}
			stepID = generated.String()
		}
		order = append(order, stepID)

		createReq.Steps = append(createReq.Steps, client.PlaybookStepInput{
			StepID:               stepID,
			StepTitle:            step.Title.ValueString(),
			StepTriggerEvent:     step.TriggerEvent.ValueString(),
			StepActionType:       step.ActionType.ValueString(),
			StepURLToTrigger:     step.URLToTrigger.ValueString(),
			FilterConfiguration:  configurationToMap(step.FilterConfiguration, &resp.Diagnostics),
			PayloadConfiguration: configurationToMap(step.PayloadConfiguration, &resp.Diagnostics),
			HeadersConfiguration: configurationToMap(step.HeadersConfiguration, &resp.Diagnostics),
		})
	}
	if resp.Diagnostics.HasError() {
		return
	}

	playbook, err := r.client.CreatePlaybook(ctx, createReq)
	if err != nil {
		addClientError(&resp.Diagnostics, "create playbook", err)
		return
	}

	// Read the planned value before state mapping replaces it with the value
	// the playbook was created with.
	wantActive := data.Active.ValueBool()

	r.setPlaybookState(ctx, &data, playbook, order, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// A playbook is always created enabled, so disabling it is a follow-up
	// write. State is recorded first: the playbook already exists, and a
	// failure here must not leave it untracked.
	if !wantActive {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		if resp.Diagnostics.HasError() {
			return
		}

		active := false
		playbook, err = r.client.UpdatePlaybook(ctx, playbook.PlaybookID, &client.PlaybookUpdateRequest{Active: &active})
		if err != nil {
			addClientError(&resp.Diagnostics, "disable playbook", err)
			return
		}

		r.setPlaybookState(ctx, &data, playbook, order, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PlaybookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PlaybookResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	playbook, err := r.client.GetPlaybook(ctx, data.PlaybookID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addClientError(&resp.Diagnostics, "read playbook", err)
		return
	}

	r.setPlaybookState(ctx, &data, playbook, stepIDs(ctx, data.Steps, &resp.Diagnostics), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PlaybookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state PlaybookResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planSteps := stepsFromModel(ctx, data.Steps, &resp.Diagnostics)
	stateSteps := stepsFromModel(ctx, state.Steps, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	title := data.Title.ValueString()
	description := data.Description.ValueString()
	active := data.Active.ValueBool()
	updateReq := &client.PlaybookUpdateRequest{
		PlaybookTitle:       &title,
		PlaybookDescription: &description,
		Active:              &active,
		ScheduleConfig:      scheduleFromModel(ctx, data.ScheduleConfig, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Step membership is fixed, so each planned step updates the step holding
	// the same position in state.
	order := make([]string, 0, len(planSteps))
	for i, step := range planSteps {
		stepID := step.StepID.ValueString()
		if stepID == "" && i < len(stateSteps) {
			stepID = stateSteps[i].StepID.ValueString()
		}

		order = append(order, stepID)

		title := step.Title.ValueString()
		trigger := step.TriggerEvent.ValueString()
		action := step.ActionType.ValueString()
		internal := step.URLToTrigger.IsNull() || step.URLToTrigger.IsUnknown()

		update := client.PlaybookStepUpdate{
			StepID:               stepID,
			StepTitle:            &title,
			StepTriggerEvent:     &trigger,
			StepActionType:       &action,
			InternalAction:       &internal,
			FilterConfiguration:  configurationToMapPtr(step.FilterConfiguration, &resp.Diagnostics),
			PayloadConfiguration: configurationToMapPtr(step.PayloadConfiguration, &resp.Diagnostics),
			HeadersConfiguration: configurationToMapPtr(step.HeadersConfiguration, &resp.Diagnostics),
		}
		if url := step.URLToTrigger.ValueString(); url != "" {
			update.StepURLToTrigger = &url
		}
		updateReq.Steps = append(updateReq.Steps, update)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	playbook, err := r.client.UpdatePlaybook(ctx, data.PlaybookID.ValueString(), updateReq)
	if err != nil {
		addClientError(&resp.Diagnostics, "update playbook", err)
		return
	}

	r.setPlaybookState(ctx, &data, playbook, order, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PlaybookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PlaybookResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePlaybook(ctx, data.PlaybookID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		addClientError(&resp.Diagnostics, "delete playbook", err)
		return
	}
}

func (r *PlaybookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("playbook_id"), req, resp)
}

// ValidateConfig rejects step and schedule combinations that cannot be
// applied as written.
func (r *PlaybookResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data PlaybookResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.Steps.IsNull() || data.Steps.IsUnknown() {
		return
	}

	steps := stepsFromModel(ctx, data.Steps, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || len(steps) == 0 {
		return
	}

	// The schedule checks compare the first step's trigger against the schedule,
	// so they wait until both are resolved. The step checks below do not.
	scheduleKnown := !steps[0].TriggerEvent.IsUnknown() && !data.ScheduleConfig.IsUnknown()
	scheduled := scheduleKnown && steps[0].TriggerEvent.ValueString() == client.ScheduledPlaybookTrigger
	hasSchedule := scheduleKnown && !data.ScheduleConfig.IsNull()

	if scheduleKnown && hasSchedule && !scheduled {
		resp.Diagnostics.AddAttributeError(path.Root("schedule_config"),
			"Schedule Requires a Scheduled Trigger",
			fmt.Sprintf("schedule_config is only valid when the first step's trigger_event is %q.", client.ScheduledPlaybookTrigger))
	}
	if scheduleKnown && scheduled && !hasSchedule {
		resp.Diagnostics.AddAttributeError(path.Root("schedule_config"),
			"Scheduled Trigger Requires a Schedule",
			fmt.Sprintf("A first step triggered by %q requires schedule_config to be set.", client.ScheduledPlaybookTrigger))
	}
	if scheduled && !steps[0].FilterConfiguration.IsNull() && !steps[0].FilterConfiguration.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("steps").AtListIndex(0).AtName("filter_configuration"),
			"Filter Is Owned by the Platform",
			"The first step of a scheduled playbook is filtered to that playbook's schedule, so filter_configuration cannot be set on it.")
	}

	// A webhook step posts to its trigger URL, so it cannot run without one.
	// The configuration is read before schema defaults are applied, so a step
	// that omits action_type carries no value here and takes the default.
	for i, step := range steps {
		if step.ActionType.IsUnknown() || step.URLToTrigger.IsUnknown() {
			continue
		}
		actionType := step.ActionType.ValueString()
		if step.ActionType.IsNull() {
			actionType = webhookActionType
		}
		if actionType == webhookActionType && step.URLToTrigger.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("steps").AtListIndex(i).AtName("url_to_trigger"),
				"Webhook Step Requires a URL",
				"A step whose action_type is \"webhook\" posts to url_to_trigger, so it must be set.")
		}
	}

	// A step may be chained to another step in the same playbook. An id that
	// belongs to no step in it would leave the step unreachable, and one used
	// twice describes two steps as the same step.
	known := map[string]bool{}
	for i, step := range steps {
		if step.StepID.IsNull() || step.StepID.IsUnknown() {
			continue
		}
		id := step.StepID.ValueString()
		if known[id] {
			resp.Diagnostics.AddAttributeError(path.Root("steps").AtListIndex(i).AtName("step_id"),
				"Duplicate Step Identifier",
				fmt.Sprintf("step_id %q is already used by another step in this playbook. "+
					"Each step needs its own identifier.", id))
		}
		known[id] = true
	}
	for i, step := range steps {
		if step.TriggerEvent.IsUnknown() {
			continue
		}
		trigger := step.TriggerEvent.ValueString()
		if !isUUID(trigger) || known[trigger] {
			continue
		}
		resp.Diagnostics.AddAttributeError(path.Root("steps").AtListIndex(i).AtName("trigger_event"),
			"Unknown Chained Step",
			fmt.Sprintf("trigger_event %q is an identifier, but no step in this playbook declares that step_id. "+
				"Chain a step by setting step_id explicitly on the step it follows.", trigger))
	}
}

// setPlaybookState sets the Terraform state from an API Playbook response.
func (r *PlaybookResource) setPlaybookState(ctx context.Context, data *PlaybookResourceModel, playbook *client.Playbook, order []string, diags *diag.Diagnostics) {
	data.PlaybookID = types.StringValue(playbook.PlaybookID)
	data.Title = types.StringValue(playbook.PlaybookTitle)
	data.Description = types.StringValue(playbook.PlaybookDescription)
	data.Active = types.BoolValue(playbook.Active)
	data.Type = types.StringValue(playbook.Type)
	data.Status = types.StringValue(playbook.Status)
	data.CreatedBy = types.StringValue(playbook.CreatedBy)
	data.CreationTimestamp = types.StringValue(playbook.CreationTimestamp)
	data.LastUpdatedBy = types.StringValue(playbook.LastUpdatedBy)
	data.LastUpdateTimestamp = types.StringValue(playbook.LastUpdateTimestamp)

	restricted, d := types.ListValueFrom(ctx, types.StringType, playbook.RestrictedFeatures)
	diags.Append(d...)
	data.RestrictedFeatures = restricted

	data.ScheduleConfig = scheduleToModel(ctx, playbook.ScheduleConfig, diags)
	data.Steps = stepsToModel(ctx, orderSteps(order, playbook.Steps), diags)
}

// orderSteps returns steps in the order the given ids define, preserving the
// order the configuration sets. Steps whose id is not listed keep their
// position relative to each other, last.
func orderSteps(ids []string, apiSteps []client.PlaybookStep) []client.PlaybookStep {
	if len(ids) == 0 {
		return apiSteps
	}

	remaining := make(map[string]client.PlaybookStep, len(apiSteps))
	for _, step := range apiSteps {
		remaining[step.StepID] = step
	}

	ordered := make([]client.PlaybookStep, 0, len(apiSteps))
	for _, id := range ids {
		if match, ok := remaining[id]; ok {
			ordered = append(ordered, match)
			delete(remaining, id)
		}
	}
	for _, step := range apiSteps {
		if _, ok := remaining[step.StepID]; ok {
			ordered = append(ordered, step)
		}
	}

	return ordered
}

// stepIDs returns the ids the steps attribute holds, skipping any not yet known.
func stepIDs(ctx context.Context, steps types.List, diags *diag.Diagnostics) []string {
	ids := make([]string, 0)
	for _, step := range stepsFromModel(ctx, steps, diags) {
		if !step.StepID.IsNull() && !step.StepID.IsUnknown() {
			ids = append(ids, step.StepID.ValueString())
		}
	}
	return ids
}

// stepMembershipRequiresReplace reports whether the planned steps carry a
// different number of steps, or a different set of declared ids, than state.
func stepMembershipRequiresReplace(ctx context.Context, req planmodifier.ListRequest, resp *listplanmodifier.RequiresReplaceIfFuncResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, state []PlaybookStepModel
	resp.Diagnostics.Append(req.PlanValue.ElementsAs(ctx, &plan, false)...)
	resp.Diagnostics.Append(req.StateValue.ElementsAs(ctx, &state, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(plan) != len(state) {
		resp.RequiresReplace = true
		return
	}

	stateIDs := map[string]bool{}
	for _, step := range state {
		stateIDs[step.StepID.ValueString()] = true
	}
	for _, step := range plan {
		if step.StepID.IsNull() || step.StepID.IsUnknown() {
			continue
		}
		if !stateIDs[step.StepID.ValueString()] {
			resp.RequiresReplace = true
			return
		}
	}
}

// scheduleRemovedRequiresReplace reports whether a schedule is being removed.
func scheduleRemovedRequiresReplace(ctx context.Context, req planmodifier.ObjectRequest, resp *objectplanmodifier.RequiresReplaceIfFuncResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	resp.RequiresReplace = !req.StateValue.IsNull() && req.PlanValue.IsNull()
}

// stepsFromModel decodes the steps attribute into its element models.
func stepsFromModel(ctx context.Context, steps types.List, diags *diag.Diagnostics) []PlaybookStepModel {
	if steps.IsNull() || steps.IsUnknown() {
		return nil
	}

	var models []PlaybookStepModel
	diags.Append(steps.ElementsAs(ctx, &models, false)...)
	return models
}

// stepsToModel builds the steps attribute from an API response.
func stepsToModel(ctx context.Context, steps []client.PlaybookStep, diags *diag.Diagnostics) types.List {
	models := make([]PlaybookStepModel, 0, len(steps))
	for _, step := range steps {
		lastRun := types.StringNull()
		if step.LastRunTimestamp != nil {
			lastRun = types.StringValue(*step.LastRunTimestamp)
		}

		// An internal step has no trigger URL of its own.
		url := types.StringNull()
		if !step.InternalAction {
			url = types.StringValue(step.StepURLToTrigger)
		}

		models = append(models, PlaybookStepModel{
			StepID:               types.StringValue(step.StepID),
			Title:                types.StringValue(step.StepTitle),
			TriggerEvent:         types.StringValue(step.StepTriggerEvent),
			ActionType:           types.StringValue(step.StepActionType),
			URLToTrigger:         url,
			InternalAction:       types.BoolValue(step.InternalAction),
			FilterConfiguration:  configurationFromMap(step.FilterConfiguration, diags),
			PayloadConfiguration: configurationFromMap(step.PayloadConfiguration, diags),
			HeadersConfiguration: configurationFromMap(step.HeadersConfiguration, diags),
			LastRunStatus:        types.StringValue(step.LastRunStatus),
			LastRunTimestamp:     lastRun,
		})
	}

	list, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: stepAttrTypes()}, models)
	diags.Append(d...)
	return list
}

// scheduleFromModel builds the schedule request from the schedule_config attribute.
func scheduleFromModel(ctx context.Context, schedule types.Object, diags *diag.Diagnostics) *client.PlaybookScheduleConfig {
	if schedule.IsNull() || schedule.IsUnknown() {
		return nil
	}

	var model PlaybookScheduleModel
	diags.Append(schedule.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil
	}

	config := &client.PlaybookScheduleConfig{
		Period:    model.Period.ValueString(),
		Time:      model.Time.ValueString(),
		Timezone:  model.Timezone.ValueString(),
		StartDate: model.StartDate.ValueString(),
	}
	// The two expiry forms are mutually exclusive. end_date is derived from
	// ends_in, so sending the derived value back alongside it is rejected.
	switch {
	case !model.EndsIn.IsNull() && !model.EndsIn.IsUnknown():
		endsIn := model.EndsIn.ValueString()
		config.EndsIn = &endsIn
	case !model.EndDate.IsNull() && !model.EndDate.IsUnknown():
		endDate := model.EndDate.ValueString()
		config.EndDate = &endDate
	}

	return config
}

// scheduleToModel builds the schedule_config attribute from an API response.
func scheduleToModel(ctx context.Context, schedule *client.PlaybookScheduleConfig, diags *diag.Diagnostics) types.Object {
	if schedule == nil {
		return types.ObjectNull(scheduleAttrTypes())
	}

	startDate := parseAPITimestamp(schedule.StartDate, diags)

	model := PlaybookScheduleModel{
		Period:    types.StringValue(schedule.Period),
		Time:      types.StringValue(schedule.Time),
		Timezone:  types.StringValue(schedule.Timezone),
		StartDate: startDate,
		EndsIn:    types.StringNull(),
		EndDate:   timetypes.NewRFC3339Null(),
	}
	if schedule.EndsIn != nil {
		model.EndsIn = types.StringValue(*schedule.EndsIn)
	}
	if schedule.EndDate != nil {
		model.EndDate = parseAPITimestamp(*schedule.EndDate, diags)
	}

	object, d := types.ObjectValueFrom(ctx, scheduleAttrTypes(), model)
	diags.Append(d...)
	return object
}

// configurationToMap decodes a configuration attribute into a request map.
func configurationToMap(configuration jsontypes.Normalized, diags *diag.Diagnostics) map[string]interface{} {
	if configuration.IsNull() || configuration.IsUnknown() {
		return nil
	}

	var decoded map[string]interface{}
	diags.Append(configuration.Unmarshal(&decoded)...)
	return decoded
}

// configurationToMapPtr decodes a configuration attribute for an update, where
// an absent value leaves the stored configuration untouched.
func configurationToMapPtr(configuration jsontypes.Normalized, diags *diag.Diagnostics) *map[string]interface{} {
	if configuration.IsNull() || configuration.IsUnknown() {
		return nil
	}

	decoded := configurationToMap(configuration, diags)
	if decoded == nil {
		decoded = map[string]interface{}{}
	}
	return &decoded
}

// configurationFromMap renders a configuration from an API response.
func configurationFromMap(configuration map[string]interface{}, diags *diag.Diagnostics) jsontypes.Normalized {
	if configuration == nil {
		configuration = map[string]interface{}{}
	}

	encoded, err := json.Marshal(configuration)
	if err != nil {
		diags.AddError("Invalid Step Configuration",
			fmt.Sprintf("Unable to read a step configuration returned by Anecdotes: %s", err))
		return jsontypes.NewNormalizedNull()
	}

	return jsontypes.NewNormalizedValue(string(encoded))
}

// isUUID reports whether s has the shape of a step identifier.
func isUUID(s string) bool {
	return uuidPattern.MatchString(s)
}

// webhookActionType is the action that posts to an outbound URL.
const webhookActionType = "webhook"

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// uuidV4Pattern matches the identifier form the platform accepts for a step.
var uuidV4Pattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

// jsonObjectValidator requires a configuration to be a JSON object.
type jsonObjectValidator struct{}

func (v jsonObjectValidator) Description(ctx context.Context) string {
	return "must be a JSON object"
}

func (v jsonObjectValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v jsonObjectValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	_, _ = decodeConfiguration(req, resp)
}

// decodeConfiguration decodes a configuration attribute, reporting anything
// that is not a JSON object.
func decodeConfiguration(req validator.StringRequest, resp *validator.StringResponse) (map[string]interface{}, bool) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return nil, false
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(req.ConfigValue.ValueString()), &decoded); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Step Configuration",
			fmt.Sprintf("Must be a JSON object: %s", err))
		return nil, false
	}
	if decoded == nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Step Configuration",
			"Must be a JSON object, for example jsonencode({ key = \"value\" }).")
		return nil, false
	}

	return decoded, true
}

// playbookConfigurationValidator rejects a configuration whose top-level
// entries would not be stored. It applies to the configurations that drop
// them, not to a filter, which keeps them.
type playbookConfigurationValidator struct{}

func (v playbookConfigurationValidator) Description(ctx context.Context) string {
	return "must be a JSON object whose top-level values are all non-empty"
}

func (v playbookConfigurationValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v playbookConfigurationValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	decoded, ok := decodeConfiguration(req, resp)
	if !ok {
		return
	}

	for key, value := range decoded {
		if key == "" {
			resp.Diagnostics.AddAttributeError(req.Path, "Empty Configuration Key",
				"A top-level key cannot be empty: Anecdotes does not store it.")
			continue
		}
		if isEmptyJSONValue(value) {
			resp.Diagnostics.AddAttributeError(req.Path, "Empty Configuration Value",
				fmt.Sprintf("The top-level key %q has an empty value. Anecdotes does not store it, so remove the key instead.", key))
		}
	}
}

// isEmptyJSONValue reports whether a top-level configuration value is empty.
func isEmptyJSONValue(value interface{}) bool {
	switch v := value.(type) {
	case nil:
		return true
	case bool:
		return !v
	case float64:
		return v == 0
	case string:
		return v == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	}
	return false
}

// utcTimestampValidator requires a timestamp to carry a UTC offset. A schedule
// is stored in UTC, so an offset timestamp would be read back in a different
// form than it was written.
type utcTimestampValidator struct{}

func (v utcTimestampValidator) Description(ctx context.Context) string {
	return "must be an RFC 3339 timestamp in UTC, for example 2026-09-07T09:30:00Z"
}

func (v utcTimestampValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v utcTimestampValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	parsed, err := time.Parse(time.RFC3339, req.ConfigValue.ValueString())
	if err != nil {
		return // the RFC3339 type reports the format error
	}
	if _, offset := parsed.Zone(); offset != 0 {
		resp.Diagnostics.AddAttributeError(req.Path, "Timestamp Must Be UTC",
			fmt.Sprintf("A schedule is stored in UTC, so %q would be read back as %s. Write it in UTC instead.",
				req.ConfigValue.ValueString(), parsed.UTC().Format(time.RFC3339)))
	}
}

// deprecatedTimezones maps the IANA names that are stored under a current name
// to that name. It mirrors the set of names the platform rewrites, so it needs
// updating whenever that set grows; a name missing here is reported as a
// mismatch at apply time instead of being rejected at plan time.
var deprecatedTimezones = map[string]string{
	"Europe/Kiev":   "Europe/Kyiv",
	"Asia/Calcutta": "Asia/Kolkata",
	"Asia/Saigon":   "Asia/Ho_Chi_Minh",
	"US/Eastern":    "America/New_York",
	"US/Central":    "America/Chicago",
	"US/Mountain":   "America/Denver",
	"US/Pacific":    "America/Los_Angeles",
}

// currentTimezoneValidator rejects a deprecated IANA name, which is stored
// under its current name and would be read back as that name.
type currentTimezoneValidator struct{}

func (v currentTimezoneValidator) Description(ctx context.Context) string {
	return "must be a current IANA timezone name"
}

func (v currentTimezoneValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v currentTimezoneValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if current, ok := deprecatedTimezones[req.ConfigValue.ValueString()]; ok {
		resp.Diagnostics.AddAttributeError(req.Path, "Deprecated Timezone Name",
			fmt.Sprintf("%q is stored as %q. Use %q instead.",
				req.ConfigValue.ValueString(), current, current))
	}
}

// parseAPITimestamp converts a timestamp reported by the API, reporting a
// value that is not RFC 3339 rather than panicking on it.
func parseAPITimestamp(value string, diags *diag.Diagnostics) timetypes.RFC3339 {
	parsed, d := timetypes.NewRFC3339Value(value)
	diags.Append(d...)
	if d.HasError() {
		return timetypes.NewRFC3339Null()
	}
	return parsed
}

// internalActionFromURL plans internal_action from the step's trigger URL, the
// same way it is derived when the step is written. Pinning it to the recorded
// value instead would report a step as external after it has been converted.
type internalActionFromURL struct{}

func (m internalActionFromURL) Description(ctx context.Context) string {
	return "derived from url_to_trigger"
}

func (m internalActionFromURL) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m internalActionFromURL) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	var url types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, req.Path.ParentPath().AtName("url_to_trigger"), &url)...)
	if resp.Diagnostics.HasError() || url.IsUnknown() {
		return
	}

	resp.PlanValue = types.BoolValue(url.IsNull())
}
