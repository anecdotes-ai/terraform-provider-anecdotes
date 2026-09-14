// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRoleDataSource_builtIn looks up a known built-in global role.
// viewer_role is one of the platform's standard roles and is not
// tenant-specific, so this does not depend on any fixture data existing.
func TestAccRoleDataSource_builtIn(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_role" "test" { role_id = "viewer_role" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.anecdotes_role.test", "name"),
					testCheckCollectionNotEmpty("data.anecdotes_role.test", "permissions"),
				),
			},
		},
	})
}
