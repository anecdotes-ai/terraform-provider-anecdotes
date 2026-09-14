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
)

// TestAccLoginSettingsResource_toggle exercises anecdotes_login_settings
// against a live tenant. This resource is a singleton with no delete — a
// destroy at the end of resource.Test's normal lifecycle would not restore
// whatever the tenant's login settings were before this test ran. So this
// captures the pre-test baseline directly and restores it itself, instead of
// relying on Terraform's destroy step to undo anything.
func TestAccLoginSettingsResource_toggle(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test — set TF_ACC=1 to run")
	}
	testAccPreCheck(t)

	c := testAccNewClient(t)
	baseline, err := c.GetLoginSettings(context.Background())
	if err != nil {
		t.Fatalf("reading baseline login settings: %v", err)
	}
	t.Cleanup(func() {
		if _, err := c.UpdateLoginSettings(context.Background(), baseline); err != nil {
			t.Errorf("cleanup: restoring baseline login settings: %v", err)
		}
	})

	toggled := !baseline.SupportTeamLogin

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_login_settings" "test" {
  support_team_login = %t
}`, toggled),
				Check: resource.TestCheckResourceAttr("anecdotes_login_settings.test", "support_team_login", fmt.Sprintf("%t", toggled)),
			},
			{
				// Un-toggle in the same test run, ahead of the cleanup restore,
				// so the resource's own Update path is exercised too (not just
				// Create), and the tenant spends as little time as possible
				// away from its original value.
				Config: fmt.Sprintf(`
resource "anecdotes_login_settings" "test" {
  support_team_login = %t
}`, baseline.SupportTeamLogin),
				Check: resource.TestCheckResourceAttr("anecdotes_login_settings.test", "support_team_login", fmt.Sprintf("%t", baseline.SupportTeamLogin)),
			},
		},
	})
}

// TestAccLoginSettingsResource_preservesUnmanagedFields confirms the
// read-merge-write in applyLoginSettings does not clobber seamless_login,
// which this resource deliberately does not expose as an attribute.
func TestAccLoginSettingsResource_preservesUnmanagedFields(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test — set TF_ACC=1 to run")
	}
	testAccPreCheck(t)

	c := testAccNewClient(t)
	baseline, err := c.GetLoginSettings(context.Background())
	if err != nil {
		t.Fatalf("reading baseline login settings: %v", err)
	}
	t.Cleanup(func() {
		if _, err := c.UpdateLoginSettings(context.Background(), baseline); err != nil {
			t.Errorf("cleanup: restoring baseline login settings: %v", err)
		}
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Manage one unrelated attribute; do not mention seamless_login
				// anywhere, since this resource's schema has no slot for it.
				Config: fmt.Sprintf(`
resource "anecdotes_login_settings" "test" {
  google_idp_enabled = %t
}`, baseline.Idps.Google),
				Check: resource.TestCheckResourceAttrSet("anecdotes_login_settings.test", "support_team_login"),
			},
		},
		CheckDestroy: func(s *terraform.State) error {
			current, err := c.GetLoginSettings(context.Background())
			if err != nil {
				return fmt.Errorf("reading login settings post-apply: %w", err)
			}
			if current.SeamlessLogin != baseline.SeamlessLogin {
				return fmt.Errorf("seamless_login changed from %t to %t even though this resource never configured it",
					baseline.SeamlessLogin, current.SeamlessLogin)
			}
			return nil
		},
	})
}
