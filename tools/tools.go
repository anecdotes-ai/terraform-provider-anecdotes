//go:build tools

package tools

import (
	// tfplugindocs generates Terraform registry documentation from provider schemas
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
