# The full key secret (key) is only ever available immediately after creation.
# Store it somewhere durable (e.g. a secrets manager) when applying this for
# real — it cannot be retrieved again afterward.
resource "anecdotes_scim_api_key" "okta" {
  api_key_name = "OKTA"
}
