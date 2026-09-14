// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccControlCategoriesDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_control_categories" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_control_categories.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_control_categories.test", "categories"),
				),
			},
		},
	})
}

func TestAccControlCategoriesDataSource_filterByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_control_categories" "test" { name_contains = "common" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountAtLeast("data.anecdotes_control_categories.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_control_categories.test", "categories"),
				),
			},
		},
	})
}

// TestAccControlCategoriesDataSource_attributeSurface asserts each mapped attribute is
// populated on at least one listed category.
func TestAccControlCategoriesDataSource_attributeSurface(t *testing.T) {
	fw := randomName("fw-cats-surface")
	cat := randomName("cat-surface")
	const addr = "data.anecdotes_control_categories.surface"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlCategoryConfig(fw, cat) + `
data "anecdotes_control_categories" "surface" {
  framework_id = anecdotes_framework.test.framework_id
  depends_on   = [anecdotes_control_category.test]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAnyAttrSet(addr, `^categories\.\d+\.category_id$`),
					testCheckAnyAttrSet(addr, `^categories\.\d+\.category_name$`),
					testCheckAnyAttrSet(addr, `^categories\.\d+\.framework_id$`),
				),
			},
		},
	})
}
