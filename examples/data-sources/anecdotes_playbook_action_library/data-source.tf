# Every action a playbook step can perform.
data "anecdotes_playbook_action_library" "all" {}

# Only the actions that are available today.
data "anecdotes_playbook_action_library" "available" {
  exclude_coming_soon = true
}

output "available_action_types" {
  value = [for a in data.anecdotes_playbook_action_library.available.actions : a.action_type]
}
