---
page_title: "Local Development - Anecdotes Provider Guide"
---

# Local Development Guide

This guide explains how to build the Anecdotes Terraform Provider locally and test it against a live Anecdotes account.

## Prerequisites

- **Go 1.25+** - [Install Go](https://golang.org/doc/install)
- **Terraform 1.0+** - [Install Terraform](https://www.terraform.io/downloads)
- **Make** - Usually pre-installed on macOS/Linux
- **Anecdotes API Key** - Admin role required

## Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/anecdotes-ai/terraform-provider-anecdotes.git
cd terraform-provider-anecdotes

# 2. Build and install locally
make install

# 3. Set your API key
export ANECDOTES_API_KEY="your-api-key-here"

# 4. Test with Terraform
cd examples/provider
terraform init
terraform plan
```

---

## Step-by-Step Instructions

### 1. Build the Provider

```bash
# Build the binary
make build

# Or build directly with Go
go build -o terraform-provider-anecdotes .
```

This creates a `terraform-provider-anecdotes` binary in the current directory.

### 2. Install for Local Terraform Use

The `make install` command installs the provider to your local Terraform plugin directory:

```bash
make install
```

This installs to:
```
~/.terraform.d/plugins/registry.terraform.io/anecdotes-ai/anecdotes/0.1.0/darwin_arm64/
```

> **Note**: The path varies by OS and architecture:
> - macOS ARM: `darwin_arm64`
> - macOS Intel: `darwin_amd64`
> - Linux: `linux_amd64`

### 3. Configure Terraform to Use Local Provider

Create a `~/.terraformrc` file (or `%APPDATA%\terraform.rc` on Windows). Use **filesystem_mirror** so Terraform does not query the registry (the provider is not published there).

**Important:** Terraform does *not* expand `$HOME` in `.terraformrc`. Use your **actual home path** (e.g. run `echo $HOME` and paste it):

```hcl
provider_installation {
  filesystem_mirror {
    path    = "/Users/YOUR_USERNAME/.terraform.d/plugins"   # replace with your $HOME path
    include = ["registry.terraform.io/anecdotes-ai/anecdotes"]
  }
  direct {
    exclude = ["registry.terraform.io/anecdotes-ai/anecdotes"]
  }
}
```

Then run `make install` from the repo root so the provider binary is placed under that path. You can then run `terraform init` and `terraform plan`/`apply` as usual.

### 4. Set Up Authentication

Get your API key from the Anecdotes platform:

1. Log into Anecdotes as an Admin
2. Go to **Administration → API Tokens**
3. Create a new token with **Admin** role
4. Copy the token

Set the environment variable:

```bash
export ANECDOTES_API_KEY="your-api-key-here"
```

Or create a `.env` file (don't commit this!):

```bash
echo 'ANECDOTES_API_KEY="your-api-key-here"' > .env
source .env
```

---

## Testing with Live Account

### Create a Test Configuration

Create a new directory for testing:

```bash
mkdir -p ~/terraform-anecdotes-test
cd ~/terraform-anecdotes-test
```

Create `main.tf`:

```hcl
terraform {
  required_providers {
    anecdotes = {
      source = "anecdotes-ai/anecdotes"
    }
  }
}

provider "anecdotes" {
  # Uses ANECDOTES_API_KEY environment variable
}

# Test: Read an existing framework
data "anecdotes_framework" "test" {
  framework_id = "1234567890"  # SOC 2 framework ID
}

output "framework_name" {
  value = data.anecdotes_framework.test.name
}

output "framework_status" {
  value = data.anecdotes_framework.test.framework_status
}

output "is_auditable" {
  value = data.anecdotes_framework.test.framework_auditable
}
```

### Run Terraform Commands

```bash
# Skip init when using dev_overrides
# terraform init  # Not needed!

# Preview changes
terraform plan

# Apply changes
terraform apply

# View outputs
terraform output

# Clean up
terraform destroy
```

---

## Testing Resource Creation

### Test Creating a Framework

```hcl
resource "anecdotes_framework" "test" {
  name        = "Terraform Test Framework"
  description = "Created by Terraform provider testing"

  framework_auditable           = false
  can_auditor_download_evidence = true
}

output "created_framework_id" {
  value = anecdotes_framework.test.framework_id
}
```

### Test Creating a Control

```hcl
# First create a control category
resource "anecdotes_control_category" "test_category" {
  framework_id = anecdotes_framework.test.framework_id
  name         = "Test Category"
}

# Then create a control referencing the category
resource "anecdotes_control" "test" {
  framework_id = anecdotes_framework.test.framework_id
  category_id  = anecdotes_control_category.test_category.category_id

  name        = "Test Control"
  description = "Created by Terraform"

  owners = ["your-email@example.com"]
  tags   = ["terraform", "test"]
}

output "created_control_id" {
  value = anecdotes_control.test.control_id
}
```

---

## Development Workflow

### Make Changes → Rebuild → Test

```bash
# 1. Edit source code
vim internal/provider/framework_resource.go

# 2. Rebuild and reinstall
make install

# 3. Test immediately (no terraform init needed with dev_overrides)
cd ~/terraform-anecdotes-test
terraform plan
```

### Enable Debug Logging

```bash
# Set Terraform log level
export TF_LOG=DEBUG

# Or just for the provider
export TF_LOG_PROVIDER=DEBUG

# Run terraform
terraform plan
```

### View Provider Logs

```bash
# Full debug output
TF_LOG=TRACE terraform apply 2>&1 | tee terraform.log

# Filter for provider-specific logs
grep -i anecdotes terraform.log
```

---

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the provider binary |
| `make install` | Build and install to local Terraform plugins |
| `make test` | Run unit tests |
| `make testacc` | Run acceptance tests (requires API key) |
| `make clean` | Remove build artifacts |
| `make fmt` | Format Go code |
| `make lint` | Run linter |

---

## Troubleshooting

### "Provider not found" Error

Make sure your `~/.terraformrc` has the correct path:

```bash
# Check where make install put the binary
ls -la ~/.terraform.d/plugins/registry.terraform.io/anecdotes-ai/anecdotes/
```

### "API key invalid" Error

Verify your API key is set:

```bash
echo $ANECDOTES_API_KEY
```

### Changes Not Taking Effect

After changing the provider code, rebuild and reinstall:

```bash
make install
terraform init -upgrade   # optional, to refresh
terraform plan
```

### Clean Slate

```bash
# Remove local state
rm -rf .terraform terraform.tfstate*

# Rebuild provider
make clean
make install
```

---

## Cleanup

When done testing with the local provider, you can remove or comment out the `filesystem_mirror` block in `~/.terraformrc` so Terraform uses the default registry again (if you publish the provider later).

And destroy any test resources:

```bash
cd ~/terraform-anecdotes-test
terraform destroy
```

