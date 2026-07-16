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
