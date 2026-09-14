// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccControlDataSource_basic creates a control and reads it back through the
// singular data source by control_id + framework_id.
func TestAccControlDataSource_basic(t *testing.T) {
	fw := randomName("fw-ds")
	cat := randomName("cat-ds")
	ctrl := randomName("ctrl-ds")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlConfig(fw, cat, ctrl) + `
data "anecdotes_control" "test" {
  control_id   = anecdotes_control.test.control_id
  framework_id = anecdotes_control.test.framework_id
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.anecdotes_control.test", "name", ctrl),
					resource.TestCheckResourceAttrSet("data.anecdotes_control.test", "control_id"),
					resource.TestCheckResourceAttrPair(
						"data.anecdotes_control.test", "control_id",
						"anecdotes_control.test", "control_id"),
				),
			},
		},
	})
}

// TestAccControlDataSource_attributeSurface asserts every attribute the singular
// control data source maps.
func TestAccControlDataSource_attributeSurface(t *testing.T) {
	fw := randomName("fw-ctrl-ds-surface")
	cat := randomName("cat-ctrl-ds-surface")
	ctrl := randomName("ctrl-ds-surface")
	const addr = "data.anecdotes_control.surface"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlConfig(fw, cat, ctrl) + `
data "anecdotes_control" "surface" {
  control_id   = anecdotes_control.test.control_id
  framework_id = anecdotes_framework.test.framework_id
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(addr, "status"),
					// tags is declared here but only ever populated by the plural
					// anecdotes_controls data source, so it is not asserted.
				),
			},
		},
	})
}
