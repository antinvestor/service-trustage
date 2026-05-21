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

package adapters

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

// trustedURLHostSuffixesEnv lists hostname suffixes (comma-separated) whose
// resolved IPs are exempted from the private-IP SSRF check. Defaults cover
// Kubernetes' in-cluster Service DNS, which intentionally resolves to
// private ClusterIPs and is the canonical way services call each other.
//
// Operators with non-default cluster domains, sidecar meshes, or
// authenticated internal registries can extend the list via env. Suffixes
// are matched case-insensitively against the *hostname* (no scheme/port);
// the leading dot is optional and added automatically if missing.
const trustedURLHostSuffixesEnv = "TRUSTAGE_URL_TRUSTED_HOST_SUFFIXES"

// defaultTrustedHostSuffixes is consulted when the env var is unset or
// empty. ".svc" + ".svc.cluster.local" together cover both short and FQDN
// forms produced by Kubernetes service discovery.
var defaultTrustedHostSuffixes = []string{".svc", ".svc.cluster.local"} //nolint:gochecknoglobals // immutable default list

// validateExternalURL checks that a URL is safe to call (no SSRF to internal networks).
//
// Hostnames matching a trusted suffix (see trustedURLHostSuffixesEnv) skip
// the private-IP check because in those cases the private address comes
// from the operator's intentional cluster topology, not user-supplied
// data steering us to an internal target.
func validateExternalURL(ctx context.Context, rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Require HTTPS or HTTP scheme.
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && scheme != "http" {
		return fmt.Errorf("unsupported scheme %q: only http and https are allowed", parsed.Scheme)
	}

	// Extract hostname (without port).
	hostname := parsed.Hostname()
	if hostname == "" {
		return errors.New("URL must have a hostname")
	}

	lower := strings.ToLower(hostname)

	// Trusted in-cluster DNS suffix → skip both the .local block below and
	// the IP-range check. ".svc.cluster.local" would otherwise be caught
	// by the ".local" suffix rule.
	if isTrustedHostSuffix(lower) {
		return nil
	}

	// Block known internal/metadata hostnames.
	if lower == "localhost" || lower == "metadata.google.internal" ||
		strings.HasSuffix(lower, ".internal") || strings.HasSuffix(lower, ".local") {
		return fmt.Errorf("URL hostname %q is not allowed: internal hostname", hostname)
	}

	// Resolve the hostname and check for private IPs.
	ips, lookupErr := net.DefaultResolver.LookupHost(ctx, hostname)
	if lookupErr != nil {
		// If DNS resolution fails, allow the request to proceed.
		// The HTTP client will fail with a more descriptive error.
		return nil //nolint:nilerr // intentional: DNS failure should not block the request
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		if isPrivateIP(ip) {
			return fmt.Errorf("URL hostname %q resolves to private IP %s", hostname, ipStr)
		}
	}

	return nil
}

// isTrustedHostSuffix reports whether the lowercased hostname matches one
// of the configured trusted suffixes. Suffix matching is exact on the
// dotted boundary so "example.svc.example.com" doesn't match ".svc".
func isTrustedHostSuffix(lowerHost string) bool {
	for _, suffix := range trustedHostSuffixes() {
		if lowerHost == strings.TrimPrefix(suffix, ".") {
			return true
		}
		if strings.HasSuffix(lowerHost, suffix) {
			return true
		}
	}
	return false
}

// trustedHostSuffixes returns the active suffix list, sourcing from env
// when set and falling back to defaultTrustedHostSuffixes otherwise.
// Suffixes are normalised to lower-case with a leading dot.
func trustedHostSuffixes() []string {
	raw := strings.TrimSpace(os.Getenv(trustedURLHostSuffixesEnv))
	if raw == "" {
		return defaultTrustedHostSuffixes
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.ToLower(strings.TrimSpace(p))
		if s == "" {
			continue
		}
		if !strings.HasPrefix(s, ".") {
			s = "." + s
		}
		out = append(out, s)
	}
	if len(out) == 0 {
		return defaultTrustedHostSuffixes
	}
	return out
}

// isPrivateIP returns true if the IP address is in a private/reserved range.
func isPrivateIP(ip net.IP) bool {
	privateRanges := []struct {
		network *net.IPNet
	}{
		{mustParseCIDR("10.0.0.0/8")},
		{mustParseCIDR("172.16.0.0/12")},
		{mustParseCIDR("192.168.0.0/16")},
		{mustParseCIDR("127.0.0.0/8")},
		{mustParseCIDR("169.254.0.0/16")}, // Link-local.
		{mustParseCIDR("100.64.0.0/10")},  // Carrier-grade NAT.
		{mustParseCIDR("::1/128")},        // IPv6 loopback.
		{mustParseCIDR("fc00::/7")},       // IPv6 unique local.
		{mustParseCIDR("fe80::/10")},      // IPv6 link-local.
		{mustParseCIDR("fd00::/8")},       // IPv6 unique local.
		{mustParseCIDR("0.0.0.0/8")},      // This network.
		{mustParseCIDR("198.18.0.0/15")},  // Benchmark testing.
		{mustParseCIDR("224.0.0.0/4")},    // Multicast.
		{mustParseCIDR("240.0.0.0/4")},    // Reserved.
	}

	for _, r := range privateRanges {
		if r.network.Contains(ip) {
			return true
		}
	}

	return false
}

func mustParseCIDR(s string) *net.IPNet {
	_, network, err := net.ParseCIDR(s)
	if err != nil {
		panic(fmt.Sprintf("invalid CIDR %q: %v", s, err))
	}

	return network
}
