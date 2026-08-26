// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testCheckRequirementViewOwnersOnPlatform reads the view straight from the
// API (bypassing Terraform state) to confirm owners set at create time were
// actually persisted by the platform, not just echoed into local state.
func testCheckRequirementViewOwnersOnPlatform(t *testing.T, resourceName string, wantOwners []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := stateAttr(s, resourceName, "requirement_id")
		if err != nil {
			return err
		}
		view, err := testAccNewClient(t).GetRequirement(context.Background(), id)
		if err != nil {
			return fmt.Errorf("reading requirement view %s: %w", id, err)
		}
		if len(view.RequirementOwners) != len(wantOwners) {
			return fmt.Errorf("expected owners %v on the platform, got %v", wantOwners, view.RequirementOwners)
		}
		for i, owner := range wantOwners {
			if view.RequirementOwners[i] != owner {
				return fmt.Errorf("expected owners %v on the platform, got %v", wantOwners, view.RequirementOwners)
			}
		}
		return nil
	}
}

// A Requirement View is linked to a control the same way any other
// requirement is: through anecdotes_mapping_control_requirement, using the
// view's own requirement_id.
func TestAccRequirementViewResource_linkedToControlViaMapping(t *testing.T) {
	fwName := randomName("fw-req-view")
	catName := randomName("cat-req-view")
	ctrlName := randomName("ctrl-req-view")
	parentName := randomName("req-view-parent")
	viewName := randomName("req-view")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlConfig(fwName, catName, ctrlName) + testAccRequirementConfig(parentName) + fmt.Sprintf(`
resource "anecdotes_requirement_view" "test" {
  parent_id = anecdotes_requirement.test.requirement_id
  view_name = %q
}

resource "anecdotes_mapping_control_requirement" "test" {
  control_id     = anecdotes_control.test.control_id
  requirement_id = anecdotes_requirement_view.test.requirement_id
}`, viewName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("anecdotes_mapping_control_requirement.test", "control_id"),
					resource.TestCheckResourceAttrSet("anecdotes_mapping_control_requirement.test", "framework_id"),
					resource.TestCheckResourceAttrPair(
						"anecdotes_mapping_control_requirement.test", "requirement_id",
						"anecdotes_requirement_view.test", "requirement_id",
					),
				),
			},
		},
	})
}

func TestAccRequirementViewResource_create(t *testing.T) {
	parentName := randomName("req-view-parent")
	viewName := randomName("req-view")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_requirement" "parent" {
  name        = %q
  description = "Parent for a Requirement View acceptance test"
}

resource "anecdotes_requirement_view" "test" {
  parent_id = anecdotes_requirement.parent.requirement_id
  view_name = %q
  owners    = ["acceptance-tests@example.com"]
}`, parentName, viewName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_requirement_view.test", "view_name", viewName),
					resource.TestCheckResourceAttr("anecdotes_requirement_view.test", "category", "Custom Requirements"),
					resource.TestCheckResourceAttrSet("anecdotes_requirement_view.test", "requirement_id"),
					resource.TestCheckResourceAttrPair(
						"anecdotes_requirement_view.test", "parent_id",
						"anecdotes_requirement.parent", "requirement_id",
					),
					resource.TestCheckResourceAttr("anecdotes_requirement_view.test", "owners.#", "1"),
					resource.TestCheckResourceAttr("anecdotes_requirement_view.test", "owners.0", "acceptance-tests@example.com"),
					testCheckRequirementViewOwnersOnPlatform(t, "anecdotes_requirement_view.test", []string{"acceptance-tests@example.com"}),
				),
			},
		},
	})
}

func TestAccRequirementViewResource_update(t *testing.T) {
	parentName := randomName("req-view-parent")
	viewName1 := randomName("req-view")
	viewName2 := randomName("req-view-upd")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_requirement" "parent" {
  name        = %q
  description = "Parent for a Requirement View acceptance test"
}

resource "anecdotes_requirement_view" "test" {
  parent_id = anecdotes_requirement.parent.requirement_id
  view_name = %q
}`, parentName, viewName1),
				Check: resource.TestCheckResourceAttr("anecdotes_requirement_view.test", "view_name", viewName1),
			},
			{
				Config: fmt.Sprintf(`
resource "anecdotes_requirement" "parent" {
  name        = %q
  description = "Parent for a Requirement View acceptance test"
}

resource "anecdotes_requirement_view" "test" {
  parent_id = anecdotes_requirement.parent.requirement_id
  view_name = %q
  category  = "Access"
}`, parentName, viewName2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("anecdotes_requirement_view.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_requirement_view.test", "view_name", viewName2),
					resource.TestCheckResourceAttr("anecdotes_requirement_view.test", "category", "Access"),
				),
			},
		},
	})
}

// requirement_parent_id is immutable at the API: changing parent_id in
// configuration must replace the view, not attempt an in-place update.
func TestAccRequirementViewResource_changingParentForcesReplace(t *testing.T) {
	parent1Name := randomName("req-view-parent1")
	parent2Name := randomName("req-view-parent2")
	viewName := randomName("req-view")
	config := func(parentAttr string) string {
		return fmt.Sprintf(`
resource "anecdotes_requirement" "parent1" {
  name        = %q
  description = "First parent"
}

resource "anecdotes_requirement" "parent2" {
  name        = %q
  description = "Second parent"
}

resource "anecdotes_requirement_view" "test" {
  parent_id = %s
  view_name = %q
}`, parent1Name, parent2Name, parentAttr, viewName)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("anecdotes_requirement.parent1.requirement_id"),
				Check: resource.TestCheckResourceAttrPair(
					"anecdotes_requirement_view.test", "parent_id",
					"anecdotes_requirement.parent1", "requirement_id",
				),
			},
			{
				Config: config("anecdotes_requirement.parent2.requirement_id"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("anecdotes_requirement_view.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.TestCheckResourceAttrPair(
					"anecdotes_requirement_view.test", "parent_id",
					"anecdotes_requirement.parent2", "requirement_id",
				),
			},
		},
	})
}

func TestAccRequirementViewResource_import(t *testing.T) {
	parentName := randomName("req-view-parent")
	viewName := randomName("req-view-imp")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_requirement" "parent" {
  name        = %q
  description = "Parent for a Requirement View acceptance test"
}

resource "anecdotes_requirement_view" "test" {
  parent_id = anecdotes_requirement.parent.requirement_id
  view_name = %q
}`, parentName, viewName),
			},
			{
				ResourceName:                         "anecdotes_requirement_view.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "requirement_id",
				ImportStateIdFunc:                    importIDFromAttr("anecdotes_requirement_view.test", "requirement_id"),
			},
		},
	})
}

// A view cannot itself be the parent of another view. The API rejects this at
// apply time; it must surface as a normal diagnostic, not a panic.
func TestAccRequirementViewResource_nestedViewRejected(t *testing.T) {
	parentName := randomName("req-view-parent")
	viewName := randomName("req-view")
	nestedName := randomName("req-view-nested")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_requirement" "parent" {
  name        = %q
  description = "Parent for a Requirement View acceptance test"
}

resource "anecdotes_requirement_view" "test" {
  parent_id = anecdotes_requirement.parent.requirement_id
  view_name = %q
}

resource "anecdotes_requirement_view" "nested" {
  parent_id = anecdotes_requirement_view.test.requirement_id
  view_name = %q
}`, parentName, viewName, nestedName),
				ExpectError: regexp.MustCompile(`(?s)create requirement view.*Cannot create a view under another view`),
			},
		},
	})
}
