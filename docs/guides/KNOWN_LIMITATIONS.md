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

## SAML configuration rename does not carry provider_id forward

`display_name` can be changed in place without replacing the resource, but
`provider_id` is derived from `display_name` only once, at creation, and does
not track a later rename. If Terraform state is lost after a rename,
`provider_id` cannot be recomputed from the current `display_name` — look it
up in the platform UI, or via the identity API's list endpoint, before
importing. A later configuration created with the original (pre-rename)
`display_name` also collides with the renamed one's `provider_id`, surfacing
as a conflict on a name nobody is currently using.

## SAML configuration delete confirms before treating a 502 as success

The platform has no reliable way to report that a SAML configuration was
already gone: an unknown `provider_id` and a genuine deletion failure both
surface the same plain-text 502. Rather than treat every 502 as success,
`terraform destroy` follows up with a read — only when that confirms the
configuration is actually gone does the destroy succeed. A 502 caused by a
real, transient failure, with the configuration still present, surfaces as an
ordinary error instead, so it still shows up on the next `terraform plan`.

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

This is also the one resource in this provider whose state file holds a
working credential, not just a reference to one. `key` is marked `Sensitive`,
so it is redacted from plan and apply output the same way the provider's own
`api_key` is — but unlike `api_key`, this value **is** written to state and
stays there for the life of the resource. Protect your state backend
accordingly (see the README's authentication and secrets notes).

## Error reporting

When the API rejects a request without structured validation details, the
provider surfaces the failure as a redacted "Anecdotes API Error" carrying the
status code and the operation that failed. Raw server responses are never
shown.
