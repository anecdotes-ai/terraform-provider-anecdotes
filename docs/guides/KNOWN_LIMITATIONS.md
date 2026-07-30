---
page_title: "Known Limitations - Anecdotes Provider Guide"
---

# Known Limitations

This guide lists the places where the provider cannot do what a Terraform user
would otherwise expect: validation that only the API can perform, attributes that
are not cleared by removing them, and objects the provider will not manage. For
how the provider reconciles changes made in the Anecdotes application, see the
Drift and Field Ownership guide.

## Validation performed only by the server

The provider validates required fields, types, and enumerated values (for example
control maturity level and auditor visibility statuses) at plan time. The
following constraints depend on live platform state and are therefore enforced
only at apply:

- **Cross-entity references** — IDs such as `framework_id`, `evidence_id`,
  `control_id`, and owner email addresses are checked for existence and permission
  by the API. A non-existent or unauthorized reference fails at apply.
- **Uniqueness / duplicates** — name uniqueness (frameworks, categories, and
  similar) is enforced by the API and cannot be detected at plan time. Creating
  an entity whose name already exists fails at apply; bring the existing entity
  under management with `terraform import` instead.

## Control status is read-only

The `anecdotes_control` resource does not manage control status — status is
computed by the platform from evidence and monitoring signals. Inspect it with
the `anecdotes_control` or `anecdotes_controls` data sources.

## Only custom controls can be managed

The `anecdotes_control` resource manages controls created through Terraform or
the Anecdotes application. Controls provided as part of a platform framework
cannot be updated or deleted, so Terraform cannot manage them: reading one — by
import or from existing state — fails with an error pointing at the data
sources. Read platform-provided controls with the `anecdotes_control` or
`anecdotes_controls` data source.

## Clearing attributes

Most optional attributes are cleared by removing them from the configuration.
The following behave differently:

- `anecdotes_framework.auditor_visible_control_statuses` and
  `auditor_visible_evidence_statuses` — removing the attribute keeps the last
  applied visibility. Set an empty set to hide every status.
- `anecdotes_requirement.category` — always has a value (default
  `Custom Requirements`); set a different category rather than removing it.

Everything else clears normally. Removing `maturity_level` clears the level on
the platform, setting a description to `""` empties it, and `owners` — on both
`anecdotes_control` and `anecdotes_requirement` — is owned by Terraform:
removing the attribute clears the owners, and owners added in the Anecdotes
application are reverted on the next apply.

## Create recovery when the outcome is unknown

If a create call returns a server error, the outcome is unknown: the object may
or may not have been created. Rather than risk a duplicate, the provider looks
the object up by name and adopts the match into state. If an unrelated object
with the same name existed at that moment, the lookup can match it instead, so
use distinctive names for Terraform-managed objects. A create that fails with an
explicit "already exists" conflict is never adopted — bring that object under
management with `terraform import`.

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

Fields with a closed value set — `anecdotes_control.maturity_level`,
`anecdotes_requirement.category`, and the auditor visibility statuses — are
validated at plan time; values outside the set are rejected before any API call.
Requirement categories match the categories available in the Requirements Hub;
to use a category that is not listed, create the requirement under
`Custom Requirements`.

## Error reporting

When the API rejects a request without structured validation details, the
provider surfaces the failure as a redacted "Anecdotes API Error" carrying the
status code and the operation that failed. Raw server responses are never
shown.
