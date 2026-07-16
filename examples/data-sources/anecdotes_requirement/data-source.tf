data "anecdotes_requirement" "access_review" {
  requirement_id = "requirement_12345"
}

output "requirement_name" {
  value = data.anecdotes_requirement.access_review.name
}
