# A requirement is a standalone object. Link it to one or more controls with
# anecdotes_mapping_control_requirement — the same requirement can satisfy
# controls in different frameworks.
resource "anecdotes_requirement" "quarterly_access_review" {
  name        = "Perform quarterly user access review"
  description = "Review all user access rights quarterly and remove inappropriate access"
  category    = "Access"
  owners      = ["security@example.com"]
}
