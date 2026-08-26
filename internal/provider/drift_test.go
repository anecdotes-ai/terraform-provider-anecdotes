// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
)

// Drift tests change managed objects out-of-band (directly through the API
// client, simulating a UI edit) and assert that the provider detects the
// change and reconciles it on the next apply.

// testAccNewClient builds a raw API client for out-of-band mutations.
func testAccNewClient(t *testing.T) *client.AnecdotesClient {
	t.Helper()
	apiURL := os.Getenv("ANECDOTES_API_URL")
	if apiURL == "" {
		apiURL = "https://api.anecdotes.ai"
	}
	c, err := client.NewAnecdotesClient(context.Background(), os.Getenv("ANECDOTES_API_KEY"), apiURL)
	if err != nil {
		t.Fatalf("failed to create API client for drift test: %v", err)
	}
	return c
}

// stateAttr reads an attribute of a resource from the current state.
func stateAttr(s *terraform.State, resourceName, attr string) (string, error) {
	rs, ok := s.RootModule().Resources[resourceName]
	if !ok {
		return "", fmt.Errorf("resource %s not found in state", resourceName)
	}
	return rs.Primary.Attributes[attr], nil
}

// TestAccDrift_FrameworkDescriptionRevert: an out-of-band change to a
// TF-owned field must surface as drift and be reverted by the next apply.
func TestAccDrift_FrameworkDescriptionRevert(t *testing.T) {
	name := randomName("fw-drift-desc")
	folder := randomName("folder-drift-desc")
	config := fmt.Sprintf(`
resource "anecdotes_framework_folder" "test" {
  name = %q
}

resource "anecdotes_framework" "test" {
  name        = %q
  description = "Managed by Terraform"
  folder_id   = anecdotes_framework_folder.test.folder_id
}`, folder, name)

	var frameworkID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_framework.test", "description", "Managed by Terraform"),
					func(s *terraform.State) error {
						id, err := stateAttr(s, "anecdotes_framework.test", "framework_id")
						frameworkID = id
						return err
					},
				),
			},
			{
				PreConfig: func() {
					c := testAccNewClient(t)
					_, err := c.UpdateFramework(context.Background(), frameworkID, &client.FrameworkUpdateRequest{
						FrameworkName:        name,
						FrameworkDescription: "Changed outside Terraform",
					})
					if err != nil {
						t.Fatalf("out-of-band description change failed: %v", err)
					}
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_framework.test", "description", "Managed by Terraform"),
				),
			},
		},
	})
}

// TestAccDrift_FrameworkFolderMoveDetected: moving a framework to another
// folder out-of-band must surface as drift and be moved back on apply.
func TestAccDrift_FrameworkFolderMoveDetected(t *testing.T) {
	folderA := randomName("folder-drift-a")
	folderB := randomName("folder-drift-b")
	fwName := randomName("fw-drift-move")

	config := fmt.Sprintf(`
resource "anecdotes_framework_folder" "a" {
  name = %q
}

resource "anecdotes_framework_folder" "b" {
  name = %q
}

resource "anecdotes_framework" "test" {
  name        = %q
  description = "Folder drift test"
  folder_id   = anecdotes_framework_folder.a.folder_id
}`, folderA, folderB, fwName)

	var frameworkID, folderAID, folderBID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						var err error
						if frameworkID, err = stateAttr(s, "anecdotes_framework.test", "framework_id"); err != nil {
							return err
						}
						if folderAID, err = stateAttr(s, "anecdotes_framework_folder.a", "folder_id"); err != nil {
							return err
						}
						folderBID, err = stateAttr(s, "anecdotes_framework_folder.b", "folder_id")
						return err
					},
				),
			},
			{
				PreConfig: func() {
					c := testAccNewClient(t)
					if err := c.MoveFrameworkFolder(context.Background(), frameworkID, folderAID, folderBID); err != nil {
						t.Fatalf("out-of-band folder move failed: %v", err)
					}
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"anecdotes_framework.test", "folder_id",
						"anecdotes_framework_folder.a", "folder_id",
					),
				),
			},
		},
	})
}

// TestAccDrift_ControlFieldsClear: a control description set back to empty and
// a removed owners attribute must both converge (no perpetual diff).
func TestAccDrift_ControlFieldsClear(t *testing.T) {
	fwName := randomName("fw-drift-clear")
	catName := randomName("cat-drift-clear")
	ctlName := randomName("ctl-drift-clear")

	base := fmt.Sprintf(`
resource "anecdotes_framework_folder" "test" {
  name = "%s-folder"
}

resource "anecdotes_framework" "test" {
  name        = %q
  description = "Control clear test"
  folder_id   = anecdotes_framework_folder.test.folder_id
}

resource "anecdotes_control_category" "test" {
  name         = %q
  framework_id = anecdotes_framework.test.framework_id
}
`, fwName, fwName, catName)

	withValues := base + fmt.Sprintf(`
resource "anecdotes_control" "test" {
  framework_id = anecdotes_framework.test.framework_id
  category_id  = anecdotes_control_category.test.category_id
  name         = %q
  description  = "To be cleared"
  owners       = ["acceptance-tests@example.com"]
}`, ctlName)

	cleared := base + fmt.Sprintf(`
resource "anecdotes_control" "test" {
  framework_id = anecdotes_framework.test.framework_id
  category_id  = anecdotes_control_category.test.category_id
  name         = %q
  description  = ""
}`, ctlName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withValues,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_control.test", "description", "To be cleared"),
					resource.TestCheckResourceAttr("anecdotes_control.test", "owners.#", "1"),
				),
			},
			{
				Config: cleared,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_control.test", "description", ""),
					resource.TestCheckNoResourceAttr("anecdotes_control.test", "owners.#"),
				),
			},
			// A follow-up plan of the same config must be empty — this is the
			// convergence guarantee (no perpetual diff after clearing).
			{
				Config:   cleared,
				PlanOnly: true,
			},
		},
	})
}

// TestAccDrift_RequirementFieldsClear: requirement description (help text) and
// an empty owners set must both converge when cleared.
func TestAccDrift_RequirementFieldsClear(t *testing.T) {
	name := randomName("req-drift-clear")

	withValue := fmt.Sprintf(`
resource "anecdotes_requirement" "test" {
  name        = %q
  description = "To be cleared"
  owners      = ["acceptance-tests@example.com"]
}`, name)

	emptied := fmt.Sprintf(`
resource "anecdotes_requirement" "test" {
  name        = %q
  description = ""
  owners      = []
}`, name)

	// The owners attribute is removed entirely: Terraform owns it, so the
	// platform must end up with no owners.
	removed := fmt.Sprintf(`
resource "anecdotes_requirement" "test" {
  name        = %q
  description = ""
}`, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withValue,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_requirement.test", "description", "To be cleared"),
					resource.TestCheckResourceAttr("anecdotes_requirement.test", "owners.#", "1"),
				),
			},
			{
				Config: emptied,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_requirement.test", "description", ""),
					resource.TestCheckResourceAttr("anecdotes_requirement.test", "owners.#", "0"),
				),
			},
			{
				Config:   emptied,
				PlanOnly: true,
			},
			{
				Config: withValue,
				Check:  resource.TestCheckResourceAttr("anecdotes_requirement.test", "owners.#", "1"),
			},
			{
				Config: removed,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("anecdotes_requirement.test", "owners.#"),
					testCheckRequirementOwnersEmpty(t, "anecdotes_requirement.test"),
				),
			},
			{
				Config:   removed,
				PlanOnly: true,
			},
		},
	})
}

// testCheckRequirementOwnersEmpty asserts the platform holds no owners for the
// requirement in state — removing the attribute must reach the API, not just
// state.
func testCheckRequirementOwnersEmpty(t *testing.T, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := stateAttr(s, resourceName, "requirement_id")
		if err != nil {
			return err
		}
		requirement, err := testAccNewClient(t).GetRequirement(context.Background(), id)
		if err != nil {
			return fmt.Errorf("reading requirement %s: %w", id, err)
		}
		if len(requirement.RequirementOwners) != 0 {
			return fmt.Errorf("expected no owners on the platform, got %v", requirement.RequirementOwners)
		}
		return nil
	}
}

// TestAccDrift_OwnersChangedOutOfBand: owners edited outside Terraform must
// surface as drift and be set back to the configuration on the next apply, on
// both resources that expose the attribute.
func TestAccDrift_OwnersChangedOutOfBand(t *testing.T) {
	fwName := randomName("fw-drift-owners")
	catName := randomName("cat-drift-owners")
	ctlName := randomName("ctl-drift-owners")
	reqName := randomName("req-drift-owners")

	const owner = "acceptance-tests@example.com"
	const intruder = "changed-outside@example.com"

	config := testAccControlCategoryConfig(fwName, catName) + fmt.Sprintf(`
resource "anecdotes_control" "test" {
  framework_id = anecdotes_framework.test.framework_id
  category_id  = anecdotes_control_category.test.category_id
  name         = %q
  owners       = [%q]
}

resource "anecdotes_requirement" "test" {
  name   = %q
  owners = [%q]
}`, ctlName, owner, reqName, owner)

	var controlID, requirementID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_control.test", "owners.0", owner),
					resource.TestCheckResourceAttr("anecdotes_requirement.test", "owners.0", owner),
					func(s *terraform.State) error {
						var err error
						if controlID, err = stateAttr(s, "anecdotes_control.test", "control_id"); err != nil {
							return err
						}
						requirementID, err = stateAttr(s, "anecdotes_requirement.test", "requirement_id")
						return err
					},
				),
			},
			{
				PreConfig: func() {
					c := testAccNewClient(t)
					if err := c.SetControlOwners(context.Background(), controlID, []string{intruder}); err != nil {
						t.Fatalf("out-of-band control owners change failed: %v", err)
					}
					intruders := []string{intruder}
					if _, err := c.UpdateRequirement(context.Background(), requirementID, &client.RequirementUpdateRequest{
						RequirementDescription: reqName,
						RequirementOwners:      &intruders,
					}); err != nil {
						t.Fatalf("out-of-band requirement owners change failed: %v", err)
					}
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_control.test", "owners.#", "1"),
					resource.TestCheckResourceAttr("anecdotes_control.test", "owners.0", owner),
					resource.TestCheckResourceAttr("anecdotes_requirement.test", "owners.#", "1"),
					resource.TestCheckResourceAttr("anecdotes_requirement.test", "owners.0", owner),
				),
			},
		},
	})
}

// TestAccDrift_MaturityLevelClearedOnRemoval: maturity level is set, changed,
// and then cleared by removing the attribute — the platform must end up with no
// level, not the last one applied.
func TestAccDrift_MaturityLevelClearedOnRemoval(t *testing.T) {
	fwName := randomName("fw-drift-mat")
	catName := randomName("cat-drift-mat")
	ctlName := randomName("ctl-drift-mat")

	withLevel := func(level string) string {
		return testAccControlCategoryConfig(fwName, catName) + fmt.Sprintf(`
resource "anecdotes_control" "test" {
  framework_id   = anecdotes_framework.test.framework_id
  category_id    = anecdotes_control_category.test.category_id
  name           = %q
  maturity_level = %q
}`, ctlName, level)
	}

	withoutLevel := testAccControlCategoryConfig(fwName, catName) + fmt.Sprintf(`
resource "anecdotes_control" "test" {
  framework_id = anecdotes_framework.test.framework_id
  category_id  = anecdotes_control_category.test.category_id
  name         = %q
}`, ctlName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withLevel("DEFINED"),
				Check:  resource.TestCheckResourceAttr("anecdotes_control.test", "maturity_level", "DEFINED"),
			},
			{
				Config: withLevel("OPTIMIZING"),
				Check:  resource.TestCheckResourceAttr("anecdotes_control.test", "maturity_level", "OPTIMIZING"),
			},
			{
				Config: withoutLevel,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("anecdotes_control.test", "maturity_level"),
					testCheckControlMaturityEmpty(t, "anecdotes_control.test"),
				),
			},
			{
				Config:   withoutLevel,
				PlanOnly: true,
			},
		},
	})
}

// testCheckControlMaturityEmpty asserts the platform holds no maturity level for
// the control in state — removing the attribute must reach the API.
func testCheckControlMaturityEmpty(t *testing.T, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := stateAttr(s, resourceName, "control_id")
		if err != nil {
			return err
		}
		level, err := testAccNewClient(t).GetControlMaturityLevel(context.Background(), id)
		if err != nil {
			return fmt.Errorf("reading maturity level of %s: %w", id, err)
		}
		if level != "" {
			return fmt.Errorf("expected no maturity level on the platform, got %q", level)
		}
		return nil
	}
}

// TestAccDrift_DeletedOutOfBand: an object deleted outside Terraform must be
// re-created by the next apply.
func TestAccDrift_DeletedOutOfBand(t *testing.T) {
	name := randomName("req-drift-del")
	config := fmt.Sprintf(`
resource "anecdotes_requirement" "test" {
  name        = %q
  description = "Delete detection"
}`, name)

	var requirementID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: func(s *terraform.State) error {
					var err error
					requirementID, err = stateAttr(s, "anecdotes_requirement.test", "requirement_id")
					return err
				},
			},
			{
				PreConfig: func() {
					c := testAccNewClient(t)
					if err := c.DeleteRequirement(context.Background(), requirementID); err != nil {
						t.Fatalf("out-of-band delete failed: %v", err)
					}
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("anecdotes_requirement.test", "requirement_id"),
					func(s *terraform.State) error {
						id, err := stateAttr(s, "anecdotes_requirement.test", "requirement_id")
						if err != nil {
							return err
						}
						if id == requirementID {
							return fmt.Errorf("expected a re-created requirement with a new ID, got the deleted one (%s)", id)
						}
						return nil
					},
				),
			},
		},
	})
}

// TestAccDrift_RequirementViewNameRevert: an out-of-band change to view_name
// (Terraform-owned) must surface as drift and be reverted by the next apply.
func TestAccDrift_RequirementViewNameRevert(t *testing.T) {
	parentName := randomName("req-view-drift-parent")
	viewName := randomName("req-view-drift")
	config := fmt.Sprintf(`
resource "anecdotes_requirement" "parent" {
  name        = %q
  description = "Parent for a Requirement View drift test"
}

resource "anecdotes_requirement_view" "test" {
  parent_id = anecdotes_requirement.parent.requirement_id
  view_name = %q
}`, parentName, viewName)

	var viewID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_requirement_view.test", "view_name", viewName),
					func(s *terraform.State) error {
						id, err := stateAttr(s, "anecdotes_requirement_view.test", "requirement_id")
						viewID = id
						return err
					},
				),
			},
			{
				PreConfig: func() {
					c := testAccNewClient(t)
					changed := "Changed outside Terraform"
					_, err := c.UpdateRequirementView(context.Background(), viewID, &client.RequirementViewUpdateRequest{
						ViewName: &changed,
					})
					if err != nil {
						t.Fatalf("out-of-band view_name change failed: %v", err)
					}
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_requirement_view.test", "view_name", viewName),
				),
			},
		},
	})
}

// TestAccDrift_RequirementViewParentDeletedIsRecreated: deleting the parent
// requirement out-of-band cascades to delete the view too. The next apply must
// re-create it rather than error.
func TestAccDrift_RequirementViewParentDeletedIsRecreated(t *testing.T) {
	parentName := randomName("req-view-drift-parent")
	viewName := randomName("req-view-drift")
	config := fmt.Sprintf(`
resource "anecdotes_requirement" "parent" {
  name        = %q
  description = "Parent for a Requirement View drift test"
}

resource "anecdotes_requirement_view" "test" {
  parent_id = anecdotes_requirement.parent.requirement_id
  view_name = %q
}`, parentName, viewName)

	var viewID, parentID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						id, err := stateAttr(s, "anecdotes_requirement_view.test", "requirement_id")
						viewID = id
						return err
					},
					func(s *terraform.State) error {
						id, err := stateAttr(s, "anecdotes_requirement.parent", "requirement_id")
						parentID = id
						return err
					},
				),
			},
			{
				PreConfig: func() {
					c := testAccNewClient(t)
					if err := c.DeleteRequirement(context.Background(), parentID); err != nil {
						t.Fatalf("out-of-band parent delete failed: %v", err)
					}
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("anecdotes_requirement_view.test", "requirement_id"),
					func(s *terraform.State) error {
						id, err := stateAttr(s, "anecdotes_requirement_view.test", "requirement_id")
						if err != nil {
							return err
						}
						if id == viewID {
							return fmt.Errorf("expected a re-created view with a new ID, got the cascade-deleted one (%s)", id)
						}
						return nil
					},
				),
			},
		},
	})
}

// TestAccDrift_RequirementViewOwnersClear: setting owners then emptying the
// set, and separately removing the attribute entirely, must both converge —
// exercising the owners state logic shared with anecdotes_control and
// anecdotes_requirement (an empty set and a removed attribute both mean "no
// owners" on the platform, but only the removed case must also clear the
// attribute from state).
func TestAccDrift_RequirementViewOwnersClear(t *testing.T) {
	parentName := randomName("req-view-drift-parent")
	viewName := randomName("req-view-drift-clear")

	withValue := fmt.Sprintf(`
resource "anecdotes_requirement" "parent" {
  name        = %q
  description = "Parent for a Requirement View owners-clear test"
}

resource "anecdotes_requirement_view" "test" {
  parent_id = anecdotes_requirement.parent.requirement_id
  view_name = %q
  owners    = ["acceptance-tests@example.com"]
}`, parentName, viewName)

	emptied := fmt.Sprintf(`
resource "anecdotes_requirement" "parent" {
  name        = %q
  description = "Parent for a Requirement View owners-clear test"
}

resource "anecdotes_requirement_view" "test" {
  parent_id = anecdotes_requirement.parent.requirement_id
  view_name = %q
  owners    = []
}`, parentName, viewName)

	removed := fmt.Sprintf(`
resource "anecdotes_requirement" "parent" {
  name        = %q
  description = "Parent for a Requirement View owners-clear test"
}

resource "anecdotes_requirement_view" "test" {
  parent_id = anecdotes_requirement.parent.requirement_id
  view_name = %q
}`, parentName, viewName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withValue,
				Check:  resource.TestCheckResourceAttr("anecdotes_requirement_view.test", "owners.#", "1"),
			},
			{
				Config: emptied,
				Check:  resource.TestCheckResourceAttr("anecdotes_requirement_view.test", "owners.#", "0"),
			},
			{
				Config:   emptied,
				PlanOnly: true,
			},
			{
				Config: withValue,
				Check:  resource.TestCheckResourceAttr("anecdotes_requirement_view.test", "owners.#", "1"),
			},
			{
				Config: removed,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("anecdotes_requirement_view.test", "owners.#"),
					testCheckRequirementViewOwnersOnPlatform(t, "anecdotes_requirement_view.test", []string{}),
				),
			},
			{
				Config:   removed,
				PlanOnly: true,
			},
		},
	})
}
