resource "anecdotes_requirement" "access_review" {
  name        = "User access is reviewed quarterly"
  description = "Access rights are reviewed and approved every quarter"
  category    = "Access"
}

# The evidence counts for the requirement only within this framework.
resource "anecdotes_mapping_requirement_evidence_scope" "sox_access_review" {
  requirement_id = anecdotes_requirement.access_review.requirement_id
  evidence_id    = "evidence_12345"
  scope_id       = "framework_67890"
  scope_type     = "framework"
}

# Evidence linked to the requirement everywhere is left out of this control.
resource "anecdotes_mapping_requirement_evidence_scope" "exclude_from_control" {
  requirement_id = anecdotes_requirement.access_review.requirement_id
  evidence_id    = "evidence_54321"
  scope_id       = "control_55312"
  scope_type     = "control"
  excluded       = true
}
