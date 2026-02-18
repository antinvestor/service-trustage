package adapters

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// validateExternalURL checks that a URL is safe to call (no SSRF to internal networks).
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

	// Block known internal/metadata hostnames.
	lower := strings.ToLower(hostname)
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
