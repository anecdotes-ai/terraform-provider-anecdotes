resource "anecdotes_mapping_control_requirement" "mfa_link" {
  control_id     = anecdotes_control.access_control.control_id
  requirement_id = anecdotes_requirement.mfa.requirement_id
}
