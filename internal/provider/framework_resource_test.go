// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFrameworkResource_create(t *testing.T) {
	name := randomName("fw")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_framework" "test" {
  name        = %q
  description = "Test framework for acceptance tests"
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_framework.test", "name", name),
					resource.TestCheckResourceAttr("anecdotes_framework.test", "description", "Test framework for acceptance tests"),
					resource.TestCheckResourceAttrSet("anecdotes_framework.test", "framework_id"),
				),
			},
		},
	})
}

func TestAccFrameworkResource_update(t *testing.T) {
	name1 := randomName("fw")
	name2 := randomName("fw-upd")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_framework" "test" {
  name        = %q
  description = "Original description"
}`, name1),
				Check: resource.TestCheckResourceAttr("anecdotes_framework.test", "description", "Original description"),
			},
			{
				Config: fmt.Sprintf(`
resource "anecdotes_framework" "test" {
  name                = %q
  description         = "Updated description"
  framework_auditable = true
}`, name2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_framework.test", "name", name2),
					resource.TestCheckResourceAttr("anecdotes_framework.test", "description", "Updated description"),
					resource.TestCheckResourceAttr("anecdotes_framework.test", "framework_auditable", "true"),
				),
			},
		},
	})
}

func TestAccFrameworkResource_import(t *testing.T) {
	name := randomName("fw-imp")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_framework" "test" {
  name        = %q
  description = "Import test"
}`, name),
			},
			{
				ResourceName:                         "anecdotes_framework.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "framework_id",
				ImportStateIdFunc:                    importIDFromAttr("anecdotes_framework.test", "framework_id"),
			},
		},
	})
}
