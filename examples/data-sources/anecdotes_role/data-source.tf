data "anecdotes_role" "viewer" {
  role_id = "viewer_role"
}

output "viewer_permissions" {
  value = data.anecdotes_role.viewer.permissions
}
