# Anecdotes Terraform Provider

Manage the [Anecdotes](https://anecdotes.ai) GRC (Governance, Risk & Compliance)
compliance program as Infrastructure as Code.

**7 resources** | **11 data sources** | Full create, read, update, delete, and import

---

## Table of Contents

- [Quick Start](#quick-start)
- [Resource Model](#resource-model)
- [Resources](#resources)
- [Data Sources](#data-sources)
- [Examples](#examples)
- [Development](#development)
- [Repository Structure](#repository-structure)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)

---

## Quick Start

### 1. Configure the provider

```hcl
terraform {
  required_providers {
    anecdotes = {
      source  = "anecdotes-ai/anecdotes"
      version = "~> 1.0"
    }
  }
}

provider "anecdotes" {
  api_key = var.anecdotes_api_key # or set the ANECDOTES_API_KEY environment variable
}
```

### 2. Get your API key

1. Log into Anecdotes as an Admin.
2. Go to **Administration > API Tokens**.
3. Create a token with the **Admin** role.

```bash
export ANECDOTES_API_KEY="your-api-key-here"
```

> **Authentication & secrets**
> - **Prefer the `ANECDOTES_API_KEY` environment variable** over hardcoding `api_key` in `.tf` files.
> - The `api_key` attribute is marked `Sensitive`, so it is redacted from plan/apply output — but it is still stored in state, so treat your state backend as sensitive.
> - **Never commit API keys, tokens, or `*.tfvars` files** containing secrets. This repository's `.gitignore` excludes `.env`, `*.env`, and `*.tfvars`.
> - Each user supplies **their own** API key; keys are not bundled with the provider.
> - The provider authenticates when it is configured, so `terraform plan` and
>   `terraform apply` need network access and valid credentials even for no-op
>   plans. `terraform validate` works offline.

The API base URL defaults to `https://api.anecdotes.ai` and can be overridden with
the `api_url` attribute or the `ANECDOTES_API_URL` environment variable.

### 3. Apply

```bash
terraform init
terraform plan
terraform apply
```

---

## Resource Model

```
anecdotes_framework_folder        # Optional folder that groups frameworks
    │
    └── anecdotes_framework        # Compliance standard (SOC 2, ISO 27001, ...)
            │
            ├── anecdotes_control_category   # Grouping of controls in a framework
            │
            └── anecdotes_control            # Control within a framework
                    │
                    └── anecdotes_mapping_control_requirement  # Links a control to requirements (M:N)
                            │
                            └── anecdotes_requirement           # Operational requirement (shared across frameworks)
                                    │
                                    └── anecdotes_mapping_requirement_evidence  # Links a requirement to evidence
```

- **Framework** — a compliance standard container (for example SOC 2, ISO 27001).
- **Framework folder** — an optional container that groups frameworks.
- **Control** — a prescriptive statement of what should be implemented, grouped by control category.
- **Requirement** — an operational action that satisfies controls; requirements can be shared across frameworks.
- **Mappings** — the M:N links between controls and requirements, and between requirements and evidence.

---

## Resources

| Resource | Description |
|----------|-------------|
| `anecdotes_framework` | A compliance framework. |
| `anecdotes_framework_folder` | A folder that groups frameworks. |
| `anecdotes_control` | A control within a framework. |
| `anecdotes_control_category` | A category grouping controls in a framework. |
| `anecdotes_requirement` | An operational requirement. |
| `anecdotes_mapping_control_requirement` | Links a control to one or more requirements. |
| `anecdotes_mapping_requirement_evidence` | Links a requirement to a piece of evidence. |

## Data Sources

| Data source | Description |
|-------------|-------------|
| `anecdotes_framework` / `anecdotes_frameworks` | Look up one framework, or list frameworks. |
| `anecdotes_control` / `anecdotes_controls` | Look up one control, or list controls in a framework. |
| `anecdotes_control_category` / `anecdotes_control_categories` | Look up one control category, or list them. |
| `anecdotes_requirement` / `anecdotes_requirements` | Look up one requirement, or list requirements. |
| `anecdotes_framework_folder` / `anecdotes_framework_folders` | Look up one framework folder, or list them. |
| `anecdotes_evidences` | List evidence (read-only). |

Per-attribute documentation is generated for every resource and data source under
[`docs/`](docs/).

---

## Examples

Runnable examples live under [`examples/`](examples/): a `provider/` configuration,
one directory per resource and data source, and a combined walkthrough at the top
level (`frameworks.tf`, `controls.tf`, `requirements.tf`, and so on).

---

## Development

### Prerequisites

- Go 1.25+
- Terraform 1.0+ on your PATH

### Build & Test

```bash
make build      # build the provider binary
make test       # unit tests (no tenant required)
make testacc    # acceptance tests (TF_ACC=1; requires an Anecdotes tenant)
make lint       # golangci-lint
make fmt        # gofmt + terraform fmt
make docs       # regenerate docs/ with tfplugindocs
```

See [docs/guides/LOCAL_DEVELOPMENT.md](docs/guides/LOCAL_DEVELOPMENT.md) for local
development setup and [docs/guides/TESTING.md](docs/guides/TESTING.md) for the
testing guide.

---

## Repository Structure

```
.
├── main.go                     # Provider entry point
├── internal/
│   ├── client/                 # Anecdotes API client
│   └── provider/               # Provider, resources (*_resource.go), data sources (*_data_source.go)
├── examples/                   # Usage examples (also rendered into docs)
├── docs/                       # Generated documentation
├── templates/                  # tfplugindocs guide templates
└── .github/workflows/          # CI (build/lint/format, docs-drift) and release
```

---

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) and the
[Code of Conduct](CODE_OF_CONDUCT.md) before opening a pull request. Security issues
should be reported privately as described in [SECURITY.md](SECURITY.md).

## License

This provider is distributed under the [Mozilla Public License 2.0](LICENSE).

## Support

- [Anecdotes Help Center](https://help.anecdotes.ai)
- Open an issue at <https://github.com/anecdotes-ai/terraform-provider-anecdotes/issues>
