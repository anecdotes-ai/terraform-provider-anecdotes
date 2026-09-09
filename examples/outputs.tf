# ============================================
# Outputs
# ============================================

output "folder_id" {
  value = anecdotes_framework_folder.anecdotes_grc_folder.folder_id
}

output "folder_name" {
  value = anecdotes_framework_folder.anecdotes_grc_folder.name
}

output "framework_id" {
  value = anecdotes_framework.anecdotes_soc_2_framework.framework_id
}

output "framework_name" {
  value = anecdotes_framework.anecdotes_soc_2_framework.name
}

output "control_id" {
  value = anecdotes_control.anecdotes_control.control_id
}

output "role_id" {
  value = anecdotes_role.anecdotes_auditor_readonly.role_id
}

output "saml_provider_id" {
  value = anecdotes_saml_configuration.anecdotes_okta_saml.provider_id
}
