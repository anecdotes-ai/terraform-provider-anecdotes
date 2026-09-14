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
			// The step checks must not wait on the schedule checks: a trigger
			// that is only known after apply must not hide a duplicate id.
			name: "playbook step identifiers are checked even when a trigger is unknown",
			config: folderConfig + `
resource "anecdotes_playbook" "test" {
  title       = "tf-test-validation-playbook"
  description = "validation"

  steps = [
    {
      step_id        = "11111111-2222-4333-8444-555555555555"
      title          = "first"
      trigger_event  = anecdotes_framework_folder.test.folder_id
      action_type    = "webhook"
      url_to_trigger = "https://example.com/validation"
    },
    {
      step_id        = "11111111-2222-4333-8444-555555555555"
      title          = "second"
      trigger_event  = "ControlStatusChanged"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/validation"
    },
  ]
}`,
			expectError: regexp.MustCompile(`(?s)Duplicate Step Identifier`),
		},
		{
			name: "playbook step defaulting to webhook requires a url",
			config: `
resource "anecdotes_playbook" "test" {
  title       = "tf-test-validation-playbook"
  description = "validation"

  steps = [
    {
      title         = "step"
      trigger_event = "ControlStatusChanged"
    },
  ]
}`,
			expectError: regexp.MustCompile(`(?s)Webhook Step Requires a URL`),
		},
		{
			name: "playbook webhook step requires a url",
			config: `
resource "anecdotes_playbook" "test" {
  title       = "tf-test-validation-playbook"
  description = "validation"

  steps = [
    {
      title         = "step"
      trigger_event = "ControlStatusChanged"
      action_type   = "webhook"
    },
  ]
}`,
			expectError: regexp.MustCompile(`(?s)Webhook Step Requires a URL`),
		},
		{
			name: "playbook step identifiers must be unique",
			config: `
resource "anecdotes_playbook" "test" {
  title       = "tf-test-validation-playbook"
  description = "validation"

  steps = [
    {
      step_id        = "11111111-2222-4333-8444-555555555555"
      title          = "first"
      trigger_event  = "ControlStatusChanged"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/validation"
    },
    {
      step_id        = "11111111-2222-4333-8444-555555555555"
      title          = "second"
      trigger_event  = "EvidenceGapDetected"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/validation"
    },
  ]
}`,
			expectError: regexp.MustCompile(`(?s)Duplicate Step Identifier`),
		},
		{
			name: "playbook schedule timezone must use a current name",
			config: `
resource "anecdotes_playbook" "test" {
  title       = "tf-test-validation-playbook"
  description = "validation"

  schedule_config = {
    period     = "day"
    time       = "09:00"
    timezone   = "US/Eastern"
    start_date = "2026-09-07T00:00:00Z"
  }

  steps = [
    {
      title          = "step"
      trigger_event  = "ScheduledPlaybookTriggered"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/validation"
    },
  ]
}`,
			expectError: regexp.MustCompile(`(?s)Deprecated Timezone Name`),
		},
		{
			name: "playbook schedule start date must be in coordinated universal time",
			config: `
resource "anecdotes_playbook" "test" {
  title       = "tf-test-validation-playbook"
  description = "validation"

  schedule_config = {
    period     = "day"
    time       = "09:00"
    start_date = "2026-09-07T09:00:00+02:00"
  }

  steps = [
    {
      title          = "step"
      trigger_event  = "ScheduledPlaybookTriggered"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/validation"
    },
  ]
}`,
			expectError: regexp.MustCompile(`(?s)Timestamp Must Be UTC`),
		},
		{
			name: "playbook schedule requires a scheduled trigger",
			config: `
resource "anecdotes_playbook" "test" {
  title       = "tf-test-validation-playbook"
  description = "validation"

  schedule_config = {
    period     = "day"
    time       = "09:00"
    start_date = "2026-09-07T00:00:00Z"
  }

  steps = [
    {
      title          = "step"
      trigger_event  = "ControlStatusChanged"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/validation"
    },
  ]
}`,
			expectError: regexp.MustCompile(`(?s)Schedule Requires a Scheduled Trigger`),
		},
		{
			name: "requirement view requires a parent_id",
			config: `
resource "anecdotes_requirement_view" "test" {
  view_name = "tf-test-validation-view"
}`,
			expectError: regexp.MustCompile(`(?s)"parent_id" is required`),
		},
		{
			name: "requirement view category must be a known category",
			config: `
resource "anecdotes_requirement_view" "test" {
  parent_id = "requirement_does_not_matter_for_plan_time_validation"
  view_name = "tf-test-validation-view"
  category  = "Not A Real Category"
}`,
			expectError: regexp.MustCompile(`(?s)category value must be\s+one of`),
		},
		{
			name: "requirement view owners must be email addresses",
			config: `
resource "anecdotes_requirement_view" "test" {
  parent_id = "requirement_does_not_matter_for_plan_time_validation"
  view_name = "tf-test-validation-view"
  owners    = ["not-an-email"]
}`,
			expectError: regexp.MustCompile(`(?s)must be a valid\s+email address`),
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

// The environment variable never reaches schema validation, so the resolved URL
// is checked while the provider is configured. This covers that wiring — the
// validator alone would leave the environment path open, which is how the
// credential would have leaked.
func TestAccValidation_RejectsPlaintextAPIURLFromEnvironment(t *testing.T) {
	t.Setenv("ANECDOTES_API_URL", "http://api.anecdotes.ai")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "anecdotes_framework_folder" "never_created" {
  name = "tf-test-plaintext-url-guard"
}`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?s)Insecure API URL.*clear text`),
			},
		},
	})
}
