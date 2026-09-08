// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AnalysisRulesDataSource{}

func NewAnalysisRulesDataSource() datasource.DataSource {
	return &AnalysisRulesDataSource{}
}

// AnalysisRulesDataSource lists analysis rules.
type AnalysisRulesDataSource struct {
	client *client.AnecdotesClient
}

// AnalysisRulesDataSourceModel describes the plural data source model.
type AnalysisRulesDataSourceModel struct {
	EvidenceID      types.String `tfsdk:"evidence_id"`
	RuleOrigin      types.String `tfsdk:"rule_origin"`
	RuleState       types.String `tfsdk:"rule_state"`
	RuleType        types.String `tfsdk:"rule_type"`
	IncludeArchived types.Bool   `tfsdk:"include_archived"`
	Rules           types.List   `tfsdk:"rules"`
	TotalCount      types.Int64  `tfsdk:"total_count"`
}

// analysisRuleAttrTypes describes one element of the `rules` list. Shared with
// the singular data source so both report a rule identically.
var analysisRuleAttrTypes = map[string]attr.Type{
	"rule_id":              types.StringType,
	"evidence_id":          types.StringType,
	"rule_name":            types.StringType,
	"rule_message":         types.StringType,
	"alert_level":          types.Int64Type,
	"rule_query_type":      types.StringType,
	"rule_query_str":       types.StringType,
	"rule_query_message":   types.StringType,
	"rule_origin":          types.StringType,
	"rule_state":           types.StringType,
	"rule_type":            types.StringType,
	"library_rule_id":      types.StringType,
	"account_scoping_type": types.StringType,
	"account_scoping_list": types.SetType{ElemType: types.StringType},
	"rule_is_archived":     types.BoolType,
	"last_updated":         types.StringType,
	"last_updated_by":      types.StringType,
}

// analysisRuleDataSourceAttributes describes one rule for a data source schema.
func analysisRuleDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"rule_id":              schema.StringAttribute{Description: "The unique identifier of the analysis rule.", Computed: true},
		"evidence_id":          schema.StringAttribute{Description: "The evidence this rule evaluates.", Computed: true},
		"rule_name":            schema.StringAttribute{Description: "The rule's display name.", Computed: true},
		"rule_message":         schema.StringAttribute{Description: "The message shown for rows the rule matches.", Computed: true},
		"alert_level":          schema.Int64Attribute{Description: "Severity: 30 (warning) or 50 (gap). 3, 5 and 10 describe an evaluation outcome rather than a configured severity.", Computed: true},
		"rule_query_type":      schema.StringAttribute{Description: "The query language: \"aql\", \"aqlext\" or \"pandas\".", Computed: true},
		"rule_query_str":       schema.StringAttribute{Description: "The query as the platform stored it, serialized as a JSON string.", Computed: true},
		"rule_query_message":   schema.StringAttribute{Description: "Message associated with the stored query.", Computed: true},
		"rule_origin":          schema.StringAttribute{Description: "Whether the rule ships with the platform (\"library\") or is authored by the account (\"custom\").", Computed: true},
		"rule_state":           schema.StringAttribute{Description: "Whether the rule is evaluated: \"active\" or \"inactive\".", Computed: true},
		"rule_type":            schema.StringAttribute{Description: "Processing type: \"uam\" or \"eid\".", Computed: true},
		"library_rule_id":      schema.StringAttribute{Description: "The library rule this rule was derived from.", Computed: true},
		"account_scoping_type": schema.StringAttribute{Description: "Which accounts the rule applies to.", Computed: true},
		"account_scoping_list": schema.SetAttribute{Description: "Service instance IDs the rule is scoped to. Unordered.", Computed: true, ElementType: types.StringType},
		"rule_is_archived":     schema.BoolAttribute{Description: "Whether the rule has been deleted. Deleting archives a rule rather than removing it.", Computed: true},
		"last_updated":         schema.StringAttribute{Description: "When the rule was last modified.", Computed: true},
		"last_updated_by":      schema.StringAttribute{Description: "Who last modified the rule.", Computed: true},
	}
}

// analysisRuleObjectValue converts one rule into a Terraform object value.
func analysisRuleObjectValue(ctx context.Context, diags *diag.Diagnostics, rule client.AnalysisRule) attr.Value {
	scoping, d := types.SetValueFrom(ctx, types.StringType, rule.AccountScopingList)
	diags.Append(d...)

	obj, d := types.ObjectValue(analysisRuleAttrTypes, map[string]attr.Value{
		"rule_id":              types.StringValue(rule.RuleID),
		"evidence_id":          types.StringValue(rule.EvidenceID),
		"rule_name":            types.StringValue(rule.RuleName),
		"rule_message":         types.StringValue(rule.RuleMessage),
		"alert_level":          types.Int64Value(rule.AlertLevel),
		"rule_query_type":      types.StringValue(rule.RuleQueryType),
		"rule_query_str":       types.StringValue(rule.RuleQueryStr),
		"rule_query_message":   types.StringValue(rule.RuleQueryMessage),
		"rule_origin":          types.StringValue(rule.RuleOrigin),
		"rule_state":           types.StringValue(rule.RuleState),
		"rule_type":            types.StringValue(rule.RuleType),
		"library_rule_id":      types.StringValue(rule.LibraryRuleID),
		"account_scoping_type": types.StringValue(rule.AccountScopingType),
		"account_scoping_list": scoping,
		"rule_is_archived":     types.BoolValue(rule.RuleIsArchived),
		"last_updated":         types.StringValue(rule.LastUpdated),
		"last_updated_by":      types.StringValue(rule.LastUpdatedBy),
	})
	diags.Append(d...)
	return obj
}

func (d *AnalysisRulesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_analysis_rules"
}

func (d *AnalysisRulesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists Anecdotes Analysis Rules, with optional filtering.",
		MarkdownDescription: `
Lists Anecdotes Analysis Rules, with optional filtering by evidence, origin, state or type.

The account's own rules and the rules shipped with the platform are returned together;
filter on ` + "`rule_origin`" + ` to separate them. Deleted rules are archived rather than
removed, and are excluded unless ` + "`include_archived`" + ` is set.
`,
		Attributes: map[string]schema.Attribute{
			"evidence_id": schema.StringAttribute{
				Description: "Only return rules attached to this evidence.",
				Optional:    true,
			},
			"rule_origin": schema.StringAttribute{
				Description: "Only return rules with this origin: \"library\" or \"custom\".",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.OneOf("library", "custom")},
			},
			"rule_state": schema.StringAttribute{
				Description: "Only return rules in this state: \"active\" or \"inactive\".",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.OneOf("active", "inactive")},
			},
			"rule_type": schema.StringAttribute{
				Description: "Only return rules of this type: \"uam\" or \"eid\".",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.OneOf("uam", "eid")},
			},
			"include_archived": schema.BoolAttribute{
				Description: "Include deleted (archived) rules. Defaults to false.",
				Optional:    true,
			},
			"rules": schema.ListNestedAttribute{
				Description:  "The analysis rules matching the filters.",
				Computed:     true,
				NestedObject: schema.NestedAttributeObject{Attributes: analysisRuleDataSourceAttributes()},
			},
			"total_count": schema.Int64Attribute{
				Description: "The number of rules returned.",
				Computed:    true,
			},
		},
	}
}

func (d *AnalysisRulesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.AnecdotesClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.AnecdotesClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *AnalysisRulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AnalysisRulesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rules, err := d.client.ListAnalysisRules(ctx)
	if err != nil {
		addClientError(&resp.Diagnostics, "list analysis rules", err)
		return
	}

	elements := make([]attr.Value, 0, len(rules))
	for _, rule := range rules {
		if !matchesAnalysisRuleFilters(&data, rule) {
			continue
		}
		elements = append(elements, analysisRuleObjectValue(ctx, &resp.Diagnostics, rule))
	}
	if resp.Diagnostics.HasError() {
		return
	}

	list, d2 := types.ListValue(types.ObjectType{AttrTypes: analysisRuleAttrTypes}, elements)
	resp.Diagnostics.Append(d2...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Rules = list
	data.TotalCount = types.Int64Value(int64(len(elements)))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// matchesAnalysisRuleFilters reports whether a rule passes the configured
// filters, which are applied here rather than in the request.
func matchesAnalysisRuleFilters(data *AnalysisRulesDataSourceModel, rule client.AnalysisRule) bool {
	if rule.RuleIsArchived && !data.IncludeArchived.ValueBool() {
		return false
	}
	if !data.EvidenceID.IsNull() && rule.EvidenceID != data.EvidenceID.ValueString() {
		return false
	}
	if !data.RuleOrigin.IsNull() && rule.RuleOrigin != data.RuleOrigin.ValueString() {
		return false
	}
	if !data.RuleState.IsNull() && rule.RuleState != data.RuleState.ValueString() {
		return false
	}
	if !data.RuleType.IsNull() && rule.RuleType != data.RuleType.ValueString() {
		return false
	}
	return true
}
