// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
				Check:  testCheckSomeEvidenceReportsInstanceIDs("data.anecdotes_evidences.all"),
			},
		},
	})
}

// testCheckSomeEvidenceReportsInstanceIDs asserts at least one evidence reports
// a non-empty service_instance_ids.
func testCheckSomeEvidenceReportsInstanceIDs(resourceAddr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceAddr]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceAddr)
		}
		count, err := strconv.Atoi(rs.Primary.Attributes["evidences.#"])
		if err != nil {
			return fmt.Errorf("reading evidences.# on %s: %w", resourceAddr, err)
		}
		for i := 0; i < count; i++ {
			n := rs.Primary.Attributes[fmt.Sprintf("evidences.%d.service_instance_ids.#", i)]
			if n != "" && n != "0" {
				return nil
			}
		}
		return fmt.Errorf("none of the %d evidences reported a service_instance_ids entry", count)
	}
}

// TestAccEvidencesDataSource_attributeSurface asserts every attribute the plural
// evidences data source maps onto each listed evidence.
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
					resource.TestCheckResourceAttrSet(addr, "evidences.0.evidence_id"),
					resource.TestCheckResourceAttrSet(addr, "evidences.0.evidence_instance_id"),
					resource.TestCheckResourceAttrSet(addr, "evidences.0.name"),
					resource.TestCheckResourceAttrSet(addr, "evidences.0.display_name"),
					resource.TestCheckResourceAttrSet(addr, "evidences.0.evidence_type"),
					resource.TestCheckResourceAttrSet(addr, "evidences.0.service_id"),
					resource.TestCheckResourceAttrSet(addr, "evidences.0.service_display_name"),
					resource.TestCheckResourceAttrSet(addr, "evidences.0.is_applicable"),
					resource.TestCheckResourceAttrSet(addr, "evidences.0.is_custom"),
					resource.TestCheckResourceAttrSet(addr, "evidences.0.is_uar"),
					resource.TestCheckResourceAttrSet(addr, "evidences.0.items_count"),
					resource.TestCheckResourceAttrSet(addr, "evidences.0.processing_state"),
					resource.TestCheckResourceAttrSet(addr, "evidences.0.entity_type"),
					resource.TestCheckResourceAttrSet(addr, "evidences.0.parent_id"),
					resource.TestCheckResourceAttrSet(addr, "evidences.0.collection_timestamp"),
					resource.TestCheckResourceAttrSet(addr, "evidences.0.service_instance_ids.#"),
					// Returned empty for evidences that were never collected on a schedule.
					testCheckAttrPresent(addr, "evidences.0.creation_time"),
					testCheckAttrPresent(addr, "evidences.0.url"),
				),
			},
		},
	})
}
