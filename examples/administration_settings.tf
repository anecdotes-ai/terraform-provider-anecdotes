# ============================================
# Administration & Settings Resources
# ============================================

# Look up a built-in base role to extend, rather than hardcoding its key.
data "anecdotes_role" "viewer" {
  role_id = "viewer_role"
}

resource "anecdotes_role" "anecdotes_auditor_readonly" {
  name        = "Auditor Read-Only"
  description = "Read-only access to a limited set of frameworks"

  extends     = [data.anecdotes_role.viewer.role_id]
  permissions = []
}

resource "anecdotes_login_settings" "anecdotes_login_settings" {
  internal_users_login        = true
  auditors_login              = true
  external_stakeholders_login = false

  google_idp_enabled    = true
  microsoft_idp_enabled = false

  support_team_login = true
}

resource "anecdotes_scim_api_key" "anecdotes_scim_key" {
  api_key_name = "OKTA"
}

# display_name must be unique per tenant: provider_id is derived from it.
resource "anecdotes_saml_configuration" "anecdotes_okta_saml" {
  display_name  = "OktaSSO"
  idp_entity_id = "http://www.okta.com/exampleIDPEntityID"
  rp_entity_id  = "https://gateway.anecdotes.ai/saml/metadata"
  sso_url       = "https://example.okta.com/app/exampleapp/sso/saml"

  x509_certificates = [
    file("${path.module}/okta-cert.pem"),
  ]
}
