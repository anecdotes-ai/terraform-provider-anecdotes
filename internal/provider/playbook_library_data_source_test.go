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

// testCheckAnyNested walks the indexed entries of a nested list attribute and
// passes when predicate accepts one of them. It reports what it saw otherwise,
// so a failure says which values were present rather than only that none
// matched.
func testCheckAnyNested(resourceAddr, listAttr, valueAttr string, predicate func(string) bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceAddr]
		if !ok {
			return fmt.Errorf("%s not found in state", resourceAddr)
		}

		count, err := strconv.Atoi(rs.Primary.Attributes[listAttr+".#"])
		if err != nil {
			return fmt.Errorf("%s has no %s.#", resourceAddr, listAttr)
		}

		seen := make([]string, 0, count)
		for i := 0; i < count; i++ {
			value := rs.Primary.Attributes[fmt.Sprintf("%s.%d.%s", listAttr, i, valueAttr)]
			if predicate(value) {
				return nil
			}
			seen = append(seen, value)
		}

		return fmt.Errorf("no %s.*.%s matched; saw %v", listAttr, valueAttr, seen)
	}
}

// testCheckNoNested is the negation of testCheckAnyNested: it passes when no
// entry matches, which is how a filter that excludes something is verified.
func testCheckNoNested(resourceAddr, listAttr, valueAttr string, predicate func(string) bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceAddr]
		if !ok {
			return fmt.Errorf("%s not found in state", resourceAddr)
		}

		count, err := strconv.Atoi(rs.Primary.Attributes[listAttr+".#"])
		if err != nil {
			return fmt.Errorf("%s has no %s.#", resourceAddr, listAttr)
		}

		for i := 0; i < count; i++ {
			key := fmt.Sprintf("%s.%d.%s", listAttr, i, valueAttr)
			if predicate(rs.Primary.Attributes[key]) {
				return fmt.Errorf("%s should have been excluded, got %q", key, rs.Primary.Attributes[key])
			}
		}

		return nil
	}
}

func TestAccPlaybookLibraryDataSource_reportsFieldsAndActions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "anecdotes_playbook_library" "test" {
  category       = "control"
  available_only = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_playbook_library.test", 0),
					resource.TestCheckResourceAttrSet("data.anecdotes_playbook_library.test", "events.0.event_type"),
					resource.TestCheckResourceAttrSet("data.anecdotes_playbook_library.test", "events.0.trigger_value"),
					// The category filter must actually filter.
					resource.TestCheckResourceAttr("data.anecdotes_playbook_library.test", "events.0.category", "control"),
					resource.TestCheckResourceAttr("data.anecdotes_playbook_library.test", "events.0.is_available", "true"),
					// An event reports the actions a step subscribing to it may
					// perform: the set that makes a step runnable.
					resource.TestCheckResourceAttrSet("data.anecdotes_playbook_library.test", "events.0.supported_actions.#"),
					// And the fields it carries, so a filter or a payload can
					// reference one instead of guessing a name.
					testCheckAnyNested("data.anecdotes_playbook_library.test", "events.0.event_fields", "field_id",
						func(v string) bool { return v != "" }),
					testCheckAnyNested("data.anecdotes_playbook_library.test", "events.0.event_fields", "is_filterable",
						func(v string) bool { return v == "true" }),
					// A filterable field reports the operator a filter must use.
					testCheckAnyNested("data.anecdotes_playbook_library.test", "events.0.event_fields", "aql_operator",
						func(v string) bool { return v != "" }),
				),
			},
		},
	})
}

func TestAccPlaybookActionLibraryDataSource_reportsRequiredFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "anecdotes_playbook_action_library" "test" {
  exclude_coming_soon = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_playbook_action_library.test", 0),
					resource.TestCheckResourceAttrSet("data.anecdotes_playbook_action_library.test", "actions.0.action_type"),
					// The filter must actually filter: nothing announced is listed.
					testCheckNoNested("data.anecdotes_playbook_action_library.test", "actions", "coming_soon",
						func(v string) bool { return v == "true" }),
					// An action reports the fields it takes, and which of them a
					// step must supply in its payload_configuration.
					testCheckAnyNested("data.anecdotes_playbook_action_library.test", "actions.0.action_fields", "field_id",
						func(v string) bool { return v != "" }),
				),
			},
		},
	})
}
