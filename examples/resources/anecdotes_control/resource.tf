resource "anecdotes_framework_folder" "compliance" {
  name = "Compliance Frameworks"
}

resource "anecdotes_framework" "soc2" {
  name        = "SOC2 Type II"
  description = "SOC 2 Type II framework"
  folder_id   = anecdotes_framework_folder.compliance.folder_id
}

resource "anecdotes_control_category" "common_criteria" {
  framework_id = anecdotes_framework.soc2.framework_id
  name         = "Common Criteria"
}

resource "anecdotes_control" "access_reviews" {
  framework_id = anecdotes_framework.soc2.framework_id
  category_id  = anecdotes_control_category.common_criteria.category_id

  name        = "CC6.1 - Access reviews are performed quarterly"
  description = "The entity performs periodic access reviews to ensure appropriate access levels"

  # Maturity level. One of: INITIAL, REPEATABLE, DEFINED, MANAGED, OPTIMIZING.
  maturity_level = "DEFINED"

  owners = ["security@example.com", "it-ops@example.com"]
}
