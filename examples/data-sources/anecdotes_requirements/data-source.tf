data "anecdotes_requirements" "all" {}

output "requirement_count" {
  value = data.anecdotes_requirements.all.total_count
}
