---
page_title: "Known Limitations - Anecdotes Provider Guide"
---

# Known Limitations

Some input validation can only be performed by the Anecdotes API, not by the
provider during `terraform plan`. In those cases an invalid configuration passes
`plan` and fails during `apply` with an actionable error from the API. This guide
lists the known cases so you can anticipate them.

## Validation performed only by the server

The provider validates required fields, types, and enumerated values (for example
control maturity level and auditor visibility statuses) at plan time. The
following constraints depend on live platform state and are therefore enforced
only at apply:

- **Cross-entity references** — IDs such as `framework_id`, `evidence_id`,
  `control_id`, and owner email addresses are checked for existence and permission
  by the API. A non-existent or unauthorized reference fails at apply.
- **Control deletion** — only custom controls can be deleted; deleting a
  non-custom control is rejected by the API.
- **Uniqueness / duplicates** — name uniqueness (frameworks, categories, and
  similar) is enforced by the API and cannot be detected at plan time. Creating
  an entity whose name already exists fails at apply; bring the existing entity
  under management with `terraform import` instead.

## Control status is read-only

The `anecdotes_control` resource does not manage control status — status is
computed by the platform from evidence and monitoring signals. Inspect it with
the `anecdotes_control` or `anecdotes_controls` data sources.

## Create recovery after ambiguous server errors

Creating a framework, control category, or requirement can occasionally return
a server error (HTTP 500) even though the object WAS created. In that case the
provider recovers its own creation by looking the object up by name. If an
unrelated object with the same name already existed at that moment, the lookup
can match it instead — use distinctive names for Terraform-managed objects to
avoid ambiguity.

## Authentication happens at provider configuration

The provider exchanges the API key for a session token when it is configured,
so `terraform plan` and `terraform apply` require network access to the
Anecdotes API and valid credentials — even for plans that change nothing.
`terraform validate` works offline.

## Feature-gated operations

Some operations require a platform feature to be enabled for your tenant. If the
feature is off, the operation fails at apply with a clear "feature not enabled"
message (HTTP 402). These are tenant-dependent and cannot be validated at plan
time.

## Enumerated values

Fields with a closed value set (such as `maturity_level` and the auditor
visibility statuses) are validated at plan time; values outside the set are
rejected before any API call. Fields whose values are tenant-defined or
discovered at runtime (such as requirement categories) are deliberately not
validated at plan time — the server remains the authority for those.

## Server errors on malformed input

A few endpoints return a generic server error (HTTP 500) for certain malformed
input instead of a structured validation error. The provider surfaces these as a
redacted "Anecdotes API Error" with the status code; the raw server response is
never shown.
