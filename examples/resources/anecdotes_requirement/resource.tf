# Create a standalone requirement
resource "anecdotes_requirement" "quarterly_access_review" {
  name        = "Perform quarterly user access review"
  description = "Review all user access rights quarterly and remove inappropriate access"
  category    = "Access Reviews"
  owner       = "security@example.com"
}

# This requirement can satisfy multiple controls across frameworks
resource "anecdotes_mapping_control_requirement" "soc2_link" {
  control_id     = anecdotes_control.soc2_access_control.control_id
  requirement_id = anecdotes_requirement.quarterly_access_review.requirement_id
}

resource "anecdotes_mapping_control_requirement" "iso_link" {
  control_id     = anecdotes_control.iso_access_control.control_id
  requirement_id = anecdotes_requirement.quarterly_access_review.requirement_id # Same requirement!
}
