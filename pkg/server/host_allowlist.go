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
	"fmt"
	"net/netip"
	"strconv"
	"strings"
)

// HostAllowlist validates HTTP Host headers against loopback, listen-host, and configured allowlist patterns.
type HostAllowlist struct {
	allowAll           bool
	exactHosts         map[string]struct{}
	exactHostPorts     map[hostPortKey]struct{}
	subdomainWildcards []string
	prefixWildcards    []string
}

type hostPortKey struct {
	host string
	port uint16
}

type parsedHost struct {
	host    string
	addr    netip.Addr
	isIP    bool
	port    uint16
	hasPort bool
}

// NewHostAllowlist constructs a HostAllowlist from the server listenHost and comma-separated rawAllowedHosts patterns.
func NewHostAllowlist(listenHost string, rawAllowedHosts string) (*HostAllowlist, error) {
	allowlist := &HostAllowlist{
		exactHosts:     make(map[string]struct{}),
		exactHostPorts: make(map[hostPortKey]struct{}),
	}

	if err := allowlist.addListenHost(listenHost); err != nil {
		return nil, err
	}

	if strings.TrimSpace(rawAllowedHosts) == "" {
		return allowlist, nil
	}

	for _, segment := range strings.Split(rawAllowedHosts, ",") {
		pattern := strings.TrimSpace(segment)
		if pattern == "" {
			return nil, fmt.Errorf("empty host pattern in allowed-hosts list")
		}
		if err := allowlist.addPattern(pattern); err != nil {
			return nil, err
		}
	}

	return allowlist, nil
}

// AllowAll reports whether the wildcard '*' pattern is configured to permit any syntactically valid Host header.
func (a *HostAllowlist) AllowAll() bool {
	return a.allowAll
}

// Allows reports whether rawHost is syntactically valid and permitted by this allowlist.
func (a *HostAllowlist) Allows(rawHost string) bool {
	parsed, err := parseHostAndPort(rawHost)
	if err != nil {
		return false
	}

	if a.allowAll {
		return true
	}

	if parsed.host == "localhost" || (parsed.isIP && parsed.addr.IsLoopback()) {
		return true
	}

	if _, ok := a.exactHosts[parsed.host]; ok {
		return true
	}

	if _, ok := a.exactHostPorts[hostPortKey{host: parsed.host, port: parsed.port}]; ok {
		return true
	}

	if parsed.isIP {
		return false
	}

	return a.matchesWildcardDomain(parsed.host)
}

func (a *HostAllowlist) addListenHost(listenHost string) error {
	trimmed := strings.TrimSpace(listenHost)
	if trimmed == "" || trimmed == "0.0.0.0" || trimmed == "::" || trimmed == "[::]" {
		return nil
	}

	// Bracket unbracketed IPv6 listenHost values so parseHostAndPort handles them uniformly.
	if addr, err := netip.ParseAddr(trimmed); err == nil && addr.Is6() {
		trimmed = "[" + addr.String() + "]"
	}

	parsed, err := parseHostAndPort(trimmed)
	if err != nil {
		return fmt.Errorf("invalid listen host %q: %w", listenHost, err)
	}
	if parsed.hasPort {
		return fmt.Errorf("listen host %q must not include a port", listenHost)
	}
	a.exactHosts[parsed.host] = struct{}{}
	return nil
}

func (a *HostAllowlist) addPattern(pattern string) error {
	if pattern == "*" {
		a.allowAll = true
		return nil
	}

	if strings.Contains(pattern, "*") {
		return a.addWildcardPattern(pattern)
	}

	parsed, err := parseHostAndPort(pattern)
	if err != nil {
		return fmt.Errorf("invalid allowed host pattern %q: %w", pattern, err)
	}
	if parsed.hasPort {
		a.exactHostPorts[hostPortKey{host: parsed.host, port: parsed.port}] = struct{}{}
		return nil
	}
	a.exactHosts[parsed.host] = struct{}{}
	return nil
}

func (a *HostAllowlist) addWildcardPattern(pattern string) error {
	if strings.Count(pattern, "*") != 1 {
		return fmt.Errorf("wildcard host pattern %q must contain exactly one '*'", pattern)
	}

	if strings.HasPrefix(pattern, "*.") {
		domain := strings.ToLower(pattern[2:])
		if !isValidDomain(domain) {
			return fmt.Errorf("invalid subdomain wildcard pattern %q", pattern)
		}
		a.subdomainWildcards = append(a.subdomainWildcards, "."+domain)
		return nil
	}

	if strings.HasPrefix(pattern, "*-") {
		suffix := strings.ToLower(pattern[1:])
		dotIdx := strings.IndexByte(suffix, '.')
		if dotIdx == -1 {
			return fmt.Errorf("invalid prefix wildcard pattern %q", pattern)
		}
		firstLabelTail := suffix[1:dotIdx]
		restDomain := suffix[dotIdx+1:]
		if !isValidDNSLabel("a-"+firstLabelTail) || !isValidDomain(restDomain) {
			return fmt.Errorf("invalid prefix wildcard pattern %q", pattern)
		}
		a.prefixWildcards = append(a.prefixWildcards, suffix)
		return nil
	}

	return fmt.Errorf("unsupported wildcard host pattern %q", pattern)
}

func (a *HostAllowlist) matchesWildcardDomain(host string) bool {
	for _, dotDomain := range a.subdomainWildcards {
		if strings.HasSuffix(host, dotDomain) {
			return true
		}
	}

	for _, suffix := range a.prefixWildcards {
		if prefix, ok := strings.CutSuffix(host, suffix); ok && !strings.Contains(prefix, ".") {
			return true
		}
	}

	return false
}

func parseHostAndPort(raw string) (parsedHost, error) {
	if strings.HasPrefix(raw, "[") {
		return parseBracketedIPv6(raw)
	}

	colonCount := strings.Count(raw, ":")
	if colonCount > 1 {
		return parsedHost{}, fmt.Errorf("unbracketed IPv6 or extra ':' in host %q", raw)
	}

	hostPart := raw
	var port uint16
	var hasPort bool
	if colonCount == 1 {
		var portPart string
		hostPart, portPart, _ = strings.Cut(raw, ":")
		parsedPort, err := parseTCPPort(portPart)
		if err != nil {
			return parsedHost{}, err
		}
		port = parsedPort
		hasPort = true
	}

	if addr, err := netip.ParseAddr(hostPart); err == nil {
		if !addr.Is4() || addr.IsUnspecified() {
			return parsedHost{}, fmt.Errorf("disallowed IP host %q", raw)
		}
		return parsedHost{
			host:    addr.String(),
			addr:    addr,
			isIP:    true,
			port:    port,
			hasPort: hasPort,
		}, nil
	}

	if !isValidDomain(hostPart) {
		return parsedHost{}, fmt.Errorf("invalid DNS host %q", raw)
	}

	return parsedHost{
		host:    strings.ToLower(hostPart),
		port:    port,
		hasPort: hasPort,
	}, nil
}

func parseBracketedIPv6(raw string) (parsedHost, error) {
	closeIdx := strings.IndexByte(raw, ']')
	if closeIdx == -1 {
		return parsedHost{}, fmt.Errorf("missing ']' in IPv6 host %q", raw)
	}

	ipStr := raw[1:closeIdx]
	rest := raw[closeIdx+1:]

	addr, err := netip.ParseAddr(ipStr)
	if err != nil || !addr.Is6() || addr.Is4In6() || addr.Zone() != "" || addr.IsUnspecified() {
		return parsedHost{}, fmt.Errorf("invalid IPv6 host %q", raw)
	}

	var port uint16
	var hasPort bool
	if rest != "" {
		if !strings.HasPrefix(rest, ":") {
			return parsedHost{}, fmt.Errorf("invalid port suffix in host %q", raw)
		}
		parsedPort, err := parseTCPPort(rest[1:])
		if err != nil {
			return parsedHost{}, err
		}
		port = parsedPort
		hasPort = true
	}

	return parsedHost{
		host:    addr.String(),
		addr:    addr,
		isIP:    true,
		port:    port,
		hasPort: hasPort,
	}, nil
}

func parseTCPPort(rawPort string) (uint16, error) {
	if rawPort == "" {
		return 0, fmt.Errorf("empty TCP port")
	}
	portNum, err := strconv.ParseUint(rawPort, 10, 64)
	if err != nil || portNum < 1 || portNum > 65535 {
		return 0, fmt.Errorf("invalid TCP port %q", rawPort)
	}
	return uint16(portNum), nil
}

func isValidDomain(domain string) bool {
	if len(domain) > 253 {
		return false
	}
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if !isValidDNSLabel(label) {
			return false
		}
	}
	return true
}

func isValidDNSLabel(label string) bool {
	if len(label) == 0 || len(label) > 63 {
		return false
	}
	for i := 0; i < len(label); i++ {
		c := label[i]
		isAlphaNum := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if isAlphaNum {
			continue
		}
		if c == '-' && i > 0 && i < len(label)-1 {
			continue
		}
		return false
	}
	return true
}
