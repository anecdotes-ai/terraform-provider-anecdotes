data "anecdotes_control_category" "common_criteria" {
  category_id = "control_category_12345"
}

output "category_name" {
  value = data.anecdotes_control_category.common_criteria.category_name
}
