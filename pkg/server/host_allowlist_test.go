// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"strings"
	"testing"
)

// TestNewHostAllowlist_InvalidPatterns verifies that NewHostAllowlist rejects malformed or disallowed listenHost and allowlist patterns.
func TestNewHostAllowlist_InvalidPatterns(t *testing.T) {
	testCases := []struct {
		name            string
		listenHost      string
		rawAllowedHosts string
	}{
		// Rejects an empty segment between commas in the allowlist string.
		{
			name:            "empty segment in comma list",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "a.example.com,,b.example.com",
		},
		// Rejects allowlist entries that include a URL scheme.
		{
			name:            "scheme prefix",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "http://example.com",
		},
		// Rejects allowlist entries containing a URL path.
		{
			name:            "path suffix",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "example.com/foo",
		},
		// Rejects allowlist entries containing userinfo.
		{
			name:            "userinfo prefix",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "user@example.com",
		},
		// Rejects allowlist entries containing internal whitespace.
		{
			name:            "embedded whitespace",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "example .com",
		},
		// Rejects allowlist entries with a trailing colon and empty port.
		{
			name:            "empty port",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "example.com:",
		},
		// Rejects allowlist entries with TCP port 0.
		{
			name:            "port zero",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "example.com:0",
		},
		// Rejects allowlist entries with TCP port exceeding 65535.
		{
			name:            "port above 65535",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "example.com:65536",
		},
		// Rejects bracketed IPv6 allowlist entries with a non-numeric port.
		{
			name:            "non-numeric port",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "[::1]:abc",
		},
		// Rejects unsupported wildcard patterns lacking a leading *. or *- prefix (including missing separators and non-leftmost wildcards).
		{
			name:            "wildcard without separator",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "*example.com",
		},
		// Rejects patterns containing multiple wildcard segments.
		{
			name:            "multiple wildcards",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "*.*.example.com",
		},
		// Rejects a bare subdomain wildcard with no domain suffix.
		{
			name:            "subdomain wildcard missing domain",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "*.",
		},
		// Rejects a prefix wildcard with no suffix after the dash.
		{
			name:            "prefix wildcard missing suffix",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "*-",
		},
		// Rejects a prefix wildcard whose suffix lacks a domain dot separator.
		{
			name:            "prefix wildcard missing domain dot",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "*-nodot",
		},
		// Rejects a prefix wildcard whose first-label tail contains an invalid underscore character.
		{
			name:            "prefix wildcard invalid character in first-label tail",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "*-bad_label.example.com",
		},
		// Rejects a prefix wildcard whose domain suffix ends with a trailing dot.
		{
			name:            "prefix wildcard trailing dot in domain suffix",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "*-dot.example.com.",
		},
		// Rejects the unspecified IPv4 wildcard address as an allowlist pattern.
		{
			name:            "wildcard IPv4 address",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "0.0.0.0",
		},
		// Rejects the unbracketed unspecified IPv6 wildcard address as an allowlist pattern.
		{
			name:            "wildcard unbracketed IPv6 address",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "::",
		},
		// Rejects the bracketed unspecified IPv6 wildcard address as an allowlist pattern.
		{
			name:            "wildcard bracketed IPv6 address",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "[::]",
		},
		// Rejects host patterns ending with a trailing dot.
		{
			name:            "trailing dot",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "example.com.",
		},
		// Rejects a listenHost that includes a TCP port suffix.
		{
			name:            "IPv4 listenHost with port",
			listenHost:      "10.2.0.2:8080",
			rawAllowedHosts: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allowlist, err := NewHostAllowlist(tc.listenHost, tc.rawAllowedHosts)

			if err == nil {
				t.Fatalf("NewHostAllowlist(%q, %q) = (%v, nil), want non-nil error", tc.listenHost, tc.rawAllowedHosts, allowlist)
			}
		})
	}
}

// TestHostAllowlist_AllowAll verifies AllowAll reports whether the * wildcard disables host filtering.
func TestHostAllowlist_AllowAll(t *testing.T) {
	testCases := []struct {
		name            string
		listenHost      string
		rawAllowedHosts string
		wantAllowAll    bool
	}{
		// Reports false when no extra allowed hosts are configured.
		{
			name:            "empty allowlist",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "",
			wantAllowAll:    false,
		},
		// Reports false when listening on wildcard 0.0.0.0 and the allowlist string contains only whitespace.
		{
			name:            "whitespace-only allowlist",
			listenHost:      "0.0.0.0",
			rawAllowedHosts: " \t ",
			wantAllowAll:    false,
		},
		// Reports false when only explicit host patterns are configured.
		{
			name:            "explicit host patterns",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "khi.example.com,*.proxy.example.com",
			wantAllowAll:    false,
		},
		// Reports true when the wildcard star pattern is configured alone.
		{
			name:            "star wildcard alone",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: "*",
			wantAllowAll:    true,
		},
		// Reports true when the wildcard star pattern appears alongside other entries.
		{
			name:            "star wildcard among other patterns",
			listenHost:      "127.0.0.1",
			rawAllowedHosts: " khi.example.com , * ",
			wantAllowAll:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allowlist, err := NewHostAllowlist(tc.listenHost, tc.rawAllowedHosts)
			if err != nil {
				t.Fatalf("NewHostAllowlist() unexpected error: %v", err)
			}

			got := allowlist.AllowAll()

			if got != tc.wantAllowAll {
				t.Errorf("AllowAll() = %v, want %v", got, tc.wantAllowAll)
			}
		})
	}
}

// TestHostAllowlist_Allows_DefaultAndListenHost verifies default loopback allowance, listenHost matching, absence of hardcoded external domains, and Host syntax validation.
func TestHostAllowlist_Allows_DefaultAndListenHost(t *testing.T) {
	testCases := []struct {
		name            string
		listenHost      string
		rawAllowedHosts string
		rawHost         string
		want            bool
	}{
		// Always allows localhost without a port regardless of wildcard listenHost.
		{
			name:       "allows localhost without port",
			listenHost: "0.0.0.0",
			rawHost:    "localhost",
			want:       true,
		},
		// Allows localhost case-insensitively with a port.
		{
			name:       "allows uppercase localhost with port",
			listenHost: "127.0.0.1",
			rawHost:    "LOCALHOST:9000",
			want:       true,
		},
		// Allows the minimum valid TCP port 1 on a loopback host.
		{
			name:       "allows minimum TCP port 1",
			listenHost: "127.0.0.1",
			rawHost:    "localhost:1",
			want:       true,
		},
		// Allows the maximum valid TCP port 65535 on a loopback host.
		{
			name:       "allows maximum TCP port 65535",
			listenHost: "127.0.0.1",
			rawHost:    "localhost:65535",
			want:       true,
		},
		// Allows 127.0.0.1 without a port.
		{
			name:       "allows IPv4 loopback without port",
			listenHost: "0.0.0.0",
			rawHost:    "127.0.0.1",
			want:       true,
		},
		// Allows any address within the 127.0.0.0/8 loopback range with a port.
		{
			name:       "allows 127.0.0.0/8 address with port",
			listenHost: "127.0.0.1",
			rawHost:    "127.0.1.1:8080",
			want:       true,
		},
		// Allows bracketed IPv6 loopback without a port.
		{
			name:       "allows IPv6 loopback without port",
			listenHost: "::",
			rawHost:    "[::1]",
			want:       true,
		},
		// Allows bracketed IPv6 loopback with a port.
		{
			name:       "allows IPv6 loopback with port",
			listenHost: "127.0.0.1",
			rawHost:    "[::1]:8080",
			want:       true,
		},
		// Allows a non-wildcard IPv4 listenHost without a port.
		{
			name:       "allows non-wildcard IPv4 listenHost without port",
			listenHost: "10.2.0.2",
			rawHost:    "10.2.0.2",
			want:       true,
		},
		// Allows a non-wildcard IPv4 listenHost with a port.
		{
			name:       "allows non-wildcard IPv4 listenHost with port",
			listenHost: "10.2.0.2",
			rawHost:    "10.2.0.2:8080",
			want:       true,
		},
		// Allows a bracketed IPv6 Host header with a port when listenHost is an unbracketed IPv6 address.
		{
			name:       "allows unbracketed IPv6 listenHost via bracketed Host",
			listenHost: "2001:db8::1",
			rawHost:    "[2001:db8::1]:8080",
			want:       true,
		},
		// Rejects a different non-loopback IP address when only listenHost is configured.
		{
			name:       "rejects non-matching IPv4 address",
			listenHost: "10.2.0.2",
			rawHost:    "10.2.0.3",
			want:       false,
		},
		// Allows a non-wildcard DNS listenHost case-insensitively when rawAllowedHosts is whitespace-only.
		{
			name:            "allows non-wildcard DNS listenHost with whitespace allowlist",
			listenHost:      "khi.internal",
			rawAllowedHosts: "  \t ",
			rawHost:         "KHI.INTERNAL:8080",
			want:            true,
		},
		// Rejects wildcard listenHost 0.0.0.0 when sent as a Host header.
		{
			name:       "rejects 0.0.0.0 even when listening on 0.0.0.0",
			listenHost: "0.0.0.0",
			rawHost:    "0.0.0.0:8080",
			want:       false,
		},
		// Rejects wildcard listenHost [::] when sent as a Host header.
		{
			name:       "rejects [::] even when listening on ::",
			listenHost: "::",
			rawHost:    "[::]:8080",
			want:       false,
		},
		// Rejects non-loopback hosts when listenHost is empty.
		{
			name:       "empty listenHost does not allow arbitrary host",
			listenHost: "",
			rawHost:    "10.2.0.2",
			want:       false,
		},
		// Rejects non-loopback hosts when listenHost is bracketed wildcard [::].
		{
			name:       "bracketed wildcard listenHost rejects [::]",
			listenHost: "[::]",
			rawHost:    "[::]",
			want:       false,
		},
		// Rejects external domains by default without explicit allowlist configuration.
		{
			name:       "does not allow unlisted example.com domain",
			listenHost: "127.0.0.1",
			rawHost:    "khi.example.com",
			want:       false,
		},
		// Rejects proxy domains by default without explicit allowlist configuration.
		{
			name:       "does not allow unlisted proxy domain",
			listenHost: "127.0.0.1",
			rawHost:    "app.proxy.example.com",
			want:       false,
		},
		// Rejects cloudshell.dev domains by default because Cloud Shell rewrites Host to loopback.
		{
			name:       "does not hardcode cloudshell.dev domain",
			listenHost: "127.0.0.1",
			rawHost:    "8080-cs-test.cloudshell.dev",
			want:       false,
		},
		// Rejects arbitrary external domains not present in the allowlist.
		{
			name:       "rejects unlisted external domain",
			listenHost: "127.0.0.1",
			rawHost:    "unlisted.example.net",
			want:       false,
		},
		// Rejects domains that use localhost only as a leftmost label of another domain.
		{
			name:       "rejects localhost label under another domain",
			listenHost: "127.0.0.1",
			rawHost:    "localhost.example.net",
			want:       false,
		},
		// Rejects domains embedding 127.0.0.1 as subdomains.
		{
			name:       "rejects 127.0.0.1 subdomain under external domain",
			listenHost: "127.0.0.1",
			rawHost:    "127.0.0.1.nip.io",
			want:       false,
		},
		// Rejects an empty Host header.
		{
			name:       "rejects empty host",
			listenHost: "127.0.0.1",
			rawHost:    "",
			want:       false,
		},
		// Rejects an unbracketed wildcard IPv6 Host header.
		{
			name:       "rejects unbracketed ::",
			listenHost: "127.0.0.1",
			rawHost:    "::",
			want:       false,
		},
		// Rejects Host headers containing a path separator.
		{
			name:       "rejects path in host",
			listenHost: "127.0.0.1",
			rawHost:    "localhost/foo",
			want:       false,
		},
		// Rejects Host headers with a non-numeric port.
		{
			name:       "rejects non-numeric port in host",
			listenHost: "127.0.0.1",
			rawHost:    "localhost:abc",
			want:       false,
		},
		// Rejects Host headers ending with a trailing dot.
		{
			name:       "rejects trailing dot on localhost",
			listenHost: "127.0.0.1",
			rawHost:    "localhost.",
			want:       false,
		},
		// Rejects IPv4-mapped IPv6 loopback addresses to keep IPv6 validation strict and symmetric.
		{
			name:       "rejects IPv4-mapped IPv6 loopback",
			listenHost: "127.0.0.1",
			rawHost:    "[::ffff:127.0.0.1]",
			want:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allowlist, err := NewHostAllowlist(tc.listenHost, tc.rawAllowedHosts)
			if err != nil {
				t.Fatalf("NewHostAllowlist() unexpected error: %v", err)
			}

			got := allowlist.Allows(tc.rawHost)

			if got != tc.want {
				t.Errorf("Allows(%q) = %v, want %v", tc.rawHost, got, tc.want)
			}
		})
	}
}

// TestHostAllowlist_Allows_Patterns verifies exact host, exact host:port, subdomain wildcard, prefix wildcard, *, and malformed bracketed IPv6 matching semantics.
func TestHostAllowlist_Allows_Patterns(t *testing.T) {
	testCases := []struct {
		name            string
		rawAllowedHosts string
		rawHost         string
		want            bool
	}{
		// Allows an exact DNS host pattern case-insensitively after trimming surrounding spaces.
		{
			name:            "exact DNS host matches case-insensitively without port",
			rawAllowedHosts: " khi.example.com ",
			rawHost:         "KHI.EXAMPLE.COM",
			want:            true,
		},
		// Allows an exact DNS host pattern with any valid TCP port when no port is pinned.
		{
			name:            "exact DNS host matches with port",
			rawAllowedHosts: "khi.example.com",
			rawHost:         "khi.example.com:8443",
			want:            true,
		},
		// Rejects subdomains when only the exact parent host is allowlisted.
		{
			name:            "exact DNS host rejects subdomain",
			rawAllowedHosts: "khi.example.com",
			rawHost:         "sub.khi.example.com",
			want:            false,
		},
		// Allows an exact IPv4 address pattern with any valid TCP port when no port is pinned.
		{
			name:            "exact IPv4 host allows port",
			rawAllowedHosts: "10.2.0.2",
			rawHost:         "10.2.0.2:8080",
			want:            true,
		},
		// Allows a bracketed IPv6 address pattern case-insensitively with a port.
		{
			name:            "exact IPv6 host matches case-insensitively with port",
			rawAllowedHosts: "[2001:db8::1]",
			rawHost:         "[2001:DB8::1]:8443",
			want:            true,
		},
		// Allows an exact IPv4 host:port pattern only when both IP and port match.
		{
			name:            "exact IPv4 host:port matches exact port",
			rawAllowedHosts: "10.2.0.2:443",
			rawHost:         "10.2.0.2:443",
			want:            true,
		},
		// Rejects an omitted port when the allowlist pattern pins a specific port.
		{
			name:            "exact IPv4 host:port rejects omitted port",
			rawAllowedHosts: "10.2.0.2:443",
			rawHost:         "10.2.0.2",
			want:            false,
		},
		// Rejects a mismatched port when the allowlist pattern pins a specific port.
		{
			name:            "exact IPv4 host:port rejects different port",
			rawAllowedHosts: "10.2.0.2:443",
			rawHost:         "10.2.0.2:8080",
			want:            false,
		},
		// Allows an exact IPv6 host:port pattern when both address and port match.
		{
			name:            "exact IPv6 host:port matches exact port",
			rawAllowedHosts: "[2001:db8::1]:8080",
			rawHost:         "[2001:DB8::1]:8080",
			want:            true,
		},
		// Rejects a mismatched port for an exact IPv6 host:port pattern.
		{
			name:            "exact IPv6 host:port rejects different port",
			rawAllowedHosts: "[2001:db8::1]:8080",
			rawHost:         "[2001:db8::1]:443",
			want:            false,
		},
		// Rejects omitted port for an exact IPv6 host:port pattern.
		{
			name:            "exact IPv6 host:port rejects omitted port",
			rawAllowedHosts: "[2001:db8::1]:8080",
			rawHost:         "[2001:db8::1]",
			want:            false,
		},
		// Allows a single subdomain label matching *.proxy.example.com.
		{
			name:            "subdomain wildcard matches single label",
			rawAllowedHosts: "*.proxy.example.com,*.dev.example.com",
			rawHost:         "app.proxy.example.com",
			want:            true,
		},
		// Allows multiple subdomain labels and a port matching *.dev.example.com case-insensitively.
		{
			name:            "subdomain wildcard matches multi-label host case-insensitively with port",
			rawAllowedHosts: "*.proxy.example.com,*.dev.example.com",
			rawHost:         "a.b.DEV.EXAMPLE.COM:8080",
			want:            true,
		},
		// Rejects the bare domain without a leading subdomain label for *.domain patterns.
		{
			name:            "subdomain wildcard rejects bare domain",
			rawAllowedHosts: "*.proxy.example.com",
			rawHost:         "proxy.example.com",
			want:            false,
		},
		// Rejects domains that concatenate a prefix without a dot separator before the allowlisted domain.
		{
			name:            "subdomain wildcard rejects missing-dot suffix mismatch",
			rawAllowedHosts: "*.proxy.example.com",
			rawHost:         "otherproxy.example.com",
			want:            false,
		},
		// Rejects domains that append an extra suffix after the allowlisted domain.
		{
			name:            "subdomain wildcard rejects trailing domain suffix",
			rawAllowedHosts: "*.proxy.example.com",
			rawHost:         "app.proxy.example.com.other.net",
			want:            false,
		},
		// Rejects an IPv4 literal Host header even when its octets match a numeric subdomain wildcard suffix.
		{
			name:            "IPv4 literal does not match numeric subdomain wildcard",
			rawAllowedHosts: "*.2.0.2",
			rawHost:         "10.2.0.2",
			want:            false,
		},
		// Allows a valid single-label prefix matching the second entry (*-dot-app.example.com) in a comma-separated allowlist.
		{
			name:            "prefix wildcard matches single label prefix",
			rawAllowedHosts: "*-dot-khi.example.com,*-dot-app.example.com",
			rawHost:         "beta-dot-app.example.com",
			want:            true,
		},
		// Allows hyphenated leftmost-label prefixes case-insensitively with a port.
		{
			name:            "prefix wildcard matches hyphenated prefix case-insensitively with port",
			rawAllowedHosts: "*-dot-khi.example.com",
			rawHost:         "V1-dot-default-dot-khi.example.com:443",
			want:            true,
		},
		// Rejects an empty wildcard prefix before the dash in *-suffix patterns.
		{
			name:            "prefix wildcard rejects empty prefix before dash",
			rawAllowedHosts: "*-dot-app.example.com",
			rawHost:         "-dot-app.example.com",
			want:            false,
		},
		// Rejects extra dot-separated subdomain labels within the * of *-suffix patterns.
		{
			name:            "prefix wildcard rejects dot inside wildcard",
			rawAllowedHosts: "*-dot-app.example.com",
			rawHost:         "sub.beta-dot-app.example.com",
			want:            false,
		},
		// Rejects extra domain suffixes appended after a prefix-wildcard domain.
		{
			name:            "prefix wildcard rejects trailing domain suffix",
			rawAllowedHosts: "*-dot-app.example.com",
			rawHost:         "beta-dot-app.example.com.other.net",
			want:            false,
		},
		// Allows arbitrary syntactically valid hosts when * disables host filtering.
		{
			name:            "star wildcard allows valid arbitrary host",
			rawAllowedHosts: "*",
			rawHost:         "arbitrary.example.com:8080",
			want:            true,
		},
		// Still rejects empty Host headers even when * disables host filtering.
		{
			name:            "star wildcard rejects empty host",
			rawAllowedHosts: "*",
			rawHost:         "",
			want:            false,
		},
		// Still rejects 0.0.0.0 wildcard IP hosts even when * disables host filtering.
		{
			name:            "star wildcard rejects 0.0.0.0",
			rawAllowedHosts: "*",
			rawHost:         "0.0.0.0:8080",
			want:            false,
		},
		// Still rejects [::] wildcard IPv6 hosts even when * disables host filtering.
		{
			name:            "star wildcard rejects [::]",
			rawAllowedHosts: "*",
			rawHost:         "[::]",
			want:            false,
		},
		// Still rejects IPv4-mapped unspecified IPv6 hosts even when * disables host filtering.
		{
			name:            "star wildcard rejects IPv4-mapped unspecified IPv6",
			rawAllowedHosts: "*",
			rawHost:         "[::ffff:0.0.0.0]",
			want:            false,
		},
		// Still rejects malformed Host syntax with paths even when * disables host filtering.
		{
			name:            "star wildcard rejects path in host",
			rawAllowedHosts: "*",
			rawHost:         "example.net/path",
			want:            false,
		},
		// Rejects a bracketed IPv6 host missing its closing bracket.
		{
			name:            "bracketed IPv6 missing closing bracket",
			rawAllowedHosts: "",
			rawHost:         "[::1",
			want:            false,
		},
		// Rejects a bracketed IPv6 host with non-colon characters after the closing bracket.
		{
			name:            "bracketed IPv6 missing colon before port",
			rawAllowedHosts: "",
			rawHost:         "[::1]8080",
			want:            false,
		},
		// Rejects a bracketed IPv4 loopback address because brackets are only valid for IPv6 literals.
		{
			name:            "bracketed IPv4 loopback",
			rawAllowedHosts: "",
			rawHost:         "[127.0.0.1]",
			want:            false,
		},
		// Rejects a bracketed IPv6 address containing a zone identifier even when wildcard '*' is enabled.
		{
			name:            "bracketed IPv6 with zone ID",
			rawAllowedHosts: "*",
			rawHost:         "[fe80::1%eth0]",
			want:            false,
		},
		// Rejects a bracketed non-IP string even when wildcard '*' is enabled.
		{
			name:            "bracketed non-IP string",
			rawAllowedHosts: "*",
			rawHost:         "[not-an-ip]",
			want:            false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allowlist, err := NewHostAllowlist("127.0.0.1", tc.rawAllowedHosts)
			if err != nil {
				t.Fatalf("NewHostAllowlist() unexpected error: %v", err)
			}

			got := allowlist.Allows(tc.rawHost)

			if got != tc.want {
				t.Errorf("Allows(%q) = %v, want %v", tc.rawHost, got, tc.want)
			}
		})
	}
}

// TestIsValidDNSLabel verifies length bounds, character classes, and hyphen placement rules for DNS labels.
func TestIsValidDNSLabel(t *testing.T) {
	testCases := []struct {
		name  string
		label string
		want  bool
	}{
		// Rejects an empty DNS label.
		{
			name:  "empty label",
			label: "",
			want:  false,
		},
		// Rejects a DNS label exceeding the 63-byte RFC maximum.
		{
			name:  "label exceeding 63 bytes",
			label: strings.Repeat("a", 64),
			want:  false,
		},
		// Accepts a DNS label at the 63-byte RFC maximum.
		{
			name:  "63-byte maximum valid label",
			label: strings.Repeat("a", 63),
			want:  true,
		},
		// Accepts a DNS label containing mixed-case ASCII letters, digits, and an interior hyphen.
		{
			name:  "mixed-case alphanumeric bounds with interior hyphen",
			label: "a-zAZ09",
			want:  true,
		},
		// Rejects a DNS label containing a non-alphanumeric underscore character.
		{
			name:  "non-alphanumeric underscore",
			label: "a_b",
			want:  false,
		},
		// Rejects a DNS label starting with a hyphen.
		{
			name:  "leading hyphen",
			label: "-a",
			want:  false,
		},
		// Rejects a DNS label ending with a hyphen.
		{
			name:  "trailing hyphen",
			label: "a-",
			want:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isValidDNSLabel(tc.label)

			if got != tc.want {
				t.Errorf("isValidDNSLabel(%q) = %v, want %v", tc.label, got, tc.want)
			}
		})
	}
}

// TestIsValidDomain verifies the 253-byte RFC maximum length boundary for domain names.
func TestIsValidDomain(t *testing.T) {
	testCases := []struct {
		name   string
		domain string
		want   bool
	}{
		// Accepts a domain whose total length equals the 253-byte RFC maximum with valid labels.
		{
			name:   "253-byte maximum valid domain",
			domain: strings.Repeat("a", 63) + "." + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 61),
			want:   true,
		},
		// Rejects a domain whose total length exceeds 253 bytes even when every label is at most 63 bytes.
		{
			name:   "domain exceeding 253 bytes",
			domain: strings.Repeat("a", 63) + "." + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 62),
			want:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isValidDomain(tc.domain)

			if got != tc.want {
				t.Errorf("isValidDomain(%d bytes) = %v, want %v", len(tc.domain), got, tc.want)
			}
		})
	}
}

// TestParseTCPPort_RejectsSignPrefix verifies that parseTCPPort rejects a leading '+' sign prefix on numeric ports.
func TestParseTCPPort_RejectsSignPrefix(t *testing.T) {
	testCases := []struct {
		name    string
		rawPort string
	}{
		// Rejects a TCP port prefixed with a plus sign.
		{
			name:    "plus-prefixed port",
			rawPort: "+8080",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			port, err := parseTCPPort(tc.rawPort)

			if err == nil {
				t.Fatalf("parseTCPPort(%q) = (%d, nil), want non-nil error", tc.rawPort, port)
			}
		})
	}
}
