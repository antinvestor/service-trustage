// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//nolint:testpackage // package-local: exercises unexported SSRF helpers.
package adapters

import (
	"context"
	"strings"
	"testing"
)

func TestValidateExternalURL_TrustedClusterSuffixesBypassPrivateIPCheck(t *testing.T) {
	t.Parallel()

	// Default suffixes (".svc" + ".svc.cluster.local") cover the canonical
	// Kubernetes Service DNS forms. These resolve to ClusterIPs in
	// private ranges; the SSRF guard must let them through so workflows
	// can target sibling in-cluster services.
	cases := []string{
		"http://opportunities-crawler.product-opportunities.svc/admin/scheduler/tick",
		"http://opportunities-crawler.product-opportunities.svc.cluster.local/admin/scheduler/tick",
		"https://service-api.gateway.svc:8443/healthz",
	}
	for _, raw := range cases {
		if err := validateExternalURL(context.Background(), raw); err != nil {
			t.Fatalf("validateExternalURL(%q) returned %v; want nil for cluster DNS", raw, err)
		}
	}
}

func TestValidateExternalURL_LocalSuffixStillBlocked(t *testing.T) {
	t.Parallel()

	// .local (mDNS) and .internal (cloud metadata) remain blocked. The
	// trusted-suffix exemption is only for entries that look like real
	// Kubernetes Service DNS.
	cases := []string{
		"http://router.local/api",
		"http://metadata.google.internal/computeMetadata/v1/",
		"http://other.internal/x",
	}
	for _, raw := range cases {
		err := validateExternalURL(context.Background(), raw)
		if err == nil {
			t.Fatalf("validateExternalURL(%q) returned nil; expected internal-hostname block", raw)
		}
		if !strings.Contains(err.Error(), "internal hostname") {
			t.Fatalf("validateExternalURL(%q) error = %q; want internal-hostname error", raw, err)
		}
	}
}

func TestTrustedHostSuffixes_FromEnv(t *testing.T) {
	// Not t.Parallel(): mutates process env.
	t.Setenv(trustedURLHostSuffixesEnv, "svc.k8s.example.com, internal.example.com")

	got := trustedHostSuffixes()
	want := []string{".svc.k8s.example.com", ".internal.example.com"}
	if len(got) != len(want) {
		t.Fatalf("trustedHostSuffixes() len = %d; want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("trustedHostSuffixes()[%d] = %q; want %q", i, got[i], want[i])
		}
	}

	// Whitespace-only env falls back to the defaults.
	t.Setenv(trustedURLHostSuffixesEnv, "   ")
	defaults := trustedHostSuffixes()
	if len(defaults) != len(defaultTrustedHostSuffixes) {
		t.Fatalf("blank env: got %d suffixes, want %d", len(defaults), len(defaultTrustedHostSuffixes))
	}
}

func TestIsTrustedHostSuffix_BoundaryMatchOnly(t *testing.T) {
	t.Parallel()

	// ".svc" must NOT match "example.svc.example.com" because the latter
	// has ".com" as its rightmost label; matching there would let an
	// attacker rebind external DNS to internal IPs via the suffix.
	if isTrustedHostSuffix("opportunities-crawler.product-opportunities.svc") != true {
		t.Fatal("expected canonical Service DNS to match")
	}
	if isTrustedHostSuffix("svc") != true {
		t.Fatal("expected bare 'svc' (the literal suffix) to match")
	}
	if isTrustedHostSuffix("example.svc.example.com") {
		t.Fatal("must not match a hostname where .svc is mid-path")
	}
	if isTrustedHostSuffix("opportunities-crawler.product-opportunities.svc.cluster.local") != true {
		t.Fatal("expected FQDN Service DNS to match")
	}
}
