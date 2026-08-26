resource "anecdotes_requirement" "acceptable_use_policy" {
  name        = "Acceptable use policy"
  description = "Defines the acceptable use of organizational assets"
  category    = "Access"
}

# A view is a requirement scoped beneath a parent — it inherits the parent's
# description, related evidences/policies, and scoping overrides on creation.
# Link it to a control with anecdotes_mapping_control_requirement just like
# any other requirement.
resource "anecdotes_requirement_view" "acceptable_use_policy_soc2" {
  parent_id = anecdotes_requirement.acceptable_use_policy.requirement_id
  view_name = "Acceptable use policy — SOC 2 scope"
  owners    = ["security@example.com"]
}
