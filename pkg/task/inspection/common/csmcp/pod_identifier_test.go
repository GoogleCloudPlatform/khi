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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParsePodIdentifier(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  *PodIdentifier
	}{
		{
			name:  "standard pod with namespace and connection id",
			input: "my-pod-1.default-123",
			want: &PodIdentifier{
				Name:         "my-pod-1",
				Namespace:    "default",
				ConnectionID: "123",
			},
		},
		{
			name:  "node prefix with hyphenated namespace and connection id",
			input: "node:auth-service-647d798687-abcde.backend-ns-61",
			want: &PodIdentifier{
				Name:         "auth-service-647d798687-abcde",
				Namespace:    "backend-ns",
				ConnectionID: "61",
			},
		},
		{
			name:  "push log without connection id",
			input: "node:payment-worker-7f5bcf84bb-fghij.payment-ns",
			want: &PodIdentifier{
				Name:         "payment-worker-7f5bcf84bb-fghij",
				Namespace:    "payment-ns",
				ConnectionID: "",
			},
		},
		{
			name:  "quoted word should return nil",
			input: `"node:payment-worker-7f5bcf84bb-fghij.payment-ns-84"`,
			want:  nil,
		},
		{
			name:  "ip address should return nil",
			input: `"192.0.2.1:41780"`,
			want:  nil,
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "no dot",
			input: "my-pod-without-namespace",
			want:  nil,
		},
		{
			name:  "multiple dots",
			input: "pod.name.too.many.dots",
			want:  nil,
		},
		{
			name:  "duration string should return nil",
			input: "100.200ms",
			want:  nil,
		},
		{
			name:  "invalid namespace with uppercase",
			input: "my-pod.Default-1",
			want:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParsePodIdentifier(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParsePodIdentifier(%q) mismatch (-want +got):\n%s", tc.input, diff)
			}
		})
	}
}

func TestIsXDSLog(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "ADS prefix", input: "ADS: new delta connection", want: true},
		{name: "CDS prefix", input: "CDS: PUSH request", want: true},
		{name: "EDS prefix", input: "EDS: PUSH request", want: true},
		{name: "LDS prefix", input: "LDS: PUSH request", want: true},
		{name: "RDS prefix", input: "RDS: PUSH request", want: true},
		{name: "SDS prefix", input: "SDS: PUSH request", want: true},
		{name: "NDS prefix", input: "NDS: PUSH request", want: true},
		{name: "WDS prefix", input: "WDS: PUSH request", want: true},
		{name: "non-XDS log", input: "Starting discovery service", want: false},
		{name: "lowercase prefix", input: "info: general log", want: false},
		{name: "non-uppercase prefix", input: "NONUPPER1: log", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsXDSLog(tc.input)
			if got != tc.want {
				t.Errorf("IsXDSLog(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestIsConnectionLog(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "delta connection", input: "ADS: new delta connection for node:p.ns-1", want: true},
		{name: "standard connection", input: "ADS: new connection for node:p.ns-1", want: true},
		{name: "terminated log", input: "ADS: p.ns-1 terminated", want: false},
		{name: "push log", input: "CDS: PUSH request", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsConnectionLog(tc.input)
			if got != tc.want {
				t.Errorf("IsConnectionLog(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestIsDisconnectionLog(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "terminated log with address", input: `ADS: "192.0.2.1:41780" p.ns-1 terminated`, want: true},
		{name: "connection log", input: "ADS: new delta connection for node:p.ns-1", want: false},
		{name: "push log", input: "CDS: PUSH request", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsDisconnectionLog(tc.input)
			if got != tc.want {
				t.Errorf("IsDisconnectionLog(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestPodIdentifier_ConnectionKey(t *testing.T) {
	testCases := []struct {
		name string
		pod  PodIdentifier
		want string
	}{
		{
			name: "standard pod identifier",
			pod: PodIdentifier{
				Namespace:    "default",
				Name:         "my-pod",
				ConnectionID: "42",
			},
			want: "default/my-pod/42",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.pod.ConnectionKey()
			if got != tc.want {
				t.Errorf("PodIdentifier.ConnectionKey(%+v) = %q, want %q", tc.pod, got, tc.want)
			}
		})
	}
}
