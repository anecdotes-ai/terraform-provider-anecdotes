---
page_title: "Drift and Field Ownership - Anecdotes Provider Guide"
---

# Drift and Field Ownership

Anecdotes objects can be changed in two places: in your Terraform configuration
and in the Anecdotes application. When the two disagree, the difference is called
drift. This guide explains what the provider does about it and how to choose the
behavior you want.

## Terraform-owned and platform-owned fields

Every attribute in this provider falls into one of two groups:

- **Terraform-owned** — the configuration is the source of truth. `terraform plan`
  reports any change made elsewhere, and `terraform apply` sets the value back to
  what the configuration says. Names, descriptions, owners, categories, folder
  placement, and auditor configuration all work this way.
- **Platform-owned** — the platform is the source of truth and the attribute is
  read-only (`Computed`). Terraform records the current value and never writes
  it. Control status and `framework_auditable` work this way.

Reverting a change is the default because it is what a configuration file means:
if `owners = ["a@example.com"]` is in the configuration, that is the intended
state, and any other value is drift.

## Analysis rule queries

`anecdotes_analysis_rule.rule_query` is Terraform-owned, and drift on it is
detected against the query the platform stored rather than the text in the
configuration. Editing a rule's query in the Anecdotes application therefore
shows up on the next plan and is reverted on apply.

A query the platform re-serialized is not drift: the provider compares the stored
query against the configured one structurally, so the platform's key order and
spacing never produce a pending change. A change in a value's *type* is a real
difference, so writing `"true"` where the platform stores `true` reports drift
that applying cannot settle. Use `jsonencode()` to get the types right.

Note that a rule condition is stored with only `operator`, `left` and `right`,
whether it is an `aql` query or the condition under an `aqlext` query's `filters`
key. The provider rejects other keys at plan time, so they never reach state and
never appear as drift.

`rule_origin`, `rule_query_str`, `rule_query_message`, `last_updated` and
`last_updated_by` are platform-owned and never written back.

## Seeing drift

`terraform plan` refreshes state and shows what an apply would change:

```console
$ terraform plan
  # anecdotes_control.access_review will be updated in-place
  ~ resource "anecdotes_control" "access_review" {
      ~ description = "Edited in the UI" -> "Quarterly access review"
    }
```

Use the exit code in automation — `2` means drift was found:

```console
$ terraform plan -detailed-exitcode
```

Deletions are detected as well: an object removed in the application is
re-created by the next apply. This includes an `anecdotes_requirement_view`
whose parent `anecdotes_requirement` was deleted — the platform cascades the
delete to every view beneath it, so the view disappears too even though nothing
targeted it directly; the next `terraform plan` shows it needing to be created
again.

## Absorbing a change made in the application

Terraform cannot edit your configuration file, so a change made in the
application resolves in one of two ways. Both are standard Terraform, not
provider features.

**Keep the platform value and stop managing that attribute** — add
`lifecycle { ignore_changes = [...] }`:

```terraform
resource "anecdotes_control" "access_review" {
  framework_id = anecdotes_framework.soc2.framework_id
  category_id  = anecdotes_control_category.access.category_id
  name         = "Quarterly Access Reviews"
  description  = "Quarterly access review"

  lifecycle {
    ignore_changes = [description]
  }
}
```

The plan reports "No changes" even when the description differs, and Terraform
stops writing that attribute. Remove the block to take the attribute back.

**Adopt the platform value into state, then update the configuration** — run a
refresh-only apply:

```console
$ terraform apply -refresh-only
```

State now holds the platform value, so the next plan shows the difference
against your configuration. Copy the value into the configuration to settle it,
or leave it and let the next apply revert it.

## Playbook step order

The order of a playbook's steps is Terraform-owned: it is the order the
configuration lists them in. The platform returns the steps of a playbook in an
order of its own that changes as steps are edited, so the provider restores the
configured order on read. A step edited in the application therefore shows as a
change to that step, never as a reordering of the list.

Because step order is restored by `step_id`, chaining a step to another one
means setting `step_id` explicitly on the step being chained to. Steps left
without an id are given one when the playbook is created.

## What drift detection does not cover

Objects created outside Terraform are not adopted: an object that no resource
block refers to is invisible to Terraform, however similar it looks to one you
manage. Bring it under management with `terraform import`.

Attributes that cannot be cleared by removing them, and the controls Terraform
cannot manage at all, are listed in the Known Limitations guide.

## Protecting objects from deletion

Removing a resource block deletes the object. Guard the ones that must not be
deleted by accident:

```terraform
resource "anecdotes_framework" "soc2" {
  name        = "SOC2 Type II"
  description = "SOC 2 Type II framework"
  folder_id   = anecdotes_framework_folder.compliance.folder_id

  lifecycle {
    prevent_destroy = true
  }
}
```
