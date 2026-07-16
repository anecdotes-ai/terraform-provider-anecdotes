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
control status and maturity level) at plan time. The following constraints depend
on live platform state and are therefore enforced only at apply:

- **Cross-entity references** — IDs such as `framework_id`, `evidence_id`,
  `control_id`, and owner email addresses are checked for existence and permission
  by the API. A non-existent or unauthorized reference fails at apply.
- **Control deletion** — only custom controls can be deleted; deleting a
  non-custom control is rejected by the API.
- **Uniqueness / duplicates** — name uniqueness (frameworks, categories, and
  similar) is enforced by the API. The provider recovers from "already exists"
  where possible but cannot detect duplicates at plan time.

## Feature-gated operations

Some operations require a platform feature to be enabled for your tenant. If the
feature is off, the operation fails at apply with a clear "feature not enabled"
message (HTTP 402). These are tenant-dependent and cannot be validated at plan
time.

## Enumerated values

Enumerated fields are validated at plan time against the set the provider supports.
That set reflects the values the API accepts for the operation, which may be a
deliberate subset of the platform's full enumeration (for example, only the
schedule frequencies an endpoint fully supports are exposed). Values outside the
supported set are rejected at plan time.

Closed value sets (statuses, types, modes) are verified against the owning
backend service and enforced at plan time. Fields whose value sets are
tenant-defined or discovered at runtime (such as requirement statuses and
categories) are intentionally not validated at plan time — the server remains
the authority for those.

## Server errors on malformed input

A few endpoints return a generic server error (HTTP 500) for certain malformed
input instead of a structured validation error. The provider surfaces these as a
redacted "Anecdotes API Error" with the status code; the raw server response is
never shown.
