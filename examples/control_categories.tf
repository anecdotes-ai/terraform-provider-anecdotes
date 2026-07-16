# ============================================
# Control Categories
# ============================================

resource "anecdotes_control_category" "anecdotes_control_category" {
  framework_id = anecdotes_framework.anecdotes_soc_2_framework.framework_id
  name         = "Anecdotes Control Category"
}
