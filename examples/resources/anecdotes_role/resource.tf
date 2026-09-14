# Look up a built-in base role to extend, rather than hardcoding its key.
data "anecdotes_role" "viewer" {
  role_id = "viewer_role"
}

resource "anecdotes_role" "auditor_readonly" {
  name        = "Auditor Read-Only"
  description = "Read-only access to a limited set of frameworks"

  # extends is what actually governs access. The platform resolves it into the
  # role's effective permissions, readable as effective_permissions.
  extends = [data.anecdotes_role.viewer.role_id]

  # permissions is omitted on purpose: the platform accepts the list but never
  # persists or enforces it, and leaving it unset keeps an imported role
  # planning clean.

  # Omit full_access_frameworks entirely for an unscoped role.
}
