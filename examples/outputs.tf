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
