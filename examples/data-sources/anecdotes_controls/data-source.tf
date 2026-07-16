data "anecdotes_controls" "soc2" {
  framework_id = "fw_12345"
}

output "control_count" {
  value = data.anecdotes_controls.soc2.total_count
}
