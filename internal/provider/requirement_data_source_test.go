// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRequirementDataSource_basic creates a requirement and reads it back
// through the singular data source by requirement_id.
func TestAccRequirementDataSource_basic(t *testing.T) {
	name := randomName("req-ds")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRequirementConfig(name) + `
data "anecdotes_requirement" "test" {
  requirement_id = anecdotes_requirement.test.requirement_id
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.anecdotes_requirement.test", "name", name),
					resource.TestCheckResourceAttrSet("data.anecdotes_requirement.test", "requirement_id"),
					resource.TestCheckResourceAttrPair(
						"data.anecdotes_requirement.test", "requirement_id",
						"anecdotes_requirement.test", "requirement_id"),
				),
			},
		},
	})
}
