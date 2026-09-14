// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFrameworksDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_frameworks" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_frameworks.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_frameworks.test", "frameworks"),
					resource.TestCheckResourceAttrSet("data.anecdotes_frameworks.test", "frameworks.0.framework_id"),
					resource.TestCheckResourceAttrSet("data.anecdotes_frameworks.test", "frameworks.0.name"),
				),
			},
		},
	})
}

func TestAccFrameworksDataSource_filterByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_frameworks" "test" { name_contains = "SOC" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_frameworks.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_frameworks.test", "frameworks"),
				),
			},
		},
	})
}

func TestAccFrameworksDataSource_filterByApplicable(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_frameworks" "test" { is_applicable = true }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_frameworks.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_frameworks.test", "frameworks"),
				),
			},
		},
	})
}

// TestAccFrameworksDataSource_attributeSurface asserts each mapped attribute is populated
// on at least one listed framework.
func TestAccFrameworksDataSource_attributeSurface(t *testing.T) {
	const addr = "data.anecdotes_frameworks.surface"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_frameworks" "surface" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan(addr, 0),
					testCheckAnyAttrSet(addr, `^frameworks\.\d+\.description$`),
					testCheckAnyAttrSet(addr, `^frameworks\.\d+\.framework_status$`),
					testCheckAnyAttrSet(addr, `^frameworks\.\d+\.framework_auditable$`),
					testCheckAnyAttrSet(addr, `^frameworks\.\d+\.is_applicable$`),
					testCheckAnyAttrSet(addr, `^frameworks\.\d+\.categories_count$`),
					testCheckAnyAttrSet(addr, `^frameworks\.\d+\.references_count$`),
					testCheckAnyAttrPresent(addr, `^frameworks\.\d+\.framework_controls_categories\.#$`),
				),
			},
		},
	})
}
