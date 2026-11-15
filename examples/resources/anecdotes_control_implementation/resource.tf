# A control provided with a framework: look it up, then manage its
# implementation text. The control itself is never created or deleted.
data "anecdotes_controls" "soc2_access" {
  framework_id  = "fw_12345"
  name_contains = "CC6.1"
}

resource "anecdotes_control_implementation" "cc6_1" {
  control_id = data.anecdotes_controls.soc2_access.controls[0].control_id

  control_implementation = <<-EOT
    Access to production systems requires single sign-on with MFA.
    Access rights are reviewed quarterly by the security team.
  EOT
}

# A control created in Terraform works the same way.
resource "anecdotes_framework_folder" "compliance" {
  name = "Compliance Frameworks"
}

resource "anecdotes_framework" "internal" {
  name        = "Internal Controls"
  description = "Controls defined by our organization"
  folder_id   = anecdotes_framework_folder.compliance.folder_id
}

resource "anecdotes_control_category" "operations" {
  framework_id = anecdotes_framework.internal.framework_id
  name         = "Operations"
}

resource "anecdotes_control" "backups" {
  framework_id = anecdotes_framework.internal.framework_id
  category_id  = anecdotes_control_category.operations.category_id
  name         = "Production data is backed up daily"
}

resource "anecdotes_control_implementation" "backups" {
  control_id             = anecdotes_control.backups.control_id
  control_implementation = "Nightly snapshots are kept for 35 days and restored in a quarterly drill."
}
