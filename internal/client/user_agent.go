// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"fmt"
	"runtime"
)

// providerRepoURL identifies this provider in its own User-Agent string.
const providerRepoURL = "https://github.com/anecdotes-ai/terraform-provider-anecdotes"

// UserAgent builds the value sent as the User-Agent header on every API
// request. It carries no credential or customer-identifying data — the API
// already derives the caller's identity from the bearer token — only what
// helps the platform distinguish Terraform-provider traffic from other API
// callers and correlate a support report to the exact provider, Terraform,
// and Go build that produced it.
//
// terraformVersion is req.TerraformVersion from the provider's Configure
// call, supplied by Terraform Core precisely for this purpose; an empty
// value (only possible if Configure is never called, e.g. a direct API
// client use) omits the segment rather than printing it blank.
func UserAgent(providerVersion, terraformVersion string) string {
	ua := fmt.Sprintf("terraform-provider-anecdotes/%s (+%s)", providerVersion, providerRepoURL)
	if terraformVersion != "" {
		ua += " Terraform/" + terraformVersion
	}
	ua += fmt.Sprintf(" %s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	return ua
}
