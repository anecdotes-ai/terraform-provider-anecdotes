// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

//go:build tools

package tools

import (
	// tfplugindocs generates Terraform registry documentation from provider schemas
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
