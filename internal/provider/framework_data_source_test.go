// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
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

// TestAccFrameworkDataSource_attributeSurface asserts the attributes the other tests for
// this data source do not cover. The framework is created with auditor configuration, so
// the auditor status objects are populated.
func TestAccFrameworkDataSource_attributeSurface(t *testing.T) {
	folderName := randomName("folder-ds-surface")
	fwName := randomName("fw-ds-surface")
	const addr = "data.anecdotes_framework.surface"

	config := fmt.Sprintf(`
resource "anecdotes_framework_folder" "surface" {
  name = %[1]q
}

resource "anecdotes_framework" "surface" {
  name                          = %[2]q
  description                   = "Attribute surface coverage"
  folder_id                     = anecdotes_framework_folder.surface.folder_id
  can_auditor_download_evidence = false
  can_auditor_view_tags         = true

  auditor_visible_control_statuses  = ["approved_by_auditor", "gap", "monitoring"]
  auditor_visible_evidence_statuses = ["auditable", "gap"]
}

data "anecdotes_framework" "surface" {
  framework_id = anecdotes_framework.surface.framework_id
}`, folderName, fwName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(addr, "framework_status"),
					resource.TestCheckResourceAttrSet(addr, "is_applicable"),
					resource.TestCheckResourceAttrSet(addr, "framework_auditable"),
					resource.TestCheckResourceAttrSet(addr, "framework_icon_id"),
					resource.TestCheckResourceAttrSet(addr, "unadopted_order"),
					resource.TestCheckResourceAttrSet(addr, "can_auditor_view_control_attachments"),
					resource.TestCheckResourceAttrSet(addr, "can_auditor_view_control_custom_fields"),
					resource.TestCheckResourceAttrSet(addr, "can_auditor_view_soa_report"),
					testCheckAttrPresent(addr, "framework_reference_field_name"),

					resource.TestCheckResourceAttr(addr, "framework_auditor_control_status.approved_by_auditor", "true"),
					resource.TestCheckResourceAttr(addr, "framework_auditor_control_status.gap", "true"),
					resource.TestCheckResourceAttr(addr, "framework_auditor_control_status.monitoring", "true"),
					resource.TestCheckResourceAttr(addr, "framework_auditor_control_status.in_progress", "false"),
					resource.TestCheckResourceAttr(addr, "framework_auditor_control_status.insufficient_data", "false"),
					resource.TestCheckResourceAttr(addr, "framework_auditor_control_status.issue", "false"),
					resource.TestCheckResourceAttr(addr, "framework_auditor_control_status.not_applicable", "false"),
					resource.TestCheckResourceAttr(addr, "framework_auditor_control_status.not_started", "false"),
					resource.TestCheckResourceAttr(addr, "framework_auditor_control_status.ready_for_audit", "false"),
					resource.TestCheckResourceAttr(addr, "framework_auditor_control_status.under_review", "false"),

					resource.TestCheckResourceAttr(addr, "framework_auditor_evidence_status.auditable", "true"),
					resource.TestCheckResourceAttr(addr, "framework_auditor_evidence_status.gap", "true"),
					resource.TestCheckResourceAttr(addr, "framework_auditor_evidence_status.not_set", "false"),
				),
			},
		},
	})
}
