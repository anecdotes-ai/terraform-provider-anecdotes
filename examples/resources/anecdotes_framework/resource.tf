# Create a custom framework with auditor configuration
resource "anecdotes_framework" "custom_security" {
  name        = "Custom Security Framework"
  description = "Internal security controls for our organization"

  # Auditor access settings
  can_auditor_download_evidence = true
  can_auditor_view_soa_report   = true

  # Control statuses visible to auditors. A status is visible when present in the set.
  # Valid members: approved_by_auditor, gap, in_progress, insufficient_data, issue,
  # monitoring, not_applicable, not_started, ready_for_audit, under_review.
  auditor_visible_control_statuses = [
    "approved_by_auditor",
    "ready_for_audit",
    "issue",
    "insufficient_data",
    "under_review",
  ]

  # Evidence statuses visible to auditors. Valid members: auditable, gap, not_set.
  auditor_visible_evidence_statuses = [
    "auditable",
    "gap",
    "not_set",
  ]
}
