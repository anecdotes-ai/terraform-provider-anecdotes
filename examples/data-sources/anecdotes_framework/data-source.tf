# Look up a framework by ID
data "anecdotes_framework" "soc2" {
  framework_id = "1234567890" # replace with your framework ID
}

# Use the framework data
output "soc2_name" {
  value = data.anecdotes_framework.soc2.name
}

output "soc2_is_auditable" {
  value = data.anecdotes_framework.soc2.framework_auditable
}

output "soc2_references" {
  value = data.anecdotes_framework.soc2.framework_references
}
