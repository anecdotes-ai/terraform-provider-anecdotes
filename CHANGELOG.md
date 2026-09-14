# Changelog

All notable changes to this provider will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## Versioning policy

| Change type | Version bump | Example |
|---|---|---|
| New resources or data sources, new required attributes | **Minor** `1.X.0` | Adding a new data source |
| Backward-incompatible schema changes, renamed/removed attributes | **Major** `X.0.0` | Renaming `framework_id` → `parent_id` |
| Bug fixes, non-breaking attribute additions, documentation updates | **Patch** `1.0.X` | Fixing a retry bug |

The provider reaches `1.0.0` upon first publication to the Terraform Registry,
signalling a stable public API. Any breaking change after that point increments
the major version.

---

## [1.2.0] - 2026-09-14

### Added

- `anecdotes_analysis_rule` resource for managing the analysis rules an account
  authors: a query evaluated against one evidence's collected data, raising a
  gap or a warning on the rows it matches. Covers the full lifecycle including
  `rule_state`, which the platform applies through a separate call. Rules
  shipped with the platform are read-only and cannot be managed.
- `anecdotes_analysis_rule` and `anecdotes_analysis_rules` data sources for
  looking up a single rule or listing rules, filterable by evidence, origin,
  state and type. Deleted rules are archived rather than removed and are
  excluded unless `include_archived` is set.
- `service_instance_ids` on the `anecdotes_evidences` data source, reporting the
  service instances that collected an evidence. These are the values an analysis
  rule's `account_scoping_list` is expressed in.
- `anecdotes_playbook` resource for managing automations that run one or more
  steps when a platform event fires or on a schedule. Steps are configured as a
  nested list; a step can be chained to another by pointing its `trigger_event`
  at that step's `step_id`. Steps can be edited in place, but a change to which
  steps a playbook has replaces the playbook, as does removing a schedule.
- `anecdotes_playbook_library` data source listing the trigger events a playbook
  step can subscribe to, filterable by category and availability.
- `anecdotes_playbook_action_library` data source listing the actions a playbook
  step can perform.
- Resources:
  - `anecdotes_role` — tenant-scoped custom RBAC role (full create, read, update,
    delete, and import support).
  - `anecdotes_login_settings` — the tenant's Login Methods settings (singleton;
    no import, since there is nothing to identify by ID).
  - `anecdotes_saml_configuration` — SAML 2.0 identity provider configuration
    (full create, read, update, delete, and import support).
  - `anecdotes_scim_api_key` — API key scoped to SCIM provisioning (create,
    read, delete, and import; no update — there is no update endpoint).
- Data sources:
  - `anecdotes_role` and `anecdotes_roles` — look up one role, or list every
    role visible to the tenant (built-in global roles plus tenant-specific
    custom roles).

### Fixed

- `anecdotes_role` create recovery no longer resolves across built-in global
  roles. A custom role's key is derived from its name rather than copied from
  it, so a custom role can share a `name` with a global role without colliding
  on a key; an ambiguous 5xx on create could therefore adopt a platform-owned
  role into state, and a later destroy would try to delete it.
- `anecdotes_role.full_access_frameworks = []` no longer fails every apply. The
  platform normalizes an empty list to null and cannot echo `[]` back, so the
  configured value was being overwritten with null and reported as an
  inconsistent result.
- `anecdotes_role.extends = []` is now rejected while planning. The platform
  substitutes `["basic_role"]` for an empty `extends` exactly as it does for an
  omitted one, so the configured value could never be honored and the apply
  failed as an inconsistent result. Omit the attribute to get the default.
- Documented that `anecdotes_role`'s `description`, `extends` and
  `full_access_frameworks` cannot be cleared by removing them from
  configuration. All three are supplied by the platform when unset, so the
  prior value is carried forward and the plan reports no changes. Each
  attribute now says so, and KNOWN_LIMITATIONS lists the value to set instead:
  `extends = ["basic_role"]` for the default inheritance and
  `full_access_frameworks = []` for an unscoped role — the latter made
  possible by the empty-list fix above.
- `anecdotes_role.permissions` is now Optional rather than Required. It is never
  read back from the platform, so `terraform import` cannot populate it and a
  configuration was previously forced to carry a value the import could not
  produce — the first plan after an import was never clean.
- The `anecdotes_role` / `anecdotes_roles` data source descriptions no longer
  recommend using them to discover `permissions` values, which the platform
  ignores, and now say that their `permissions` is the resolved set the resource
  exposes as `effective_permissions`.
- Creating a control or a control category no longer records an empty id when
  the API answers with a body that parses but carries no id. The control
  reports the failure, and the category is recovered by name.

---

## [1.1.1] - 2026-08-30

### Changed

- Every API request now sends a `User-Agent` header identifying the provider
  version, Terraform CLI version, and Go runtime/platform (for example,
  `terraform-provider-anecdotes/1.1.1 (+https://github.com/anecdotes-ai/terraform-provider-anecdotes)
  Terraform/1.9.0 go1.25.13 darwin/arm64`), so a support report can be
  correlated to the exact build that produced it. It carries no credential or
  customer-identifying data.

## [1.1.0] - 2026-08-26

### Added

- `anecdotes_requirement_view` resource: manages a Requirement View, a
  requirement scoped beneath a parent requirement (`parent_id`, immutable),
  with its own `view_name`, `category`, and `owners`. Supports create, read,
  update, delete, and import.
- `parent_id` and `view_name` attributes on the `anecdotes_requirement` and
  `anecdotes_requirements` data sources, to reveal whether a looked-up
  requirement is a Requirement View.

### Fixed

- `anecdotes_requirement` now rejects a Requirement View's id, mirroring the
  existing check in `anecdotes_requirement_view` against a standalone
  requirement's id. Previously importing a view under `anecdotes_requirement`
  silently succeeded and misrepresented the object.

## [1.0.0]

Initial public release of the Anecdotes Terraform Provider, covering the core
compliance surface of the Anecdotes GRC platform.

### Added

- Provider configuration with `api_key` and `api_url` attributes, backed by the
  `ANECDOTES_API_KEY` and `ANECDOTES_API_URL` environment variables. The API key
  is exchanged for a short-lived bearer token that is refreshed automatically.
  The base URL must use `https` — plain `http` is accepted only for localhost, so
  a long-lived credential is never sent in clear text — and the provider does not
  follow redirects, so it cannot be forwarded to another host.
- Resources:
  - `anecdotes_framework`
  - `anecdotes_framework_folder`
  - `anecdotes_control`
  - `anecdotes_control_category`
  - `anecdotes_requirement`
  - `anecdotes_mapping_control_requirement`
  - `anecdotes_mapping_requirement_evidence`
- Data sources:
  - `anecdotes_framework` and `anecdotes_frameworks`
  - `anecdotes_control` and `anecdotes_controls`
  - `anecdotes_control_category` and `anecdotes_control_categories`
  - `anecdotes_requirement` and `anecdotes_requirements`
  - `anecdotes_framework_folder` and `anecdotes_framework_folders`
  - `anecdotes_evidences` (read-only)
- Full create, read, update, delete, and import support for the resources above
  (the two mapping resources support create, read, delete, and import).
- Plan-time validation of enumerated attributes (control maturity level,
  requirement category, framework auditor visibility, and data-source filters),
  and typed, redacted API error handling.
- Generated documentation for every resource and data source, runnable examples,
  and guides for drift and field ownership, local development, testing, and
  known limitations.
