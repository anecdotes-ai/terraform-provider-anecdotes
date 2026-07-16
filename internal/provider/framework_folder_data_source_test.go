// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccFrameworkFolderDataSource_basic creates a framework folder and reads it
// back through the singular data source by name.
func TestAccFrameworkFolderDataSource_basic(t *testing.T) {
	name := randomName("folder-ds")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_framework_folder" "test" {
  name = %q
}

data "anecdotes_framework_folder" "test" {
  name = anecdotes_framework_folder.test.name
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.anecdotes_framework_folder.test", "name", name),
					resource.TestCheckResourceAttrSet("data.anecdotes_framework_folder.test", "folder_id"),
					resource.TestCheckResourceAttrPair(
						"data.anecdotes_framework_folder.test", "folder_id",
						"anecdotes_framework_folder.test", "folder_id"),
				),
			},
		},
	})
}
