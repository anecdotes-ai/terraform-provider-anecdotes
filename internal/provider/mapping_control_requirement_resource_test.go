// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
				ResourceName:      "anecdotes_mapping_control_requirement.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
