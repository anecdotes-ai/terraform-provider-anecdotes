# ============================================
# Framework Resources
# ============================================

resource "anecdotes_framework" "anecdotes_soc_2_framework" {
  name        = "Anecdotes SOC 2"
  description = "This is Anecdotes' SOC 2 framework"
  folder_id   = anecdotes_framework_folder.anecdotes_grc_folder.folder_id

  # Basic auditor config
  can_auditor_download_evidence = true
  can_auditor_view_soa_report   = true
}
