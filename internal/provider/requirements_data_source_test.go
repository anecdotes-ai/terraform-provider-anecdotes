// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRequirementsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_requirements" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_requirements.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_requirements.test", "requirements"),
				),
			},
		},
	})
}

func TestAccRequirementsDataSource_filterByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_requirements" "test" { name_contains = "password" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_requirements.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_requirements.test", "requirements"),
				),
			},
		},
	})
}

func TestAccRequirementsDataSource_filterByIsCustom(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_requirements" "test" { is_custom = true }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountAtLeast("data.anecdotes_requirements.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_requirements.test", "requirements"),
				),
			},
		},
	})
}
