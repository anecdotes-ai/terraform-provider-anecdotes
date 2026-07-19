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

## [1.0.0]

Initial public release of the Anecdotes Terraform Provider, covering the core
compliance surface of the Anecdotes GRC platform.

### Added

- Provider configuration with `api_key` and `api_url` attributes, backed by the
  `ANECDOTES_API_KEY` and `ANECDOTES_API_URL` environment variables. The API key
  is exchanged for a short-lived bearer token that is refreshed automatically.
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
  framework auditor visibility, and data-source filters), and typed, redacted
  API error handling.
- Generated documentation for every resource and data source, runnable examples,
  and local-development, testing, and known-limitations guides.
