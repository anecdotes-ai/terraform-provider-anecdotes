resource "anecdotes_framework" "soc2" {
  name      = "SOC2 Type II"
  folder_id = var.folder_id
}

resource "anecdotes_control_category" "access_management" {
  framework_id = anecdotes_framework.soc2.framework_id
  name         = "Access Management"
}

resource "anecdotes_control" "access_review" {
  framework_id = anecdotes_framework.soc2.framework_id
  category_id  = anecdotes_control_category.access_management.category_id
  name         = "Quarterly Access Reviews"
}
