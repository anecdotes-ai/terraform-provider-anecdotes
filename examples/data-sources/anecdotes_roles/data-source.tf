# Discover every built-in global role and its key, to reference in a custom
# role's `extends` without hardcoding role keys.
data "anecdotes_roles" "built_in" {
  is_custom = false
}

output "built_in_role_keys" {
  value = [for r in data.anecdotes_roles.built_in.roles : r.role_id]
}
