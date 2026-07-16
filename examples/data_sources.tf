# ============================================
# Data Sources (read-only lookups)
# ============================================

# List all frameworks
data "anecdotes_frameworks" "all" {}

# List all evidences from a specific service
data "anecdotes_evidences" "github" {
  service_id = "github"
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
