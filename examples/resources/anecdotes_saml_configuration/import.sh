# Import a SAML configuration by its provider_id.
#
# provider_id is derived from display_name only once, at creation — it is not
# recomputable from the current display_name after a rename. If you don't
# already have the provider_id (e.g. state was lost after a rename), look it
# up in the platform UI or via the identity API's list endpoint first.
terraform import anecdotes_saml_configuration.example saml.a0000000000
