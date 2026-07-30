// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccFrameworkResource_create(t *testing.T) {
	name := randomName("fw")
	folder := randomName("folder-fw")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_framework_folder" "test" {
  name = %q
}

resource "anecdotes_framework" "test" {
  name        = %q
  description = "Test framework for acceptance tests"
  folder_id   = anecdotes_framework_folder.test.folder_id
}`, folder, name),
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
	folder := randomName("folder-fw-upd")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_framework_folder" "test" {
  name = %q
}

resource "anecdotes_framework" "test" {
  name        = %q
  description = "Original description"
  folder_id   = anecdotes_framework_folder.test.folder_id
}`, folder, name1),
				Check: resource.TestCheckResourceAttr("anecdotes_framework.test", "description", "Original description"),
			},
			{
				Config: fmt.Sprintf(`
resource "anecdotes_framework_folder" "test" {
  name = %q
}

resource "anecdotes_framework" "test" {
  name                  = %q
  description           = "Updated description"
  can_auditor_view_tags = true
  folder_id             = anecdotes_framework_folder.test.folder_id
}`, folder, name2),
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
	folder := randomName("folder-fw-imp")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_framework_folder" "test" {
  name = %q
}

resource "anecdotes_framework" "test" {
  name        = %q
  description = "Import test"
  folder_id   = anecdotes_framework_folder.test.folder_id
}`, folder, name),
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

// TestAccFrameworkResource_folderMove verifies that changing folder_id updates
// the framework in place rather than replacing it.
func TestAccFrameworkResource_folderMove(t *testing.T) {
	folderA := randomName("folder-move-a")
	folderB := randomName("folder-move-b")
	fwName := randomName("fw-move")

	configTemplate := `
resource "anecdotes_framework_folder" "a" {
  name = %q
}

resource "anecdotes_framework_folder" "b" {
  name = %q
}

resource "anecdotes_framework" "test" {
  name        = %q
  description = "Folder move test"
  folder_id   = anecdotes_framework_folder.%s.folder_id
}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configTemplate, folderA, folderB, fwName, "a"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"anecdotes_framework.test", "folder_id",
						"anecdotes_framework_folder.a", "folder_id",
					),
				),
			},
			{
				Config: fmt.Sprintf(configTemplate, folderA, folderB, fwName, "b"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("anecdotes_framework.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"anecdotes_framework.test", "folder_id",
						"anecdotes_framework_folder.b", "folder_id",
					),
				),
			},
		},
	})
}

// TestAccFrameworkResource_auditorConfig covers the auditor-visibility surface
// (booleans, both status sets, folder placement) that create applies via
// follow-up calls, and verifies it all round-trips through import.
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
			},
		},
	})
}
