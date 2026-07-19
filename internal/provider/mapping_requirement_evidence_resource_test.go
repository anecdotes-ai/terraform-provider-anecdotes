// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
				ResourceName: "anecdotes_mapping_requirement_evidence.test",
				ImportState:  true,
				// The resource has no id attribute; build the documented
				// composite import ID from state.
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["anecdotes_mapping_requirement_evidence.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					return rs.Primary.Attributes["requirement_id"] + "/" + rs.Primary.Attributes["evidence_id"], nil
				},
				ImportStateVerify: true,
			},
		},
	})
}
