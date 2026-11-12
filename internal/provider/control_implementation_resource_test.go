// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"slices"
	"testing"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const controlImplementationAddr = "anecdotes_control_implementation.test"

// testAccEmptyPlatformControl returns the ID of a control provided by a
// platform framework that has no implementation text and is not one of
// exclude, and clears whatever text the test leaves on it. It skips the test
// when the tenant has no such control.
func testAccEmptyPlatformControl(t *testing.T, exclude ...string) string {
	t.Helper()

	_, control := testAccFindPlatformControl(t, func(c client.Control) bool {
		return c.ControlImplementation == "" && !slices.Contains(exclude, c.ControlID)
	})
	controlID := control.ControlID
	c := testAccNewClient(t)
	t.Cleanup(func() {
		if err := c.SetControlImplementation(context.Background(), controlID, ""); err != nil {
			t.Errorf("clearing implementation text on control %s: %v", controlID, err)
		}
	})
	return controlID
}

// testAccSkipUnlessAcceptance skips a test that queries the tenant while its
// configuration is being built, before resource.Test would skip it.
func testAccSkipUnlessAcceptance(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test - set TF_ACC=1 to run")
	}
	testAccPreCheck(t)
}

// testCheckControlImplementationInAPI reads the control from the platform and
// compares its implementation text, so the check does not depend on state.
func testCheckControlImplementationInAPI(t *testing.T, controlID func(*terraform.State) (string, error), want string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := controlID(s)
		if err != nil {
			return err
		}
		control, err := testAccNewClient(t).GetControl(context.Background(), "", id)
		if err != nil {
			return fmt.Errorf("reading control %s: %w", id, err)
		}
		if control.ControlImplementation != want {
			return fmt.Errorf("control %s: expected implementation text %q on the platform, got %q", id, want, control.ControlImplementation)
		}
		return nil
	}
}

func fixedControlID(id string) func(*terraform.State) (string, error) {
	return func(*terraform.State) (string, error) { return id, nil }
}

func controlIDFromState(s *terraform.State) (string, error) {
	return stateAttr(s, controlImplementationAddr, "control_id")
}

func testAccControlImplementationConfig(controlID, text string) string {
	return fmt.Sprintf(`
resource "anecdotes_control_implementation" "test" {
  control_id             = %q
  control_implementation = %q
}`, controlID, text)
}

func TestAccControlImplementationResource_platformControl(t *testing.T) {
	testAccSkipUnlessAcceptance(t)
	controlID := testAccEmptyPlatformControl(t)
	otherControlID := testAccEmptyPlatformControl(t, controlID)

	// Ends with a line break, as a heredoc does.
	first := "First line.\nSecond line.\n\nFourth line, after a blank line.\n"
	second := "Access rights are reviewed quarterly."

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testCheckControlImplementationInAPI(t, fixedControlID(controlID), ""),
			testCheckControlImplementationInAPI(t, fixedControlID(otherControlID), ""),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccControlImplementationConfig(controlID, first),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(controlImplementationAddr, "control_id", controlID),
					resource.TestCheckResourceAttr(controlImplementationAddr, "control_implementation", first),
					testCheckControlImplementationInAPI(t, controlIDFromState, first),
				),
			},
			{
				Config: testAccControlImplementationConfig(controlID, second),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(controlImplementationAddr, "control_implementation", second),
					testCheckControlImplementationInAPI(t, controlIDFromState, second),
				),
			},
			{
				ResourceName:                         controlImplementationAddr,
				ImportState:                          true,
				ImportStateId:                        controlID,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "control_id",
			},
			{
				// Changing control_id replaces the resource: the text moves to the
				// new control and is cleared from the old one.
				Config: testAccControlImplementationConfig(otherControlID, second),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(controlImplementationAddr, "control_id", otherControlID),
					testCheckControlImplementationInAPI(t, fixedControlID(otherControlID), second),
					testCheckControlImplementationInAPI(t, fixedControlID(controlID), ""),
				),
			},
		},
	})
}

// A custom control managed by anecdotes_control can carry an implementation
// resource too: updating the control leaves its implementation text in place.
func TestAccControlImplementationResource_customControl(t *testing.T) {
	fwName := randomName("fw-impl")
	catName := randomName("cat-impl")
	ctrlName := randomName("ctrl-impl")

	config := func(name, text string) string {
		return testAccControlConfig(fwName, catName, name) + fmt.Sprintf(`
resource "anecdotes_control_implementation" "test" {
  control_id             = anecdotes_control.test.control_id
  control_implementation = %q
}`, text)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(ctrlName, "Nightly snapshots are kept for 35 days."),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(controlImplementationAddr, "control_id", "anecdotes_control.test", "control_id"),
					resource.TestCheckResourceAttr(controlImplementationAddr, "control_implementation", "Nightly snapshots are kept for 35 days."),
					testCheckControlImplementationInAPI(t, controlIDFromState, "Nightly snapshots are kept for 35 days."),
				),
			},
			{
				Config: config(ctrlName, "Snapshots are restored in a quarterly drill."),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(controlImplementationAddr, "control_implementation", "Snapshots are restored in a quarterly drill."),
					testCheckControlImplementationInAPI(t, controlIDFromState, "Snapshots are restored in a quarterly drill."),
				),
			},
			{
				Config: config(ctrlName+"-renamed", "Snapshots are restored in a quarterly drill."),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_control.test", "name", ctrlName+"-renamed"),
					testCheckControlImplementationInAPI(t, controlIDFromState, "Snapshots are restored in a quarterly drill."),
				),
			},
		},
	})
}

// A control that already has implementation text is never overwritten by a
// create: it must be imported, after which the text round-trips unchanged,
// HTML included, and the configuration can replace it.
func TestAccControlImplementationResource_existingTextRequiresImport(t *testing.T) {
	testAccSkipUnlessAcceptance(t)
	controlID := testAccEmptyPlatformControl(t)

	existing := "<p><strong>Existing</strong> implementation, written in the application.</p>"
	if err := testAccNewClient(t).SetControlImplementation(context.Background(), controlID, existing); err != nil {
		t.Fatalf("setting existing implementation text: %v", err)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckControlImplementationInAPI(t, fixedControlID(controlID), ""),
		Steps: []resource.TestStep{
			{
				Config:      testAccControlImplementationConfig(controlID, "Written by Terraform."),
				ExpectError: regexp.MustCompile(`(?s)already has implementation text.*terraform import`),
			},
			{
				Config:             testAccControlImplementationConfig(controlID, existing),
				ResourceName:       controlImplementationAddr,
				ImportState:        true,
				ImportStateId:      controlID,
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected one imported instance, got %d", len(states))
					}
					if got := states[0].Attributes["control_implementation"]; got != existing {
						return fmt.Errorf("expected imported text %q, got %q", existing, got)
					}
					return nil
				},
			},
			{
				Config:   testAccControlImplementationConfig(controlID, existing),
				PlanOnly: true,
			},
			{
				Config: testAccControlImplementationConfig(controlID, "Written by Terraform."),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(controlImplementationAddr, "control_implementation", "Written by Terraform."),
					testCheckControlImplementationInAPI(t, controlIDFromState, "Written by Terraform."),
				),
			},
		},
	})
}

func TestAccControlImplementationResource_emptyValuesRejected(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccControlImplementationConfig("control_1000000000000", ""),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?s)control_implementation.*string length must be at least 1`),
			},
			{
				Config:      testAccControlImplementationConfig("", "Written by Terraform."),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?s)control_id.*string length must be at least 1`),
			},
		},
	})
}

// A control deleted outside Terraform drops out of state on the next refresh,
// and the following apply recreates the control and sets its text again.
func TestAccControlImplementationResource_controlDeletedOutsideTerraform(t *testing.T) {
	fwName := randomName("fw-impl-gone")
	catName := randomName("cat-impl-gone")
	ctrlName := randomName("ctrl-impl-gone")
	text := "Set again after the control was recreated."

	config := testAccControlConfig(fwName, catName, ctrlName) + fmt.Sprintf(`
resource "anecdotes_control_implementation" "test" {
  control_id             = anecdotes_control.test.control_id
  control_implementation = %q
}`, text)

	var frameworkID, deletedControlID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: func(s *terraform.State) error {
					var err error
					if frameworkID, err = stateAttr(s, "anecdotes_control.test", "framework_id"); err != nil {
						return err
					}
					deletedControlID, err = stateAttr(s, "anecdotes_control.test", "control_id")
					return err
				},
			},
			{
				PreConfig: func() {
					if err := testAccNewClient(t).DeleteControl(context.Background(), frameworkID, deletedControlID); err != nil {
						t.Fatalf("deleting the control outside Terraform: %v", err)
					}
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(controlImplementationAddr, "control_id", "anecdotes_control.test", "control_id"),
					func(s *terraform.State) error {
						id, err := controlIDFromState(s)
						if err != nil {
							return err
						}
						if id == deletedControlID {
							return fmt.Errorf("expected a recreated control, still on deleted control %s", id)
						}
						return nil
					},
					testCheckControlImplementationInAPI(t, controlIDFromState, text),
				),
			},
		},
	})
}

func TestAccControlImplementationResource_controlNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccControlImplementationConfig("control_1000000000000", "Written by Terraform."),
				ExpectError: regexp.MustCompile(`Control Not Found`),
			},
		},
	})
}

// Both control data sources report the implementation text of a control that
// has one, and null for a control that has none.
func TestAccControlDataSources_controlImplementation(t *testing.T) {
	fwName := randomName("fw-impl-ds")
	catName := randomName("cat-impl-ds")
	ctrlName := randomName("ctrl-impl-ds")
	text := "Implementation text read through the data sources."

	config := testAccControlConfig(fwName, catName, ctrlName) + fmt.Sprintf(`
resource "anecdotes_control" "without_text" {
  name         = "%s-empty"
  framework_id = anecdotes_framework.test.framework_id
  category_id  = anecdotes_control_category.test.category_id
}

resource "anecdotes_control_implementation" "test" {
  control_id             = anecdotes_control.test.control_id
  control_implementation = %q
}

data "anecdotes_control" "with_text" {
  framework_id = anecdotes_framework.test.framework_id
  control_id   = anecdotes_control.test.control_id
  depends_on   = [anecdotes_control_implementation.test]
}

data "anecdotes_control" "without_text" {
  framework_id = anecdotes_framework.test.framework_id
  control_id   = anecdotes_control.without_text.control_id
}

data "anecdotes_controls" "all" {
  framework_id = anecdotes_framework.test.framework_id
  depends_on   = [anecdotes_control_implementation.test, anecdotes_control.without_text]
}`, ctrlName, text)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.anecdotes_control.with_text", "control_implementation", text),
					resource.TestCheckNoResourceAttr("data.anecdotes_control.without_text", "control_implementation"),
					testCheckControlsListImplementation("anecdotes_control.test", text),
					testCheckControlsListImplementation("anecdotes_control.without_text", ""),
				),
			},
		},
	})
}

// testCheckControlsListImplementation finds a control in the anecdotes_controls
// data source by its ID and compares its implementation text. An empty want
// means the attribute must be null.
func testCheckControlsListImplementation(controlAddr, want string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		controlID, err := stateAttr(s, controlAddr, "control_id")
		if err != nil {
			return err
		}
		list, ok := s.RootModule().Resources["data.anecdotes_controls.all"]
		if !ok {
			return fmt.Errorf("data.anecdotes_controls.all not found in state")
		}
		attrs := list.Primary.Attributes
		for i := 0; ; i++ {
			id, found := attrs[fmt.Sprintf("controls.%d.control_id", i)]
			if !found {
				return fmt.Errorf("control %s not found in data.anecdotes_controls.all", controlID)
			}
			if id != controlID {
				continue
			}
			got, set := attrs[fmt.Sprintf("controls.%d.control_implementation", i)]
			if want == "" && set {
				return fmt.Errorf("control %s: expected no implementation text, got %q", controlID, got)
			}
			if want != "" && got != want {
				return fmt.Errorf("control %s: expected implementation text %q, got %q", controlID, want, got)
			}
			return nil
		}
	}
}

// Two resources on the same control in one configuration: one sets the text
// and the other is refused, instead of both applying and overwriting each other
// on every apply.
func TestAccControlImplementationResource_twoResourcesOneControl(t *testing.T) {
	testAccSkipUnlessAcceptance(t)
	controlID := testAccEmptyPlatformControl(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckControlImplementationInAPI(t, fixedControlID(controlID), ""),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_control_implementation" "a" {
  control_id             = %q
  control_implementation = "Text A"
}

resource "anecdotes_control_implementation" "b" {
  control_id             = %q
  control_implementation = "Text B"
}`, controlID, controlID),
				ExpectError: regexp.MustCompile(`Control Already Has Implementation Text`),
			},
		},
	})
}
