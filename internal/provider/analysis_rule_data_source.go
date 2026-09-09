// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AnalysisRuleDataSource{}

func NewAnalysisRuleDataSource() datasource.DataSource {
	return &AnalysisRuleDataSource{}
}

// AnalysisRuleDataSource looks up a single analysis rule.
type AnalysisRuleDataSource struct {
	client *client.AnecdotesClient
}

// AnalysisRuleDataSourceModel describes the singular data source model.
// The rule's own fields are flattened onto the data source rather than nested,
// matching the singular lookups elsewhere in this provider.
type AnalysisRuleDataSourceModel struct {
	RuleID     types.String `tfsdk:"rule_id"`
	EvidenceID types.String `tfsdk:"evidence_id"`

	RuleName           types.String `tfsdk:"rule_name"`
	RuleMessage        types.String `tfsdk:"rule_message"`
	AlertLevel         types.Int64  `tfsdk:"alert_level"`
	RuleQueryType      types.String `tfsdk:"rule_query_type"`
	RuleQueryStr       types.String `tfsdk:"rule_query_str"`
	RuleQueryMessage   types.String `tfsdk:"rule_query_message"`
	RuleOrigin         types.String `tfsdk:"rule_origin"`
	RuleState          types.String `tfsdk:"rule_state"`
	RuleType           types.String `tfsdk:"rule_type"`
	LibraryRuleID      types.String `tfsdk:"library_rule_id"`
	AccountScopingType types.String `tfsdk:"account_scoping_type"`
	AccountScopingList types.Set    `tfsdk:"account_scoping_list"`
	RuleIsArchived     types.Bool   `tfsdk:"rule_is_archived"`
	LastUpdated        types.String `tfsdk:"last_updated"`
	LastUpdatedBy      types.String `tfsdk:"last_updated_by"`
}

func (d *AnalysisRuleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_analysis_rule"
}

func (d *AnalysisRuleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := analysisRuleDataSourceAttributes()

	// rule_id identifies the rule being looked up; evidence_id narrows the
	// lookup to one evidence, which avoids reading the platform's full rule
	// library when it is known.
	attrs["rule_id"] = schema.StringAttribute{
		Description: "The unique identifier of the analysis rule to look up.",
		Required:    true,
	}
	attrs["evidence_id"] = schema.StringAttribute{
		Description: "The evidence the rule is attached to. Optional, and only a hint: supplying it makes the lookup cheaper.",
		Optional:    true,
		Computed:    true,
	}

	resp.Schema = schema.Schema{
		Description: "Looks up a single Anecdotes Analysis Rule by ID.",
		MarkdownDescription: `
Looks up a single Anecdotes Analysis Rule by ID, whether it is one the account authored
or one shipped with the platform.

Archived (deleted) rules are not returned. Use the ` + "`anecdotes_analysis_rules`" + `
data source with ` + "`include_archived`" + ` to read those.
`,
		Attributes: attrs,
	}
}

func (d *AnalysisRuleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, "Data Source", &resp.Diagnostics)
}

func (d *AnalysisRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AnalysisRuleDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := d.client.GetAnalysisRule(ctx, data.EvidenceID.ValueString(), data.RuleID.ValueString())
	if err != nil {
		addClientError(&resp.Diagnostics, "read analysis rule", err)
		return
	}

	scoping, diags := types.SetValueFrom(ctx, types.StringType, rule.AccountScopingList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.RuleID = types.StringValue(rule.RuleID)
	data.EvidenceID = types.StringValue(rule.EvidenceID)
	data.RuleName = types.StringValue(rule.RuleName)
	data.RuleMessage = types.StringValue(rule.RuleMessage)
	data.AlertLevel = types.Int64Value(rule.AlertLevel)
	data.RuleQueryType = types.StringValue(rule.RuleQueryType)
	data.RuleQueryStr = types.StringValue(rule.RuleQueryStr)
	data.RuleQueryMessage = types.StringValue(rule.RuleQueryMessage)
	data.RuleOrigin = types.StringValue(rule.RuleOrigin)
	data.RuleState = types.StringValue(rule.RuleState)
	data.RuleType = types.StringValue(rule.RuleType)
	data.LibraryRuleID = types.StringValue(rule.LibraryRuleID)
	data.AccountScopingType = types.StringValue(rule.AccountScopingType)
	data.AccountScopingList = scoping
	data.RuleIsArchived = types.BoolValue(rule.RuleIsArchived)
	data.LastUpdated = types.StringValue(rule.LastUpdated)
	data.LastUpdatedBy = types.StringValue(rule.LastUpdatedBy)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
