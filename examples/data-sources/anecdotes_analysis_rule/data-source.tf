# Look up a single rule by ID. Supplying evidence_id is optional and only makes
# the lookup cheaper, by avoiding a read of the platform's full rule library.
data "anecdotes_analysis_rule" "example" {
  rule_id = "analysis_rule_1234567890123"
}

output "rule_query" {
  value = data.anecdotes_analysis_rule.example.rule_query_str
}
