data "anecdotes_evidences" "policies" {
  name_contains = "All policies"
}

# Every rule attached to one evidence, the account's own and the platform's.
data "anecdotes_analysis_rules" "for_evidence" {
  evidence_id = data.anecdotes_evidences.policies.evidences[0].evidence_id
}

# Only the rules the account authored. These are the ones
# anecdotes_analysis_rule can manage.
data "anecdotes_analysis_rules" "custom" {
  rule_origin = "custom"
}

# Library rules that are currently switched off.
data "anecdotes_analysis_rules" "disabled_library_rules" {
  rule_origin = "library"
  rule_state  = "inactive"
}

output "disabled_library_rule_names" {
  value = [for r in data.anecdotes_analysis_rules.disabled_library_rules.rules : r.rule_name]
}
