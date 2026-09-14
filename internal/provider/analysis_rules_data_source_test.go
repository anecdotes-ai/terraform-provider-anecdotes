// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccAnalysisRulesDataSource_basic reads the unfiltered list and checks the
// nested attributes are populated, not merely present.
func TestAccAnalysisRulesDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_analysis_rules" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_analysis_rules.all", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_analysis_rules.all", "rules"),
					resource.TestCheckResourceAttrSet("data.anecdotes_analysis_rules.all", "rules.0.rule_id"),
					resource.TestCheckResourceAttrSet("data.anecdotes_analysis_rules.all", "rules.0.evidence_id"),
					resource.TestCheckResourceAttrSet("data.anecdotes_analysis_rules.all", "rules.0.rule_origin"),
				),
			},
		},
	})
}

// TestAccAnalysisRulesDataSource_filtersExclude checks each filter actually
// narrows the result. A filter that is wired up but never applied would still
// return rows and pass a "some rows came back" assertion.
func TestAccAnalysisRulesDataSource_filtersExclude(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAnalysisRuleConfig(randomName("tf-acc-rule-filter"), testAQLQuery, "") + `
data "anecdotes_analysis_rules" "all" {
  depends_on = [anecdotes_analysis_rule.test]
}

data "anecdotes_analysis_rules" "library" {
  rule_origin = "library"
  depends_on  = [anecdotes_analysis_rule.test]
}

data "anecdotes_analysis_rules" "active" {
  rule_state = "active"
}

data "anecdotes_analysis_rules" "with_archived" {
  include_archived = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Every library rule is a rule, and the platform ships rules
					// the account did not author, so this is a strict subset only
					// if the filter runs.
					testCheckCountStrictlyLess("data.anecdotes_analysis_rules.library", "data.anecdotes_analysis_rules.all", true),
					testCheckAllAttrEquals("data.anecdotes_analysis_rules.library", "rules", "rule_origin", "library"),
					testCheckAllAttrEquals("data.anecdotes_analysis_rules.active", "rules", "rule_state", "active"),
					// Deleting archives rather than removes, so including archived
					// rules widens the result. Asserting only that it does not
					// shrink would pass even if the flag were ignored, so this
					// also finds an archived rule among the rows it returned.
					testCheckCountStrictlyLess("data.anecdotes_analysis_rules.all", "data.anecdotes_analysis_rules.with_archived", false),
					testCheckSomeRowHasAttr("data.anecdotes_analysis_rules.with_archived", "rules", "rule_is_archived", "true"),
					testCheckNoRowHasAttr("data.anecdotes_analysis_rules.all", "rules", "rule_is_archived", "true"),
				),
			},
		},
	})
}

// TestAccAnalysisRuleDataSource_singular looks a rule up by id.
func TestAccAnalysisRuleDataSource_singular(t *testing.T) {
	name := randomName("tf-acc-rule-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAnalysisRuleConfig(name, testAQLQuery, "") + `
data "anecdotes_analysis_rule" "by_id" {
  rule_id     = anecdotes_analysis_rule.test.rule_id
  evidence_id = anecdotes_analysis_rule.test.evidence_id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.anecdotes_analysis_rule.by_id", "rule_name", name),
					resource.TestCheckResourceAttr("data.anecdotes_analysis_rule.by_id", "rule_origin", "custom"),
					resource.TestCheckResourceAttr("data.anecdotes_analysis_rule.by_id", "rule_is_archived", "false"),
					resource.TestCheckResourceAttrSet("data.anecdotes_analysis_rule.by_id", "rule_query_str"),
				),
			},
		},
	})
}

// testCheckCountStrictlyLess compares the total_count of two data sources.
func testCheckCountStrictlyLess(smaller, larger string, strict bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		a, err := intAttr(s, smaller, "total_count")
		if err != nil {
			return err
		}
		b, err := intAttr(s, larger, "total_count")
		if err != nil {
			return err
		}
		if strict && a >= b {
			return fmt.Errorf("%s returned %d rules and %s returned %d; the filter excluded nothing", smaller, a, larger, b)
		}
		if a > b {
			return fmt.Errorf("%s returned %d rules, more than %s at %d", smaller, a, larger, b)
		}
		return nil
	}
}

// testCheckAllAttrEquals asserts every element of a list attribute holds want.
func testCheckAllAttrEquals(resourceAddr, listAttr, field, want string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceAddr]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceAddr)
		}
		count, err := strconv.Atoi(rs.Primary.Attributes[listAttr+".#"])
		if err != nil {
			return fmt.Errorf("reading %s.# on %s: %w", listAttr, resourceAddr, err)
		}
		if count == 0 {
			return fmt.Errorf("%s returned no rows, so the %s filter proves nothing", resourceAddr, field)
		}
		for i := 0; i < count; i++ {
			key := fmt.Sprintf("%s.%d.%s", listAttr, i, field)
			if got := rs.Primary.Attributes[key]; got != want {
				return fmt.Errorf("%s on %s = %q, want %q", key, resourceAddr, got, want)
			}
		}
		return nil
	}
}

func intAttr(s *terraform.State, resourceAddr, attr string) (int, error) {
	rs, ok := s.RootModule().Resources[resourceAddr]
	if !ok {
		return 0, fmt.Errorf("resource %s not found in state", resourceAddr)
	}
	v, err := strconv.Atoi(rs.Primary.Attributes[attr])
	if err != nil {
		return 0, fmt.Errorf("reading %s on %s: %w", attr, resourceAddr, err)
	}
	return v, nil
}

// testCheckSomeRowHasAttr asserts at least one element of a list attribute holds
// want, which is how an inclusive filter proves it admitted something.
func testCheckSomeRowHasAttr(resourceAddr, listAttr, field, want string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceAddr]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceAddr)
		}
		count, err := strconv.Atoi(rs.Primary.Attributes[listAttr+".#"])
		if err != nil {
			return fmt.Errorf("reading %s.# on %s: %w", listAttr, resourceAddr, err)
		}
		for i := 0; i < count; i++ {
			if rs.Primary.Attributes[fmt.Sprintf("%s.%d.%s", listAttr, i, field)] == want {
				return nil
			}
		}
		return fmt.Errorf("no row on %s has %s = %q, so the filter admitted nothing", resourceAddr, field, want)
	}
}

// testCheckNoRowHasAttr asserts no element of a list attribute holds want, which
// is how the same filter proves it excluded something.
func testCheckNoRowHasAttr(resourceAddr, listAttr, field, want string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceAddr]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceAddr)
		}
		count, err := strconv.Atoi(rs.Primary.Attributes[listAttr+".#"])
		if err != nil {
			return fmt.Errorf("reading %s.# on %s: %w", listAttr, resourceAddr, err)
		}
		for i := 0; i < count; i++ {
			key := fmt.Sprintf("%s.%d.%s", listAttr, i, field)
			if rs.Primary.Attributes[key] == want {
				return fmt.Errorf("%s on %s = %q, but the filter should have excluded it", key, resourceAddr, want)
			}
		}
		return nil
	}
}
