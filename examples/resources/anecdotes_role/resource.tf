# Look up a built-in base role to extend, rather than hardcoding its key.
data "anecdotes_role" "viewer" {
  role_id = "viewer_role"
}

resource "anecdotes_role" "auditor_readonly" {
  name        = "Auditor Read-Only"
  description = "Read-only access to a limited set of frameworks"

  extends     = [data.anecdotes_role.viewer.role_id]
  permissions = []

  # Omit full_access_frameworks entirely for an unscoped role.
}
