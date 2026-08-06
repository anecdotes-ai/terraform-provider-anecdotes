// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccControlResource_create(t *testing.T) {
	fwName := randomName("fw")
	catName := randomName("cat")
	ctrlName := randomName("ctrl")
	const owner = "acceptance-tests@example.com"

	config := testAccControlCategoryConfig(fwName, catName) + fmt.Sprintf(`
resource "anecdotes_control" "test" {
  name         = %q
  framework_id = anecdotes_framework.test.framework_id
  category_id  = anecdotes_control_category.test.category_id
  owners       = [%q]
}`, ctrlName, owner)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_control.test", "name", ctrlName),
					resource.TestCheckResourceAttrSet("anecdotes_control.test", "control_id"),
					resource.TestCheckResourceAttrSet("anecdotes_control.test", "framework_id"),
					resource.TestCheckResourceAttrSet("anecdotes_control.test", "category_id"),
					// Assert against the platform, not only against state: a
					// value dropped on write and absent on read agrees with
					// state while never reaching the platform.
					testCheckControlOwnersOnPlatform(t, "anecdotes_control.test", owner),
				),
			},
		},
	})
}

// testCheckControlOwnersOnPlatform reads the control back from the API and
// asserts the owners a create applied are actually stored.
func testCheckControlOwnersOnPlatform(t *testing.T, resourceName string, want ...string) resource.TestCheckFunc {
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
		if len(control.ControlOwners) != len(want) {
			return fmt.Errorf("expected %d owners on the platform, got %v", len(want), control.ControlOwners)
		}
		for i, w := range want {
			if control.ControlOwners[i] != w {
				return fmt.Errorf("owner %d: expected %q on the platform, got %q", i, w, control.ControlOwners[i])
			}
		}
		return nil
	}
}

func TestAccControlResource_update(t *testing.T) {
	fwName := randomName("fw")
	catName := randomName("cat")
	ctrlName1 := randomName("ctrl")
	ctrlName2 := randomName("ctrl-upd")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlConfig(fwName, catName, ctrlName1),
				Check:  resource.TestCheckResourceAttr("anecdotes_control.test", "name", ctrlName1),
			},
			{
				Config: testAccControlCategoryConfig(fwName, catName) + fmt.Sprintf(`
resource "anecdotes_control" "test" {
  name         = %q
  description  = "Updated control"
  framework_id = anecdotes_framework.test.framework_id
  category_id  = anecdotes_control_category.test.category_id
}`, ctrlName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_control.test", "name", ctrlName2),
					resource.TestCheckResourceAttr("anecdotes_control.test", "description", "Updated control"),
				),
			},
		},
	})
}

func TestAccControlResource_import(t *testing.T) {
	fwName := randomName("fw")
	catName := randomName("cat")
	ctrlName := randomName("ctrl-imp")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlConfig(fwName, catName, ctrlName),
			},
			{
				ResourceName:                         "anecdotes_control.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "control_id",
				ImportStateIdFunc:                    importIDComposite("anecdotes_control.test", "framework_id", "control_id"),
			},
		},
	})
}

// TestAccControlResource_platformProvidedRejected: a control that belongs to a
// platform framework cannot be updated or deleted, so importing one must fail
// rather than put an unmanageable object into state.
func TestAccControlResource_platformProvidedRejected(t *testing.T) {
	fwName := randomName("fw-ootb")
	catName := randomName("cat-ootb")
	ctrlName := randomName("ctrl-ootb")

	// Resolved in PreCheck so the tenant is only queried once the acceptance
	// gate has passed.
	var importID string

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			importID = testAccPlatformControlImportID(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccControlConfig(fwName, catName, ctrlName),
			},
			{
				Config:            testAccControlConfig(fwName, catName, ctrlName),
				ResourceName:      "anecdotes_control.test",
				ImportState:       true,
				ImportStateIdFunc: func(*terraform.State) (string, error) { return importID, nil },
				ImportStateVerify: false,
				ExpectError:       regexp.MustCompile(`(?s)cannot be managed by Terraform`),
			},
		},
	})
}

// testAccPlatformControlImportID returns the import ID of a control provided by
// a platform framework, skipping the test when the tenant has none.
func testAccPlatformControlImportID(t *testing.T) string {
	t.Helper()

	c := testAccNewClient(t)
	frameworks, err := c.ListFrameworks(context.Background())
	if err != nil {
		t.Fatalf("listing frameworks: %v", err)
	}
	for _, framework := range frameworks {
		if !framework.IsApplicable {
			continue
		}
		controls, err := c.ListControls(context.Background(), framework.FrameworkID)
		if err != nil {
			continue
		}
		for _, control := range controls {
			if control.ControlIsCustom != nil && !*control.ControlIsCustom {
				return framework.FrameworkID + "/" + control.ControlID
			}
		}
	}
	t.Skip("no platform-provided control on this tenant")
	return ""
}
