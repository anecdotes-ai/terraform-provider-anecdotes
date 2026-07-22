// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccControlCategoryDataSource_basic creates a control category and reads it
// back through the singular data source by category_id.
func TestAccControlCategoryDataSource_basic(t *testing.T) {
	fw := randomName("fw-ds")
	cat := randomName("cat-ds")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlCategoryConfig(fw, cat) + `
data "anecdotes_control_category" "test" {
  category_id = anecdotes_control_category.test.category_id
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.anecdotes_control_category.test", "category_name", cat),
					resource.TestCheckResourceAttrSet("data.anecdotes_control_category.test", "framework_id"),
					resource.TestCheckResourceAttrPair(
						"data.anecdotes_control_category.test", "category_id",
						"anecdotes_control_category.test", "category_id"),
				),
			},
		},
	})
}
