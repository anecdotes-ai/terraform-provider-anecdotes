# ============================================
# Requirement Resources
# ============================================

resource "anecdotes_requirement" "test_requirement" {
  name        = "Test Requirement from Terraform"
  description = "<p>This requirement was created by the Terraform provider. Safe to delete.</p>"
  category    = "Terraform Tests"

  # Optional: Set owners
  # owners = ["user@example.com"]
}
