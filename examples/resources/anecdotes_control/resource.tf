resource "anecdotes_framework" "soc2" {
  name = "SOC2 Type II"
}

resource "anecdotes_control" "access_reviews" {
  framework_id = anecdotes_framework.soc2.framework_id

  name        = "CC6.1 - Access reviews are performed quarterly"
  description = "The entity performs periodic access reviews to ensure appropriate access levels"
  category    = "Common Criteria"

  # Maturity level. One of: INITIAL, REPEATABLE, DEFINED, MANAGED, OPTIMIZING.
  maturity_level = "DEFINED"

  owners = ["security@example.com", "it-ops@example.com"]
}
