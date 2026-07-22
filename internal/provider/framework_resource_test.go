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
  name                  = %q
  description           = "Updated description"
  can_auditor_view_tags = true
}`, name2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_framework.test", "name", name2),
					resource.TestCheckResourceAttr("anecdotes_framework.test", "description", "Updated description"),
					resource.TestCheckResourceAttr("anecdotes_framework.test", "can_auditor_view_tags", "true"),
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

// TestAccFrameworkResource_auditorConfig covers the auditor-visibility surface
// (booleans, both status sets, folder placement) that create applies via
// follow-up calls. It import-verifies all but folder_id (not returned on read).
func TestAccFrameworkResource_auditorConfig(t *testing.T) {
	folderName := randomName("folder-audit")
	fwName := randomName("fw-audit")
	config := fmt.Sprintf(`
resource "anecdotes_framework_folder" "test" {
  name = %q
}

resource "anecdotes_framework" "test" {
  name                          = %q
  description                   = "Auditor configuration coverage"
  folder_id                     = anecdotes_framework_folder.test.folder_id
  can_auditor_download_evidence = false
  can_auditor_view_tags         = true

  auditor_visible_control_statuses  = ["approved_by_auditor", "gap", "monitoring"]
  auditor_visible_evidence_statuses = ["auditable", "gap"]
}`, folderName, fwName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_framework.test", "can_auditor_download_evidence", "false"),
					resource.TestCheckResourceAttr("anecdotes_framework.test", "can_auditor_view_tags", "true"),
					resource.TestCheckResourceAttr("anecdotes_framework.test", "auditor_visible_control_statuses.#", "3"),
					resource.TestCheckResourceAttr("anecdotes_framework.test", "auditor_visible_evidence_statuses.#", "2"),
					resource.TestCheckResourceAttrSet("anecdotes_framework.test", "folder_id"),
				),
			},
			{
				ResourceName:                         "anecdotes_framework.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "framework_id",
				ImportStateIdFunc:                    importIDFromAttr("anecdotes_framework.test", "framework_id"),
				// folder_id is not returned by the framework read, so it cannot
				// round-trip on import (documented known limitation).
				ImportStateVerifyIgnore: []string{"folder_id"},
			},
		},
	})
}
