// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

// testAccEvidenceScopeIDs reads the ID of a requirement resource and the ID of
// the prerequisite framework or control, selected by scopeType.
func testAccEvidenceScopeIDs(s *terraform.State, requirementAddr, scopeType string) (requirementID, scopeID string, err error) {
	requirementID, err = stateAttr(s, requirementAddr, "requirement_id")
	if err != nil {
		return "", "", err
	}
	scopeID, err = stateAttr(s, "anecdotes_"+scopeType+".test", scopeType+"_id")
	return requirementID, scopeID, err
}

// testAccOtherEvidenceID returns the ID of an evidence in the account other
// than the one testAccEvidenceID returns. It skips the test when there is none.
func testAccOtherEvidenceID(t *testing.T) string {
	t.Helper()

	evidences, err := testAccNewClient(t).ListEvidences(context.Background())
	if err != nil {
		t.Fatalf("listing evidences: %v", err)
	}
	for _, evidence := range evidences {
		if evidence.EvidenceID != "" && evidence.EvidenceID != testAccEvidenceID() {
			return evidence.EvidenceID
		}
	}
	t.Skip("the account has no second evidence")
	return ""
}

// testCheckEvidenceScopeInAPI reads the requirement from the platform and
// compares the test evidence's entry in the framework or control scope, so the
// check does not depend on the scope resource's state.
func testCheckEvidenceScopeInAPI(t *testing.T, scopeType, want string) resource.TestCheckFunc {
	return testCheckEvidenceScopeEntryInAPI(t, "anecdotes_requirement.test", scopeType, testAccEvidenceID(), want)
}

// testCheckEvidenceScopeEntryInAPI is testCheckEvidenceScopeInAPI for a given
// requirement resource and evidence ID. It fails when the requirement itself
// does not exist.
func testCheckEvidenceScopeEntryInAPI(t *testing.T, requirementAddr, scopeType, evidenceID, want string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		requirementID, scopeID, err := testAccEvidenceScopeIDs(s, requirementAddr, scopeType)
		if err != nil {
			return err
		}
		c := testAccNewClient(t)
		if _, err := c.GetRequirement(context.Background(), requirementID); err != nil {
			return fmt.Errorf("reading requirement %s: %w", requirementID, err)
		}
		got := scopeAbsent
		scope, err := c.GetRequirementEvidenceScope(context.Background(), requirementID, scopeID, evidenceID)
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
			return fmt.Errorf("requirement %s, %s scope %s, evidence %s: expected the entry to be %s on the platform, got %s", requirementID, scopeType, scopeID, evidenceID, want, got)
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
		return a["requirement_id"] + "/" + a["scope_id"] + "/" + a["evidence_id"], nil
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
		requirementID, frameworkID, err = testAccEvidenceScopeIDs(s, "anecdotes_requirement.test", "framework")
		return err
	}
	importID := func(*terraform.State) (string, error) {
		return requirementID + "/" + frameworkID + "/" + testAccEvidenceID(), nil
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
	requirement := testAccRequirementConfig(randomName("req-scope-unknown"))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: requirement + fmt.Sprintf(`
resource "anecdotes_mapping_requirement_evidence_scope" "test" {
  requirement_id = anecdotes_requirement.test.requirement_id
  evidence_id    = %q
  scope_id       = "framework_1000000000000"
  scope_type     = "framework"
}`, testAccEvidenceID()),
				ExpectError: regexp.MustCompile(`Evidence Scope Not Stored`),
			},
			{
				// The failed apply left no scope on the requirement.
				Config: requirement,
				Check: func(s *terraform.State) error {
					requirementID, err := stateAttr(s, "anecdotes_requirement.test", "requirement_id")
					if err != nil {
						return err
					}
					req, err := testAccNewClient(t).GetRequirement(context.Background(), requirementID)
					if err != nil {
						return fmt.Errorf("reading requirement %s: %w", requirementID, err)
					}
					if len(req.RequirementScopingOverrides) != 0 {
						return fmt.Errorf("requirement %s: expected no scopes on the platform, got %v", requirementID, req.RequirementScopingOverrides)
					}
					return nil
				},
			},
		},
	})
}

// Changing the evidence, the scope or the requirement replaces the resource:
// the old entry is removed and the new one is added.
func TestAccMappingRequirementEvidenceScopeResource_replaceOnChange(t *testing.T) {
	testAccSkipUnlessAcceptance(t)
	testAccEvidencePreCheck(t)
	firstEvidence, secondEvidence := testAccEvidenceID(), testAccOtherEvidenceID(t)

	prerequisites := testAccEvidenceScopePrerequisites("scope-replace") + fmt.Sprintf(`
resource "anecdotes_requirement" "other" {
  name        = %q
  description = "Acceptance test requirement"
  category    = "Custom Requirements"
}`, randomName("req-scope-replace-other"))

	config := func(requirement, evidenceID, scopeType string) string {
		return prerequisites + fmt.Sprintf(`
resource "anecdotes_mapping_requirement_evidence_scope" "test" {
  requirement_id = anecdotes_requirement.%s.requirement_id
  evidence_id    = %q
  scope_id       = anecdotes_%s.test.%s_id
  scope_type     = %q
}`, requirement, evidenceID, scopeType, scopeType, scopeType)
	}

	replaced := resource.ConfigPlanChecks{
		PreApply: []plancheck.PlanCheck{
			plancheck.ExpectResourceAction(evidenceScopeAddr, plancheck.ResourceActionDestroyBeforeCreate),
		},
	}
	entry := func(requirement, scopeType, evidenceID, want string) resource.TestCheckFunc {
		return testCheckEvidenceScopeEntryInAPI(t, "anecdotes_requirement."+requirement, scopeType, evidenceID, want)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("test", firstEvidence, "control"),
				Check:  entry("test", "control", firstEvidence, scopeIncluded),
			},
			{
				Config:           config("test", secondEvidence, "control"),
				ConfigPlanChecks: replaced,
				Check: resource.ComposeAggregateTestCheckFunc(
					entry("test", "control", firstEvidence, scopeAbsent),
					entry("test", "control", secondEvidence, scopeIncluded),
				),
			},
			{
				Config:           config("test", secondEvidence, "framework"),
				ConfigPlanChecks: replaced,
				Check: resource.ComposeAggregateTestCheckFunc(
					entry("test", "control", secondEvidence, scopeAbsent),
					entry("test", "framework", secondEvidence, scopeIncluded),
				),
			},
			{
				Config:           config("other", secondEvidence, "framework"),
				ConfigPlanChecks: replaced,
				Check: resource.ComposeAggregateTestCheckFunc(
					entry("test", "framework", secondEvidence, scopeAbsent),
					entry("other", "framework", secondEvidence, scopeIncluded),
				),
			},
		},
	})
}

// A requirement deleted outside Terraform drops the scope out of state on the
// next refresh, and the following apply recreates the requirement and the scope.
func TestAccMappingRequirementEvidenceScopeResource_requirementDeletedOutsideTerraform(t *testing.T) {
	config := testAccEvidenceScopePrerequisites("scope-gone") + testAccEvidenceScopeFrameworkConfig(false)

	var deletedRequirementID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: func(s *terraform.State) (err error) {
					deletedRequirementID, err = stateAttr(s, "anecdotes_requirement.test", "requirement_id")
					return err
				},
			},
			{
				PreConfig: func() {
					if err := testAccNewClient(t).DeleteRequirement(context.Background(), deletedRequirementID); err != nil {
						t.Fatalf("deleting the requirement outside Terraform: %v", err)
					}
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(evidenceScopeAddr, plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(evidenceScopeAddr, "requirement_id", "anecdotes_requirement.test", "requirement_id"),
					func(s *terraform.State) error {
						id, err := stateAttr(s, evidenceScopeAddr, "requirement_id")
						if err != nil {
							return err
						}
						if id == deletedRequirementID {
							return fmt.Errorf("expected a recreated requirement, still on deleted requirement %s", id)
						}
						return nil
					},
					testCheckEvidenceScopeInAPI(t, "framework", scopeIncluded),
				),
			},
		},
	})
}

func TestAccMappingRequirementEvidenceScopeResource_requirementNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "anecdotes_mapping_requirement_evidence_scope" "test" {
  requirement_id = "requirement_1000000000000"
  evidence_id    = "1000000000"
  scope_id       = "control_1000000000000"
  scope_type     = "control"
}`,
				ExpectError: regexp.MustCompile(`Requirement Not Found`),
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

// A scope_id that is unknown at plan time is still checked against scope_type
// during the apply, before the resource is created.
func TestAccMappingRequirementEvidenceScopeResource_scopeTypeCheckedAtApply(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccEvidencePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEvidenceScopePrerequisites("scope-type") + fmt.Sprintf(`
resource "anecdotes_mapping_requirement_evidence_scope" "test" {
  requirement_id = anecdotes_requirement.test.requirement_id
  evidence_id    = %q
  scope_id       = anecdotes_framework.test.framework_id
  scope_type     = "control"
}`, testAccEvidenceID()),
				ExpectError: regexp.MustCompile(`Scope Type Does Not Match Scope ID`),
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
				ImportStateId: "requirement_1000000000000/framework/1000000000/1000000000",
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

func TestCheckEvidenceScopeType(t *testing.T) {
	cases := []struct {
		scopeID, scopeType string
		wantError          bool
	}{
		{"framework_1000000000000", client.EvidenceScopeFramework, false},
		{"control_1000000000000", client.EvidenceScopeControl, false},
		{"framework_1000000000000", client.EvidenceScopeControl, true},
		{"control_1000000000000", client.EvidenceScopeFramework, true},
		{"1000000000", client.EvidenceScopeFramework, false},
		{"1000000000", client.EvidenceScopeControl, false},
	}
	for _, tc := range cases {
		var diags diag.Diagnostics
		checkEvidenceScopeType(&diags, MappingRequirementEvidenceScopeResourceModel{
			ScopeID:   types.StringValue(tc.scopeID),
			ScopeType: types.StringValue(tc.scopeType),
		})
		if diags.HasError() != tc.wantError {
			t.Errorf("scope_id %s with scope_type %s: expected error = %t, got %v", tc.scopeID, tc.scopeType, tc.wantError, diags)
		}
	}
}

// Each failure of a scope write is reported with its own summary, and the
// framework note is added only for a framework scope.
func TestAddEvidenceScopeWriteError(t *testing.T) {
	notStored := fmt.Errorf("%w: evidence e1 in scope s1 of requirement r1", client.ErrEvidenceScopeNotStored)
	cases := []struct {
		name      string
		scopeType string
		err       error
		summary   string
		detail    string
		wantNote  bool
	}{
		{"requirement not found", client.EvidenceScopeFramework, fmt.Errorf("failed to read requirement r1: %w", client.ErrNotFound), "Requirement Not Found", "No requirement with ID r1", false},
		{"scope type mismatch", client.EvidenceScopeFramework, fmt.Errorf("%w: scope s1 of requirement r1 is recorded as a control scope", client.ErrEvidenceScopeTypeMismatch), "Scope Type Does Not Match", "recorded as a control scope", false},
		{"framework scope not stored", client.EvidenceScopeFramework, notStored, "Evidence Scope Not Stored", "evidence e1 in scope s1 of requirement r1", true},
		{"control scope not stored", client.EvidenceScopeControl, notStored, "Evidence Scope Not Stored", "evidence e1 in scope s1 of requirement r1", false},
		{"other error", client.EvidenceScopeFramework, errors.New("connection refused"), "Anecdotes API Error", "connection refused", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := MappingRequirementEvidenceScopeResourceModel{
				RequirementID: types.StringValue("r1"),
				ScopeID:       types.StringValue("s1"),
				EvidenceID:    types.StringValue("e1"),
				ScopeType:     types.StringValue(tc.scopeType),
			}
			var diags diag.Diagnostics
			addEvidenceScopeWriteError(&diags, "add evidence scope", data, tc.err)

			if len(diags) != 1 {
				t.Fatalf("expected one diagnostic, got %d", len(diags))
			}
			if got := diags[0].Summary(); got != tc.summary {
				t.Errorf("summary: expected %q, got %q", tc.summary, got)
			}
			detail := diags[0].Detail()
			if !strings.Contains(detail, tc.detail) {
				t.Errorf("detail: expected it to contain %q, got %q", tc.detail, detail)
			}
			if hasNote := strings.Contains(detail, frameworkScopeStoredNote); hasNote != tc.wantNote {
				t.Errorf("detail: expected framework note present = %t, got %q", tc.wantNote, detail)
			}
		})
	}
}
