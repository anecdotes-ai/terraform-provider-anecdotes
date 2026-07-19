// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMappingControlRequirementResource_create(t *testing.T) {
	fwName := randomName("fw")
	catName := randomName("cat")
	ctrlName := randomName("ctrl")
	reqName := randomName("req")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlConfig(fwName, catName, ctrlName) + testAccRequirementConfig(reqName) + `
resource "anecdotes_mapping_control_requirement" "test" {
  control_id     = anecdotes_control.test.control_id
  requirement_id = anecdotes_requirement.test.requirement_id
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("anecdotes_mapping_control_requirement.test", "control_id"),
					resource.TestCheckResourceAttrSet("anecdotes_mapping_control_requirement.test", "framework_id"),
				),
			},
		},
	})
}

func TestAccMappingControlRequirementResource_import(t *testing.T) {
	fwName := randomName("fw")
	catName := randomName("cat")
	ctrlName := randomName("ctrl")
	reqName := randomName("req")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlConfig(fwName, catName, ctrlName) + testAccRequirementConfig(reqName) + `
resource "anecdotes_mapping_control_requirement" "test" {
  control_id     = anecdotes_control.test.control_id
  requirement_id = anecdotes_requirement.test.requirement_id
}`,
			},
			{
				ResourceName: "anecdotes_mapping_control_requirement.test",
				ImportState:  true,
				// The resource has no id attribute; build the documented
				// composite import ID from state.
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["anecdotes_mapping_control_requirement.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					return rs.Primary.Attributes["control_id"] + "/" + rs.Primary.Attributes["requirement_id"], nil
				},
				ImportStateVerify: true,
			},
		},
	})
}
