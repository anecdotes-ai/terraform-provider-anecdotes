// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMappingControlRequirementResource_create(t *testing.T) {
	fwName := randomName("fw")
	catName := randomName("cat")
	ctrlName := randomName("ctrl")
	reqName := randomName("req")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlConfig(fwName, catName, ctrlName) + testAccRequirementConfig(reqName) + `
resource "anecdotes_mapping_control_requirement" "test" {
  control_id     = anecdotes_control.test.control_id
  requirement_id = anecdotes_requirement.test.requirement_id
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("anecdotes_mapping_control_requirement.test", "control_id"),
					resource.TestCheckResourceAttrSet("anecdotes_mapping_control_requirement.test", "framework_id"),
				),
			},
		},
	})
}

func TestAccMappingControlRequirementResource_import(t *testing.T) {
	fwName := randomName("fw")
	catName := randomName("cat")
	ctrlName := randomName("ctrl")
	reqName := randomName("req")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlConfig(fwName, catName, ctrlName) + testAccRequirementConfig(reqName) + `
resource "anecdotes_mapping_control_requirement" "test" {
  control_id     = anecdotes_control.test.control_id
  requirement_id = anecdotes_requirement.test.requirement_id
}`,
			},
			{
				ResourceName:                         "anecdotes_mapping_control_requirement.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "control_id",
				ImportStateIdFunc:                    importIDComposite("anecdotes_mapping_control_requirement.test", "control_id", "requirement_id"),
			},
		},
	})
}

// TestAccMappingControlRequirementResource_manyToOneControl links several
// requirements to a single control in one apply and then unlinks some of them.
// Every mapping rewrites the control's whole requirement list, so this is the
// shape that loses links if the writes are not serialized.
func TestAccMappingControlRequirementResource_manyToOneControl(t *testing.T) {
	fwName := randomName("fw-many")
	catName := randomName("cat-many")
	ctrlName := randomName("ctrl-many")
	reqName := randomName("req-many")

	config := func(count int) string {
		return testAccControlConfig(fwName, catName, ctrlName) + fmt.Sprintf(`
resource "anecdotes_requirement" "many" {
  count       = %d
  name        = "%s-${count.index}"
  description = "Acceptance test requirement"
}

resource "anecdotes_mapping_control_requirement" "many" {
  count          = %d
  control_id     = anecdotes_control.test.control_id
  requirement_id = anecdotes_requirement.many[count.index].requirement_id
}`, count, reqName, count)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(5),
				Check:  testCheckControlRequirementCount(t, "anecdotes_control.test", 5),
			},
			{
				Config: config(2),
				Check:  testCheckControlRequirementCount(t, "anecdotes_control.test", 2),
			},
		},
	})
}

// testCheckControlRequirementCount asserts how many requirements the platform
// has linked to the control in state, which is what parallel link writes race
// over.
func testCheckControlRequirementCount(t *testing.T, resourceName string, want int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		controlID, err := stateAttr(s, resourceName, "control_id")
		if err != nil {
			return err
		}
		frameworkID, err := stateAttr(s, resourceName, "framework_id")
		if err != nil {
			return err
		}
		control, err := testAccNewClient(t).GetControl(context.Background(), frameworkID, controlID)
		if err != nil {
			return fmt.Errorf("reading control %s: %w", controlID, err)
		}
		if len(control.RequirementIDs) != want {
			return fmt.Errorf("expected %d linked requirements, got %d: %v",
				want, len(control.RequirementIDs), control.RequirementIDs)
		}
		return nil
	}
}
