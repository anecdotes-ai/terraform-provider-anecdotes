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
