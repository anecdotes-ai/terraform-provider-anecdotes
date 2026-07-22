// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRequirementsDataSource_basic creates a requirement and confirms the
// plural data source lists it. include_unlinked ensures the standalone
// requirement is counted even before it is linked to a control.
func TestAccRequirementsDataSource_basic(t *testing.T) {
	name := randomName("req")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRequirementConfig(name) + `
data "anecdotes_requirements" "test" {
  include_unlinked = true
  depends_on       = [anecdotes_requirement.test]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_requirements.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_requirements.test", "requirements"),
				),
			},
		},
	})
}

// TestAccRequirementsDataSource_filterByName confirms the name_contains filter
// matches the created requirement.
func TestAccRequirementsDataSource_filterByName(t *testing.T) {
	name := randomName("req-filter")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRequirementConfig(name) + fmt.Sprintf(`
data "anecdotes_requirements" "test" {
  include_unlinked = true
  name_contains    = %q
  depends_on       = [anecdotes_requirement.test]
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_requirements.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_requirements.test", "requirements"),
				),
			},
		},
	})
}

// TestAccRequirementsDataSource_filterByIsCustom confirms the is_custom filter
// returns the created (custom) requirement.
func TestAccRequirementsDataSource_filterByIsCustom(t *testing.T) {
	name := randomName("req-custom")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRequirementConfig(name) + `
data "anecdotes_requirements" "test" {
  is_custom        = true
  include_unlinked = true
  depends_on       = [anecdotes_requirement.test]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_requirements.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_requirements.test", "requirements"),
				),
			},
		},
	})
}
