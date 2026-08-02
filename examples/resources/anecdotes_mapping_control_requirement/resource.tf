resource "anecdotes_framework_folder" "compliance" {
  name = "Compliance Frameworks"
}

resource "anecdotes_framework" "soc2" {
  name        = "SOC2 Type II"
  description = "SOC 2 Type II framework"
  folder_id   = anecdotes_framework_folder.compliance.folder_id
}

resource "anecdotes_control_category" "access_management" {
  framework_id = anecdotes_framework.soc2.framework_id
  name         = "Access Management"
}

resource "anecdotes_control" "access_control" {
  framework_id = anecdotes_framework.soc2.framework_id
  category_id  = anecdotes_control_category.access_management.category_id
  name         = "Logical access is restricted to authorized users"
}

resource "anecdotes_requirement" "mfa" {
  name        = "Multi-factor authentication is enforced"
  description = "MFA is required for all users with access to production systems"
  category    = "Access"
}

# The link itself. A requirement can be linked to controls in several
# frameworks by repeating this resource with a different control_id.
resource "anecdotes_mapping_control_requirement" "mfa_link" {
  control_id     = anecdotes_control.access_control.control_id
  requirement_id = anecdotes_requirement.mfa.requirement_id
}
