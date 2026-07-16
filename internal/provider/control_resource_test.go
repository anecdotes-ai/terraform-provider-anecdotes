// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccControlResource_create(t *testing.T) {
	fwName := randomName("fw")
	catName := randomName("cat")
	ctrlName := randomName("ctrl")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlConfig(fwName, catName, ctrlName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_control.test", "name", ctrlName),
					resource.TestCheckResourceAttrSet("anecdotes_control.test", "control_id"),
					resource.TestCheckResourceAttrSet("anecdotes_control.test", "framework_id"),
					resource.TestCheckResourceAttrSet("anecdotes_control.test", "category_id"),
				),
			},
		},
	})
}

func TestAccControlResource_update(t *testing.T) {
	fwName := randomName("fw")
	catName := randomName("cat")
	ctrlName1 := randomName("ctrl")
	ctrlName2 := randomName("ctrl-upd")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlConfig(fwName, catName, ctrlName1),
				Check:  resource.TestCheckResourceAttr("anecdotes_control.test", "name", ctrlName1),
			},
			{
				Config: testAccControlCategoryConfig(fwName, catName) + fmt.Sprintf(`
resource "anecdotes_control" "test" {
  name         = %q
  description  = "Updated control"
  framework_id = anecdotes_framework.test.framework_id
  category_id  = anecdotes_control_category.test.category_id
}`, ctrlName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_control.test", "name", ctrlName2),
					resource.TestCheckResourceAttr("anecdotes_control.test", "description", "Updated control"),
				),
			},
		},
	})
}

func TestAccControlResource_import(t *testing.T) {
	fwName := randomName("fw")
	catName := randomName("cat")
	ctrlName := randomName("ctrl-imp")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlConfig(fwName, catName, ctrlName),
			},
			{
				ResourceName: "anecdotes_control.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["anecdotes_control.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: anecdotes_control.test")
					}
					return rs.Primary.Attributes["framework_id"] + "/" + rs.Primary.Attributes["control_id"], nil
				},
				ImportStateVerify: true,
			},
		},
	})
}
