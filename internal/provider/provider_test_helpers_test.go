// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"anecdotes": providerserver.NewProtocol6WithError(New("test")()),
}

// importIDFromAttr builds an ImportStateIdFunc that reads a single attribute
// from the named resource's state. These resources expose no synthetic "id",
// so imports must be driven by the real identifier attribute.
func importIDFromAttr(resourceName, attr string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}
		return rs.Primary.Attributes[attr], nil
	}
}

// importIDComposite builds an ImportStateIdFunc that joins two attributes with a
// slash, matching the composite import format of the mapping resources.
func importIDComposite(resourceName, attr1, attr2 string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}
		return rs.Primary.Attributes[attr1] + "/" + rs.Primary.Attributes[attr2], nil
	}
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("ANECDOTES_API_KEY") == "" {
		t.Fatal("ANECDOTES_API_KEY must be set for acceptance tests")
	}
}

// testCheckTotalCountGreaterThan asserts total_count > minValue.
func testCheckTotalCountGreaterThan(resourceAddr string, minValue int) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith(resourceAddr, "total_count",
		func(value string) error {
			count, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("total_count is not an integer: %s", value)
			}
			if count <= minValue {
				return fmt.Errorf("expected total_count > %d, got %d", minValue, count)
			}
			return nil
		})
}

// testCheckTotalCountAtLeast asserts total_count >= minValue.
func testCheckTotalCountAtLeast(resourceAddr string, minValue int) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith(resourceAddr, "total_count",
		func(value string) error {
			count, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("total_count is not an integer: %s", value)
			}
			if count < minValue {
				return fmt.Errorf("expected total_count >= %d, got %d", minValue, count)
			}
			return nil
		})
}

// testCheckListCountMatchesTotalCount verifies listAttr.# equals total_count.
func testCheckListCountMatchesTotalCount(resourceAddr, listAttr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceAddr]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceAddr)
		}
		totalCount := rs.Primary.Attributes["total_count"]
		listCount := rs.Primary.Attributes[listAttr+".#"]
		if totalCount != listCount {
			return fmt.Errorf("%s.# = %s but total_count = %s", listAttr, listCount, totalCount)
		}
		return nil
	}
}

// randomName generates a unique test resource name to avoid collisions.
func randomName(prefix string) string {
	return fmt.Sprintf("tf-test-%s-%d", prefix, rand.Intn(99999))
}

// ==================== Shared HCL Config Builders ====================
// Each returns a self-contained HCL snippet for use as a prerequisite.

func testAccFrameworkConfig(name string) string {
	return fmt.Sprintf(`
resource "anecdotes_framework" "test" {
  name        = %q
  description = "Acceptance test framework"
}`, name)
}

func testAccControlCategoryConfig(fwName, catName string) string {
	return testAccFrameworkConfig(fwName) + fmt.Sprintf(`
resource "anecdotes_control_category" "test" {
  name         = %q
  framework_id = anecdotes_framework.test.framework_id
}`, catName)
}

func testAccControlConfig(fwName, catName, ctrlName string) string {
	return testAccControlCategoryConfig(fwName, catName) + fmt.Sprintf(`
resource "anecdotes_control" "test" {
  name         = %q
  framework_id = anecdotes_framework.test.framework_id
  category_id  = anecdotes_control_category.test.category_id
}`, ctrlName)
}

func testAccRequirementConfig(name string) string {
	return fmt.Sprintf(`
resource "anecdotes_requirement" "test" {
  name        = %q
  description = "Acceptance test requirement"
  category    = "Custom Requirements"
}`, name)
}

// testAccEvidenceID returns the id of a pre-existing evidence used by acceptance
// tests. Evidence is read-only in this provider (there is no evidence resource),
// so acceptance tests that need an evidence reference read one from the
// environment. It is empty when the ANECDOTES_TEST_EVIDENCE_ID env var is unset;
// acceptance tests that require it should gate on testAccEvidencePreCheck.
func testAccEvidenceID() string {
	return os.Getenv("ANECDOTES_TEST_EVIDENCE_ID")
}

// testAccEvidencePreCheck fails fast (before any plan/apply) when an acceptance
// test needs a pre-existing evidence id but ANECDOTES_TEST_EVIDENCE_ID is not set.
func testAccEvidencePreCheck(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	if testAccEvidenceID() == "" {
		t.Fatal("ANECDOTES_TEST_EVIDENCE_ID must be set for acceptance tests that reference an existing evidence")
	}
}
