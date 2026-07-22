# Look up a framework folder by name and read the frameworks it contains.
data "anecdotes_framework_folder" "security" {
  name = "Security Frameworks"
}

output "folder_frameworks" {
  value = data.anecdotes_framework_folder.security.frameworks_list
}
