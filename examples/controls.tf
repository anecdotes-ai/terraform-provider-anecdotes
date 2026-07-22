# ============================================
# Controls
# ============================================

resource "anecdotes_control" "anecdotes_control" {
  framework_id = anecdotes_framework.anecdotes_soc_2_framework.framework_id
  category_id  = anecdotes_control_category.anecdotes_control_category.category_id

  name        = "TF-001 - Test Control from Terraform"
  description = "This control was created by the Terraform provider. Safe to delete."

  # Maturity level. One of: INITIAL, REPEATABLE, DEFINED, MANAGED, OPTIMIZING.
  maturity_level = "INITIAL"

  # Optional: Add owners (use valid emails from your Anecdotes account)
  # owners = ["user@example.com"]

  # Optional: Add tags
  tags = ["terraform", "test", "automated"]
}
