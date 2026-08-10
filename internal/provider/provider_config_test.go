// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// A provider value that is only known at apply must be reported as unknown, not
// as missing. terraform_data.output is computed, so it is unknown during plan.
func TestAccProviderConfig_UnknownValuesAreReportedAsUnknown(t *testing.T) {
	cases := []struct {
		name        string
		config      string
		expectError *regexp.Regexp
	}{
		{
			name: "unknown api_key",
			config: `
resource "terraform_data" "key" {
  input = "value-known-only-at-apply"
}

provider "anecdotes" {
  api_key = terraform_data.key.output
}

data "anecdotes_frameworks" "test" {}
`,
			expectError: regexp.MustCompile(`(?s)Unknown API Key`),
		},
		{
			name: "unknown api_url",
			config: `
resource "terraform_data" "url" {
  input = "https://api.anecdotes.ai"
}

provider "anecdotes" {
  api_url = terraform_data.url.output
}

data "anecdotes_frameworks" "test" {}
`,
			expectError: regexp.MustCompile(`(?s)Unknown API URL`),
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
