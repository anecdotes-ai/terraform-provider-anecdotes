// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	evidenceScopeAddr        = "anecdotes_mapping_requirement_evidence_scope.test"
	evidenceScopeControlAddr = "anecdotes_mapping_requirement_evidence_scope.control"
)

// Entry states for testCheckEvidenceScopeInAPI.
const (
	scopeAbsent   = "absent"
	scopeIncluded = "included"
	scopeExcluded = "excluded"
)

// testAccEvidenceScopePrerequisites returns a framework, a control in it and a
// requirement for the scope tests.
func testAccEvidenceScopePrerequisites(suffix string) string {
	return testAccControlConfig(randomName("fw-"+suffix), randomName("cat-"+suffix), randomName("ctrl-"+suffix)) +
		testAccRequirementConfig(randomName("req-"+suffix))
}

func testAccEvidenceScopeFrameworkConfig(excluded bool) string {
	return fmt.Sprintf(`
resource "anecdotes_mapping_requirement_evidence_scope" "test" {
  requirement_id = anecdotes_requirement.test.requirement_id
  evidence_id    = %q
  scope_id       = anecdotes_framework.test.framework_id
  scope_type     = "framework"
  excluded       = %t
}`, testAccEvidenceID(), excluded)
}

func testAccEvidenceScopeControlConfig() string {
	return fmt.Sprintf(`
resource "anecdotes_mapping_requirement_evidence_scope" "control" {
  requirement_id = anecdotes_requirement.test.requirement_id
  evidence_id    = %q
  scope_id       = anecdotes_control.test.control_id
  scope_type     = "control"
}`, testAccEvidenceID())
}

// testAccEvidenceScopeIDs reads the requirement ID and the scope ID from the
// prerequisite resources. scopeType selects the framework or the control.
func testAccEvidenceScopeIDs(s *terraform.State, scopeType string) (requirementID, scopeID string, err error) {
	requirementID, err = stateAttr(s, "anecdotes_requirement.test", "requirement_id")
	if err != nil {
		return "", "", err
	}
	scopeID, err = stateAttr(s, "anecdotes_"+scopeType+".test", scopeType+"_id")
	return requirementID, scopeID, err
}

// testCheckEvidenceScopeInAPI reads the requirement from the platform and
// compares the test evidence's entry in the framework or control scope, so the
// check does not depend on the scope resource's state.
func testCheckEvidenceScopeInAPI(t *testing.T, scopeType, want string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		requirementID, scopeID, err := testAccEvidenceScopeIDs(s, scopeType)
		if err != nil {
			return err
		}
		got := scopeAbsent
		scope, err := testAccNewClient(t).GetRequirementEvidenceScope(context.Background(), requirementID, scopeID, testAccEvidenceID())
		switch {
		case client.IsNotFound(err):
		case err != nil:
			return fmt.Errorf("reading evidence scope: %w", err)
		case scope.ScopeType != scopeType:
			return fmt.Errorf("scope %s: expected scope type %q on the platform, got %q", scopeID, scopeType, scope.ScopeType)
		case scope.Included:
			got = scopeIncluded
		default:
			got = scopeExcluded
		}
		if got != want {
			return fmt.Errorf("requirement %s, %s scope %s: expected the evidence entry to be %s on the platform, got %s", requirementID, scopeType, scopeID, want, got)
		}
		return nil
	}
}

func evidenceScopeImportIDFromState(addr string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[addr]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", addr)
		}
		a := rs.Primary.Attributes
		return a["requirement_id"] + "/" + a["scope_type"] + "/" + a["scope_id"] + "/" + a["evidence_id"], nil
	}
}

func TestAccMappingRequirementEvidenceScopeResource_framework(t *testing.T) {
	prerequisites := testAccEvidenceScopePrerequisites("scope-fw")
	updatedInPlace := resource.ConfigPlanChecks{
		PreApply: []plancheck.PlanCheck{
			plancheck.ExpectResourceAction(evidenceScopeAddr, plancheck.ResourceActionUpdate),
		},
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// excluded is omitted, so it takes its default.
				Config: prerequisites + fmt.Sprintf(`
resource "anecdotes_mapping_requirement_evidence_scope" "test" {
  requirement_id = anecdotes_requirement.test.requirement_id
  evidence_id    = %q
  scope_id       = anecdotes_framework.test.framework_id
  scope_type     = "framework"
}`, testAccEvidenceID()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(evidenceScopeAddr, "requirement_id", "anecdotes_requirement.test", "requirement_id"),
					resource.TestCheckResourceAttrPair(evidenceScopeAddr, "scope_id", "anecdotes_framework.test", "framework_id"),
					resource.TestCheckResourceAttr(evidenceScopeAddr, "evidence_id", testAccEvidenceID()),
					resource.TestCheckResourceAttr(evidenceScopeAddr, "scope_type", "framework"),
					resource.TestCheckResourceAttr(evidenceScopeAddr, "excluded", "false"),
					testCheckEvidenceScopeInAPI(t, "framework", scopeIncluded),
				),
			},
			{
				Config:   prerequisites + testAccEvidenceScopeFrameworkConfig(false),
				PlanOnly: true,
			},
			{
				Config:           prerequisites + testAccEvidenceScopeFrameworkConfig(true),
				ConfigPlanChecks: updatedInPlace,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(evidenceScopeAddr, "excluded", "true"),
					testCheckEvidenceScopeInAPI(t, "framework", scopeExcluded),
				),
			},
			{
				ResourceName:                         evidenceScopeAddr,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "requirement_id",
				ImportStateIdFunc:                    evidenceScopeImportIDFromState(evidenceScopeAddr),
			},
			{
				Config:           prerequisites + testAccEvidenceScopeFrameworkConfig(false),
				ConfigPlanChecks: updatedInPlace,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(evidenceScopeAddr, "excluded", "false"),
					testCheckEvidenceScopeInAPI(t, "framework", scopeIncluded),
				),
			},
			{
				// Removing the resource removes the entry and keeps the requirement.
				Config: prerequisites,
				Check:  testCheckEvidenceScopeInAPI(t, "framework", scopeAbsent),
			},
		},
	})
}

// A framework scope, a control scope and a global link on the same requirement
// are written in one apply and none of them replaces another. Removing one
// scope keeps the other.
func TestAccMappingRequirementEvidenceScopeResource_sameRequirement(t *testing.T) {
	prerequisites := testAccEvidenceScopePrerequisites("scope-both")
	globalLink := fmt.Sprintf(`
resource "anecdotes_mapping_requirement_evidence" "test" {
  requirement_id = anecdotes_requirement.test.requirement_id
  evidence_id    = %q
}`, testAccEvidenceID())

	checkGlobalLink := func(s *terraform.State) error {
		requirementID, err := stateAttr(s, "anecdotes_requirement.test", "requirement_id")
		if err != nil {
			return err
		}
		return testAccNewClient(t).GetRequirementEvidenceLink(context.Background(), requirementID, testAccEvidenceID())
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: prerequisites + globalLink + testAccEvidenceScopeFrameworkConfig(true) + testAccEvidenceScopeControlConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(evidenceScopeControlAddr, "scope_id", "anecdotes_control.test", "control_id"),
					resource.TestCheckResourceAttr(evidenceScopeControlAddr, "scope_type", "control"),
					resource.TestCheckResourceAttr(evidenceScopeControlAddr, "excluded", "false"),
					testCheckEvidenceScopeInAPI(t, "framework", scopeExcluded),
					testCheckEvidenceScopeInAPI(t, "control", scopeIncluded),
					checkGlobalLink,
				),
			},
			{
				ResourceName:                         evidenceScopeControlAddr,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "scope_id",
				ImportStateIdFunc:                    evidenceScopeImportIDFromState(evidenceScopeControlAddr),
			},
			{
				Config: prerequisites + globalLink + testAccEvidenceScopeControlConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckEvidenceScopeInAPI(t, "framework", scopeAbsent),
					testCheckEvidenceScopeInAPI(t, "control", scopeIncluded),
					checkGlobalLink,
				),
			},
			{
				Config: prerequisites + testAccEvidenceScopeControlConfig(),
				Check:  testCheckEvidenceScopeInAPI(t, "control", scopeIncluded),
			},
		},
	})
}

// An entry that already exists is never changed by a create: it must be
// imported, after which the configuration can change it.
func TestAccMappingRequirementEvidenceScopeResource_existingEntryRequiresImport(t *testing.T) {
	prerequisites := testAccEvidenceScopePrerequisites("scope-import")

	var requirementID, frameworkID string
	captureIDs := func(s *terraform.State) (err error) {
		requirementID, frameworkID, err = testAccEvidenceScopeIDs(s, "framework")
		return err
	}
	importID := func(*terraform.State) (string, error) {
		return requirementID + "/framework/" + frameworkID + "/" + testAccEvidenceID(), nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: prerequisites,
				Check:  captureIDs,
			},
			{
				PreConfig: func() {
					err := testAccNewClient(t).SetRequirementEvidenceScope(context.Background(),
						requirementID, frameworkID, client.EvidenceScopeFramework, testAccEvidenceID(), false)
					if err != nil {
						t.Fatalf("adding the existing scope entry: %v", err)
					}
				},
				Config:      prerequisites + testAccEvidenceScopeFrameworkConfig(false),
				ExpectError: regexp.MustCompile(`(?s)Evidence Already Scoped.*terraform import`),
			},
			{
				Config:             prerequisites + testAccEvidenceScopeFrameworkConfig(true),
				ResourceName:       evidenceScopeAddr,
				ImportState:        true,
				ImportStateIdFunc:  importID,
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					// The state also holds the prerequisite resources.
					for _, state := range states {
						a := state.Attributes
						if a["scope_id"] == "" {
							continue
						}
						if a["excluded"] != "true" || a["scope_type"] != "framework" {
							return fmt.Errorf("expected an excluded framework entry, got excluded=%q scope_type=%q", a["excluded"], a["scope_type"])
						}
						return nil
					}
					return fmt.Errorf("the imported evidence scope was not found in state")
				},
			},
			{
				Config:   prerequisites + testAccEvidenceScopeFrameworkConfig(true),
				PlanOnly: true,
			},
			{
				Config: prerequisites + testAccEvidenceScopeFrameworkConfig(false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(evidenceScopeAddr, "excluded", "false"),
					testCheckEvidenceScopeInAPI(t, "framework", scopeIncluded),
				),
			},
		},
	})
}

// A framework scope the platform does not store fails the apply.
func TestAccMappingRequirementEvidenceScopeResource_frameworkNotStored(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRequirementConfig(randomName("req-scope-unknown")) + fmt.Sprintf(`
resource "anecdotes_mapping_requirement_evidence_scope" "test" {
  requirement_id = anecdotes_requirement.test.requirement_id
  evidence_id    = %q
  scope_id       = "framework_1000000000000"
  scope_type     = "framework"
}`, testAccEvidenceID()),
				ExpectError: regexp.MustCompile(`Evidence Scope Not Stored`),
			},
		},
	})
}

// An evidence ID and a control scope ID that match no existing object are
// stored as written.
func TestAccMappingRequirementEvidenceScopeResource_unknownIDsStored(t *testing.T) {
	const controlID, evidenceID = "control_1000000000000", "1000000000"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRequirementConfig(randomName("req-scope-unknown-ids")) + fmt.Sprintf(`
resource "anecdotes_mapping_requirement_evidence_scope" "test" {
  requirement_id = anecdotes_requirement.test.requirement_id
  evidence_id    = %q
  scope_id       = %q
  scope_type     = "control"
}`, evidenceID, controlID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(evidenceScopeAddr, "scope_id", controlID),
					resource.TestCheckResourceAttr(evidenceScopeAddr, "evidence_id", evidenceID),
					func(s *terraform.State) error {
						requirementID, err := stateAttr(s, "anecdotes_requirement.test", "requirement_id")
						if err != nil {
							return err
						}
						scope, err := testAccNewClient(t).GetRequirementEvidenceScope(context.Background(), requirementID, controlID, evidenceID)
						if err != nil {
							return fmt.Errorf("reading evidence scope: %w", err)
						}
						if !scope.Included || scope.ScopeType != client.EvidenceScopeControl {
							return fmt.Errorf("expected an included control entry on the platform, got %+v", scope)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccMappingRequirementEvidenceScopeResource_invalidValuesRejected(t *testing.T) {
	config := func(requirementID, evidenceID, scopeID, scopeType string) string {
		return fmt.Sprintf(`
resource "anecdotes_mapping_requirement_evidence_scope" "test" {
  requirement_id = %q
  evidence_id    = %q
  scope_id       = %q
  scope_type     = %q
}`, requirementID, evidenceID, scopeID, scopeType)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config("requirement_1000000000000", "1000000000", "framework_1000000000000", "policy"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?s)scope_type.*value must be one of`),
			},
			{
				Config:      config("requirement_1000000000000", "1000000000", "framework_1000000000000", "control"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Scope Type Does Not Match Scope ID`),
			},
			{
				Config:      config("requirement_1000000000000", "1000000000", "control_1000000000000", "framework"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Scope Type Does Not Match Scope ID`),
			},
			{
				Config:      config("", "1000000000", "1000000000", "framework"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?s)requirement_id.*string length must be at least 1`),
			},
			{
				Config:      config("requirement_1000000000000", "", "1000000000", "framework"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?s)evidence_id.*string length must be at least 1`),
			},
			{
				Config:      config("requirement_1000000000000", "1000000000", "", "framework"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?s)scope_id.*string length must be at least 1`),
			},
			{
				Config:        config("requirement_1000000000000", "1000000000", "1000000000", "framework"),
				ResourceName:  evidenceScopeAddr,
				ImportState:   true,
				ImportStateId: "requirement_1000000000000/policy/1000000000/1000000000",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
			{
				Config:        config("requirement_1000000000000", "1000000000", "1000000000", "framework"),
				ResourceName:  evidenceScopeAddr,
				ImportState:   true,
				ImportStateId: "requirement_1000000000000/1000000000",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}
