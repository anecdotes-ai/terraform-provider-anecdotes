// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strings"
	"testing"
)

// The API key is sent on the first request of every session, so a plaintext base
// URL leaks a long-lived credential. Loopback is exempt so a local mock works —
// which is also what the client's own httptest-backed tests rely on.
func TestCheckAPIURL(t *testing.T) {
	cases := []struct {
		url      string
		accepted bool
	}{
		{"https://api.anecdotes.ai", true},
		{"https://api.anecdotes.ai/", true},
		{"https://eu.api.anecdotes.ai", true},
		{"http://api.anecdotes.ai", false},
		{"http://anything.example.com", false},
		{"http://localhost", true},
		{"http://localhost:8080", true},
		{"http://127.0.0.1:1234", true},
		{"http://[::1]:1234", true},
		{"ftp://api.anecdotes.ai", false},
		{"api.anecdotes.ai", false}, // no scheme
	}

	for _, c := range cases {
		summary, detail := checkAPIURL(c.url)
		accepted := summary == ""
		if accepted != c.accepted {
			t.Errorf("%q: expected accepted=%t, got accepted=%t (%s: %s)",
				c.url, c.accepted, accepted, summary, detail)
		}
	}
}

func TestCheckAPIURL_ExplainsTheRisk(t *testing.T) {
	summary, detail := checkAPIURL("http://api.anecdotes.ai")
	if summary != "Insecure API URL" {
		t.Errorf("unexpected summary: %q", summary)
	}
	for _, want := range []string{"clear text", "https"} {
		if !strings.Contains(detail, want) {
			t.Errorf("the message should mention %q, got: %s", want, detail)
		}
	}
}
