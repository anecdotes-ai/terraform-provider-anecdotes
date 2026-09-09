# Every trigger event available to this account.
data "anecdotes_playbook_library" "all" {}

# Only the control events that can be used right now.
data "anecdotes_playbook_library" "control_events" {
  category       = "control"
  available_only = true
}

output "control_trigger_values" {
  value = [for e in data.anecdotes_playbook_library.control_events.events : e.trigger_value]
}
