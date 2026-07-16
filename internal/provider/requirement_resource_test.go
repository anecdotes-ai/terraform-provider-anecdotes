// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRequirementResource_create(t *testing.T) {
	name := randomName("req")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_requirement" "test" {
  name        = %q
  description = "Test requirement"
  category    = "Custom Requirements"
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_requirement.test", "name", name),
					resource.TestCheckResourceAttrSet("anecdotes_requirement.test", "requirement_id"),
				),
			},
		},
	})
}

func TestAccRequirementResource_update(t *testing.T) {
	name1 := randomName("req")
	name2 := randomName("req-upd")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_requirement" "test" {
  name        = %q
  description = "Original"
}`, name1),
				Check: resource.TestCheckResourceAttr("anecdotes_requirement.test", "name", name1),
			},
			{
				Config: fmt.Sprintf(`
resource "anecdotes_requirement" "test" {
  name        = %q
  description = "Updated description"
}`, name2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_requirement.test", "name", name2),
					resource.TestCheckResourceAttr("anecdotes_requirement.test", "description", "Updated description"),
				),
			},
		},
	})
}

func TestAccRequirementResource_import(t *testing.T) {
	name := randomName("req-imp")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_requirement" "test" {
  name        = %q
  description = "Import test"
}`, name),
			},
			{
				ResourceName:      "anecdotes_requirement.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
