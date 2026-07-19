---
page_title: "Local Development - Anecdotes Provider Guide"
---

# Local Development Guide

How to build the provider from source and run it against a live Anecdotes account.

## Prerequisites

- **Go 1.25+** — [install Go](https://golang.org/doc/install)
- **Terraform 1.0+** — [install Terraform](https://www.terraform.io/downloads)
- **Make** — pre-installed on macOS/Linux
- **Anecdotes API key** — Admin role required

## Build and install

```bash
git clone https://github.com/anecdotes-ai/terraform-provider-anecdotes.git
cd terraform-provider-anecdotes
make install
```

`make install` builds the binary and places it in your local Terraform plugin
directory:

```
~/.terraform.d/plugins/registry.terraform.io/anecdotes-ai/anecdotes/<version>/<os_arch>/
```

`<os_arch>` is your platform, for example `darwin_arm64` or `linux_amd64`.

## Point Terraform at the local build

Create `~/.terraformrc` (`%APPDATA%\terraform.rc` on Windows) with a
`filesystem_mirror`, so Terraform loads the local build instead of querying the
registry. Terraform does **not** expand `$HOME` here — paste your real home path:

```hcl
provider_installation {
  filesystem_mirror {
    path    = "/Users/YOUR_USERNAME/.terraform.d/plugins"
    include = ["registry.terraform.io/anecdotes-ai/anecdotes"]
  }
  direct {
    exclude = ["registry.terraform.io/anecdotes-ai/anecdotes"]
  }
}
```

## Authenticate

Create an API token in the Anecdotes platform (**Administration → API Tokens**,
Admin role) and export it:

```bash
export ANECDOTES_API_KEY="your-api-key-here"
```

## Try it

```bash
cd examples/provider
terraform init
terraform plan
```

The `examples/` directory contains a runnable configuration for every resource
and data source.

## Iterate

After changing provider code, rebuild and re-run — Terraform picks up the new
binary from the mirror path:

```bash
make install
terraform plan
```

For verbose logs:

```bash
TF_LOG_PROVIDER=DEBUG terraform plan
```

## Make targets

| Command | Description |
|---------|-------------|
| `make build` | Build the provider binary |
| `make install` | Build and install into the local plugin directory |
| `make test` | Run unit tests |
| `make testacc` | Run acceptance tests (requires a live account) |
| `make fmt` | Format Go code and examples |
| `make lint` | Run the linter |
| `make docs` | Regenerate documentation |
| `make clean` | Remove build artifacts |

## Troubleshooting

- **Provider not found** — confirm the binary exists under
  `~/.terraform.d/plugins/registry.terraform.io/anecdotes-ai/anecdotes/` and that
  `~/.terraformrc` uses your real home path.
- **Authentication failed** — confirm `echo $ANECDOTES_API_KEY` prints your key
  and that the token has the Admin role.
- **Changes not taking effect** — re-run `make install`, then `terraform plan`.

## Cleanup

Destroy any test resources with `terraform destroy`, and remove the
`filesystem_mirror` block from `~/.terraformrc` when you want Terraform to use
the published provider again.
