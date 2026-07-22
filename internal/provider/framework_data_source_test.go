// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccFrameworkDataSource_basic creates a framework and reads it back through
// the singular data source by framework_id, asserting the looked-up attributes match.
func TestAccFrameworkDataSource_basic(t *testing.T) {
	name := randomName("fw-ds")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFrameworkConfig(name) + `
data "anecdotes_framework" "test" {
  framework_id = anecdotes_framework.test.framework_id
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.anecdotes_framework.test", "name", name),
					resource.TestCheckResourceAttrSet("data.anecdotes_framework.test", "framework_id"),
					resource.TestCheckResourceAttrPair(
						"data.anecdotes_framework.test", "framework_id",
						"anecdotes_framework.test", "framework_id"),
				),
			},
		},
	})
}
