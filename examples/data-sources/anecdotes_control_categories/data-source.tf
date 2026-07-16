data "anecdotes_control_categories" "soc2" {
  framework_id = "fw_12345"
}

output "categories" {
  value = data.anecdotes_control_categories.soc2.categories
}
