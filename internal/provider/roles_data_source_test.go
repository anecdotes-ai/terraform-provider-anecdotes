// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRolesDataSource_builtInOnly(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_roles" "test" { is_custom = false }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_roles.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_roles.test", "roles"),
					resource.TestCheckResourceAttr("data.anecdotes_roles.test", "roles.0.is_custom", "false"),
				),
			},
		},
	})
}

func TestAccRolesDataSource_nameContains(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_roles" "test" { name_contains = "viewer" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_roles.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_roles.test", "roles"),
				),
			},
		},
	})
}
