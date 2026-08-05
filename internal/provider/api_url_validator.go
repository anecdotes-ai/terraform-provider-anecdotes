// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// httpsURLValidator requires a URL to use HTTPS. The API key is sent on the
// first request of every session, so a plaintext base URL puts a long-lived
// credential on the wire; a redirect to HTTPS afterwards does not undo that.
// Loopback addresses are allowed so a local mock can be used in development.
type httpsURLValidator struct{}

func requireHTTPSURL() validator.String {
	return httpsURLValidator{}
}

func (v httpsURLValidator) Description(ctx context.Context) string {
	return "must be an https URL, or an http URL on localhost (for use with a local mock)"
}

func (v httpsURLValidator) MarkdownDescription(ctx context.Context) string {
	return "must be an `https` URL, or an `http` URL on `localhost` (for use with a local mock)"
}

func (v httpsURLValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}

	if summary, detail := checkAPIURL(value); summary != "" {
		resp.Diagnostics.AddAttributeError(req.Path, summary, detail)
	}
}

// checkAPIURL applies the scheme rule to a base URL from any source — the
// provider configuration or the environment variable, which schema validation
// never sees. An empty summary means the URL is acceptable.
func checkAPIURL(value string) (summary, detail string) {
	parsed, err := url.Parse(value)
	if err != nil {
		return "Invalid API URL", fmt.Sprintf("%q could not be parsed as a URL: %s", value, err)
	}

	switch parsed.Scheme {
	case "https":
		return "", ""
	case "http":
		if isLoopbackHost(parsed.Hostname()) {
			return "", ""
		}
		return "Insecure API URL", fmt.Sprintf(
			"%q uses http, which would send the Anecdotes API key over the network in clear text on "+
				"every session. Use https instead. Plain http is accepted only for localhost, for use "+
				"with a local mock.", value)
	default:
		return "Invalid API URL", fmt.Sprintf("%q must use https, but uses %q.", value, parsed.Scheme)
	}
}

// isLoopbackHost reports whether host refers to the local machine.
func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// Compile-time check that the validator satisfies the interface.
var _ validator.String = httpsURLValidator{}
