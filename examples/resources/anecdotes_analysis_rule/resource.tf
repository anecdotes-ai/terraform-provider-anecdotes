# An analysis rule is scoped to one evidence, so start by locating it.
data "anecdotes_evidences" "policies" {
  name_contains = "All policies"
}

locals {
  policies_evidence = data.anecdotes_evidences.policies.evidences[0]
}

# Raise a gap on every policy that has not been approved. The left-hand side is
# a column of the evidence's collected data, so it has to match that evidence's
# own columns.
resource "anecdotes_analysis_rule" "unapproved_policies" {
  evidence_id  = local.policies_evidence.evidence_id
  rule_name    = "Policy is not approved"
  rule_message = "This policy is still in draft or awaiting approval."
  alert_level  = 50

  rule_query = jsonencode({
    left     = "Policy Status"
    operator = "IsIn"
    right    = ["Draft", "Pending Approval"]
  })
}

# A rule can be authored and left switched off until it is ready.
resource "anecdotes_analysis_rule" "staged" {
  evidence_id = local.policies_evidence.evidence_id
  rule_name   = "Policy owner missing"
  alert_level = 30
  rule_state  = "inactive"

  rule_query = jsonencode({
    left     = "Policy Owner"
    operator = "IsIn"
    right    = [""]
  })
}

# Scoping a rule to particular accounts. The IDs come from the evidence itself;
# the platform rejects any that do not belong to the evidence's service.
data "anecdotes_evidences" "slack_admins" {
  service_id    = "slack"
  name_contains = "admin users"
}

resource "anecdotes_analysis_rule" "slack_admins_one_account" {
  evidence_id  = data.anecdotes_evidences.slack_admins.evidences[0].evidence_id
  rule_name    = "Slack administrator present"
  rule_message = "Review whether this account still needs administrator access."

  account_scoping_type = "included_accounts"
  account_scoping_list = data.anecdotes_evidences.slack_admins.evidences[0].service_instance_ids

  rule_query = jsonencode({
    left     = "Is Admin?"
    operator = "IsIn"
    right    = ["TRUE"]
  })
}
