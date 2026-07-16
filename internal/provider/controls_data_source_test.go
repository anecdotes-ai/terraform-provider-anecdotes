// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccControlsDataSource_basic(t *testing.T) {
	// Use a known framework with controls (the chained approach may pick a framework with 0 controls)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_controls" "test" { framework_id = "1234567890" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_controls.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_controls.test", "controls"),
					resource.TestCheckResourceAttrSet("data.anecdotes_controls.test", "controls.0.control_id"),
					resource.TestCheckResourceAttrSet("data.anecdotes_controls.test", "controls.0.name"),
				),
			},
		},
	})
}

func TestAccControlsDataSource_withKnownFrameworkID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_controls" "test" { framework_id = "1234567890" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_controls.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_controls.test", "controls"),
				),
			},
		},
	})
}

func TestAccControlsDataSource_filterByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "anecdotes_controls" "test" {
  framework_id  = "1234567890"
  name_contains = "access"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_controls.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_controls.test", "controls"),
				),
			},
		},
	})
}
