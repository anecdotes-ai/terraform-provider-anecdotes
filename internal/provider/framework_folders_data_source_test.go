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

// TestAccFrameworkFoldersDataSource_attributeSurface asserts the folders data source
// reports the frameworks each folder holds.
func TestAccFrameworkFoldersDataSource_attributeSurface(t *testing.T) {
	name := randomName("fw-folders-surface")
	const addr = "data.anecdotes_framework_folders.surface"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFrameworkConfig(name) + `
data "anecdotes_framework_folders" "surface" {
  depends_on = [anecdotes_framework.test]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckAnyAttrSet(addr, `^folders\.\d+\.frameworks_list\.#$`),
				),
			},
		},
	})
}
