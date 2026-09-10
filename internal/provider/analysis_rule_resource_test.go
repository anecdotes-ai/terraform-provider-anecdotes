// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const testAQLQuery = `{"left":"Min Password Length","operator":"IsIn","right":["1","2"]}`

func testAccAnalysisRuleConfig(name, query, extra string) string {
	return fmt.Sprintf(`
resource "anecdotes_analysis_rule" "test" {
  evidence_id  = %[1]q
  rule_name    = %[2]q
  rule_message = "managed by acceptance tests"
  rule_query   = %[3]q
%[4]s
}
`, testAccEvidenceID(), name, query, extra)
}

// testCheckAnalysisRuleOnPlatform reads the rule straight from the API and hands
// it to check. Terraform state is populated by the provider's own Read, so a bug
// there can make a state-only assertion pass while the platform holds something
// else entirely.
func testCheckAnalysisRuleOnPlatform(t *testing.T, resourceAddr string, check func(*client.AnalysisRule) error) resource.TestCheckFunc {
	t.Helper()
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceAddr]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceAddr)
		}

		ruleID := rs.Primary.Attributes["rule_id"]
		rule, err := testAccNewClient(t).GetAnalysisRule(context.Background(), rs.Primary.Attributes["evidence_id"], ruleID)
		if err != nil {
			return fmt.Errorf("reading analysis rule %s from the platform: %w", ruleID, err)
		}
		if err := check(rule); err != nil {
			return fmt.Errorf("the platform's copy of analysis rule %s: %w", ruleID, err)
		}
		return nil
	}
}

// TestAccAnalysisRule_basic covers create, update, and import.
func TestAccAnalysisRule_basic(t *testing.T) {
	name := randomName("tf-acc-rule")
	renamed := name + "-renamed"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAnalysisRuleConfig(name, testAQLQuery, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("anecdotes_analysis_rule.test", "rule_id"),
					resource.TestCheckResourceAttr("anecdotes_analysis_rule.test", "rule_name", name),
					// The platform records every rule created this way as custom
					// and active regardless of what the request asks for.
					resource.TestCheckResourceAttr("anecdotes_analysis_rule.test", "rule_origin", "custom"),
					resource.TestCheckResourceAttr("anecdotes_analysis_rule.test", "rule_state", "active"),
					resource.TestCheckResourceAttr("anecdotes_analysis_rule.test", "alert_level", "50"),
					resource.TestCheckResourceAttr("anecdotes_analysis_rule.test", "rule_query_type", "aql"),
					resource.TestCheckResourceAttr("anecdotes_analysis_rule.test", "account_scoping_type", "all_accounts"),
					testCheckAnalysisRuleOnPlatform(t, "anecdotes_analysis_rule.test", func(rule *client.AnalysisRule) error {
						if rule.RuleName != name {
							return fmt.Errorf("rule_name = %q, want %q", rule.RuleName, name)
						}
						return nil
					}),
				),
			},
			{
				Config: testAccAnalysisRuleConfig(renamed, testAQLQuery, "  alert_level = 30"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_analysis_rule.test", "rule_name", renamed),
					resource.TestCheckResourceAttr("anecdotes_analysis_rule.test", "alert_level", "30"),
					testCheckAnalysisRuleOnPlatform(t, "anecdotes_analysis_rule.test", func(rule *client.AnalysisRule) error {
						if rule.RuleName != renamed {
							return fmt.Errorf("rule_name = %q, want %q", rule.RuleName, renamed)
						}
						if rule.AlertLevel != 30 {
							return fmt.Errorf("alert_level = %d, want 30", rule.AlertLevel)
						}
						// An update must never blank the stored query.
						if rule.RuleQueryStr == "" || rule.RuleQueryStr == "null" {
							return fmt.Errorf("rule_query_str = %q, want the configured query", rule.RuleQueryStr)
						}
						return nil
					}),
				),
			},
			{
				ResourceName:                         "anecdotes_analysis_rule.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "rule_id",
				ImportStateIdFunc:                    importIDFromAttr("anecdotes_analysis_rule.test", "rule_id"),
				// An imported rule has no configuration to compare against, so
				// rule_query arrives rendered the way the platform stores it
				// rather than the way it was originally written. The two are the
				// same query; only the text differs.
				ImportStateVerifyIgnore: []string{"rule_query"},
			},
		},
	})
}

// TestAccAnalysisRule_storedQueryIsNotPerpetualDrift pins the property that
// matters for rule_query: the platform stores the query with its own key order
// and spacing, and that difference must never surface as a pending change.
//
// Note this is about the platform's rendering, not the configuration's. A
// practitioner who reorders the keys in their own configuration does get an
// update planned, because Terraform compares configuration to state as text.
func TestAccAnalysisRule_storedQueryIsNotPerpetualDrift(t *testing.T) {
	name := randomName("tf-acc-rule-json")
	config := testAccAnalysisRuleConfig(name, testAQLQuery, "")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: testCheckAnalysisRuleOnPlatform(t, "anecdotes_analysis_rule.test", func(rule *client.AnalysisRule) error {
					// Guard the premise: if the platform ever stored the query
					// byte-for-byte as sent, this test would pass for the wrong
					// reason and stop protecting anything.
					if rule.RuleQueryStr == testAQLQuery {
						return fmt.Errorf("the platform stored the query unchanged, so this test no longer exercises re-rendering")
					}
					return nil
				}),
			},
			{
				// Re-planning the same configuration against the stored form must
				// produce nothing to do.
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// TestAccAnalysisRule_stateToggle covers rule_state, which the platform moves
// through a dedicated endpoint rather than the update body.
func TestAccAnalysisRule_stateToggle(t *testing.T) {
	name := randomName("tf-acc-rule-state")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAnalysisRuleConfig(name, testAQLQuery, `  rule_state = "inactive"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_analysis_rule.test", "rule_state", "inactive"),
					// The state endpoint answers 200 with an empty body both for a
					// no-op and for an unknown id, so only the platform's own read
					// shows whether the change actually landed.
					testCheckAnalysisRuleOnPlatform(t, "anecdotes_analysis_rule.test", func(rule *client.AnalysisRule) error {
						if rule.RuleState != "inactive" {
							return fmt.Errorf("rule_state = %q, want inactive", rule.RuleState)
						}
						return nil
					}),
				),
			},
			{
				Config: testAccAnalysisRuleConfig(name, testAQLQuery, `  rule_state = "active"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_analysis_rule.test", "rule_state", "active"),
					testCheckAnalysisRuleOnPlatform(t, "anecdotes_analysis_rule.test", func(rule *client.AnalysisRule) error {
						if rule.RuleState != "active" {
							return fmt.Errorf("rule_state = %q, want active", rule.RuleState)
						}
						return nil
					}),
				),
			},
		},
	})
}

// TestAccAnalysisRule_scopedToServiceInstances applies the account scoping path
// end to end, taking the instance IDs from the evidences data source rather than
// hard-coding them. This is what ties service_instance_ids to the thing it
// exists for: the platform rejects a scoping list whose instances do not belong
// to the evidence's service, so an attribute reporting the wrong IDs fails here.
func TestAccAnalysisRule_scopedToServiceInstances(t *testing.T) {
	name := randomName("tf-acc-rule-scoped")

	config := fmt.Sprintf(`
data "anecdotes_evidences" "all" {}

locals {
  target = one([
    for e in data.anecdotes_evidences.all.evidences : e
    if e.evidence_id == %[1]q
  ])
}

resource "anecdotes_analysis_rule" "scoped" {
  evidence_id = %[1]q
  rule_name   = %[2]q
  rule_query  = %[3]q

  account_scoping_type = "included_accounts"
  account_scoping_list = local.target.service_instance_ids
}
`, testAccEvidenceID(), name, testAQLQuery)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_analysis_rule.scoped", "account_scoping_type", "included_accounts"),
					resource.TestCheckResourceAttr("anecdotes_analysis_rule.scoped", "account_scoping_list.#", "1"),
					testCheckAnalysisRuleOnPlatform(t, "anecdotes_analysis_rule.scoped", func(rule *client.AnalysisRule) error {
						if len(rule.AccountScopingList) == 0 {
							return fmt.Errorf("account_scoping_list is empty; the platform kept none of the configured instances")
						}
						return nil
					}),
				),
			},
		},
	})
}

// TestAccAnalysisRule_aqlextLifecycle covers a rule written in a query language
// other than the default. The query is read according to rule_query_type, so an
// update that omits it is rejected by the platform: this exercises create and
// update together, which a rule using the default language would not catch.
func TestAccAnalysisRule_aqlextLifecycle(t *testing.T) {
	name := randomName("tf-acc-rule-aqlext")
	query := `{"manipulations":[{"operator_name":"JoinLeftMinusRight","other_instance_id":"UAM","on_left_column":"Email","on_right_column":"label: Email","original_columns_only":true}]}`

	cfg := func(ruleName string) string {
		return fmt.Sprintf(`
resource "anecdotes_analysis_rule" "aqlext" {
  evidence_id     = %[1]q
  rule_name       = %[2]q
  rule_query_type = "aqlext"
  rule_query      = %[3]q
}
`, testAccEvidenceID(), ruleName, query)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(name),
				Check:  resource.TestCheckResourceAttr("anecdotes_analysis_rule.aqlext", "rule_query_type", "aqlext"),
			},
			{
				// A rename still sends the whole query, so this is where an
				// update missing the query language would fail.
				Config: cfg(name + "-renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_analysis_rule.aqlext", "rule_name", name+"-renamed"),
					testCheckAnalysisRuleOnPlatform(t, "anecdotes_analysis_rule.aqlext", func(rule *client.AnalysisRule) error {
						if rule.RuleQueryType != "aqlext" {
							return fmt.Errorf("rule_query_type = %q after update, want aqlext", rule.RuleQueryType)
						}
						if !strings.Contains(rule.RuleQueryStr, "JoinLeftMinusRight") {
							return fmt.Errorf("the update rewrote the stored query: %s", rule.RuleQueryStr)
						}
						return nil
					}),
				),
			},
		},
	})
}

// TestAccAnalysisRule_rejectsUnsupportedQueryKeys: an "aql" query is stored with
// only operator, left and right, so extra keys are refused before apply.
func TestAccAnalysisRule_rejectsUnsupportedQueryKeys(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAnalysisRuleConfig(randomName("tf-acc-rule-keys"),
					`{"operator":"IsIn","left":"Policy Status","right":["Draft"],"aql_column_name":"X"}`, ""),
				ExpectError: regexp.MustCompile(`Unsupported Keys In Rule Query`),
			},
		},
	})
}

// TestAccAnalysisRule_rejectsAllAccountsWithScopingList: the platform discards a
// scoping list when the rule applies to every account, so the combination is
// refused before apply rather than silently dropped.
func TestAccAnalysisRule_rejectsAllAccountsWithScopingList(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAnalysisRuleConfig(randomName("tf-acc-rule-scope"), testAQLQuery,
					"  account_scoping_type = \"all_accounts\"\n  account_scoping_list = [\"some-instance\"]"),
				ExpectError: regexp.MustCompile(`Unexpected Account Scoping List`),
			},
		},
	})
}

// TestAccAnalysisRule_rejectsUnknownScopingInstance: an instance that does not
// belong to the evidence's service is dropped by the platform, and the provider
// names it rather than leaving an unexplained mismatch.
func TestAccAnalysisRule_rejectsUnknownScopingInstance(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAnalysisRuleConfig(randomName("tf-acc-rule-badinst"), testAQLQuery,
					"  account_scoping_type = \"included_accounts\"\n  account_scoping_list = [\"not-a-real-instance\"]"),
				ExpectError: regexp.MustCompile(`Account scoping list is not valid`),
			},
		},
	})
}

// TestAccAnalysisRule_libraryRuleRejected: a rule shipped with the platform
// cannot be updated or deleted, so importing one must fail rather than produce a
// resource whose every later operation misbehaves.
func TestAccAnalysisRule_libraryRuleRejected(t *testing.T) {
	var libraryRuleID string

	// The import step needs this address to exist in the configuration; the
	// block is never applied.
	importConfig := fmt.Sprintf(`
data "anecdotes_analysis_rules" "library" {
  rule_origin = "library"
}

resource "anecdotes_analysis_rule" "imported" {
  evidence_id = %[1]q
  rule_name   = "placeholder for the import address"
  rule_query  = %[2]q
}
`, testAccEvidenceID(), testAQLQuery)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_analysis_rules" "library" { rule_origin = "library" }`,
				Check: func(s *terraform.State) error {
					rs, ok := s.RootModule().Resources["data.anecdotes_analysis_rules.library"]
					if !ok {
						return fmt.Errorf("library rules data source not found in state")
					}
					libraryRuleID = rs.Primary.Attributes["rules.0.rule_id"]
					if libraryRuleID == "" {
						return fmt.Errorf("no library rule available to import")
					}
					return nil
				},
			},
			{
				Config:            importConfig,
				ResourceName:      "anecdotes_analysis_rule.imported",
				ImportState:       true,
				ImportStateIdFunc: func(*terraform.State) (string, error) { return libraryRuleID, nil },
				ExpectError:       regexp.MustCompile(`cannot be managed by Terraform`),
			},
		},
	})
}

// TestAccAnalysisRule_reportsPartiallyRejectedScoping covers the case
// checkScopingAccepted exists for: a list the platform accepts but shortens.
// A list with no valid entry at all is refused outright by the platform and
// never reaches that code, so it is exercised separately.
func TestAccAnalysisRule_reportsPartiallyRejectedScoping(t *testing.T) {
	config := fmt.Sprintf(`
data "anecdotes_evidences" "all" {}

locals {
  target = one([
    for e in data.anecdotes_evidences.all.evidences : e
    if e.evidence_id == %[1]q
  ])
}

resource "anecdotes_analysis_rule" "partial" {
  evidence_id = %[1]q
  rule_name   = %[2]q
  rule_query  = %[3]q

  account_scoping_type = "included_accounts"
  # One instance the evidence really reports, plus one the platform will drop.
  account_scoping_list = concat(local.target.service_instance_ids, ["not-a-real-instance"])
}
`, testAccEvidenceID(), randomName("tf-acc-rule-partial"), testAQLQuery)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Account Scoping List Not Accepted`),
			},
		},
	})
}

// TestAccAnalysisRule_rejectsScopingListWithoutType: leaving account_scoping_type
// unset applies its default of all_accounts, under which the platform keeps no
// list, so a configured list must still be refused.
func TestAccAnalysisRule_rejectsScopingListWithoutType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAnalysisRuleConfig(randomName("tf-acc-rule-notype"), testAQLQuery,
					"  account_scoping_list = [\"some-instance\"]"),
				ExpectError: regexp.MustCompile(`Unexpected Account Scoping List`),
			},
		},
	})
}

// TestAccAnalysisRule_pandasLifecycle covers the third query language through
// create and update. A pandas query is an expression string rather than an
// object, and it is stored lower-cased, so it round-trips only when written that
// way.
func TestAccAnalysisRule_pandasLifecycle(t *testing.T) {
	name := randomName("tf-acc-rule-pandas")
	query := "`policy status` == \"draft\""

	cfg := func(ruleName string) string {
		return fmt.Sprintf(`
resource "anecdotes_analysis_rule" "pandas" {
  evidence_id     = %[1]q
  rule_name       = %[2]q
  rule_query_type = "pandas"
  rule_query      = jsonencode(%[3]q)
}
`, testAccEvidenceID(), ruleName, query)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(name),
				Check:  resource.TestCheckResourceAttr("anecdotes_analysis_rule.pandas", "rule_query_type", "pandas"),
			},
			{
				Config: cfg(name + "-renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_analysis_rule.pandas", "rule_name", name+"-renamed"),
					testCheckAnalysisRuleOnPlatform(t, "anecdotes_analysis_rule.pandas", func(rule *client.AnalysisRule) error {
						if rule.RuleQueryType != "pandas" {
							return fmt.Errorf("rule_query_type = %q after update, want pandas", rule.RuleQueryType)
						}
						if !strings.Contains(rule.RuleQueryStr, "policy status") {
							return fmt.Errorf("the update rewrote the stored query: %s", rule.RuleQueryStr)
						}
						return nil
					}),
				),
			},
		},
	})
}

// TestAccAnalysisRule_rejectsMixedCasePandasQuery: the platform lowercases a
// pandas query, so anything else is refused before it can be stored as a
// different expression than the one written.
func TestAccAnalysisRule_rejectsMixedCasePandasQuery(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_analysis_rule" "pandas" {
  evidence_id     = %[1]q
  rule_name       = %[2]q
  rule_query_type = "pandas"
  rule_query      = jsonencode("`+"`Policy Status`"+` == \"Draft\"")
}
`, testAccEvidenceID(), randomName("tf-acc-rule-pandas-case")),
				ExpectError: regexp.MustCompile(`Rule Query Must Be Lower Case`),
			},
		},
	})
}

// TestAccAnalysisRule_attributeSurface asserts every attribute the resource
// exposes, including the ones no scenario test happens to touch. A test that
// only sets an attribute in HCL proves nothing about how it is read back.
func TestAccAnalysisRule_attributeSurface(t *testing.T) {
	name := randomName("tf-acc-rule-surface")
	const addr = "anecdotes_analysis_rule.surface"

	config := fmt.Sprintf(`
resource "anecdotes_analysis_rule" "surface" {
  evidence_id     = %[1]q
  rule_name       = %[2]q
  rule_message    = "every attribute is asserted"
  alert_level     = 30
  rule_query_type = "aql"
  rule_query      = %[3]q
  rule_type       = "eid"
  library_rule_id = "library-rule-reference"
  rule_state      = "inactive"

  account_scoping_type = "all_accounts"
}
`, testAccEvidenceID(), name, testAQLQuery)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Terraform-owned
					resource.TestCheckResourceAttr(addr, "evidence_id", testAccEvidenceID()),
					resource.TestCheckResourceAttr(addr, "rule_name", name),
					resource.TestCheckResourceAttr(addr, "rule_message", "every attribute is asserted"),
					resource.TestCheckResourceAttr(addr, "alert_level", "30"),
					resource.TestCheckResourceAttr(addr, "rule_query_type", "aql"),
					resource.TestCheckResourceAttr(addr, "rule_type", "eid"),
					resource.TestCheckResourceAttr(addr, "library_rule_id", "library-rule-reference"),
					resource.TestCheckResourceAttr(addr, "rule_state", "inactive"),
					resource.TestCheckResourceAttr(addr, "account_scoping_type", "all_accounts"),
					// The query is JSON, so it is compared as JSON rather than text.
					testCheckJSONAttr(addr, "rule_query", testAQLQuery),
					// Platform-owned
					resource.TestCheckResourceAttrSet(addr, "rule_id"),
					resource.TestCheckResourceAttrSet(addr, "rule_query_str"),
					resource.TestCheckResourceAttrSet(addr, "last_updated"),
					resource.TestCheckResourceAttrSet(addr, "last_updated_by"),
					resource.TestCheckResourceAttr(addr, "rule_origin", "custom"),
					// The platform does not set a query message when a rule is
					// created, so the attribute is absent rather than empty.
					resource.TestCheckNoResourceAttr(addr, "rule_query_message"),
					testCheckAnalysisRuleOnPlatform(t, addr, func(rule *client.AnalysisRule) error {
						if rule.RuleType != "eid" {
							return fmt.Errorf("rule_type = %q, want eid", rule.RuleType)
						}
						if rule.LibraryRuleID != "library-rule-reference" {
							return fmt.Errorf("library_rule_id = %q, want library-rule-reference", rule.LibraryRuleID)
						}
						if rule.RuleMessage != "every attribute is asserted" {
							return fmt.Errorf("rule_message = %q", rule.RuleMessage)
						}
						return nil
					}),
				),
			},
		},
	})
}

// TestAccAnalysisRule_writeOnceAttributesSettleOnUpdate: an unchanged plan being
// empty says nothing about what an update reports. The attributes the platform
// writes once must already be known while planning an update, or every change
// reports them as pending.
func TestAccAnalysisRule_writeOnceAttributesSettleOnUpdate(t *testing.T) {
	const addr = "anecdotes_analysis_rule.test"
	name := randomName("tf-acc-rule-settled")

	settled := func(attr string) plancheck.PlanCheck {
		return plancheck.ExpectKnownValue(addr, tfjsonpath.New(attr), knownvalue.NotNull())
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccAnalysisRuleConfig(name, testAQLQuery, "")},
			{
				Config: testAccAnalysisRuleConfig(name+"-renamed", testAQLQuery, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						settled("rule_id"),
						settled("evidence_id"),
						settled("rule_origin"),
					},
				},
			},
		},
	})
}

// TestAccAnalysisRule_immutableFieldsReplace covers the attributes that cannot be
// changed on an existing rule. Each must plan a replacement: the platform ignores
// them on update, so a change that planned as an update would appear to succeed
// while leaving the rule as it was.
func TestAccAnalysisRule_immutableFieldsReplace(t *testing.T) {
	const addr = "anecdotes_analysis_rule.test"
	name := randomName("tf-acc-rule-replace")

	replaced := resource.ConfigPlanChecks{
		PreApply: []plancheck.PlanCheck{
			plancheck.ExpectResourceAction(addr, plancheck.ResourceActionDestroyBeforeCreate),
		},
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccAnalysisRuleConfig(name, testAQLQuery, "")},
			{
				// rule_type is not applied on update.
				Config:           testAccAnalysisRuleConfig(name, testAQLQuery, `  rule_type = "uam"`),
				ConfigPlanChecks: replaced,
				Check:            resource.TestCheckResourceAttr(addr, "rule_type", "uam"),
			},
			{
				// library_rule_id is recorded once, at creation.
				Config:           testAccAnalysisRuleConfig(name, testAQLQuery, "  rule_type = \"uam\"\n  library_rule_id = \"some-library-rule\""),
				ConfigPlanChecks: replaced,
				Check:            resource.TestCheckResourceAttr(addr, "library_rule_id", "some-library-rule"),
			},
			{
				// The query language decides how the query is read, so it cannot
				// change under an existing query.
				Config: fmt.Sprintf(`
resource "anecdotes_analysis_rule" "test" {
  evidence_id     = %[1]q
  rule_name       = %[2]q
  rule_query_type = "aqlext"
  rule_query      = %[3]q
}
`, testAccEvidenceID(), name, `{"manipulations":[]}`),
				ConfigPlanChecks: replaced,
				Check:            resource.TestCheckResourceAttr(addr, "rule_query_type", "aqlext"),
			},
		},
	})
}

// TestAccAnalysisRule_rejectsInvalidEnumValues checks each enumerated attribute
// refuses a value outside its set while planning, rather than storing something
// the platform will not act on.
func TestAccAnalysisRule_rejectsInvalidEnumValues(t *testing.T) {
	cases := []struct {
		name  string
		extra string
		want  string
	}{
		{"alert level outside the settable pair", "  alert_level = 10", `alert_level value must be one of`},
		{"unknown query language", `  rule_query_type = "sql"`, `rule_query_type value must be one of`},
		{"unknown rule type", `  rule_type = "unknown"`, `rule_type value must be one of`},
		{"unknown rule state", `  rule_state = "paused"`, `rule_state value must be one of`},
		{"unknown scoping type", `  account_scoping_type = "some_accounts"`, `account_scoping_type value must be one of`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccEvidencePreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      testAccAnalysisRuleConfig(randomName("tf-acc-rule-enum"), testAQLQuery, tc.extra),
						ExpectError: regexp.MustCompile(tc.want),
					},
				},
			})
		})
	}
}

// TestAccAnalysisRule_removingNameKeepsPlatformValue: the platform ignores an
// emptied name, so removing the attribute has to settle on the value it holds.
// Reporting the configuration's null instead fails the apply and leaves state
// disagreeing with the configuration with no way back.
func TestAccAnalysisRule_removingNameKeepsPlatformValue(t *testing.T) {
	name := randomName("tf-acc-rule-keepname")
	const addr = "anecdotes_analysis_rule.test"

	without := fmt.Sprintf(`
resource "anecdotes_analysis_rule" "test" {
  evidence_id = %[1]q
  rule_query  = %[2]q
}
`, testAccEvidenceID(), testAQLQuery)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAnalysisRuleConfig(name, testAQLQuery, ""),
				Check:  resource.TestCheckResourceAttr(addr, "rule_name", name),
			},
			{
				Config: without,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "rule_name", name),
					resource.TestCheckResourceAttr(addr, "rule_message", "managed by acceptance tests"),
				),
			},
		},
	})
}

// TestAccAnalysisRule_rejectsEmptyName: the platform keeps the value it holds
// rather than storing an empty one, so an explicitly empty name would be applied
// as something other than what was written. Removing the attribute is the
// supported way to leave it alone.
func TestAccAnalysisRule_rejectsEmptyName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_analysis_rule" "test" {
  evidence_id = %[1]q
  rule_name   = ""
  rule_query  = %[2]q
}
`, testAccEvidenceID(), testAQLQuery),
				ExpectError: regexp.MustCompile(`string length must be at least 1`),
			},
		},
	})
}
