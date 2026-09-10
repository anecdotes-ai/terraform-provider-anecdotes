// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// samlTestDisplayName strips randomName's hyphens: display_name is validated
// against `^\w+$` (letters, digits, underscore only).
func samlTestDisplayName(prefix string) string {
	return strings.ReplaceAll(randomName(prefix), "-", "")
}

func testAccSamlConfigurationConfig(displayName string) string {
	return fmt.Sprintf(`
resource "anecdotes_saml_configuration" "test" {
  display_name      = %q
  idp_entity_id     = "TfAccTestIDP"
  rp_entity_id      = "TfAccTestSP"
  sso_url           = "https://idp.example.com/sso/saml"
  x509_certificates = ["-----BEGIN CERTIFICATE-----\nMIIBTEST\n-----END CERTIFICATE-----"]
}`, displayName)
}

func TestAccSamlConfigurationResource_create(t *testing.T) {
	name := samlTestDisplayName("saml")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSamlConfigurationConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_saml_configuration.test", "display_name", name),
					resource.TestCheckResourceAttrSet("anecdotes_saml_configuration.test", "provider_id"),
					resource.TestCheckResourceAttrSet("anecdotes_saml_configuration.test", "idp_type"),
				),
			},
		},
	})
}

// TestAccSamlConfigurationResource_renamePreservesProviderID is the
// regression test for the rename/provider_id caveat: display_name changes in
// place (no replace), and provider_id — despite being derived from
// display_name — stays exactly what it was at creation.
func TestAccSamlConfigurationResource_renamePreservesProviderID(t *testing.T) {
	name1 := samlTestDisplayName("samlrename")
	name2 := samlTestDisplayName("samlrenamed")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSamlConfigurationConfig(name1),
				Check:  resource.TestCheckResourceAttr("anecdotes_saml_configuration.test", "display_name", name1),
			},
			{
				Config: testAccSamlConfigurationConfig(name2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_saml_configuration.test", "display_name", name2),
					resource.TestCheckResourceAttrSet("anecdotes_saml_configuration.test", "provider_id"),
				),
			},
		},
	})
}

func TestAccSamlConfigurationResource_update(t *testing.T) {
	name := samlTestDisplayName("samlupd")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSamlConfigurationConfig(name),
				Check:  resource.TestCheckResourceAttr("anecdotes_saml_configuration.test", "sso_url", "https://idp.example.com/sso/saml"),
			},
			{
				Config: fmt.Sprintf(`
resource "anecdotes_saml_configuration" "test" {
  display_name      = %q
  idp_entity_id     = "TfAccTestIDP"
  rp_entity_id      = "TfAccTestSP"
  sso_url           = "https://idp.example.com/sso/saml-updated"
  x509_certificates = ["-----BEGIN CERTIFICATE-----\nMIIBTEST\n-----END CERTIFICATE-----"]
}`, name),
				Check: resource.TestCheckResourceAttr("anecdotes_saml_configuration.test", "sso_url", "https://idp.example.com/sso/saml-updated"),
			},
		},
	})
}

func TestAccSamlConfigurationResource_import(t *testing.T) {
	name := samlTestDisplayName("samlimp")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSamlConfigurationConfig(name),
			},
			{
				ResourceName:                         "anecdotes_saml_configuration.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "provider_id",
				ImportStateIdFunc:                    importIDFromAttr("anecdotes_saml_configuration.test", "provider_id"),
			},
		},
	})
}
