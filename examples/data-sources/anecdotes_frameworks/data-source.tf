# List all frameworks
data "anecdotes_frameworks" "all" {}

output "framework_count" {
  value = data.anecdotes_frameworks.all.total_count
}

output "frameworks" {
  value = data.anecdotes_frameworks.all.frameworks
}

# List only adopted frameworks
data "anecdotes_frameworks" "adopted" {
  is_applicable = true
}

output "adopted_frameworks" {
  value = data.anecdotes_frameworks.adopted.frameworks
}

# Search by name
data "anecdotes_frameworks" "nist" {
  name_contains = "NIST"
}

output "nist_frameworks" {
  value = data.anecdotes_frameworks.nist.frameworks
}
