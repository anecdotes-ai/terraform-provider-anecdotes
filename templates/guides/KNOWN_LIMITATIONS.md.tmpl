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

## Requirements provided by the platform cannot be deleted

The `anecdotes_requirement` resource manages both custom requirements and the
requirements Anecdotes provides — setting owners or a description on a provided
requirement works normally. Deleting one does not: only custom requirements can
be deleted, so `terraform destroy` on a provided requirement fails with an
explanatory error rather than reporting a success that did not happen. Remove it
from state with `terraform state rm` if Terraform should stop managing it.

## Requirement Views

`anecdotes_requirement_view` manages a requirement scoped beneath a parent
requirement (`parent_id`). A few things are specific to views:

- **`parent_id` is immutable.** The API rejects any attempt to change it after
  creation; changing it in configuration replaces the view rather than updating
  it in place.
- **Some fields are inherited from the parent and are not exposed on the view.**
  On creation, the platform copies the parent's description, related evidences
  and policies, and scoping overrides onto the view — these are not attributes
  of `anecdotes_requirement_view` and cannot be set independently through
  Terraform. `category` and `owners` are not inherited; they behave exactly like
  on `anecdotes_requirement`.
- **Deleting the parent deletes its views.** If a parent `anecdotes_requirement`
  managed elsewhere is destroyed, every view beneath it is deleted too; the next
  `terraform plan` for that view shows it needing to be created again rather than
  failing.

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
management with `terraform import`. This recovery does not apply to
`anecdotes_requirement_view`: `view_name` is not unique across views, so a
by-name lookup could adopt the wrong one. A create error on a view always
surfaces as-is.

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

Requirement categories are the categories Anecdotes defines, the same list the
Requirements Hub offers. Requirements that do not fit one of them belong under
`Custom Requirements`.

## SAML configuration display name must be unique

`anecdotes_saml_configuration`'s `provider_id` is derived from `display_name`, so
two configurations with the same `display_name` collide on the same identifier.
Creating one fails with a conflict error rather than a generic uniqueness error
— rename the configuration to resolve it.

## SAML configuration delete is best-effort

The platform has no reliable way to report that a SAML configuration was
already gone: an unknown `provider_id` and a genuine deletion failure both
surface the same server error. `terraform destroy` treats that ambiguous
response as success rather than getting permanently stuck on a configuration
that was removed outside Terraform. If the delete genuinely failed for another
reason, the configuration will still show up on the next `terraform plan`.

## Login Methods settings is a singleton, without import

`anecdotes_login_settings` manages the tenant's one login configuration, which
always exists on the platform — there is nothing to look up by ID, so
`terraform import` is not supported for this resource. Removing the resource
block only stops Terraform from managing the settings; it does not reset or
clear them.

## SCIM API key secret is available only once

`anecdotes_scim_api_key`'s `key` attribute holds the full secret only in the
response to the create call — every later read from the platform returns just
the last 8 characters. Store the value somewhere durable when you apply the
resource; it cannot be retrieved again afterward, including via
`terraform import` (an imported key's `key` attribute holds only the truncated
value).

## Error reporting

When the API rejects a request without structured validation details, the
provider surfaces the failure as a redacted "Anecdotes API Error" carrying the
status code and the operation that failed. Raw server responses are never
shown.
