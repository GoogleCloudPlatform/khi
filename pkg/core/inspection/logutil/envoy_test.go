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

package logutil

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseEnvoyResponseFlags(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  EnvoyResponseFlags
	}{
		{
			name:  "known single flag",
			input: "UH",
			want:  EnvoyResponseFlags{EnvoyResponseFlagNoHealthyUpstream},
		},
		{
			name:  "no error flag (-)",
			input: "-",
			want:  EnvoyResponseFlags{EnvoyResponseFlagNoError},
		},
		{
			name:  "empty string",
			input: "",
			want:  EnvoyResponseFlags{},
		},
		{
			name:  "comma separated multiple known flags",
			input: "UH,URX",
			want:  EnvoyResponseFlags{EnvoyResponseFlagNoHealthyUpstream, EnvoyResponseFlagUpstreamRetryLimitExceeded},
		},
		{
			name:  "comma separated flags with unknown flag",
			input: "UH,UNKNOWN_FLAG,URX",
			want:  EnvoyResponseFlags{EnvoyResponseFlagNoHealthyUpstream, "UNKNOWN_FLAG", EnvoyResponseFlagUpstreamRetryLimitExceeded},
		},
		{
			name:  "comma separated flags with whitespace",
			input: " UF , NR ",
			want:  EnvoyResponseFlags{EnvoyResponseFlagUpstreamConnectionFailure, EnvoyResponseFlagNoRouteFound},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseEnvoyResponseFlags(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseEnvoyResponseFlags() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEnvoyResponseFlags_String(t *testing.T) {
	testCases := []struct {
		name  string
		input EnvoyResponseFlags
		want  string
	}{
		{
			name:  "empty flags",
			input: EnvoyResponseFlags{},
			want:  "",
		},
		{
			name:  "no error flag (-)",
			input: EnvoyResponseFlags{EnvoyResponseFlagNoError},
			want:  "-",
		},
		{
			name:  "single flag",
			input: EnvoyResponseFlags{EnvoyResponseFlagNoHealthyUpstream},
			want:  "UH",
		},
		{
			name:  "multiple flags",
			input: EnvoyResponseFlags{EnvoyResponseFlagNoHealthyUpstream, EnvoyResponseFlagUpstreamRetryLimitExceeded},
			want:  "UH,URX",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.input.String()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("EnvoyResponseFlags.String() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEnvoyResponseFlags_Summary(t *testing.T) {
	testCases := []struct {
		name  string
		input EnvoyResponseFlags
		want  string
	}{
		{
			name:  "empty flags",
			input: EnvoyResponseFlags{},
			want:  "",
		},
		{
			name:  "no error flag (-)",
			input: EnvoyResponseFlags{EnvoyResponseFlagNoError},
			want:  "OK",
		},
		{
			name:  "single known flag",
			input: EnvoyResponseFlags{EnvoyResponseFlagNoHealthyUpstream},
			want:  "No healthy upstream",
		},
		{
			name:  "multiple known flags",
			input: EnvoyResponseFlags{EnvoyResponseFlagNoHealthyUpstream, EnvoyResponseFlagUpstreamRetryLimitExceeded},
			want:  "No healthy upstream, Upstream retry limit exceeded",
		},
		{
			name:  "unknown flag",
			input: EnvoyResponseFlags{"UNKNOWN_FLAG"},
			want:  "UNKNOWN_FLAG",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.input.Summary()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("EnvoyResponseFlags.Summary() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEnvoyResponseFlags_HasError(t *testing.T) {
	testCases := []struct {
		name  string
		input EnvoyResponseFlags
		want  bool
	}{
		{
			name:  "empty flags",
			input: EnvoyResponseFlags{},
			want:  false,
		},
		{
			name:  "no error flag",
			input: EnvoyResponseFlags{EnvoyResponseFlagNoError},
			want:  false,
		},
		{
			name:  "cache filter response flag",
			input: EnvoyResponseFlags{EnvoyResponseFlagResponseFromCacheFilter},
			want:  false,
		},
		{
			name:  "delay injected response flag",
			input: EnvoyResponseFlags{EnvoyResponseFlagDelayInjected},
			want:  false,
		},
		{
			name:  "error response flag",
			input: EnvoyResponseFlags{EnvoyResponseFlagNoHealthyUpstream},
			want:  true,
		},
		{
			name:  "mixed non-error and error response flags",
			input: EnvoyResponseFlags{EnvoyResponseFlagResponseFromCacheFilter, EnvoyResponseFlagNoHealthyUpstream},
			want:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.input.HasError()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("EnvoyResponseFlags.HasError() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEnvoyResponseFlags_Contains(t *testing.T) {
	testCases := []struct {
		name       string
		input      EnvoyResponseFlags
		targetFlag EnvoyResponseFlag
		want       bool
	}{
		{
			name:       "empty flags",
			input:      EnvoyResponseFlags{},
			targetFlag: EnvoyResponseFlagNoHealthyUpstream,
			want:       false,
		},
		{
			name:       "flag present",
			input:      EnvoyResponseFlags{EnvoyResponseFlagNoHealthyUpstream, EnvoyResponseFlagUpstreamRetryLimitExceeded},
			targetFlag: EnvoyResponseFlagNoHealthyUpstream,
			want:       true,
		},
		{
			name:       "flag absent",
			input:      EnvoyResponseFlags{EnvoyResponseFlagNoHealthyUpstream},
			targetFlag: EnvoyResponseFlagNoRouteFound,
			want:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.input.Contains(tc.targetFlag)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("EnvoyResponseFlags.Contains() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatEnvoySummary(t *testing.T) {
	testCases := []struct {
		name          string
		statusCode    int
		method        string
		requestURL    string
		responseFlags EnvoyResponseFlags
		want          string
	}{
		{
			name:          "standard summary without response flags",
			statusCode:    200,
			method:        "GET",
			requestURL:    "http://example.com/api/v1/users",
			responseFlags: EnvoyResponseFlags{EnvoyResponseFlagNoError},
			want:          "200 GET http://example.com/api/v1/users",
		},
		{
			name:          "summary with empty response flags",
			statusCode:    200,
			method:        "GET",
			requestURL:    "http://example.com/api/v1/users",
			responseFlags: EnvoyResponseFlags{},
			want:          "200 GET http://example.com/api/v1/users",
		},
		{
			name:          "summary with single response flag",
			statusCode:    503,
			method:        "GET",
			requestURL:    "http://10.4.0.5/",
			responseFlags: EnvoyResponseFlags{EnvoyResponseFlagUpstreamConnectionFailure},
			want:          "【Upstream connection failure(UF)】503 GET http://10.4.0.5/",
		},
		{
			name:          "summary with multiple response flags",
			statusCode:    503,
			method:        "GET",
			requestURL:    "/productpage",
			responseFlags: EnvoyResponseFlags{EnvoyResponseFlagNoHealthyUpstream, EnvoyResponseFlagUpstreamRetryLimitExceeded},
			want:          "【No healthy upstream, Upstream retry limit exceeded(UH,URX)】503 GET /productpage",
		},
		{
			name:          "summary with unknown response flag",
			statusCode:    500,
			method:        "POST",
			requestURL:    "/submit",
			responseFlags: EnvoyResponseFlags{"CUSTOM_FLAG"},
			want:          "【CUSTOM_FLAG(CUSTOM_FLAG)】500 POST /submit",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatEnvoySummary(tc.statusCode, tc.method, tc.requestURL, tc.responseFlags)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("FormatEnvoySummary() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
