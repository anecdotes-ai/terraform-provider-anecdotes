// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &AnalysisRuleResource{}
var _ resource.ResourceWithImportState = &AnalysisRuleResource{}
var _ resource.ResourceWithValidateConfig = &AnalysisRuleResource{}

// Alert levels a rule may be authored with. The platform's alert scale also
// includes 3, 5 and 10, which describe the outcome of an evaluation rather than
// the severity a rule raises, so they are not offered here.
const (
	alertLevelWarning int64 = 30
	alertLevelGap     int64 = 50
)

// analysisRuleOriginLibrary marks a rule shipped with the platform rather than
// authored by the account.
const analysisRuleOriginLibrary = "library"

func NewAnalysisRuleResource() resource.Resource {
	return &AnalysisRuleResource{}
}

// AnalysisRuleResource defines the resource implementation.
type AnalysisRuleResource struct {
	client *client.AnecdotesClient
}

// AnalysisRuleResourceModel describes the resource data model.
type AnalysisRuleResourceModel struct {
	RuleID     types.String `tfsdk:"rule_id"`
	EvidenceID types.String `tfsdk:"evidence_id"`

	RuleName      types.String         `tfsdk:"rule_name"`
	RuleMessage   types.String         `tfsdk:"rule_message"`
	AlertLevel    types.Int64          `tfsdk:"alert_level"`
	RuleQueryType types.String         `tfsdk:"rule_query_type"`
	RuleQuery     jsontypes.Normalized `tfsdk:"rule_query"`
	RuleType      types.String         `tfsdk:"rule_type"`
	LibraryRuleID types.String         `tfsdk:"library_rule_id"`
	RuleState     types.String         `tfsdk:"rule_state"`

	AccountScopingType types.String `tfsdk:"account_scoping_type"`
	AccountScopingList types.Set    `tfsdk:"account_scoping_list"`

	RuleQueryStr     types.String `tfsdk:"rule_query_str"`
	RuleQueryMessage types.String `tfsdk:"rule_query_message"`
	RuleOrigin       types.String `tfsdk:"rule_origin"`
	LastUpdated      types.String `tfsdk:"last_updated"`
	LastUpdatedBy    types.String `tfsdk:"last_updated_by"`
}

func (r *AnalysisRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_analysis_rule"
}

func (r *AnalysisRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a custom Anecdotes Analysis Rule.",
		MarkdownDescription: `
Manages a custom Anecdotes Analysis Rule: a query evaluated against one evidence's
collected data, raising a gap or a warning on the rows that match.

## Relationships

` + "```" + `
Evidence (anecdotes_evidences data source)
  └── Analysis Rule (anecdotes_analysis_rule)   ← this resource
` + "```" + `

## Scope

This resource manages rules the account authors ( ` + "`rule_origin`" + ` ` + "`custom`" + ` ).
Rules shipped with the platform ( ` + "`rule_origin`" + ` ` + "`library`" + ` ) cannot be
created, updated or deleted through it; use the ` + "`anecdotes_analysis_rules`" + ` data
source to read them.
`,
		Attributes: map[string]schema.Attribute{
			"rule_id": schema.StringAttribute{
				Description: "The unique identifier of the analysis rule.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"evidence_id": schema.StringAttribute{
				Description: "The evidence this rule evaluates. Changing it replaces the rule.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rule_name": schema.StringAttribute{
				Description: "The rule's display name. Removing the attribute keeps the current value, which the platform will not clear.",
				Optional:    true,
				Computed:    true,
			},
			"rule_message": schema.StringAttribute{
				Description: "The message shown for rows the rule matches. Removing the attribute keeps the current value, which the platform will not clear.",
				Optional:    true,
				Computed:    true,
			},
			"alert_level": schema.Int64Attribute{
				Description: "Severity raised by the rule: 30 (warning) or 50 (gap). Defaults to 50.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(alertLevelGap),
				Validators: []validator.Int64{
					int64validator.OneOf(alertLevelWarning, alertLevelGap),
				},
			},
			"rule_query_type": schema.StringAttribute{
				Description: "The query language of `rule_query`: \"aql\", \"aqlext\" or \"pandas\". Defaults to \"aql\". Changing it replaces the rule.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("aql"),
				Validators: []validator.String{
					stringvalidator.OneOf("aql", "aqlext", "pandas"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rule_query": schema.StringAttribute{
				Description: "The rule query, as a JSON object. Its shape depends on `rule_query_type`.",
				MarkdownDescription: "The rule query, as a JSON object. Its shape depends on `rule_query_type`.\n\n" +
					"Compared as JSON rather than as text, so key order and whitespace do not " +
					"produce a diff. Value types are compared as written: `true` and `\"true\"` " +
					"are different, and the platform stores the typed form, so a quoted boolean " +
					"or number produces a permanent diff.",
				Required:   true,
				CustomType: jsontypes.NormalizedType{},
			},
			"rule_type": schema.StringAttribute{
				Description: "Optional processing type: \"uam\" or \"eid\". Changing it replaces the rule.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("uam", "eid"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"library_rule_id": schema.StringAttribute{
				Description: "The library rule this rule was derived from, recorded for reference. Changing it replaces the rule.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rule_state": schema.StringAttribute{
				Description: "Whether the rule is evaluated: \"active\" or \"inactive\". Defaults to \"active\".",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("active"),
				Validators: []validator.String{
					stringvalidator.OneOf("active", "inactive"),
				},
			},
			"account_scoping_type": schema.StringAttribute{
				Description: "Which accounts the rule applies to: \"all_accounts\", \"included_accounts\" or \"excluded_accounts\". Defaults to \"all_accounts\".",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("all_accounts"),
				Validators: []validator.String{
					stringvalidator.OneOf("all_accounts", "included_accounts", "excluded_accounts"),
				},
			},
			"account_scoping_list": schema.SetAttribute{
				Description: "Service instance IDs the rule is scoped to. Required when `account_scoping_type` is \"included_accounts\" or \"excluded_accounts\", and rejected by the platform if none of them belong to the evidence's service. The `anecdotes_evidences` data source reports the available IDs.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"rule_query_str": schema.StringAttribute{
				Description: "The query as the platform stored it, serialized as a JSON string.",
				Computed:    true,
			},
			"rule_query_message": schema.StringAttribute{
				Description: "Message associated with the stored query.",
				Computed:    true,
			},
			"rule_origin": schema.StringAttribute{
				Description: "Whether the rule ships with the platform (\"library\") or is authored by the account (\"custom\").",
				Computed:    true,
				// A rule's origin is fixed when it is created, so keeping the
				// known value stops every update reporting it as pending.
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "When the rule was last modified.",
				Computed:    true,
			},
			"last_updated_by": schema.StringAttribute{
				Description: "Who last modified the rule.",
				Computed:    true,
			},
		},
	}
}

// aqlQueryKeys are the keys an "aql" rule query is stored with. The platform
// keeps these and discards anything else, so a query carrying extra keys would
// apply as something other than what was written.
var aqlQueryKeys = map[string]struct{}{"operator": {}, "left": {}, "right": {}}

// collectUnsupportedAQLKeys walks a decoded "aql" query and records the paths of
// any keys outside aqlQueryKeys. Nested conditions use the same shape, so the
// walk recurses through objects and arrays alike.
func collectUnsupportedAQLKeys(v interface{}, at string, found *[]string) {
	switch node := v.(type) {
	case map[string]interface{}:
		for key, child := range node {
			where := strings.TrimPrefix(at+"."+key, ".")
			if _, ok := aqlQueryKeys[key]; !ok {
				*found = append(*found, where)
				continue
			}
			collectUnsupportedAQLKeys(child, where, found)
		}
	case []interface{}:
		for i, child := range node {
			collectUnsupportedAQLKeys(child, fmt.Sprintf("%s[%d]", at, i), found)
		}
	}
}

// collectUnsupportedAQLExtKeys walks an "aqlext" query and checks every
// condition it carries: the one under "filters", the one a manipulation carries
// in "filter_on_other", and both again inside a query built on top of another.
func collectUnsupportedAQLExtKeys(v interface{}, at string, found *[]string) {
	obj, ok := v.(map[string]interface{})
	if !ok {
		return
	}

	prefix := ""
	if at != "" {
		prefix = at + "."
	}

	if filters, ok := obj["filters"]; ok {
		collectUnsupportedAQLKeys(filters, prefix+"filters", found)
	}
	if manipulations, ok := obj["manipulations"].([]interface{}); ok {
		for i, m := range manipulations {
			step, ok := m.(map[string]interface{})
			if !ok {
				continue
			}
			if on, ok := step["filter_on_other"]; ok {
				collectUnsupportedAQLKeys(on, fmt.Sprintf("%smanipulations[%d].filter_on_other", prefix, i), found)
			}
		}
	}
	if base, ok := obj["base"]; ok {
		collectUnsupportedAQLExtKeys(base, prefix+"base", found)
	}
}

// validateRuleQuery rejects a query carrying keys the platform does not store,
// so the mismatch is reported against the attribute at plan time rather than as
// an unexplained difference after the rule is written.
//
// An "aql" query is a condition and is checked whole. An "aqlext" query is a
// different structure, but the condition under its "filters" key is an ordinary
// AQL condition and is stored the same way, so that subtree is checked too. A
// "pandas" query is an expression string and is checked for case instead.
func validateRuleQuery(data *AnalysisRuleResourceModel, diags *diag.Diagnostics) {
	if data.RuleQuery.IsNull() || data.RuleQuery.IsUnknown() || data.RuleQueryType.IsUnknown() {
		return
	}
	// An unset rule_query_type takes the schema default.
	queryType := data.RuleQueryType.ValueString()
	if data.RuleQueryType.IsNull() {
		queryType = "aql"
	}

	var decoded interface{}
	if err := json.Unmarshal([]byte(data.RuleQuery.ValueString()), &decoded); err != nil {
		// Malformed JSON is already reported by the attribute's own type.
		return
	}

	var found []string
	switch queryType {
	case "aql":
		collectUnsupportedAQLKeys(decoded, "", &found)
	case "aqlext":
		collectUnsupportedAQLExtKeys(decoded, "", &found)
	case "pandas":
		// A "pandas" query is a expression string, and it is stored lowercased.
		// Writing it in any other case would store a different expression than
		// the one configured, including inside quoted values, so the difference
		// is reported here rather than after the rule is written.
		if expr, ok := decoded.(string); ok && expr != strings.ToLower(expr) {
			diags.AddAttributeError(
				path.Root("rule_query"),
				"Rule Query Must Be Lower Case",
				fmt.Sprintf("A \"pandas\" rule query is stored lower-cased, so this query would be "+
					"stored as something other than what was written:\n\n  configured: %s\n  stored:     %s\n\n"+
					"Write the query in lower case. Note that quoted values are lowered too, so a "+
					"comparison against mixed-case data cannot be expressed this way.",
					expr, strings.ToLower(expr)),
			)
		}
		return
	default:
		return
	}
	if len(found) == 0 {
		return
	}
	sort.Strings(found)

	diags.AddAttributeError(
		path.Root("rule_query"),
		"Unsupported Keys In Rule Query",
		fmt.Sprintf("A rule condition is stored with only operator, left and right. "+
			"These keys would be discarded: %s\n\nRemove them. Note that \"aqlext\" is a "+
			"different structure rather than a richer condition, so moving a condition there "+
			"does not preserve them.", strings.Join(found, ", ")),
	)
}

// ValidateConfig rejects an account scoping configuration the platform cannot
// satisfy, so it surfaces at plan time rather than part-way through an apply.
func (r *AnalysisRuleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data AnalysisRuleResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateRuleQuery(&data, &resp.Diagnostics)

	// An unknown value is resolved during apply and cannot be judged here.
	if data.AccountScopingList.IsUnknown() {
		return
	}
	listed := !data.AccountScopingList.IsNull() && len(data.AccountScopingList.Elements()) > 0

	// An unset account_scoping_type takes the schema default.
	scopingType := data.AccountScopingType.ValueString()
	if data.AccountScopingType.IsNull() {
		scopingType = "all_accounts"
	}
	if data.AccountScopingType.IsUnknown() {
		return
	}

	switch scopingType {
	case "included_accounts", "excluded_accounts":
		if !listed {
			resp.Diagnostics.AddAttributeError(
				path.Root("account_scoping_list"),
				"Missing Account Scoping List",
				fmt.Sprintf("account_scoping_type is %q, which scopes the rule to specific accounts, "+
					"so account_scoping_list must name at least one service instance ID.", scopingType),
			)
		}
	case "all_accounts":
		// A rule that applies to every account carries no list, and the platform
		// discards one that is sent, so accepting it here would apply a
		// configuration the platform will not hold.
		if listed {
			resp.Diagnostics.AddAttributeError(
				path.Root("account_scoping_list"),
				"Unexpected Account Scoping List",
				"account_scoping_type is \"all_accounts\" (the default), which applies the rule to "+
					"every account, "+
					"so account_scoping_list must not be set. Remove it, or set account_scoping_type "+
					"to \"included_accounts\" or \"excluded_accounts\".",
			)
		}
	}
}

func (r *AnalysisRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, "Resource", &resp.Diagnostics)
}

// applyAnalysisRule maps the platform's view of a rule onto the resource model.
//
// rule_query is populated from the query the platform stored rather than left
// as configured, so a query edited outside Terraform shows up as a difference.
func applyAnalysisRule(ctx context.Context, diags *diag.Diagnostics, data *AnalysisRuleResourceModel, rule *client.AnalysisRule) {
	data.RuleID = types.StringValue(rule.RuleID)
	data.EvidenceID = types.StringValue(rule.EvidenceID)

	data.RuleName = stringOrNull(rule.RuleName)
	data.RuleMessage = stringOrNull(rule.RuleMessage)
	data.AlertLevel = types.Int64Value(rule.AlertLevel)
	data.RuleQueryType = stringOrNull(rule.RuleQueryType)
	data.RuleType = stringOrNull(rule.RuleType)
	data.LibraryRuleID = stringOrNull(rule.LibraryRuleID)
	data.RuleState = stringOrNull(rule.RuleState)

	data.AccountScopingType = stringOrNull(rule.AccountScopingType)
	data.AccountScopingList = stringSetFromAPI(ctx, diags, data.AccountScopingList, rule.AccountScopingList)

	// The platform re-serializes the query with its own key order and spacing.
	// Assigning it unconditionally is safe: the framework compares this response
	// against the prior value and keeps the prior one when the two mean the same
	// thing, so a rule that was merely re-rendered does not read as a change,
	// while a query edited outside Terraform does.
	if rule.RuleQueryStr != "" {
		data.RuleQuery = jsontypes.NewNormalizedValue(rule.RuleQueryStr)
	}

	data.RuleQueryStr = stringOrNull(rule.RuleQueryStr)
	data.RuleQueryMessage = stringOrNull(rule.RuleQueryMessage)
	data.RuleOrigin = stringOrNull(rule.RuleOrigin)
	data.LastUpdated = stringOrNull(rule.LastUpdated)
	data.LastUpdatedBy = stringOrNull(rule.LastUpdatedBy)
}

// scopingListFor returns the account scoping IDs to send. A rule that applies to
// every account carries no list, and the platform clears one that is sent, so
// nothing is sent in that case.
func scopingListFor(ctx context.Context, diags *diag.Diagnostics, data *AnalysisRuleResourceModel) []string {
	if data.AccountScopingType.ValueString() == "all_accounts" {
		return nil
	}
	if data.AccountScopingList.IsNull() || data.AccountScopingList.IsUnknown() {
		return nil
	}
	var list []string
	diags.Append(data.AccountScopingList.ElementsAs(ctx, &list, false)...)
	return list
}

// checkScopingAccepted reports the configured account scoping IDs the platform
// did not keep. It only keeps instances belonging to the evidence's service, and
// a silently shortened list would otherwise surface as an unexplained mismatch
// between the plan and the applied result.
func checkScopingAccepted(ctx context.Context, diags *diag.Diagnostics, data *AnalysisRuleResourceModel, rule *client.AnalysisRule) {
	if data.AccountScopingType.ValueString() == "all_accounts" {
		return
	}
	configured := scopingListFor(ctx, diags, data)
	if len(configured) == 0 || diags.HasError() {
		return
	}

	kept := make(map[string]struct{}, len(rule.AccountScopingList))
	for _, id := range rule.AccountScopingList {
		kept[id] = struct{}{}
	}

	var rejected []string
	for _, id := range configured {
		if _, ok := kept[id]; !ok {
			rejected = append(rejected, id)
		}
	}
	if len(rejected) == 0 {
		return
	}

	diags.AddAttributeError(
		path.Root("account_scoping_list"),
		"Account Scoping List Not Accepted",
		fmt.Sprintf("The platform did not accept these service instance IDs for evidence %s: %s\n\n"+
			"A rule can only be scoped to instances of the service that collects its evidence, and only "+
			"while those instances are installed. The `anecdotes_evidences` data source reports the "+
			"instances that collected each evidence.",
			rule.EvidenceID, strings.Join(rejected, ", ")),
	)
}

func (r *AnalysisRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AnalysisRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.AnalysisRuleCreateRequest{
		EvidenceID:         data.EvidenceID.ValueString(),
		RuleName:           data.RuleName.ValueString(),
		RuleMessage:        data.RuleMessage.ValueString(),
		AlertLevel:         data.AlertLevel.ValueInt64(),
		RuleQueryType:      data.RuleQueryType.ValueString(),
		RuleQuery:          []byte(data.RuleQuery.ValueString()),
		RuleType:           data.RuleType.ValueString(),
		LibraryRuleID:      data.LibraryRuleID.ValueString(),
		AccountScopingType: data.AccountScopingType.ValueString(),
		AccountScopingList: scopingListFor(ctx, &resp.Diagnostics, &data),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateAnalysisRule(ctx, createReq)
	if err != nil {
		addClientError(&resp.Diagnostics, "create analysis rule", err)
		return
	}

	// A new rule is always created active, so an inactive rule takes a second
	// call. The rule itself exists once the call above succeeds, so a failure
	// here is reported as a warning: the next plan sees rule_state differ and
	// retries just the state change. Reporting an error instead would mark the
	// rule for replacement, so the next apply would build a new one rather than
	// finish configuring this one.
	stateApplied := true
	if data.RuleState.ValueString() == "inactive" {
		if err := r.client.SetAnalysisRuleState(ctx, created.RuleID, "inactive"); err != nil {
			stateApplied = false
			resp.Diagnostics.AddWarning(
				"Analysis Rule Created But Not Deactivated",
				fmt.Sprintf("Rule %s was created but could not be set to inactive: %s\n\n"+
					"The rule is recorded in state as configured and is currently active on the "+
					"platform. The next plan reports the difference and applies the state change "+
					"on its own.", created.RuleID, err),
			)
		}
	}

	// The create response predates the state change, and the state endpoint's
	// own response cannot be relied on, so the rule is read back.
	rule, err := r.client.GetAnalysisRule(ctx, created.EvidenceID, created.RuleID)
	if err != nil {
		addClientError(&resp.Diagnostics, "read analysis rule after create", err)
		return
	}

	checkScopingAccepted(ctx, &resp.Diagnostics, &data, rule)

	plannedState := data.RuleState
	applyAnalysisRule(ctx, &resp.Diagnostics, &data, rule)
	// Recording the state the platform reports here would contradict the plan
	// and fail the apply, marking the rule for replacement. Keeping the planned
	// value lets the apply finish; the next read finds the rule still active and
	// plans the state change again. That read is what surfaces it, so a plan
	// that skips refreshing will not show the difference.
	if !stateApplied {
		data.RuleState = plannedState
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AnalysisRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AnalysisRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GetAnalysisRule(ctx, data.EvidenceID.ValueString(), data.RuleID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addClientError(&resp.Diagnostics, "read analysis rule", err)
		return
	}

	// Only rules the account authored can be updated or deleted. Catching a
	// library rule here covers the way one realistically arrives in state, an
	// import, and stops it before an update or destroy acts on a rule this
	// resource cannot manage.
	if rule.RuleOrigin == analysisRuleOriginLibrary {
		resp.Diagnostics.AddError(
			"Analysis rule cannot be managed by Terraform",
			fmt.Sprintf(
				"Analysis rule %s is provided by the Anecdotes platform. Only rules the account authors "+
					"can be updated or deleted, so Terraform cannot manage it.\n\n"+
					"Read it with the anecdotes_analysis_rule or anecdotes_analysis_rules data source instead. "+
					"If it is already in state, remove it with `terraform state rm` before applying again.",
				rule.RuleID,
			),
		)
		return
	}

	applyAnalysisRule(ctx, &resp.Diagnostics, &data, rule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AnalysisRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state AnalysisRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ruleID := state.RuleID.ValueString()

	// The query is sent on every update, whether or not it changed: the
	// platform assigns the rule's stored query from this request each time.
	updateReq := client.AnalysisRuleUpdateRequest{
		RuleName:    data.RuleName.ValueString(),
		RuleMessage: data.RuleMessage.ValueString(),
		AlertLevel:  data.AlertLevel.ValueInt64(),
		// The query and the language it is written in travel together: the
		// query is read according to this field, so sending one without the
		// other would reinterpret it.
		RuleQueryType:      data.RuleQueryType.ValueString(),
		RuleQuery:          []byte(data.RuleQuery.ValueString()),
		AccountScopingType: data.AccountScopingType.ValueString(),
		AccountScopingList: scopingListFor(ctx, &resp.Diagnostics, &data),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.UpdateAnalysisRule(ctx, ruleID, updateReq); err != nil {
		addClientError(&resp.Diagnostics, "update analysis rule", err)
		return
	}

	// State moves through its own endpoint and is not part of the body above.
	if data.RuleState.ValueString() != state.RuleState.ValueString() {
		if err := r.client.SetAnalysisRuleState(ctx, ruleID, data.RuleState.ValueString()); err != nil {
			addClientError(&resp.Diagnostics, "set analysis rule state", err)
			return
		}
	}

	rule, err := r.client.GetAnalysisRule(ctx, data.EvidenceID.ValueString(), ruleID)
	if err != nil {
		addClientError(&resp.Diagnostics, "read analysis rule after update", err)
		return
	}

	checkScopingAccepted(ctx, &resp.Diagnostics, &data, rule)
	applyAnalysisRule(ctx, &resp.Diagnostics, &data, rule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AnalysisRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AnalysisRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAnalysisRule(ctx, data.RuleID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		addClientError(&resp.Diagnostics, "delete analysis rule", err)
	}
}

// ImportState imports a rule by its id. The evidence is not part of the import
// address: the rule reports its own evidence, and the read falls back to a
// full scan when no evidence is known yet.
func (r *AnalysisRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("rule_id"), req, resp)
}
