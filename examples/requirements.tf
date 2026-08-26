# ============================================
# Requirement Resources
# ============================================

resource "anecdotes_requirement" "test_requirement" {
  name        = "Test Requirement from Terraform"
  description = "<p>This requirement was created by the Terraform provider. Safe to delete.</p>"
  category    = "Custom Requirements"

  # Optional: Set owners
  # owners = ["user@example.com"]
}

# A view is a requirement scoped beneath a parent — it inherits the parent's
# description, related evidences/policies, and scoping overrides on creation.
# parent_id is immutable: changing it replaces the view.
resource "anecdotes_requirement_view" "test_requirement_view" {
  parent_id = anecdotes_requirement.test_requirement.requirement_id
  view_name = "Test Requirement View from Terraform"
  category  = "Custom Requirements"

  # Optional: Set owners
  # owners = ["user@example.com"]
}
