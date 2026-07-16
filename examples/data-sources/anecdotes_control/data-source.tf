data "anecdotes_control" "access_reviews" {
  control_id   = "control_12345"
  framework_id = "framework_12345"
}

output "control_name" {
  value = data.anecdotes_control.access_reviews.name
}
