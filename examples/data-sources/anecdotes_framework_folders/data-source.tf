data "anecdotes_framework_folders" "all" {}

output "folder_count" {
  value = data.anecdotes_framework_folders.all.total_count
}
