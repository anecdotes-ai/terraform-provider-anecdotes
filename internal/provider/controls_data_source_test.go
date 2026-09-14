// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccControlsDataSource_basic creates a control and confirms the plural data
// source lists it for its framework.
func TestAccControlsDataSource_basic(t *testing.T) {
	fw := randomName("fw")
	cat := randomName("cat")
	ctrl := randomName("ctrl")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlConfig(fw, cat, ctrl) + `
data "anecdotes_controls" "test" {
  framework_id = anecdotes_control.test.framework_id
  depends_on   = [anecdotes_control.test]
}`,
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

// TestAccControlsDataSource_filterByName confirms the name_contains filter
// matches the created control.
func TestAccControlsDataSource_filterByName(t *testing.T) {
	fw := randomName("fw")
	cat := randomName("cat")
	ctrl := randomName("ctrl-filter")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlConfig(fw, cat, ctrl) + fmt.Sprintf(`
data "anecdotes_controls" "test" {
  framework_id  = anecdotes_control.test.framework_id
  name_contains = %q
  depends_on    = [anecdotes_control.test]
}`, ctrl),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_controls.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_controls.test", "controls"),
				),
			},
		},
	})
}

// TestAccControlsDataSource_attributeSurface asserts every attribute the plural
// controls data source maps onto each listed control.
func TestAccControlsDataSource_attributeSurface(t *testing.T) {
	fw := randomName("fw-ctrls-surface")
	cat := randomName("cat-ctrls-surface")
	ctrl := randomName("ctrl-surface")
	const addr = "data.anecdotes_controls.surface"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlConfig(fw, cat, ctrl) + `
data "anecdotes_controls" "surface" {
  framework_id = anecdotes_framework.test.framework_id
  depends_on   = [anecdotes_control.test]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan(addr, 0),
					testCheckAnyAttrSet(addr, `^controls\.\d+\.framework_id$`),
					testCheckAnyAttrSet(addr, `^controls\.\d+\.category_id$`),
					testCheckAnyAttrSet(addr, `^controls\.\d+\.status$`),
					// Empty on a control created without them.
					testCheckAnyAttrPresent(addr, `^controls\.\d+\.category$`),
					testCheckAnyAttrPresent(addr, `^controls\.\d+\.description$`),
					testCheckAnyAttrPresent(addr, `^controls\.\d+\.owners\.#$`),
					testCheckAnyAttrPresent(addr, `^controls\.\d+\.tags\.#$`),
				),
			},
		},
	})
}
