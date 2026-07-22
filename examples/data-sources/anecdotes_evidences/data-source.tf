data "anecdotes_evidences" "all" {}

data "anecdotes_evidences" "aws" {
  service_id = "aws_guard_duty"
}

output "evidence_count" {
  value = data.anecdotes_evidences.all.total_count
}
