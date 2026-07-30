// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// These tests assert that invalid configurations are rejected during plan, so
// no API call is made and nothing is created on the tenant. Each case is a
// plan-only step with an expected error.

func TestAccValidation_RejectedAtPlanTime(t *testing.T) {
	folder := randomName("folder-validation")

	folderConfig := fmt.Sprintf(`
resource "anecdotes_framework_folder" "test" {
  name = %q
}`, folder)

	cases := []struct {
		name        string
		config      string
		expectError *regexp.Regexp
	}{
		{
			name: "framework description cannot be empty",
			config: folderConfig + `
resource "anecdotes_framework" "test" {
  name        = "tf-test-validation-fw"
  description = ""
  folder_id   = anecdotes_framework_folder.test.folder_id
}`,
			expectError: regexp.MustCompile(`(?s)description.*string length must be\s+at least 1`),
		},
		{
			name: "framework requires a folder",
			config: `
resource "anecdotes_framework" "test" {
  name        = "tf-test-validation-fw"
  description = "Missing folder"
}`,
			expectError: regexp.MustCompile(`(?s)"folder_id" is required`),
		},
		{
			name: "framework folder name cannot be empty",
			config: `
resource "anecdotes_framework_folder" "test" {
  name = ""
}`,
			expectError: regexp.MustCompile(`(?s)name.*string length must be\s+between 1 and 255`),
		},
		{
			name: "requirement category must be a known category",
			config: `
resource "anecdotes_requirement" "test" {
  name     = "tf-test-validation-req"
  category = "Not A Real Category"
}`,
			expectError: regexp.MustCompile(`(?s)category value must be\s+one of`),
		},
		{
			name: "requirement owners must be email addresses",
			config: `
resource "anecdotes_requirement" "test" {
  name   = "tf-test-validation-req"
  owners = ["not-an-email"]
}`,
			expectError: regexp.MustCompile(`(?s)must be a valid\s+email address`),
		},
		{
			name: "control owners must be email addresses",
			config: folderConfig + `
resource "anecdotes_framework" "test" {
  name        = "tf-test-validation-fw"
  description = "Owner validation"
  folder_id   = anecdotes_framework_folder.test.folder_id
}

resource "anecdotes_control_category" "test" {
  name         = "tf-test-validation-cat"
  framework_id = anecdotes_framework.test.framework_id
}

resource "anecdotes_control" "test" {
  framework_id = anecdotes_framework.test.framework_id
  category_id  = anecdotes_control_category.test.category_id
  name         = "tf-test-validation-ctl"
  owners       = ["not-an-email"]
}`,
			expectError: regexp.MustCompile(`(?s)must be a valid\s+email address`),
		},
		{
			name: "control maturity level must be a known level",
			config: folderConfig + `
resource "anecdotes_framework" "test" {
  name        = "tf-test-validation-fw"
  description = "Maturity validation"
  folder_id   = anecdotes_framework_folder.test.folder_id
}

resource "anecdotes_control_category" "test" {
  name         = "tf-test-validation-cat"
  framework_id = anecdotes_framework.test.framework_id
}

resource "anecdotes_control" "test" {
  framework_id   = anecdotes_framework.test.framework_id
  category_id    = anecdotes_control_category.test.category_id
  name           = "tf-test-validation-ctl"
  maturity_level = "ADVANCED"
}`,
			expectError: regexp.MustCompile(`(?s)maturity_level value must be\s+one of`),
		},
		{
			name: "framework auditor visible statuses must be known statuses",
			config: folderConfig + `
resource "anecdotes_framework" "test" {
  name                             = "tf-test-validation-fw"
  description                      = "Auditor validation"
  folder_id                        = anecdotes_framework_folder.test.folder_id
  auditor_visible_control_statuses = ["everything"]
}`,
			expectError: regexp.MustCompile(`(?s)value must be\s+one of`),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      c.config,
						PlanOnly:    true,
						ExpectError: c.expectError,
					},
				},
			})
		})
	}
}
