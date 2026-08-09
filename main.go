// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Generate the provider documentation. The directive lives in the root package so
// `go generate ./...` runs tfplugindocs from the repository root, where examples/,
// templates/, and the docs/ output directory live. `go tool` runs the version
// recorded by the tool directive in go.mod.
//go:generate go tool tfplugindocs generate --provider-name anecdotes

// Provider version - will be set during build
var version = "dev"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/anecdotes-ai/anecdotes",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
