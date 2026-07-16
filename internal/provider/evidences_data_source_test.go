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
