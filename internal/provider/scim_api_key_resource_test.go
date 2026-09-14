// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccScimApiKeyResource_create(t *testing.T) {
	name := randomName("scim")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_scim_api_key" "test" {
  api_key_name = %q
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_scim_api_key.test", "api_key_name", name),
					resource.TestCheckResourceAttrSet("anecdotes_scim_api_key.test", "key_id"),
					resource.TestCheckResourceAttrSet("anecdotes_scim_api_key.test", "key"),
				),
			},
		},
	})
}

// TestAccScimApiKeyResource_rename confirms api_key_name is RequiresReplace:
// there is no update endpoint, so a rename must destroy and recreate rather
// than attempt an in-place update (which Update() explicitly refuses).
func TestAccScimApiKeyResource_rename(t *testing.T) {
	name1 := randomName("scim-rename")
	name2 := randomName("scim-renamed")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_scim_api_key" "test" {
  api_key_name = %q
}`, name1),
				Check: resource.TestCheckResourceAttr("anecdotes_scim_api_key.test", "api_key_name", name1),
			},
			{
				Config: fmt.Sprintf(`
resource "anecdotes_scim_api_key" "test" {
  api_key_name = %q
}`, name2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("anecdotes_scim_api_key.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.TestCheckResourceAttr("anecdotes_scim_api_key.test", "api_key_name", name2),
			},
		},
	})
}

func TestAccScimApiKeyResource_import(t *testing.T) {
	name := randomName("scim-imp")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_scim_api_key" "test" {
  api_key_name = %q
}`, name),
			},
			{
				ResourceName:                         "anecdotes_scim_api_key.test",
				ImportState:                          true,
				ImportStateVerifyIdentifierAttribute: "key_id",
				ImportStateIdFunc:                    importIDFromAttr("anecdotes_scim_api_key.test", "key_id"),
				// key is not verified: import can only recover the platform's
				// truncated (last 8 characters) value, never the full secret
				// this resource's state held from the original create response.
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"},
			},
		},
	})
}
