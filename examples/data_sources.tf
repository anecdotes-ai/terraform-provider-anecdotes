# ============================================
# Data Sources (read-only lookups)
# ============================================

# List all frameworks
data "anecdotes_frameworks" "all" {}

# List all evidences from a specific service
data "anecdotes_evidences" "github" {
  service_id = "github"
}

# List every built-in global role, to discover valid `extends` keys
data "anecdotes_roles" "built_in" {
  is_custom = false
}

# ============================================
# Data Source Outputs
# ============================================

output "framework_count" {
  value = data.anecdotes_frameworks.all.total_count
}

output "github_evidence_count" {
  value = data.anecdotes_evidences.github.total_count
}

output "built_in_role_keys" {
  value = [for r in data.anecdotes_roles.built_in.roles : r.role_id]
}
