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

package csmcp

import (
	"strings"
)

// PodIdentifier holds the namespace, name, and an optional connection ID of a Pod.
type PodIdentifier struct {
	Namespace    string
	Name         string
	ConnectionID string
}

// ParsePodIdentifier parses a word to extract Pod details (name, namespace, optional connection ID).
// It returns nil if the word does not match the expected format.
func ParsePodIdentifier(word string) *PodIdentifier {
	word = strings.TrimPrefix(word, "node:")

	if strings.Count(word, ".") != 1 {
		return nil
	}

	dotIdx := strings.IndexByte(word, '.')
	if dotIdx <= 0 || dotIdx == len(word)-1 {
		return nil
	}

	podName := word[:dotIdx]
	if !isValidPodName(podName) {
		return nil
	}

	rest := word[dotIdx+1:]
	namespace, connectionID := extractNamespaceAndConnectionID(rest)
	if !isValidNamespace(namespace) {
		return nil
	}

	return &PodIdentifier{
		Name:         podName,
		Namespace:    namespace,
		ConnectionID: connectionID,
	}
}

// IsXDSLog checks if the message starts with an xDS prefix (e.g., ADS:, CDS:).
func IsXDSLog(msg string) bool {
	idx := strings.IndexByte(msg, ':')
	if idx <= 0 || idx > 10 {
		return false
	}
	prefix := msg[:idx]
	for i := 0; i < len(prefix); i++ {
		c := prefix[i]
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

// IsConnectionLog checks if the message indicates an established xDS connection.
func IsConnectionLog(msg string) bool {
	return strings.Contains(msg, "new delta connection for") || strings.Contains(msg, "new connection for")
}

// IsDisconnectionLog checks if the message indicates a terminated xDS connection.
func IsDisconnectionLog(msg string) bool {
	return strings.Contains(msg, "terminated")
}

// ConnectionKey returns a unique key for identifying a pod's connection instance.
func (p PodIdentifier) ConnectionKey() string {
	return p.Namespace + "/" + p.Name + "/" + p.ConnectionID
}

// isValidPodName checks if the given string is a valid Kubernetes pod name component.
func isValidPodName(name string) bool {
	if len(name) == 0 {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return true
}

// isValidNamespace checks if the given string is a valid Kubernetes namespace component.
func isValidNamespace(namespace string) bool {
	if len(namespace) == 0 || !(namespace[0] >= 'a' && namespace[0] <= 'z') {
		return false
	}
	for i := 0; i < len(namespace); i++ {
		c := namespace[i]
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return true
}

// extractNamespaceAndConnectionID separates the namespace and connection ID from rest.
func extractNamespaceAndConnectionID(rest string) (string, string) {
	lastDashIdx := strings.LastIndexByte(rest, '-')
	if lastDashIdx > 0 && lastDashIdx < len(rest)-1 {
		possibleConnID := rest[lastDashIdx+1:]
		if isAllDigits(possibleConnID) {
			return rest[:lastDashIdx], possibleConnID
		}
	}
	return rest, ""
}

// isAllDigits checks if the given string consists solely of numeric digits.
func isAllDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
