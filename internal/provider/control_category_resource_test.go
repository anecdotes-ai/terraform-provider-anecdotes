// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccControlCategoryResource_create(t *testing.T) {
	fwName := randomName("fw")
	catName := randomName("cat")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlCategoryConfig(fwName, catName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_control_category.test", "name", catName),
					resource.TestCheckResourceAttrSet("anecdotes_control_category.test", "category_id"),
					resource.TestCheckResourceAttrSet("anecdotes_control_category.test", "framework_id"),
				),
			},
		},
	})
}

func TestAccControlCategoryResource_update(t *testing.T) {
	fwName := randomName("fw")
	catName1 := randomName("cat")
	catName2 := randomName("cat-upd")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlCategoryConfig(fwName, catName1),
				Check:  resource.TestCheckResourceAttr("anecdotes_control_category.test", "name", catName1),
			},
			{
				Config: testAccFrameworkConfig(fwName) + fmt.Sprintf(`
resource "anecdotes_control_category" "test" {
  name         = %q
  framework_id = anecdotes_framework.test.framework_id
}`, catName2),
				Check: resource.TestCheckResourceAttr("anecdotes_control_category.test", "name", catName2),
			},
		},
	})
}

func TestAccControlCategoryResource_import(t *testing.T) {
	fwName := randomName("fw")
	catName := randomName("cat-imp")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlCategoryConfig(fwName, catName),
			},
			{
				ResourceName:      "anecdotes_control_category.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
