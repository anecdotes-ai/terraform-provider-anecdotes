// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEvidencesDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_evidences" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan("data.anecdotes_evidences.test", 0),
					resource.TestCheckResourceAttrSet("data.anecdotes_evidences.test", "evidences.0.evidence_id"),
					resource.TestCheckResourceAttrSet("data.anecdotes_evidences.test", "evidences.0.name"),
				),
			},
		},
	})
}

func TestAccEvidencesDataSource_filterByService(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_evidences" "test" { service_id = "github" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.anecdotes_evidences.test", "total_count"),
				),
			},
		},
	})
}

func TestAccEvidencesDataSource_filterByType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_evidences" "test" { evidence_type = "MANUAL" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.anecdotes_evidences.test", "total_count"),
				),
			},
		},
	})
}

// TestAccEvidencesDataSource_reportsServiceInstanceIDs checks that
// service_instance_ids is populated, not merely declared. These IDs are what an
// analysis rule's account_scoping_list is expressed in, and the platform rejects
// a list it does not recognise, so an empty attribute would be silently useless.
func TestAccEvidencesDataSource_reportsServiceInstanceIDs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_evidences" "all" {}`,
				Check:  testCheckAnyAttrSet("data.anecdotes_evidences.all", `^evidences\.\d+\.service_instance_ids\.\d+$`),
			},
		},
	})
}

// TestAccEvidencesDataSource_attributeSurface asserts each mapped attribute is populated
// on at least one listed evidence.
func TestAccEvidencesDataSource_attributeSurface(t *testing.T) {
	const addr = "data.anecdotes_evidences.surface"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "anecdotes_evidences" "surface" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckTotalCountGreaterThan(addr, 0),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.evidence_id$`),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.evidence_instance_id$`),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.name$`),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.display_name$`),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.evidence_type$`),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.service_id$`),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.service_display_name$`),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.is_applicable$`),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.is_custom$`),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.is_uar$`),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.items_count$`),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.alert_level$`),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.processing_state$`),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.entity_type$`),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.parent_id$`),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.collection_timestamp$`),
					testCheckAnyAttrSet(addr, `^evidences\.\d+\.service_instance_ids\.\d+$`),
					testCheckAnyAttrPresent(addr, `^evidences\.\d+\.creation_time$`),
					testCheckAnyAttrPresent(addr, `^evidences\.\d+\.url$`),
				),
			},
		},
	})
}
