// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestNoRawAPIClientErrors guards the error-handling contract: any failure from an
// Anecdotes API call must be surfaced via addClientError (which decodes, classifies,
// and redacts the message), never via a raw resp.Diagnostics.AddError that embeds the
// error directly. Raw AddError is still fine for *local* failures (JSON/ID parsing,
// file reads, model mapping, client construction) — those are listed in localErrAllow.
//
// The test scans the provider package source for AddError(...) calls that reference an
// error value and fails on any that is not in the allowlist. If you add a new local
// (non-API) error site, add its summary here with a justification; if it reports an API
// call, use addClientError instead.
func TestNoRawAPIClientErrors(t *testing.T) {
	// Summaries of known LOCAL (non-API) raw AddError sites. Each reports a failure that
	// did not come from a client/API call, so structured API decoding does not apply.
	localErrAllow := map[string]string{
		"Invalid rule_query":                    "json.Unmarshal of user rule_query",
		"Invalid data JSON":                     "json.Unmarshal of user data attribute",
		"Invalid filter_aql":                    "json.Unmarshal of user filter_aql",
		"File Read Error":                       "os.ReadFile of a local file",
		"File Error":                            "os.ReadFile of a local file",
		"Mapping Error":                         "local model->state mapping",
		"Error generating folder ID":            "uuid.GenerateUUID (local)",
		"Configuration Error":                   "local request building (buildCreateRequest)",
		"Unable to Create Anecdotes API Client": "client construction, before any API call",
	}

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	errToken := regexp.MustCompile(`\berr\b`)
	summaryRe := regexp.MustCompile(`"((?:[^"\\]|\\.)*)"`)

	var violations []string
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, rerr := os.ReadFile(f)
		if rerr != nil {
			t.Fatalf("read %s: %v", f, rerr)
		}
		for _, call := range addErrorCalls(string(src)) {
			if !errToken.MatchString(call.args) {
				continue // not reporting an error value
			}
			if strings.Contains(strings.ToLower(call.args), "configure type") {
				continue // framework wiring (%T of ProviderData), not an API error
			}
			m := summaryRe.FindStringSubmatch(call.args)
			summary := ""
			if m != nil {
				summary = m[1]
			}
			if _, ok := localErrAllow[summary]; ok {
				continue
			}
			violations = append(violations, f+": AddError("+summary+") passes an error directly — use addClientError, or allowlist it if it is a local (non-API) error.")
		}
	}

	if len(violations) > 0 {
		t.Fatalf("found %d raw API-error AddError site(s):\n  %s", len(violations), strings.Join(violations, "\n  "))
	}
}

type addErrorCall struct{ args string }

// addErrorCalls returns the argument text of every AddError(...) call in src, balancing
// parentheses so multi-line calls are captured whole.
func addErrorCalls(src string) []addErrorCall {
	var out []addErrorCall
	const marker = "AddError("
	for i := 0; ; {
		idx := strings.Index(src[i:], marker)
		if idx < 0 {
			break
		}
		start := i + idx + len(marker)
		depth := 1
		j := start
		for ; j < len(src) && depth > 0; j++ {
			switch src[j] {
			case '(':
				depth++
			case ')':
				depth--
			}
		}
		out = append(out, addErrorCall{args: src[start : j-1]})
		i = j
	}
	return out
}
