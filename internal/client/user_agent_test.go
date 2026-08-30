// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

func TestUserAgent_IncludesProviderTerraformAndRuntime(t *testing.T) {
	ua := UserAgent("1.2.3", "1.9.0")

	want := fmt.Sprintf("terraform-provider-anecdotes/1.2.3 (+%s) Terraform/1.9.0 %s %s/%s",
		providerRepoURL, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	if ua != want {
		t.Errorf("UserAgent(\"1.2.3\", \"1.9.0\") = %q, want %q", ua, want)
	}
}

// A direct client user (not going through the provider's Configure, which is
// the only caller that has a Terraform version to report) should get a valid
// header with the Terraform segment cleanly omitted, not a dangling
// "Terraform/" with nothing after it.
func TestUserAgent_OmitsTerraformSegmentWhenVersionUnknown(t *testing.T) {
	ua := UserAgent("1.2.3", "")

	if strings.Contains(ua, "Terraform/") {
		t.Errorf("UserAgent with no Terraform version should omit the segment entirely, got %q", ua)
	}
	want := fmt.Sprintf("terraform-provider-anecdotes/1.2.3 (+%s) %s %s/%s",
		providerRepoURL, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	if ua != want {
		t.Errorf("UserAgent(\"1.2.3\", \"\") = %q, want %q", ua, want)
	}
}

// The API key exchange and every ordinary API call must carry the
// User-Agent the caller configured — this is the only thing that lets the
// platform tell provider traffic apart from any other API caller.
func TestClient_SendsConfiguredUserAgent(t *testing.T) {
	const wantUA = "terraform-provider-anecdotes/9.9.9-test"

	var exchangeUA, requestUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/apikey/exchange") {
			exchangeUA = r.Header.Get("User-Agent")
			_, _ = w.Write([]byte("test-token"))
			return
		}
		requestUA = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c, err := NewAnecdotesClient(context.Background(), "test-key", srv.URL, wantUA)
	if err != nil {
		t.Fatalf("NewAnecdotesClient: %v", err)
	}
	if exchangeUA != wantUA {
		t.Errorf("token exchange User-Agent = %q, want %q", exchangeUA, wantUA)
	}

	if _, err := c.ListFrameworks(context.Background()); err != nil {
		t.Fatalf("ListFrameworks: %v", err)
	}
	if requestUA != wantUA {
		t.Errorf("API request User-Agent = %q, want %q", requestUA, wantUA)
	}
}
