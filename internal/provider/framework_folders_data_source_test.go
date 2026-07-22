// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFrameworkFoldersDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_framework_folders" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_framework_folders.test", 0),
					testCheckListCountMatchesTotalCount("data.anecdotes_framework_folders.test", "folders"),
					resource.TestCheckResourceAttrSet("data.anecdotes_framework_folders.test", "folders.0.folder_id"),
					resource.TestCheckResourceAttrSet("data.anecdotes_framework_folders.test", "folders.0.name"),
				),
			},
		},
	})
}
