# display_name must be unique per tenant: provider_id is derived from it, so a
# duplicate display_name is rejected as a conflict rather than creating a second
# configuration.
resource "anecdotes_saml_configuration" "okta" {
  display_name  = "OktaSSO"
  idp_entity_id = "http://www.okta.com/exampleIDPEntityID"
  rp_entity_id  = "https://gateway.anecdotes.ai/saml/metadata"
  sso_url       = "https://example.okta.com/app/exampleapp/sso/saml"

  x509_certificates = [
    file("${path.module}/okta-cert.pem"),
  ]
}
