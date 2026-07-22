// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFrameworkFolderResource_create(t *testing.T) {
	name := randomName("folder")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "anecdotes_framework_folder" "test" { name = %q }`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_framework_folder.test", "name", name),
					resource.TestCheckResourceAttrSet("anecdotes_framework_folder.test", "folder_id"),
				),
			},
		},
	})
}

func TestAccFrameworkFolderResource_update(t *testing.T) {
	name1 := randomName("folder")
	name2 := randomName("folder-upd")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "anecdotes_framework_folder" "test" { name = %q }`, name1),
				Check:  resource.TestCheckResourceAttr("anecdotes_framework_folder.test", "name", name1),
			},
			{
				Config: fmt.Sprintf(`resource "anecdotes_framework_folder" "test" { name = %q }`, name2),
				Check:  resource.TestCheckResourceAttr("anecdotes_framework_folder.test", "name", name2),
			},
		},
	})
}

func TestAccFrameworkFolderResource_import(t *testing.T) {
	name := randomName("folder-imp")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "anecdotes_framework_folder" "test" { name = %q }`, name),
			},
			{
				ResourceName:                         "anecdotes_framework_folder.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "folder_id",
				ImportStateIdFunc:                    importIDFromAttr("anecdotes_framework_folder.test", "folder_id"),
			},
		},
	})
}
