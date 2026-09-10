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
- `anecdotes_analysis_rule.rule_name` and `rule_message` — the API ignores an
  empty value on update, so these can be changed but not cleared once set.
  Removing the attribute keeps the value the platform holds rather than failing;
  set a new value to change it.

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

## Playbook steps cannot be added or removed

A playbook's steps can be edited in place, but the platform has no way to add a
step to an existing playbook or to remove one from it. Changing how many steps a
playbook has, or which `step_id`s it carries, replaces the playbook. The
replacement is a new playbook with a new `playbook_id`.

## A playbook schedule cannot be removed

`schedule_config` can be added to a playbook at creation and changed afterwards,
but it cannot be cleared. Removing the attribute replaces the playbook.

A schedule is only meaningful when the first step's `trigger_event` is
`ScheduledPlaybookTriggered`; the two are validated together at plan time. The
platform filters that step to the playbook's own schedule, so its
`filter_configuration` cannot be set.

## Playbook payload and header configurations drop empty top-level values

`payload_configuration` and `headers_configuration` are JSON objects, and the
platform does not store a top-level entry whose key is empty or whose value is
`false`, `null`, `0`, `""`, `[]` or `{}`. Nested values are stored as written.
The provider rejects such an entry at plan time rather than letting it disappear
on apply; remove the key instead. `filter_configuration` is not affected: it
keeps empty top-level values.

## Removing a playbook step configuration leaves it in place

The three step configuration attributes are optional and computed, so removing
one from the configuration leaves the stored value untouched rather than
clearing it. Set it to `jsonencode({})` to clear it.

## Playbooks are read by listing them

The platform has no endpoint that reads a single playbook, so every read fetches
all of them and selects one. A configuration managing many playbooks therefore
issues one full listing per playbook per plan.

## Playbook step retries are not configurable

Playbook steps run once. There is no retry setting to configure, so the provider
does not expose one.

## A playbook step's action and payload are not validated against its trigger

Two rules govern a step, and the platform enforces neither. A step that breaks
either one is accepted and stored, but never runs, and the application shows the
playbook as an empty row.

1. `action_type` must be one the trigger supports. Every event lists its own
   `supported_actions`, and they differ: `EvidenceGapDetected` supports
   `create_finding` but not `create_task`, for example. The provider validates
   `action_type` against the full set of actions the platform defines, which is
   wider than the set any single trigger accepts.
2. The action's required fields must be present in `payload_configuration`.
   `create_finding` requires `title`, `severity` and `reported_by`;
   `webhook` requires `url_to_trigger` instead, which is its own attribute.

Both sets are readable before writing a step. `anecdotes_playbook_library`
reports `supported_actions` and `event_fields` for each event, and
`anecdotes_playbook_action_library` reports `action_fields` with `is_required`
for each action.

A field referenced from a filter or a payload template must also be one the
event carries: `event_fields` lists them, along with which are filterable and
the operator a filter on each must use.

## An unrecognised playbook trigger is reported on apply, not on plan

`action_type` is checked while planning, because the set of actions the platform
defines is fixed. `trigger_event` is not: which events exist, and which are
available, differs between accounts, so a list held in the provider would reject
values that are valid elsewhere and accept values that are unavailable here.

A misspelled event name therefore passes `terraform plan` and fails `terraform
apply`, naming the attribute and the reason:

```
Error: Invalid Configuration
  Unable to create playbook: steps.0.step_trigger_event: Invalid event trigger id
```

Nothing is created when this happens. Read the value from
`anecdotes_playbook_library` rather than writing it by hand to avoid the round
trip.

## Some playbook actions are not yet available

`action_type` accepts every action the platform defines, including ones it has
announced but not yet made available. A step configured with one of those is
stored but does not run. The `anecdotes_playbook_action_library` data source
reports `coming_soon` for each action; check it before relying on one.

## A playbook schedule is written in UTC

`start_date` and `end_date` are stored in UTC, and `timezone` is stored under
the current IANA name for the zone. Timestamps carrying a non-UTC offset, and
deprecated zone names such as `US/Eastern`, are rejected at plan time rather
than being read back in a different form.

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

## Analysis Rules

**Only custom rules can be managed.** `anecdotes_analysis_rule` manages rules the
account authors (`rule_origin` `custom`). The rules shipped with the platform
(`rule_origin` `library`) cannot be created, updated or deleted through it, and
the provider refuses to read one into state — importing a library rule fails with
an error rather than producing a resource that cannot be applied. Read them with
the `anecdotes_analysis_rules` data source.

**Deleting archives rather than removes.** The API has no hard delete: destroying
a rule marks it archived, and it keeps being returned by the underlying read
endpoints. The provider treats an archived rule as absent, so Terraform converges
correctly, but the rule remains visible in the Anecdotes application and in the
`anecdotes_analysis_rules` data source when `include_archived` is set.

**`rule_query` is reparsed by the platform, not merely reformatted.** A rule
condition is stored with only `operator`, `left` and `right`; any other key is
discarded. The provider rejects such a query at plan time rather than letting it
apply as something other than what was written. This covers an `aql` query, and
the condition under an `aqlext` query's `filters` key, which is stored the same
way.

**A `pandas` query is stored lower-cased.** The whole expression is lowered,
quoted values included, so `` `Policy Status` == "Draft" `` is stored as
`` `policy status` == "draft" ``. The provider rejects a query that is not
already lower case, because it would otherwise store an expression other than the
one written. A comparison against mixed-case data cannot be expressed this way.

`aqlext` is a different structure rather than a richer condition: it is built
from `base`, `manipulations` and `filters`. An `aql`-shaped query submitted as
`aqlext` is rejected by the platform, so it is not a way to keep keys that `aql`
discards.

What survives is re-serialized with the platform's own key order and spacing.
That never shows up as a pending change: the provider recognises a stored query
that means the same thing as the configured one and leaves the configuration's
version in state.

Two consequences follow. Editing the key order in your own configuration *does*
plan an update, because Terraform compares configuration to state as text; the
apply is harmless. And an imported rule arrives formatted the way the platform
stores it, since there is no configuration yet to compare against.

Value *types* are compared as written: the platform stores typed values, so a
quoted boolean or number (`"true"` rather than `true`) produces a difference that
never settles. Prefer `jsonencode()` over a heredoc string, which gets the types
right for you.

**`alert_level` accepts more values than a rule should carry.** The API accepts
3, 5 and 10 in addition to 30 and 50, but those describe the outcome of an
evaluation rather than a severity a rule raises. The provider rejects them at
plan time.

**`account_scoping_list` is filtered by the platform.** Only instances of the
service that collects the rule's evidence are kept, and only while they are
installed; a request naming none of them is rejected outright. When the platform
drops an ID the provider reports which one, because the resulting configuration
would no longer match what was applied.

The `anecdotes_evidences` data source reports `service_instance_ids`, the
instances that *collected* each evidence. That is a close guide rather than an
exact list of what a rule will accept: an instance that collected the evidence
but has since been uninstalled still appears there and is refused by the rule.

The list also cannot be emptied while `account_scoping_type` stays
`included_accounts` or `excluded_accounts`, for the same reason `rule_name` and
`rule_message` cannot be cleared. Set `account_scoping_type` to `all_accounts`,
which clears the list as part of the change.

**`rule_state` is applied by a separate call.** Creating an inactive rule takes a
create followed by a state change. If the state change fails, the rule still
exists and is recorded as configured, with a warning saying so; the next plan
reports the difference against the platform and applies the state change on its
own.

## Enumerated values

Fields with a closed value set — `anecdotes_control.maturity_level`,
`anecdotes_requirement.category`, and the auditor visibility statuses — are
validated at plan time; values outside the set are rejected before any API call.

Requirement categories are the categories Anecdotes defines, the same list the
Requirements Hub offers. Requirements that do not fit one of them belong under
`Custom Requirements`.

## Error reporting

When the API rejects a request without structured validation details, the
provider surfaces the failure as a redacted "Anecdotes API Error" carrying the
status code and the operation that failed. Raw server responses are never
shown.
