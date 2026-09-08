# Login Settings is a singleton: exactly one instance manages the whole tenant's
# Login Methods configuration. Any attribute left unset here is not managed by
# Terraform and keeps whatever value the platform already has.
resource "anecdotes_login_settings" "this" {
  internal_users_login        = true
  auditors_login              = true
  external_stakeholders_login = false

  google_idp_enabled    = true
  microsoft_idp_enabled = false

  support_team_login = true
}
