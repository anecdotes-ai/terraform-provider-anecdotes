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
