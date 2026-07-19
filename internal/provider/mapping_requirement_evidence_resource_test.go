// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMappingRequirementEvidenceResource_create(t *testing.T) {
	reqName := randomName("req")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRequirementConfig(reqName) + fmt.Sprintf(`
resource "anecdotes_mapping_requirement_evidence" "test" {
  requirement_id = anecdotes_requirement.test.requirement_id
  evidence_id    = %q
}`, testAccEvidenceID()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("anecdotes_mapping_requirement_evidence.test", "requirement_id"),
				),
			},
		},
	})
}

func TestAccMappingRequirementEvidenceResource_import(t *testing.T) {
	reqName := randomName("req")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRequirementConfig(reqName) + fmt.Sprintf(`
resource "anecdotes_mapping_requirement_evidence" "test" {
  requirement_id = anecdotes_requirement.test.requirement_id
  evidence_id    = %q
}`, testAccEvidenceID()),
			},
			{
				ResourceName:                         "anecdotes_mapping_requirement_evidence.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "requirement_id",
				ImportStateIdFunc:                    importIDComposite("anecdotes_mapping_requirement_evidence.test", "requirement_id", "evidence_id"),
			},
		},
	})
}
